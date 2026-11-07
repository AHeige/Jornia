"""Owned native-chat wire formats. Checked UTF-8 data only; never Lua source."""
import json
import re
import struct
import sys
import pathlib
import zlib
sys.path.insert(0,str(pathlib.Path(__file__).resolve().parents[1]/'experiments/ocr'))
from bridge import checksum,navigation


def expand_snapshot(value):
    if value is None:return None
    if not isinstance(value,list) or len(value) not in (6,7):raise ValueError('snapshot shape')
    from bridge import expand_snapshot as base
    rows=value[5]
    if not isinstance(rows,list):raise ValueError('quest shape')
    simple=[]
    for row in rows:
        if not isinstance(row,list) or len(row) not in (4,5):raise ValueError('quest shape')
        simple.append(row[:4])
    snap=base(value[:5]+[simple])
    for row,q in zip(rows,snap['quests']):
        if len(row)==5:
            if not isinstance(row[4],list) or len(row[4])>8:raise ValueError('objective shape')
            objectives=[]
            for o in row[4]:
                if not isinstance(o,list) or len(o) not in (4,5) or any(type(n) is not int for n in o[:4]) or not 1<=o[0]<=32 or not -1<=o[1]<=2147483647 or not -1<=o[2]<=2147483647 or o[3] not in (0,1):raise ValueError('objective bounds')
                goal=dict(index=o[0],finished=bool(o[3]))
                if len(o)==5:
                    if not isinstance(o[4],str) or len(o[4].encode())>80:raise ValueError('objective text bounds')
                    goal['text']=o[4]
                if o[1]>=0:goal['fulfilled']=o[1]
                if o[2]>=0:goal['required']=o[2]
                objectives.append(goal)
            q['objectives']=objectives
    if len(value)==7:
        meta=value[6]
        if not isinstance(meta,dict) or set(meta)!={'c','r','e','p','m'}:raise ValueError('character shape')
        if any(type(meta[k]) is not int or not 0<=meta[k]<=100 for k in ('c','r')):raise ValueError('identity bounds')
        gear=meta['e'];skills=meta['p'];missing=meta['m']
        if not isinstance(gear,list) or len(gear)>19 or not isinstance(skills,list) or len(skills)>12 or not isinstance(missing,list) or len(missing)>8 or any(x not in ('quests','equipment','professions','class','race') for x in missing):raise ValueError('character bounds')
        slots=set();items=[]
        for row in gear:
            if not isinstance(row,list) or len(row)!=2 or any(type(n) is not int for n in row) or not 1<=row[0]<=19 or row[0] in slots or not 1<=row[1]<=2147483647:raise ValueError('item bounds')
            slots.add(row[0]);items.append(dict(slot=row[0],item_id=row[1]))
        professions=[]
        for row in skills:
            if not isinstance(row,list) or len(row)!=3 or any(type(n) is not int for n in row) or not 1<=row[0]<=2147483647 or not 0<=row[1]<=row[2]<=10000:raise ValueError('skill bounds')
            professions.append(dict(skill_id=row[0],rank=row[1],max_rank=row[2]))
        snap.update(class_id=meta['c'],race_id=meta['r'],equipment=items,professions=professions,missing=missing)
    return snap

def parse(lines,identities):
    text='\n'.join(lines)
    m=re.search(r'^WISPBEGIN3\s+(\d{24})\s*(\d{5})\s*(\d{8})\s*(\d{5})\n([\s0-9a-fA-F]+)\nWISPEND$',text,re.M)
    if not m:return None
    ident,binding,nonce,digest,encoded=m.groups()
    encoded=''.join(encoded.split())
    try:
        if not 2<=len(encoded)<=1280 or len(encoded)%2:return None
        raw=bytes.fromhex(encoded).decode('utf8')
        if checksum(ident+binding+nonce+raw)!=int(digest):return None
        value=json.loads(raw)
        if not isinstance(value,dict):return None
        if set(value)=={'a'}:
            a=value['a']
            if not isinstance(a,dict) or set(a)!={'pad','cipher'} or not isinstance(a['pad'],str) or not re.fullmatch('[0-9a-f]{16}',a['pad']) or not isinstance(a['cipher'],str) or not re.fullmatch('[0-9a-f]{64}',a['cipher']):return None
            return dict(id=ident,binding=binding,nonce=nonce,activation=a)
        if set(value)!={'q','s'}:return None
        q=value['q']
        if not isinstance(q,str) or not q.strip() or len(q.encode())>160:return None
        snapshot=expand_snapshot(value['s'])
        matches=[g for g in identities if re.fullmatch(r'Player-[A-Za-z0-9-]+',g) and checksum(g)==int(binding)]
        if len(matches)!=1:return None
        guid=matches[0]
        packet=dict(id=ident,binding=binding,nonce=nonce,guid=guid,character_id=identities[guid],question=q)
        if snapshot is not None:packet['snapshot']=snapshot
        return packet
    except (ValueError,TypeError,UnicodeError):return None


def reply(packet,result,error=None):
    text=error if error else result.get('text','')
    if not isinstance(text,str) or not text:raise ValueError('No reply text')
    raw=text.encode('utf8')
    if len(raw)>1536:
        error='This reply is too long for the in-game channel. Read it in Wisp desktop.'
        raw=error.encode()
    credit=not error and any(str(s.get('id','')).startswith('foreverdb:') for s in result.get('sources') or [])
    sources=min(len(result.get('sources') or []),65535)
    action=None if error else navigation(packet,result)
    quest,expires=action if action else (0,0)
    return bytes([1,bool(error)])+packet['id'].encode('ascii')+struct.pack('>H',len(raw))+raw+struct.pack('>BHII',int(credit),sources,quest,expires)


def frame(payload,nonce,binding):
    if len(payload)>4096:raise ValueError('Reply payload bound exceeded')
    body=b'WB\x01'+struct.pack('>IHH',int(nonce),int(binding),len(payload))+payload
    body+=struct.pack('>I',zlib.adler32(body))
    return body+bytes((-len(body))%8)
