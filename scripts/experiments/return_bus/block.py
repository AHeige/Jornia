#!/usr/bin/env python3
"""Wisp's eight-byte parallel return experiment. Stdlib, no network/AI calls."""
import argparse
import json
import pathlib
import queue
import re
import signal
import struct
import subprocess
import threading
import time
import zlib
from bus import Carrier,check,restore

BLOCK=8
MAX_PAYLOAD=1024


def packet(payload,nonce,binding):
    if len(payload)>MAX_PAYLOAD:raise ValueError('Payload exceeds 1024-byte lab limit')
    body=b'WB\x01'+struct.pack('>IHH',int(nonce),int(binding),len(payload))+payload
    wire=body+struct.pack('>I',zlib.adler32(body))
    return wire+bytes((-len(wire))%BLOCK)


def receipt(lines):
    text=' '.join(' '.join(lines).split())
    # Vision sometimes drops spaces between digits; fixed widths preserve fields.
    # Rebuild canonical spaces before checking the receipt checksum.
    m=re.search(r'\b(WBREADY|WBACK|WBDONE|WBFAIL)\s*(\d{8})\s*(\d{5})\s*(\d{3})\s*(\d{4})\s*(\d{10})\s*(\d{5})\b',text)
    if not m:return None
    f=m.groups()
    if check(' '.join(f[:-1]))!=int(f[-1]):return None
    return dict(kind=f[0],nonce=f[1],binding=f[2],seq=int(f[3]),value=int(f[4]),digest=int(f[5]))


def validate(got,nonce,binding,index,wire,payload,fault):
    """Classify a correlated receipt; never advance twice on a stale OCR image."""
    if got['binding']!=binding or got['nonce']!=nonce:return 'ignore'
    if got['seq']<index:return 'ignore'
    if got['seq']!=index:raise RuntimeError('Unexpected block sequence')
    count=len(wire)//BLOCK
    if got['kind']=='WBFAIL':
        if fault=='stale' and index==1 and got['value']==6:return 'rejected'
        if fault in ('crc','padding') and index==count and got['value']==3:return 'rejected'
        raise RuntimeError('Receiver rejected the transfer')
    if got['kind']=='WBDONE':
        crc=zlib.adler32(wire[:11+len(payload)])
        if fault or index!=count or got['value']!=len(payload) or got['digest']!=crc:raise RuntimeError('Unexpected final receipt')
        return 'done'
    if got['kind']!='WBACK':return 'ignore'
    if index>=count or got['value']!=index*BLOCK or got['digest']!=zlib.adler32(wire[:index*BLOCK]):
        raise RuntimeError('Block prefix receipt mismatch')
    return 'advance'


def run(args):
    payload=args.text.encode('utf8')
    if len(payload)>MAX_PAYLOAD:raise ValueError('Payload too long')
    carrier=Carrier(args.root,width=64)
    process=None;nonce=None;wire=None;index=0;started=None;deadline=time.monotonic()+60;ocr=[]
    try:
        process=subprocess.Popen([str(args.capture)],stdout=subprocess.PIPE,text=True)
        incoming=queue.Queue()
        def reader():
            for line in process.stdout:incoming.put(line)
            incoming.put(None)
        threading.Thread(target=reader,daemon=True).start()
        while True:
            if time.monotonic()>deadline:raise TimeoutError('Checked receipt timed out; no automatic resend')
            try:line=incoming.get(timeout=.5)
            except queue.Empty:continue
            if line is None:raise RuntimeError('Capture helper stopped')
            sample=json.loads(line)
            if sample.get('error'):raise RuntimeError(sample['error'])
            got=receipt(sample.get('lines',[]))
            if not got or got['binding']!=args.binding:continue
            if nonce is None:
                if got['kind']!='WBREADY' or got['seq']!=0 or got['value']!=BLOCK or got['digest']!=0:continue
                nonce=got['nonce']
                encoded_nonce=int(nonce) if args.fault!='stale' else int(nonce)^1
                wire=packet(payload,encoded_nonce,args.binding)
                if args.fault=='crc':
                    at=11+len(payload)+3;wire=wire[:at]+bytes([wire[at]^1])+wire[at+1:]
                if args.fault=='padding':
                    if len(wire)==11+len(payload)+4:raise ValueError('This payload has no padding to corrupt')
                    wire=wire[:-1]+b'\x01'
                started=time.monotonic();index=1
                carrier.publish(int.from_bytes(wire[:BLOCK],'little'));deadline=time.monotonic()+15
                print(json.dumps({'status':'sending_blocks','nonce':nonce,'payload_bytes':len(payload),'wire_bytes':len(wire),'blocks':len(wire)//BLOCK}),flush=True)
                continue
            if got['nonce']!=nonce:raise RuntimeError('Receiver transfer changed')
            action=validate(got,nonce,args.binding,index,wire,payload,args.fault)
            if action=='ignore':continue
            ocr.append(round(sample.get('ocr_ms',0),1))
            if action in ('done','rejected'):
                duration=time.monotonic()-started
                print(json.dumps({'status':'delivered' if action=='done' else 'fault_rejected','fault':args.fault,'nonce':nonce,
                    'payload_bytes':len(payload),'wire_bytes':len(wire),'blocks':index,'elapsed_ms':round(duration*1000,1),
                    'payload_bytes_per_second':round(len(payload)/duration,2),'ocr_ms':ocr}),flush=True)
                return
            index+=1
            carrier.publish(int.from_bytes(wire[(index-1)*BLOCK:index*BLOCK],'little'));deadline=time.monotonic()+15
    finally:
        try:carrier.restore()
        finally:
            if process:process.terminate();process.wait(timeout=5)
        print(json.dumps({'status':'carrier_files_restored','count':65}),flush=True)


if __name__=='__main__':
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('action',choices=['run','restore'])
    p.add_argument('--root',type=pathlib.Path,required=True)
    p.add_argument('--binding')
    p.add_argument('--text',default='Hi')
    p.add_argument('--fault',choices=['crc','stale','padding'])
    p.add_argument('--capture',type=pathlib.Path,default=pathlib.Path('.cache/ocr-test/wisp-ocr-capture').resolve())
    a=p.parse_args()
    if a.action=='run' and not re.fullmatch(r'\d{5}',a.binding or ''):p.error('run requires the five-digit binding shown by Lua')
    def interrupt(*_):raise KeyboardInterrupt
    signal.signal(signal.SIGTERM,interrupt)
    if a.action=='restore':restore(a.root,width=64)
    else:run(a)
