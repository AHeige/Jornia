#!/usr/bin/env python3
"""Bounded live-model evaluation using synthetic quest context, never player journals."""
import argparse, json, pathlib, time, urllib.request, urllib.error
p=argparse.ArgumentParser();p.add_argument('--base',required=True);p.add_argument('--output',type=pathlib.Path,required=True);p.add_argument('--case',action='append',dest='selected');a=p.parse_args()
if not a.base.startswith('http://127.0.0.1:'):raise SystemExit('Use a private loopback tunnel')
def snapshot(quests):return dict(captured_at=int(time.time()),level=16,zone='Synthetic Test Zone',client_build='70790',truncated=False,quests=quests)
quest=dict(id=376,title='Synthetic Collection Test',complete=False,waypoint=False,objectives=[dict(index=1,text='Synthetic paws: 4/6',fulfilled=4,required=6,finished=False)])
cases=[('remaining','How many more synthetic paws do I need?',lambda:snapshot([quest])),('next','What should I do next in this synthetic test?',lambda:snapshot([quest])),('ready','What should I do next?',lambda:snapshot([dict(id=376,title='Synthetic Collection Test',complete=True,waypoint=False)])),('missing','What should I do next?',lambda:None),('empty','Which quests do I have active?',lambda:snapshot([])),('stale','Next?',lambda:dict(snapshot([quest]),captured_at=int(time.time())-120))]
rich=dict(quest,description='Synthetic paws are found west of Test Chapel. Bring them back to Synthetic Keeper afterwards.',objective_text='Collect six synthetic paws for Synthetic Keeper.',waypoint_text='West of Test Chapel')
cases += [('directions','Where should I go and what should I do there?',lambda:snapshot([rich])),('recipient','Who should I return to afterwards?',lambda:snapshot([rich])),('continuation_missing','What is the name of the quest I unlock after Synthetic Collection Test?',lambda:snapshot([rich]))]
if a.selected:cases=[c for c in cases if c[0] in a.selected]
rows=[]
for index,(label,question,make) in enumerate(cases):
 if index:time.sleep(11)
 body=dict(question=question,mode='generate',game_version='forever');s=make()
 if s is not None:body['snapshot']=s
 began=time.monotonic();row=dict(case=label,synthetic=True)
 try:
  req=urllib.request.Request(a.base+'/api/wisp/ask',json.dumps(body).encode(),{'Content-Type':'application/json'})
  with urllib.request.urlopen(req,timeout=65) as response:answer=json.load(response)
  row['answer']=answer
  ids=answer.get('evidence_ids') or []
  failures=[]
  if any('foreverdb' in json.dumps(x).lower() for x in answer.get('sources') or []):failures.append('paused provider')
  if label in ('remaining','next','ready') and (answer.get('status')!='answered' or 'state:quest:376' not in ids):failures.append('quest grounding')
  if label=='remaining' and not any(x in answer.get('text','').lower() for x in ['2','two']):failures.append('remaining count')
  if label=='empty' and (answer.get('status')!='answered' or 'state:snapshot' not in ids):failures.append('empty snapshot not explained')
  if label=='ready' and any(x in answer.get('text','').lower() for x in ['do not invent','suggest checking']):failures.append('internal instruction leaked')
  if label=='recipient' and ('Synthetic Keeper' not in answer.get('text','') or 'state:quest:376' not in ids):failures.append('observed requester missing')
  if label=='directions' and ('west' not in answer.get('text','').lower() or 'state:quest:376' not in ids):failures.append('observed direction missing')
  if label=='continuation_missing' and not any(x in answer.get('text','').lower() for x in ['unconfirmed','unknown','not confirmed','not supplied',"don't",'cannot',"can't",'not verified','no verified','not available','not provided']):failures.append('missing continuation not explained')
  if label=='stale':failures.append('stale accepted')
  row.update(passed=not failures,failures=failures)
 except urllib.error.HTTPError as error:row.update(http_status=error.code,passed=label=='stale' and error.code==400)
 except Exception as error:row.update(error=type(error).__name__,passed=False)
 row['elapsed_ms']=round((time.monotonic()-began)*1000);rows.append(row)
 a.output.parent.mkdir(parents=True,exist_ok=True);a.output.write_text(json.dumps(dict(cases=rows,passed=sum(x['passed'] for x in rows),total=len(rows)),indent=2)+'\n')
 print(label,row['passed'],row['elapsed_ms'],flush=True)
raise SystemExit(0 if all(x['passed'] for x in rows) else 1)
