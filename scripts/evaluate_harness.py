#!/usr/bin/env python3
"""Five bounded real-model tests with synthetic per-question state. No retries."""
import argparse,json,pathlib,time,urllib.request
p=argparse.ArgumentParser();p.add_argument('--base',required=True);p.add_argument('--output',type=pathlib.Path,required=True);a=p.parse_args()
if not a.base.startswith('http://127.0.0.1:'):raise SystemExit('Use private API tunnel')
cases=[('fr','Que dois-je faire maintenant selon cet instantané de test ?'),('es','¿Qué debo hacer ahora según esta captura de prueba?'),('ja','このテスト用スナップショットでは、次に何をすればいいですか？'),('ar','ماذا أفعل الآن حسب لقطة الاختبار هذه؟'),('sv','Vad gör jag nu enligt denna testögonblicksbild?')]
results=[]
for lang,question in cases:
 snapshot=dict(captured_at=int(time.time()),level=16,zone='Ashenvale',client_build='70790',truncated=True,quests=[dict(id=700001,title='Synthetic completed quest',complete=True,waypoint=True),dict(id=700002,title='Synthetic active quest',complete=False,waypoint=False)])
 payload=dict(question=question,mode='generate',game_version='forever',snapshot=snapshot)
 req=urllib.request.Request(a.base+'/api/wisp/ask',json.dumps(payload).encode(),{'Content-Type':'application/json'})
 try:
  with urllib.request.urlopen(req,timeout=40) as r:answer=json.load(r)
  ok=answer['status']=='answered' and 'state:quest:700001' in answer['evidence_ids'] and len(answer.get('actions',[]))==1 and answer['actions'][0]['quest_id']==700001 and len(answer.get('tool_trace',[]))==4
  results.append(dict(language=lang,passed=ok,answer=answer))
 except Exception as error:results.append(dict(language=lang,passed=False,error=type(error).__name__))
 a.output.parent.mkdir(parents=True,exist_ok=True);a.output.write_text(json.dumps(dict(cases=results,passed=sum(r['passed'] for r in results),total=len(cases)),ensure_ascii=False,indent=2)+'\n')
 print(lang,'PASS' if results[-1]['passed'] else 'FAIL',flush=True)
 if len(results)<len(cases):time.sleep(11)
raise SystemExit(0 if all(r['passed'] for r in results) else 1)
