#!/usr/bin/env python3
"""Validate reviewed, versioned knowledge packages. Does not scrape or infer completeness."""
import hashlib
import json
from pathlib import Path

root=Path(__file__).resolve().parents[1]
entities={};edges={};sources={};revisions=[]
for file in sorted((root/'internal/storage/datasets').glob('*.json')):
    if file.name.startswith('._'):continue
    data=json.loads(file.read_text())
    assert data['version']=='forever' and data['dataset_revision'],file
    revisions.append({'revision':data['dataset_revision'],'sha256':hashlib.sha256(file.read_bytes()).hexdigest()})
    for source in data['sources']:
        assert source['id'] not in sources,source['id']
        assert source['validation_status'] in ['source_checked','verified'],source['id']
        assert source['uri'].startswith('https://') and source['license'] and source['revision'],source['id']
        for field in ['checked_at','effective_version','verification_method','attribution','license_uri','redistribution','review_sha256']:
            assert source['provenance'].get(field),(source['id'],field)
        sources[source['id']]=source
    for entity in data['entities']:
        assert entity['id'] not in entities,entity['id']
        assert entity['name'] and entity['kind'] and entity['source_id'] in sources,entity['id']
        entities[entity['id']]=entity
    for edge in data['edges']:
        assert edge['id'] not in edges,edge['id']
        assert edge['source_id'] in sources,edge['id']
        edges[edge['id']]=edge
for edge in edges.values():
    assert edge['subject_id'] in entities and edge['object_id'] in entities,edge['id']
# Serving uses SQL anchor retrieval and a bounded two-hop neighborhood, not a whole-corpus prompt.
report={'complete':False,'entities':len(entities),'edges':len(edges),'sources':len(sources),'revisions':revisions,
        'scope':'Ashenvale geography, neighboring zones, and selected NPC/quest relationships',
        'unknown_total_game_entities':True}
print(json.dumps(report,indent=2))
