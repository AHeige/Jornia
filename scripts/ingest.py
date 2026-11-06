#!/usr/bin/env python3
"""Run a bounded local ingestion batch. Reads only the named development credentials."""
from pathlib import Path
import argparse
import os
import subprocess

root = Path(__file__).resolve().parents[1]
parser = argparse.ArgumentParser()
parser.add_argument('--max-requests', type=int, default=200)
args = parser.parse_args()
if not 1 <= args.max_requests <= 2000:
    parser.error('max-requests must be 1..2000')
env = os.environ.copy()
if (root / '.env').exists():
    for line in (root / '.env').read_text().splitlines():
        key, separator, value = line.partition('=')
        if separator and key in {'BLIZZARD_CLIENT_ID', 'BLIZZARD_CLIENT_SECRET', 'BLIZZARD_REGION', 'BLIZZARD_LOCALE'}:
            env.setdefault(key, value.strip().strip('\"\x27'))
env.setdefault('DATABASE_URL', 'postgres://wisp:wisp-local-only@127.0.0.1:55471/wisp?sslmode=disable')
env['GOCACHE'] = str(root / '.cache/go-build')
env['GOMODCACHE'] = str(root / '.cache/go-mod')
raise SystemExit(subprocess.run(['go', 'run', './apps/ingest', '-max-requests', str(args.max_requests)], cwd=root, env=env).returncode)
