#!/usr/bin/env python3
"""Own finite-session OCR bridge; stdlib only. No screenshots or question text in logs."""
import argparse
import json
import pathlib
import re
import secrets
import shutil
import subprocess
import threading
import time
import urllib.request


def checksum(text):
    return sum(i * b for i, b in enumerate(text.encode('ascii'), 1)) % 65521


def parse(lines, nonce, identities):
    # Never repair identifiers, infer missing words, or trust OCR confidence alone.
    text = '\n'.join(lines)
    match = re.search(r'^WISPBEGIN\s+(\d{24})\s+(\d{5})\s+(\d{5})\n(.+?)\nWISPEND$', text, re.M | re.S)
    if not match:
        return None
    ident, binding, check, question = match.groups()
    question = ' '.join(question.split())
    if not ident.startswith(nonce) or not 1 <= len(question) <= 160 or '|' in question:
        return None
    try:
        if checksum(ident + binding + question) != int(check):
            return None
        matches = [guid for guid in identities if re.fullmatch(r'Player-[A-Za-z0-9-]+', guid) and checksum(guid) == int(binding)]
    except UnicodeEncodeError:
        return None
    if len(matches) != 1:
        return None
    guid = matches[0]
    return {'id': ident, 'guid': guid, 'character_id': identities[guid], 'question': question}


def atomic(path, text):
    temp = path.with_suffix('.tmp')
    with open(temp, 'w', encoding='utf8') as f:
        f.write(text)
    temp.replace(path)


def publish(addons, packet, status, text):
    # Only hex data enters executable Lua. Bind replies to both request and GUID.
    encoded = text.encode('utf8')[:16000].decode('utf8', errors='ignore').encode().hex()
    body = f'WispOCRReceive("{packet["id"]}","{packet["guid"]}","{status}","{encoded}")\n'
    for n in range(1, 97):
        atomic(addons / f'WispOCRInbox{n:03d}' / 'Inbox.lua', body)


def api(session, path, data=None):
    body = None if data is None else json.dumps(data).encode()
    req = urllib.request.Request(session['base_url'] + path, data=body, headers={
        'X-Wisp-Token': session['token'], 'Origin': session['base_url'], 'Content-Type': 'application/json'})
    with urllib.request.urlopen(req, timeout=6) as response:
        return json.load(response)


def complete(session, addons, packet, evidence):
    started = time.monotonic()
    try:
        # Stable operation ID survives reader restart: existing receipts prevent paid replay.
        publish(addons, packet, 'sending', '')
        receipt = api(session, '/local/ask', {
            'mutation_id': 'ocr-' + packet['id'], 'character_id': packet['character_id'],
            'question': packet['question'], 'ai': True})
        publish(addons, packet, 'thinking', '')
        while receipt['status'] == 'pending' and time.monotonic() - started < 48:
            time.sleep(.3)
            receipt = api(session, '/local/operation?id=ocr-' + packet['id'])
        if receipt['status'] != 'completed':
            raise RuntimeError(receipt.get('error') or 'No answer confirmed. No automatic resend.')
        result = receipt['result']
        text = result.get('text', '')
        warnings = result.get('warnings') or []
        sources = result.get('sources') or []
        if warnings:
            text += '\n\n' + '\n'.join(warnings)
        if sources:
            text += '\n\nSources: ' + '; '.join(str(s.get('title') or s.get('url') or s.get('id') or 'World knowledge') for s in sources)
        text += '\n\nLab timing: OCR %.0f ms; API %.1f s. Images stayed local.' % (evidence['ocr_ms'], time.monotonic()-started)
        publish(addons, packet, 'done', text)
        evidence.update(status='reply_files_written', route=result.get('route'), model=result.get('model'), backend_ms=result.get('latency_ms'))
    except Exception as error:
        # Generic transport message; do not leak tokens/URLs or pretend a failed request was unsent.
        publish(addons, packet, 'error', 'No confirmed reply. Check the Wisp desktop chat for this question. It will not be sent again automatically.')
        evidence.update(status='error', error_type=type(error).__name__)
    evidence['api_and_reply_ms'] = round((time.monotonic() - started) * 1000, 1)
    print(json.dumps(evidence), flush=True)


