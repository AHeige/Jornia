#!/usr/bin/env python3
"""First-party byte-wide presence bus; native Vision supplies local OCR receipts.
No network/AI calls. Only own lab assets are renamed, then restored exactly.
"""
import argparse
import hashlib
import json
import pathlib
import queue
import threading
import re
import signal
import struct
import subprocess
import time
import zlib


def check(text):
    return sum(i*b for i,b in enumerate(text.encode('ascii'),1)) % 65521


def packet(payload):
    if len(payload)>96: raise ValueError('Lab payload limit is 96 bytes')
    body=b'WP'+bytes([len(payload)])+payload
    return body+struct.pack('>I',zlib.adler32(body))


def receipt(lines):
    text=' '.join(' '.join(lines).split())
    match=re.search(r'\b(WISPREADY|WISPACK|WISPDONE|WISPFAIL) (\d{8}) (\d{5}) (\d{3}) (\d{3}) (\d{10}) (\d{5})\b',text)
    if not match: return None
    fields=match.groups(); raw=' '.join(fields[:-1])
    if check(raw)!=int(fields[-1]): return None
    return dict(kind=fields[0],nonce=fields[1],binding=fields[2],seq=int(fields[3]),value=int(fields[4]),digest=int(fields[5]))


class Carrier:
    def __init__(self,root,width=8):
        if width not in (8,64): raise ValueError("Only the two fixed lab carrier banks are allowed")
        self.width=width
        self.root=root.resolve(strict=True)
        if not (self.root/'WTF').is_dir(): raise ValueError('Expected WoW client root')
        addons=self.root/'Interface/AddOns'
        self.data=[]
        for n in (range(81,89) if width==8 else range(1,65)):
            name=f'WispOCRInbox{n:03d}'
            self.data.append(addons/name/(name+'_Camelot.toc'))
        self.clock=addons/'WispBridgeLab/wispreturn_pulse.wav'
        self.paths=self.data+[self.clock]
        self.original={}
        for p in self.paths:
            if p.parent.is_symlink() or p.is_symlink() or not p.is_file() or self.hold(p).exists():
                raise ValueError('Carrier not in clean, recoverable state; use restore after inspection')
            raw=p.read_bytes()
            if p==self.clock and raw!=b'': raise ValueError('Clock must be our zero-byte probe asset')
            if p!=self.clock and (b'## LoadOnDemand: 1' not in raw or b'## Dependencies: WispOCR' not in raw):
                raise ValueError('Only own lab TOCs are permitted')
            self.original[p]=hashlib.sha256(raw).hexdigest()
        self.phase=True

    @staticmethod
    def hold(p): return p.with_name(p.name+'.wispbus-hold')

    def present(self,p,want):
        held=self.hold(p)
        if want:
            if held.exists():
                if p.exists(): raise ValueError('Both carrier and held original exist; preserving both')
                held.rename(p)
        elif p.exists():
            if held.exists(): raise ValueError('Held original exists; no overwrite')
            p.rename(held)

    def publish(self,value):
        if not 0<=value<(1<<self.width): raise ValueError('Value exceeds carrier width')
        for i,p in enumerate(self.data): self.present(p,bool(value & (1<<i)))
        # Clock changes LAST. Data is frozen until a matching checked OCR receipt.
        self.phase=not self.phase
        self.present(self.clock,self.phase)

    def restore(self):
        for p in self.paths: self.present(p,True)
        for p,digest in self.original.items():
            if hashlib.sha256(p.read_bytes()).hexdigest()!=digest: raise ValueError('Restored hash mismatch')


def restore(root,width=8):
    # Recovery also after SIGKILL. Fixed own paths only, never paths from a manifest.
    if width not in (8,64): raise ValueError('Unknown bank width')
    addons=root.resolve(strict=True)/'Interface/AddOns'
    indexes=range(81,89) if width==8 else range(1,65)
    paths=[addons/f'WispOCRInbox{n:03d}'/f'WispOCRInbox{n:03d}_Camelot.toc' for n in indexes]
    paths.append(addons/'WispBridgeLab/wispreturn_pulse.wav')
    for p in paths:
        held=Carrier.hold(p)
        if p.parent.is_symlink() or p.is_symlink() or held.is_symlink(): raise ValueError('Unexpected symlink')
        if held.exists():
            if p.exists(): raise ValueError('Both files exist; no overwrite')
            held.rename(p)


