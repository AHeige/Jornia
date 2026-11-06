#!/usr/bin/env python3
"""Compare the original fixture with reviewed knowledge using the running API. No inference calls."""
import json
import os
from pathlib import Path
import urllib.request

root = Path(__file__).resolve().parents[1]
fixture = json.loads((root / 'internal/storage/fixtures/ashenvale.json').read_text())
packages=[json.loads(p.read_text()) for p in sorted((root/'internal/storage/datasets').glob('*.json')) if not p.name.startswith('._')]
reviewed={'version':'forever','dataset_revision':'+'.join(p['dataset_revision'] for p in packages),
          'entities':[e for p in packages for e in p['entities']], 'edges':[e for p in packages for e in p['edges']], 'sources':[e for p in packages for e in p['sources']]}
base = os.environ.get('WISP_API_URL', 'http://127.0.0.1:8090').rstrip('/')

def ask(question, version):
    payload = {'question':question, 'game_version':version}
    request = urllib.request.Request(base + '/api/wisp/ask', json.dumps(payload).encode(), {'Content-Type':'application/json'})
    with urllib.request.urlopen(request, timeout=10) as response:
        result = json.load(response)
    if result['model'] is not None or result['estimated_cost_eur'] != 0:
        raise RuntimeError('Comparison unexpectedly invoked inference')
    return {key:result[key] for key in ['status','text','evidence_ids','warnings']} | {
        'source_ids':[source['id'] for source in result['sources']],
        'source_statuses':sorted({source['validation_status'] for source in result['sources']})}

questions = [
    'Where is Ashenvale?', 'Where is Astranaar?', 'Where is Splintertree Post?',
    'Where is The Zoram Strand?', 'Tell me about Ashenvale', 'Where is Forest Song?',
    "Where is Zoram'gar Outpost?", 'Where is Blackfathom Deeps?',
    'Where is Raene Wolfrunner?', "Who gives Raene's Cleansing?",
    'What places are in Ashenvale?', 'What quests are in Ashenvale?',
    'What level should I be in Ashenvale?', 'Tell me about Hidden Story',
]
old_ids = {e['id'] for e in fixture['entities']}
new_ids = {e['id'] for e in reviewed['entities']}
old_edges = {(e['subject_id'],e['predicate'],e['object_id']) for e in fixture['edges']}
new_edges = {(e['subject_id'],e['predicate'],e['object_id']) for e in reviewed['edges']}
results = [{'question':q,'fixture':ask(q,fixture['version']),'reviewed':ask(q,reviewed['version'])} for q in questions]
report = {
    'review_date':'2026-11-05', 'fixture_version':fixture['version'],
    'reviewed_version':reviewed['version'], 'dataset_revision':reviewed['dataset_revision'],
    'assurance':'Community source-index review; no independent game-client verification.',
    'counts':{'fixture_entities':len(old_ids),'reviewed_entities':len(new_ids),'reviewed_edges':len(new_edges),'reviewed_sources':len(reviewed['sources'])},
    'shared_entities':sorted(old_ids & new_ids), 'added_entities':sorted(new_ids-old_ids),
    'fixture_only_entities':sorted(old_ids-new_ids),
    'shared_relationships':sorted(old_edges & new_edges),
    'newly_answered':sum(r['fixture']['status']=='unknown' and r['reviewed']['status']=='answered' for r in results),
    'questions':results,
}
output=root/'docs/ashenvale-comparison.json'
output.write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
for r in results:
    print(f"{r['fixture']['status']:8s} -> {r['reviewed']['status']:8s} {r['question']}")
print(json.dumps(report['counts']))
print('Newly answered questions:',report['newly_answered'])
print('Report:',output)
