#!/usr/bin/env python3
"""Run fixture and sourced Forever acceptance checks against the running API; --live makes one paid provider call."""
import argparse
import json
import os
from pathlib import Path
import urllib.request

parser = argparse.ArgumentParser()
parser.add_argument('--live', action='store_true')
parser.add_argument('--suite', choices=['all','fixture','forever'], default='all')
args = parser.parse_args()
base = os.environ.get('WISP_API_URL', 'http://127.0.0.1:8090').rstrip('/')
root = Path(__file__).resolve().parents[1]
cases = []
if args.suite in ('all','fixture'):
    for case in json.loads((root / 'tests/evaluations/ashenvale.json').read_text()):
        case.setdefault('game_version','fixture-ashenvale-v1')
        cases.append(case)
if args.suite in ('all','forever'):
    cases.extend(json.loads((root / 'tests/evaluations/ashenvale-forever.json').read_text()))
if args.live:
    cases = [{'question': 'Briefly explain where Raene Wolfrunner is, using only the supplied community sources.', 'mode': 'generate', 'status': 'answered', 'contains': 'Astranaar', 'game_version':'forever'}]
failures = 0
for case in cases:
    payload = {k: case[k] for k in ('question', 'game_version', 'mode') if k in case}
    payload.setdefault('game_version', 'forever')
    request = urllib.request.Request(base + '/api/wisp/ask', json.dumps(payload).encode(), {'Content-Type': 'application/json'})
    try:
        with urllib.request.urlopen(request, timeout=40) as response:
            result = json.load(response)
        assert result['status'] == case['status'], result
        assert case.get('contains', '').lower() in result['text'].lower(), result
        assert 'Hidden Story' not in result['text'], result
        if result['status'] == 'answered':
            assert result['sources'] and result['evidence_ids'], result
            assert result['warnings'], result
            if payload['game_version'] == 'forever':
                assert 'fixture' not in result['text'].lower(), result
                for source in result['sources']:
                    assert source['validation_status'] == 'source_checked', source
                    assert source.get('provenance', {}).get('checked_at') == '2026-11-05', source
                    assert source['uri'].startswith('https://'), source
        else:
            assert not result['evidence_ids'] and not result['sources'], result
        if args.live:
            assert result['model'] and result['route'] == 'berget', result
            print(json.dumps(result, indent=2))
        else:
            assert result['model'] is None and result['estimated_cost_eur'] == 0, result
        print('PASS:', case['question'])
    except Exception as error:
        failures += 1
        print('FAIL:', case['question'], str(error))
print(f'{len(cases) - failures}/{len(cases)} passed')
raise SystemExit(1 if failures else 0)