def run(args):
    payload=args.text.encode('utf8'); wire=packet(payload)
    if args.corrupt: wire=wire[:-1]+bytes([wire[-1]^1])
    carrier=Carrier(args.root)
    process=None;start=None;nonce=None;seq=0;last_byte=None;next_deadline=time.monotonic()+60
    samples=[]
    try:
        process=subprocess.Popen([str(args.capture)],stdout=subprocess.PIPE,text=True)
        incoming=queue.Queue()
        def read():
            for line in process.stdout: incoming.put(line)
            incoming.put(None)
        threading.Thread(target=read,daemon=True).start()
        while True:
            if time.monotonic()>next_deadline: raise TimeoutError('Receipt timeout; no automatic resend')
            try: line=incoming.get(timeout=.5)
            except queue.Empty: continue
            if line is None: break
            result=json.loads(line)
            if result.get('error'): raise RuntimeError(result['error'])
            got=receipt(result.get('lines',[]))
            if not got or got['binding']!=args.binding: continue
            if nonce is None:
                if got['kind']!='WISPREADY' or got['seq']!=0 or got['value']!=0 or got['digest']!=0:continue
                nonce=got['nonce'];start=time.monotonic();seq=1;last_byte=wire[0]
                carrier.publish(last_byte);next_deadline=time.monotonic()+15
                print(json.dumps({'status':'sending','nonce':nonce,'wire_bytes':len(wire)}),flush=True)
                continue
            if got['nonce']!=nonce: raise RuntimeError('Receiver session changed')
            if got['seq']<seq:continue
            if got['seq']!=seq:raise RuntimeError('Unexpected receipt sequence')
            elapsed=(time.monotonic()-start)*1000
            if got['kind']=='WISPFAIL':
                if args.corrupt and seq==len(wire) and got['value']==3:
                    print(json.dumps({'status':'corruption_rejected','wire_bytes':seq,'elapsed_ms':round(elapsed,1)}),flush=True);return
                raise RuntimeError('Receiver rejected frame')
            if got['kind']=='WISPDONE':
                expected=struct.unpack('>I',wire[-4:])[0]
                if args.corrupt or seq!=len(wire) or got['value']!=len(payload) or got['digest']!=expected:
                    raise RuntimeError('Unexpected final receipt')
                print(json.dumps({'status':'delivered','wire_bytes':seq,'payload_bytes':len(payload),'elapsed_ms':round(elapsed,1),
                    'payload_bytes_per_second':round(len(payload)/(elapsed/1000),2),'ocr_ms':samples,'nonce':nonce}),flush=True);return
            if got['kind']!='WISPACK':continue
            if got['value']!=last_byte or got['digest']!=zlib.adler32(wire[:seq]): raise RuntimeError('Byte receipt mismatch')
            samples.append(round(result.get('ocr_ms',0),1))
            seq+=1
            if seq>len(wire):raise RuntimeError('Missing final receipt')
            last_byte=wire[seq-1];carrier.publish(last_byte);next_deadline=time.monotonic()+15
        raise RuntimeError('Capture helper ended')
    finally:
        try: carrier.restore()
        finally:
            if process:
                process.terminate();process.wait(timeout=5)
        print(json.dumps({'status':'carrier_files_restored'}),flush=True)


if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action',choices=['run','restore'])
    parser.add_argument('--root',type=pathlib.Path,required=True)
    parser.add_argument('--binding',help='five-digit binding shown in the Lua READY receipt')
    parser.add_argument('--text',default='Hi')
    parser.add_argument('--corrupt',action='store_true')
    parser.add_argument('--capture',type=pathlib.Path,default=pathlib.Path('.cache/ocr-test/wisp-ocr-capture').resolve())
    args=parser.parse_args()
    if args.action=='run' and not re.fullmatch(r'\d{5}',args.binding or ''): parser.error('run requires a five-digit binding')
    def interrupt(_signum,_frame):raise KeyboardInterrupt
    signal.signal(signal.SIGTERM,interrupt)
    if args.action=='restore':restore(args.root)
    else:run(args)
