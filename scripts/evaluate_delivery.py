#!/usr/bin/env python3
"""Bounded operator evaluation. Local/private API only; synthetic personal context.
No retry of model calls. --live permits at most 18 deliberate Berget requests.
Outputs test questions and replies, never credentials or real player journals.
"""
import argparse, json, os, pathlib, time, urllib.request, urllib.error
p=argparse.ArgumentParser();p.add_argument('--base',default='http://127.0.0.1:18092');p.add_argument('--live',action='store_true');p.add_argument('--catalog',type=pathlib.Path);p.add_argument('--output',type=pathlib.Path,required=True);a=p.parse_args()
if not a.base.startswith('http://127.0.0.1:'):raise SystemExit('Use an explicit private loopback API tunnel')
now=int(time.time())
profile={'evidence_id':'character:profile','level':15,'race':'Undead','class':'Warlock','fetched_at':now,'last_login':now-3*86400,'expires_at':now+86400,'items':[{'id':700001,'slot':'MAIN_HAND','name':'Synthetic Training Wand','quality':'COMMON'}]}
journal=[{'id':'memory:1','kind':'level_up','level':16,'zone':'Ashenvale','observed_at':now-3600},{'id':'memory:2','kind':'quest_completed','quest_id':1033,'level':15,'zone':'Ashenvale','subzone':'Astranaar','observed_at':now-7200}]
cases=[]
def case(group,q,status='answered',contains=None,**fields):
 cases.append({'group':group,'question':q,'status':status,'contains':contains,'fields':fields})
if a.catalog:
 for e in json.loads(a.catalog.read_text()):
  for prefix in ['Tell me about ','Vad är ']:case('catalog',prefix+e['name'],'answered',e['name'])
for q in ["What is the exact live drop chance for every item in Ashenvale?","Can you confirm every Forever quest is imported?","What is my current inventory?","Which quests do I have active right now?","Where is Unknown Wisp Test Village?","Who gives Nonexistent Wisp Quest?","Vad har jag i väskan just nu?","Vilka quests har jag aktiva just nu?","What comes after Nonexistent Wisp Quest?","Where is My Made Up Dungeon?"]:
 case('missing',q,'unknown')
for version in ['retail','classic','era','not-a-game']:
 case('version','Where is Astranaar?','unknown',game_version=version)
if a.live:
 for q,expected,word in [
  ('Where is Raene Wolfrunner in the supplied Ashenvale sources?','answered','Astranaar'),
  ("Who gives Bathran's Hair according to the supplied sources?",'answered','Orendil'),
  ("What comes after Orendil’s Cure in the supplied chain?",'answered','Elune'),
  ('What required level does the supplied API record give Tough Jerky?','answered','1'),
  ('What does the supplied Tough Jerky tooltip say it restores?','answered','58'),
  ('What quality does the supplied API record give Tough Jerky?','answered','Common'),
  ('What are the exact live coordinates of Raene Wolfrunner?','unknown',None),
  ("What is the exact drop chance for Bathran’s Hair?",'unknown',None)]:
  case('live-world',q,expected,word,mode='generate')
 for q,word in [
  ('In this synthetic test, what class does the dated profile record?','Warlock'),
  ('In this synthetic test, what race does the dated profile record?','Undead'),
  ('Which weapon is listed in this synthetic dated profile?','Synthetic Training Wand'),
  ('What level does the older synthetic profile record, and is it live?','15'),
  ('Vilken klass visar denna syntetiska sparade profil?','Warlock')]:
  case('live-character',q,'answered',word,mode='generate',character=profile)
 for q in ['Which quests are active right now on this test character?','How much gold is in the bags of this test character?','Which talents has this test character chosen?']:
  case('live-character-missing',q,'unknown',mode='generate',character=profile)
 case('live-journey','In this synthetic test, what level was most recently recorded by the journal?','answered','16',mode='generate',journal=journal,synthetic_journal=True,character=profile)
 case('live-journey','Vilket quest-ID har den syntetiska journalen registrerat som slutfört?','answered','1033',mode='generate',journal=journal,synthetic_journal=True)
results=[];last_live=0;cost=0.0
for c in cases:
 fields=c['fields'];live=fields.get('mode')=='generate'
 if live:
  delay=11-(time.monotonic()-last_live)
  if delay>0:time.sleep(delay)
  last_live=time.monotonic()
 payload={'question':c['question'],'game_version':'forever','mode':'auto'}|fields
 row={k:v for k,v in c.items() if k!='fields'}
 try:
  req=urllib.request.Request(a.base+'/api/wisp/ask',json.dumps(payload).encode(),{'Content-Type':'application/json'})
  with urllib.request.urlopen(req,timeout=40) as response:r=json.load(response)
  row['answer']=r
  failures=[]
  if r['status']!=c['status']:failures.append('status')
  if c['contains'] and c['contains'].casefold() not in r['text'].casefold():failures.append('expected fact')
  if r['status']=='answered' and (not r.get('sources') or not r.get('evidence_ids')):failures.append('missing evidence')
  if r['status']=='unknown' and (r.get('sources') or r.get('evidence_ids')):failures.append('unknown has citations')
  if c['group'].startswith('live-character') and r['status']=='answered' and 'character:profile' not in r['evidence_ids']:failures.append('missing character evidence')
  if c['group']=='live-journey' and r['status']=='answered' and not any(x.startswith('memory:') for x in r['evidence_ids']):failures.append('missing journey evidence')
  cost+=r.get('estimated_cost_eur') or 0
  row.update(passed=not failures,failures=failures)
 except Exception as error:row.update(passed=False,error=type(error).__name__)
 results.append(row);a.output.write_text(json.dumps({'cases':results,'passed':sum(x['passed'] for x in results),'total':len(results),'estimated_cost_eur':cost},ensure_ascii=False,indent=2)+'\n')
 print(('PASS' if row['passed'] else 'FAIL'),c['group'],c['question'],flush=True)
 if cost>.05:raise SystemExit('Evaluation cost guard reached; no more calls')
print(f"{sum(x['passed'] for x in results)}/{len(results)} passed, known estimated cost EUR {cost:.6f}")
raise SystemExit(0 if all(x['passed'] for x in results) else 1)
