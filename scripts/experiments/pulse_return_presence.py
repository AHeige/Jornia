#!/usr/bin/env python3
"""Pulse only Jornia's disposable, pre-existing zero-byte research asset.
No WoW input, settings, network, imports or inference. Lua must already be probing.
"""
import argparse
import json
import pathlib
import time


def pulse(root, cycles, half_period):
    root = root.resolve(strict=True)
    lab = root / 'Interface/AddOns/WispBridgeLab'
    target = lab / 'wispreturn_pulse.wav'
    if not (root / 'WTF').is_dir() or lab.is_symlink() or not lab.is_dir():
        raise ValueError('Expected installed Jornia research lab')
    if target.is_symlink() or not target.is_file() or target.stat().st_size != 0:
        raise ValueError('Expected only our existing zero-byte pulse asset')
    if not 1 <= cycles <= 20 or not .25 <= half_period <= 2 or cycles * half_period * 2 > 40:
        raise ValueError('Bounded experiment only: at most 20 cycles and 40 seconds')
    events = []
    try:
        for cycle in range(cycles):
            target.unlink()
            events.append({'cycle': cycle + 1, 'present': False, 'monotonic': time.monotonic()})
            time.sleep(half_period)
            with target.open('xb'):
                pass
            events.append({'cycle': cycle + 1, 'present': True, 'monotonic': time.monotonic()})
            time.sleep(half_period)
    finally:
        if not target.exists():
            with target.open('xb'):
                pass
    return events


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--root', type=pathlib.Path, required=True)
    parser.add_argument('--cycles', type=int, default=10)
    parser.add_argument('--half-period', type=float, default=.5)
    args = parser.parse_args()
    print(json.dumps(pulse(args.root, args.cycles, args.half_period), indent=2))
