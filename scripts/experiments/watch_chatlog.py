#!/usr/bin/env python3
"""Read-only bounded tail. Persist only exact Wisp test markers, never unrelated log content."""
import argparse
import datetime
import json
import os
from pathlib import Path
import re
import time
from zoneinfo import ZoneInfo

p = argparse.ArgumentParser()
p.add_argument('--log', type=Path, required=True)
p.add_argument('--run', required=True)
p.add_argument('--output', type=Path, required=True)
p.add_argument('--duration', type=float, default=900)
p.add_argument('--protocol', choices=['chat', 'native'], default='chat')
a = p.parse_args()
if not re.fullmatch(r'[0-9a-f]{8,32}', a.run):
    p.error('Invalid run ID')
if a.output.exists():
    p.error('Refusing to overwrite evidence')
a.output.parent.mkdir(parents=True, exist_ok=True)
pattern = re.compile(rb'WISPCHAT1:' + a.run.encode() + rb':([AB]):(\d+):([0-9.]+):(\d+)') if a.protocol == 'chat' else re.compile(rb'WISPNATIVE1:' + a.run.encode() + rb':([0-9.]+):(\d+)')
native = re.compile(rb'^(\d+)/(\d+)\s+(\d+):(\d+):(\d+)\.(\d+)')
offset = a.log.stat().st_size if a.log.exists() else 0
initial_offset = offset
inode = None
partial = b''
started = time.monotonic()
previous = started
max_gap = 0
count = 0
last_heartbeat = started
with a.output.open('x') as evidence:
    os.chmod(a.output, 0o600)
    def emit(obj):
        evidence.write(json.dumps(obj) + '\n')
        evidence.flush()
        print(json.dumps(obj), flush=True)
    emit(dict(event='start', protocol=a.protocol, run=a.run, wall=time.time(), mono_ns=time.monotonic_ns(), initial_offset=initial_offset, poll_seconds=0.05, timezone='Europe/Stockholm'))
    try:
        while time.monotonic() - started < a.duration:
            now = time.monotonic()
            max_gap = max(max_gap, now - previous)
            previous = now
            try:
                stat = a.log.stat()
                if (inode is not None and stat.st_ino != inode) or stat.st_size < offset:
                    emit(dict(event='file_replaced_or_truncated', wall=time.time()))
                    offset, partial = 0, b''
                inode = stat.st_ino
                if stat.st_size > offset:
                    with a.log.open('rb') as stream:
                        stream.seek(offset)
                        chunk = stream.read(64 << 10)
                        offset = stream.tell()
                    observed_wall, observed_mono = time.time(), time.monotonic_ns()
                    lines = (partial + chunk).split(b'\n')
                    partial = lines.pop()[-65536:]
                    for line in lines:
                        marker = pattern.search(line)
                        if not marker:
                            continue
                        row = dict(event='marker', protocol=a.protocol, observed_wall=observed_wall, observed_mono_ns=observed_mono)
                        if a.protocol == 'chat':
                            row.update(variant=marker[1].decode(), sequence=int(marker[2]), sent_game_uptime=float(marker[3]), sent_game_epoch=int(marker[4]))
                        else:
                            row.update(sent_game_uptime=float(marker[1]), sent_game_epoch=int(marker[2]))
                        stamp = native.search(line)
                        if stamp:
                            month, day, hour, minute, second = map(int, stamp.groups()[:5])
                            fraction = stamp[6].decode()
                            row['native_timestamp_prefix'] = stamp[0].decode()
                            try:
                                year = datetime.datetime.now(ZoneInfo('Europe/Stockholm')).year
                                event_time = datetime.datetime(year,month,day,hour,minute,second,int((fraction+'000000')[:6]),tzinfo=ZoneInfo('Europe/Stockholm')).timestamp()
                                row['native_wall_assuming_local_timezone'] = event_time
                                row['observed_minus_native_seconds'] = observed_wall - event_time
                                row['native_resolution_seconds'] = 10 ** -len(fraction)
                            except ValueError:
                                row['native_timestamp_invalid'] = True
                        count += 1
                        emit(row)
            except FileNotFoundError:
                pass
            if now - last_heartbeat >= 15:
                emit(dict(event='heartbeat', wall=time.time(), markers=count, offset=offset, max_poll_gap_seconds=max_gap))
                last_heartbeat = now
            time.sleep(0.05)
    except KeyboardInterrupt:
        pass
    finally:
        emit(dict(event='stop',wall=time.time(),markers=count,max_poll_gap_seconds=max_gap))
