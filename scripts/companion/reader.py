#!/usr/bin/env python3
"""Mac native companion bridge. Local OCR + API receipts + reusable owned file bus."""
import argparse
import fcntl
import json
import pathlib
import queue
import re
import signal
import subprocess
import sys
import threading
import time
import urllib.request
from protocol import parse,parse_observation,normalize_ocr_lines,qr_lines,reply,frame
from invite import InvitePads
sys.path.insert(0,str(pathlib.Path(__file__).resolve().parents[1]/'experiments/return_bus'))
from bus import Carrier
from block import receipt,validate
sys.path.insert(0,str(pathlib.Path(__file__).resolve().parents[1]/'experiments/ocr'))
from bridge import api


class ChatCarrier(Carrier):
    def __init__(self,root):
        super().__init__(root,width=64)
        self.thinking=self.root/'Interface/AddOns/WispOCRInbox065/WispOCRInbox065_Camelot.toc'
        self.heartbeat=self.root/'Interface/AddOns/WispOCRInbox066/WispOCRInbox066_Camelot.toc'
        for p in [self.thinking,self.heartbeat]:
            if p.is_symlink() or p.parent.is_symlink() or self.hold(p).exists() or not p.is_file():raise ValueError('Status carrier requires recovery')
            self.paths.append(p)
            self.original[p]=__import__('hashlib').sha256(p.read_bytes()).hexdigest()


def transfer(carrier,packet,payload,samples,stop):
    wire=frame(payload,packet['nonce'],packet['binding']);index=1
    carrier.phase=True
    carrier.publish(int.from_bytes(wire[:8],'little'))
    began=time.monotonic();deadline=began+12
    while time.monotonic()<deadline:
        if stop.is_set():raise InterruptedError('Reader stopped')
        try:sample=samples.get(timeout=.3)
        except queue.Empty:continue
        got=receipt(qr_lines(sample) or []) if 'qr' in sample else receipt(normalize_ocr_lines(sample.get('lines',[])))
        if not got:continue
        action=validate(got,packet['nonce'],packet['binding'],index,wire,payload,'')
        if action=='ignore':continue
        if action=='done':return round((time.monotonic()-began)*1000,1),index
        index+=1
        carrier.publish(int.from_bytes(wire[(index-1)*8:index*8],'little'));deadline=time.monotonic()+12
    raise TimeoutError('Reply receipt not confirmed; no automatic resend')


def complete(session,carrier,packet,samples,ocr_ms,stop,pads):
    started=time.monotonic();status='not_confirmed';result={};failure=None
    try:
        if packet.get('activation'):
            token=pads.decrypt(packet)
            try:accepted=api(session,'/local/access',dict(mutation_id='native-connect-'+packet['id'],token=token))
            finally:token=None
            carrier.present(carrier.thinking,False)
            if accepted['status']!='completed':raise RuntimeError('Invite was not verified')
            result=dict(text='AI connected. You can now ask Jornia.',route='connection')
        else:
            if packet.get('registration'):
                registered=api(session,'/local/native-character',dict(mutation_id='native-register-'+packet['id'],**packet['registration']))
                identity=registered.get('result') or {}
                if registered['status']!='completed' or not identity.get('character_id') or identity.get('excluded'):raise RuntimeError('Character registration not confirmed')
                packet['character_id']=identity['character_id']
                session['identities'][packet['guid']]=packet['character_id']
            payload=dict(mutation_id='native-'+packet['id'],character_id=packet['character_id'],question=packet['question'],ai=True)
            if packet.get('snapshot') is not None:payload.update(snapshot=packet['snapshot'],snapshot_approved=True)
            if stop.is_set():raise InterruptedError('Reader stopped')
            accepted=api(session,'/local/ask',payload)
            carrier.present(carrier.thinking,False)
            while accepted['status']=='pending' and time.monotonic()-started<45 and not stop.is_set():
                time.sleep(.3);accepted=api(session,'/local/operation?id=native-'+packet['id'])
            if accepted['status']!='completed':raise RuntimeError('No completed local receipt')
            result=accepted['result']
    except Exception as error:
        failure='Invite could not be verified. Check the code and Jornia desktop connection.' if packet.get('activation') else 'No confirmed reply. Check Jornia desktop; your draft is kept. No automatic resend.'
        print(json.dumps(dict(status='api_not_confirmed',error_type=type(error).__name__,request_id=packet['id'])),flush=True)
    try:
        encoded=reply(packet,result,failure)
        ms,blocks=transfer(carrier,packet,encoded,samples,stop)
        status='delivered'
        print(json.dumps(dict(status=status,request_id=packet['id'],ocr_ms=ocr_ms,return_ms=ms,blocks=blocks,elapsed_ms=round((time.monotonic()-started)*1000,1),route=result.get('route'))),flush=True)
    except Exception as error:
        print(json.dumps(dict(status='delivery_not_confirmed',request_id=packet['id'],error_type=type(error).__name__)),flush=True)
    finally:
        # Keep the independent heartbeat live; restore only the data/status lines.
        for path in carrier.data+[carrier.clock,carrier.thinking]:carrier.present(path,True)
        carrier.phase=True


