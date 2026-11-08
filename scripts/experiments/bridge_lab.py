#!/usr/bin/env python3
"""Controlled LoD inbox experiment. Writes only our fixed addon paths; never runs WoW."""
import argparse
import pathlib
import shutil
import time

p = argparse.ArgumentParser()
p.add_argument('action', choices=['install', 'reply'])
p.add_argument('--root', type=pathlib.Path, required=True)
p.add_argument('--slot', type=int, choices=range(1, 5), default=1)
p.add_argument('--text', default='A fresh local test reply, written after WoW started.')
a = p.parse_args()
if not (a.root / 'WTF').is_dir():
    p.error('Expected a WoW client directory containing WTF')
addons = a.root / 'Interface' / 'AddOns'
if a.action == 'install':
    source = pathlib.Path(__file__).resolve().parents[2] / 'apps/bridge-lab/WispBridgeLab'
    if (addons / 'WispBridgeLab').exists():
        p.error('Lab already installed; refusing to reset consumed experiment slots')
    shutil.copytree(source, addons / 'WispBridgeLab')
    for slot in range(1, 5):
        name = f'WispBridgeInbox{slot:02d}'
        target = addons / name
        target.mkdir()
        (target / f'{name}_Camelot.toc').write_text(f'## Interface: 16001\n## Title: Jornia research inbox {slot}\n## LoadOnDemand: 1\n## Dependencies: WispBridgeLab\nInbox.lua\n')
        (target / 'Inbox.lua').write_text('WispBridgeLabReceive("696e697469616c", "5374617274757020636f6e74726f6c20746578742e")\n')
    print('Installed inert lab + four one-shot inboxes. Start WoW, then write a reply before consuming slot 1.')
else:
    data = a.text.encode('utf-8')
    if len(data) > 12000:
        p.error('Maximum reply size is 12000 UTF-8 bytes')
    request = f'file-probe-{time.time_ns()}'
    target = addons / f'WispBridgeInbox{a.slot:02d}' / 'Inbox.lua'
    if not target.is_file():
        p.error('Install the lab before starting WoW')
    # Untrusted reply text becomes hex data, never executable Lua.
    temporary = target.with_suffix('.lua.tmp')
    temporary.write_text(f'WispBridgeLabReceive("{request.encode().hex()}", "{data.hex()}")\n')
    temporary.replace(target)
    print('Wrote', request, 'to slot', a.slot, 'at', time.time())
    print(f'Now run /wispbridge inbox {a.slot} in game, WITHOUT /reload first.')