def install(root):
    if subprocess.run(['pgrep', '-x', 'World of Warcraft'], stdout=subprocess.DEVNULL).returncode == 0:
        raise SystemExit('Game is running. No reload/restart is permitted by this installer.')
    if not (root / 'WTF').is_dir():
        raise SystemExit('Expected client directory with WTF')
    addons = root / 'Interface/AddOns'
    if (addons / 'WispOCR').exists():
        raise SystemExit('Existing experiment preserved; refusing to reset slots/session.')
    nonce = ''.join(secrets.choice('23456789') for _ in range(8))
    source = pathlib.Path(__file__).resolve().parents[3] / 'apps/ocr-lab/WispOCR'
    shutil.copytree(source, addons / 'WispOCR')
    atomic(addons / 'WispOCR/Config.lua', f'WispOCRSession = "{nonce}"\n')
    for n in range(1, 97):
        name = f'WispOCRInbox{n:03d}'
        folder = addons / name
        folder.mkdir()
        (folder / f'{name}_Camelot.toc').write_text(f'## Interface: 16001\n## Title: Wisp OCR inbox {n}\n## LoadOnDemand: 1\n## Dependencies: WispOCR\nInbox.lua\n')
        (folder / 'Inbox.lua').write_text('-- Waiting. This one-shot slot contains no reply.\n')
    print('Installed Wisp OCR Lab and 96 finite inbox slots. No other addons changed.')


def run(args):
    path = args.session
    if path.stat().st_mode & 0o077:
        raise SystemExit('Session must be owner-only (0600).')
    session = json.loads(path.read_text())
    # Never permit session configuration to turn this into a remote token sender.
    if not re.fullmatch(r'http://127\.0\.0\.1:\d+', session['base_url']):
        raise SystemExit('Expected numeric loopback session')
    addons = args.root / 'Interface/AddOns'
    nonce = re.fullmatch(r'WispOCRSession = "(\d{8})"\s*', (addons / 'WispOCR/Config.lua').read_text()).group(1)
    identities = session['identities']
    if not identities:
        raise SystemExit('No verified local character identities. Import a character before this lab.')
    seen, last, count, worker = set(), None, 0, None
    last_rejection = 0
    process = subprocess.Popen([str(args.capture)], stdout=subprocess.PIPE, text=True)
    try:
        for line in process.stdout:
            result = json.loads(line)
            if result.get('error'):
                print(json.dumps(result), flush=True); break
            if result.get('ready'):
                print('Ready. Open /wispocr in WoW; only the visible question is sent on Enter.', flush=True); continue
            packet = parse(result.get('lines', []), nonce, identities)
            if packet is None:
                if result.get('lines') and time.monotonic() - last_rejection > 5:
                    print(json.dumps({'status': 'outbox_not_validated', 'ocr_ms': round(result['ocr_ms'], 1)}), flush=True)
                    last_rejection = time.monotonic()
                last, count = None, 0; continue
            signature = (packet['id'], packet['guid'], packet['question'])
            count = count + 1 if signature == last else 1
            last = signature
            if count < 2 or packet['id'] in seen or (worker and worker.is_alive()):
                continue
            seen.add(packet['id'])
            evidence = {'request_id': packet['id'], 'ocr_ms': round(result['ocr_ms'], 1),
                        'capture_ocr_ms': round(result['capture_ocr_ms'], 1), 'matching_reads': count}
            worker = threading.Thread(target=complete, args=(session, addons, packet, evidence))
            worker.start()
    finally:
        process.terminate(); process.wait(timeout=5)
        if worker:
            worker.join(timeout=55)


if __name__ == '__main__':
    p = argparse.ArgumentParser()
    p.add_argument('action', choices=['install', 'run'])
    p.add_argument('--root', type=pathlib.Path, required=True)
    p.add_argument('--session', type=pathlib.Path, default=pathlib.Path.home() / 'Library/Application Support/Wisp/ocr-session.json')
    p.add_argument('--capture', type=pathlib.Path, default=pathlib.Path('.cache/ocr-test/wisp-ocr-capture').resolve())
    a = p.parse_args()
    install(a.root) if a.action == 'install' else run(a)
