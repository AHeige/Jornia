#!/usr/bin/env python3
"""Summarise our saved Lua receipts and filtered observer events, never raw chat."""
import argparse
import json
from pathlib import Path
import statistics

p = argparse.ArgumentParser()
p.add_argument('folder', type=Path)
a = p.parse_args()
run = json.loads((a.folder / 'addon-receipts.json').read_text())
rows = []
for file in a.folder.glob('disk-observations*.jsonl'):
    rows += [json.loads(line) for line in file.read_text().splitlines()]
markers = [r for r in rows if r['event'] == 'marker']
starts = [r for r in rows if r['event'] == 'start']
clock_offsets = [r['wall'] - r['mono_ns'] / 1e9 for r in starts]
clock_offset = statistics.median(clock_offsets)
samples = []
for sample in run['samples']:
    hits = [r for r in markers if r['sequence'] == sample['sequence'] and r['variant'] == sample['variant']]
    first = min(hits, key=lambda r: r['observed_mono_ns']) if hits else None
    native_times = [r['native_wall_assuming_local_timezone'] for r in hits if 'native_wall_assuming_local_timezone' in r]
    receipt = dict(sequence=sample['sequence'],variant=sample['variant'],mode=sample.get('toggle_mode','direct' if sample['variant']=='B' else 'normal'),received='received_uptime' in sample)
    if receipt['received']:
        receipt['send_to_receive_seconds'] = sample['received_uptime'] - sample['sent_uptime']
    if first:
        receipt['first_observed_wall'] = first['observed_wall']
        receipt['matching_log_rows'] = len(hits)
        if native_times:
            native = min(native_times)
            receipt['native_to_disk_observation_seconds'] = first['observed_wall'] - native
            receipt['native_vs_game_clock_alignment_seconds'] = native - sample['received_uptime'] - clock_offset
    if 'toggle_after_uptime' in sample:
        receipt['receive_to_toggle_end_seconds'] = sample['toggle_after_uptime'] - sample['received_uptime']
        receipt['toggle_interval_seconds'] = sample['toggle_after_uptime'] - sample['toggle_before_uptime']
    samples.append(receipt)
summary = dict(run=run['id'],client_version=run['version'],client_build=run['build'],original_logging=run['original_logging'],stop_reason=run.get('stop_reason'),stopped_epoch=run.get('stopped_epoch'),samples=samples,matching_log_rows=len(markers),unique_first_disk_times=sorted(set(s['first_observed_wall'] for s in samples if 'first_observed_wall' in s)),max_poll_gap_seconds=max(r.get('max_poll_gap_seconds',0) for r in rows),host_wall_minus_monotonic_offsets=clock_offsets)
output = a.folder / 'summary.json'
output.write_text(json.dumps(summary,indent=2))
print(json.dumps(summary,indent=2))
