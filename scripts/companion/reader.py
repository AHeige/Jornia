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
from protocol import parse,reply,frame
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
        got=receipt(sample.get('lines',[]))
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
            result=dict(text='AI connected. You can now ask Wisp.',route='connection')
        else:
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
        failure='Invite could not be verified. Check the code and Wisp desktop connection.' if packet.get('activation') else 'No confirmed reply. Check Wisp desktop; your draft is kept. No automatic resend.'
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


def run(a):
    if a.session.stat().st_mode&0o077:raise ValueError('Session must be owner-only')
    session=json.loads(a.session.read_text())
    if not re.fullmatch(r'http://127\.0\.0\.1:\d+',session['base_url']):raise ValueError('Expected loopback session')
    lock=open(a.session.with_name('native-reader.lock'),'a')
    fcntl.flock(lock,fcntl.LOCK_EX|fcntl.LOCK_NB)
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
        process=subprocess.Popen([str(a.capture),'--card'],stdout=subprocess.PIPE,text=True)
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
            packet=parse(sample.get('lines',[]),session['identities'])
            if packet is None:last=None;count=0;continue
            signature=json.dumps(packet,sort_keys=True)
            count=count+1 if signature==last else 1;last=signature
            if count<2 or packet['id'] in seen:continue
            seen.add(packet['id'])
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
    p=argparse.ArgumentParser();p.add_argument('--root',type=pathlib.Path,required=True);p.add_argument('--session',type=pathlib.Path,default=pathlib.Path.home()/'Library/Application Support/Wisp/ocr-session.json');p.add_argument('--capture',type=pathlib.Path,default=pathlib.Path('.cache/ocr-test/wisp-companion-capture').resolve());a=p.parse_args()
    def interrupted(*_):raise KeyboardInterrupt
    signal.signal(signal.SIGTERM,interrupted)
    try:run(a)
    except KeyboardInterrupt:pass