def recover_owned(root):
    """Recover fixed owned carriers after interruption; never overwrite ambiguous files."""
    from bus import restore
    restore(root,width=64)
    addons=root.resolve(strict=True)/'Interface/AddOns'
    for n in (65,66):
        folder=addons/f'WispOCRInbox{n:03d}'
        p=folder/(folder.name+'_Camelot.toc');held=Carrier.hold(p)
        if folder.is_symlink() or p.is_symlink() or held.is_symlink():raise ValueError('Recovery symlink rejected')
        if held.exists():
            if p.exists():raise ValueError('Ambiguous status carrier; originals preserved')
            if b'## Dependencies: WispOCR' not in held.read_bytes():raise ValueError('Unexpected carrier original')
            held.rename(p)
    for n in range(67,81):
        folder=addons/f'WispOCRInbox{n:03d}'
        p=folder/'Inbox.lua';held=folder/'Inbox.lua.wispconnect-hold'
        if folder.is_symlink() or p.is_symlink() or held.is_symlink():raise ValueError('Recovery symlink rejected')
        if not held.exists():continue
        if not p.is_file() or not re.fullmatch(r'WispConnectPads=WispConnectPads or \{\};WispConnectPads\[\d+\]=\{id="[a-f0-9]{16}",hex="[a-f0-9]{64}"\}\n',p.read_text()):
            raise ValueError('Invite file changed; originals preserved')
        held.replace(p)
    print('Owned transport files restored. No journal or player data changed.',flush=True)


def run(a):
    if a.session.stat().st_mode&0o077:raise ValueError('Session must be owner-only')
    session=json.loads(a.session.read_text())
    if not re.fullmatch(r'http://127\.0\.0\.1:\d+',session['base_url']):raise ValueError('Expected loopback session')
    lock=open(a.session.with_name('native-reader.lock'),'a')
    fcntl.flock(lock,fcntl.LOCK_EX|fcntl.LOCK_NB)
    if a.recover:
        try:recover_owned(a.root)
        finally:lock.close()
        return
    carrier=ChatCarrier(a.root);pads=InvitePads(a.root);stop=threading.Event();process=None;worker=None;last_capture=0;announced=False
    def beat():
        phase=True
        while not stop.wait(1):
            phase=not phase
            try:carrier.present(carrier.heartbeat,phase if time.monotonic()-last_capture<3 else True)
            except Exception:stop.set();return
    heartbeat=threading.Thread(target=beat);heartbeat.start()
    seen=set();last=None;count=0;samples=queue.Queue(maxsize=32)
    try:
        process=subprocess.Popen([str(a.capture)]+(['--qr'] if a.qr else ['--card','--fast']),stdout=subprocess.PIPE,text=True)
        for line in process.stdout:
            sample=json.loads(line)
            if sample.get('error'):raise RuntimeError(sample['error'])
            if sample.get('ready'):print('Local OCR helper started. Waiting for a game frame.',flush=True);continue
            if sample.get('capture_tick'):
                last_capture=time.monotonic()
                if not announced:print('Capture active. Native chat is ready; only submitted questions are sent.',flush=True);announced=True
                continue
            if worker and worker.is_alive():
                if samples.full():
                    try:samples.get_nowait()
                    except queue.Empty:pass
                samples.put_nowait(sample);continue
            packet=parse_observation(sample,session['identities'])
            if packet is None:last=None;count=0;continue
            signature=json.dumps(packet,sort_keys=True)
            count=count+1 if signature==last else 1;last=signature
            if count<2 or packet['id'] in seen:continue
            seen.add(packet['id'])
            if a.qr:
                snap=packet.get('snapshot') or {}
                print(json.dumps(dict(status='qr_accepted',request_id=packet['id'],decode_ms=round(sample.get('ocr_ms',0),1),crop_width=sample.get('crop_width'),crop_height=sample.get('crop_height'),has_context=bool(snap),quests=len(snap.get('quests',[])))),flush=True)
            while not samples.empty():samples.get_nowait()
            worker=threading.Thread(target=complete,args=(session,carrier,packet,samples,round(sample.get('ocr_ms',0),1),stop,pads));worker.start()
    finally:
        stop.set();heartbeat.join(timeout=3)
        if worker:
            worker.join(timeout=15)
            if worker.is_alive():raise RuntimeError('Worker still active; preserve carrier files for explicit recovery')
        if process:process.terminate();process.wait(timeout=5)
        carrier.restore();pads.restore();lock.close()
        print('Native reader stopped; owned carrier files restored.',flush=True)


if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('--qr',action='store_true',help='Decode the small in-chat QR tile instead of the laboratory OCR card');p.add_argument('--root',type=pathlib.Path,required=True);p.add_argument('--recover',action='store_true',help='Recover only owned interrupted carrier files, then exit');p.add_argument('--session',type=pathlib.Path,default=next((x for x in [pathlib.Path.home()/'Library/Application Support/Jornia/ocr-session.json',pathlib.Path.home()/'Library/Application Support/Wisp/ocr-session.json'] if x.exists()),pathlib.Path.home()/'Library/Application Support/Jornia/ocr-session.json'));p.add_argument('--capture',type=pathlib.Path,default=pathlib.Path('.cache/ocr-test/wisp-companion-capture').resolve());a=p.parse_args()
    def interrupted(*_):raise KeyboardInterrupt
    signal.signal(signal.SIGTERM,interrupted)
    try:run(a)
    except KeyboardInterrupt:pass
