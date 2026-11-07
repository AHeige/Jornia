import {settings,companion,journey,openImport} from './ui.mjs';
import {spawn} from 'node:child_process';
import {once} from 'node:events';
import fs from 'node:fs/promises';
import path from 'node:path';
import http from 'node:http';
import {fileURLToPath} from 'node:url';
import {chromium} from 'playwright';
const root=path.resolve(path.dirname(fileURLToPath(import.meta.url)),'../..');
const data=await fs.mkdtemp(path.join(root,'.cache/personal-'));let child,browser;const requests=[];const access='b'.repeat(64);
const backend=http.createServer(async(req,res)=>{res.setHeader('content-type','application/json');if(req.headers.authorization!=='Bearer '+access){res.writeHead(401);res.end('{}');return;}if(req.url==='/healthz'){res.end('{"status":"ok"}');return;}let body='';for await(const part of req)body+=part;const payload=JSON.parse(body);requests.push(payload);res.end(JSON.stringify({status:'answered',text:payload.journal?.length?'Demo data: your recorded journey passes through Ashenvale.':'Ashenvale is a zone.',route:'berget',model:'synthetic-test-model',game_version:'forever',evidence_ids:payload.journal?.length?['memory:1']:[],sources:payload.journal?.length?[{id:'memory:1',title:'Selected local journal excerpt',uri:'wisp-local:excerpt/memory:1',revision:'test',validation_status:'user_imported'}]:[],warnings:['Synthetic integration test.']}));});backend.listen(0,'127.0.0.1');await once(backend,'listening');
function ensure(x,m){if(!x)throw Error(m);}
try{
 child=spawn(path.join(root,'.cache/wisp-companion'),['-no-browser','-listen','127.0.0.1:0','-data-dir',data,'-backend','http://127.0.0.1:'+backend.address().port],{stdio:['ignore','pipe','pipe']});const base=await new Promise((resolve,reject)=>{child.stdout.on('data',x=>{const m=String(x).match(/Wisp: (http:\/\/[^\s]+)/);if(m)resolve(m[1]);});child.once('exit',code=>reject(Error('exit '+code)));});
 browser=await chromium.launch({headless:true});const page=await browser.newPage({viewport:{width:1200,height:1050}});await page.goto(base);
 await settings(page);await page.locator('#access-code').fill(access);await page.locator('#access-save').click();await page.getByText('Wisp is connected.',{exact:false}).waitFor();ensure(await page.locator('#access-code').inputValue()==='','Credential remains visible');
 await page.locator('#demo-start').click();await page.locator('#demo-banner').waitFor({state:'visible'});await companion(page);await page.locator('#ai').check();await page.getByText('recorded observations. Sent only',{exact:false}).waitFor();
 await page.locator('#question').fill('Tell me about Ashenvale');await page.locator('#ask-submit').click();await page.getByRole('log').getByText('Ashenvale is a zone.',{exact:true}).waitFor();ensure(!requests[0].journal?.length,'Implicit journal upload');
 await page.locator('#ai').uncheck();await page.locator('#ai').check();await page.getByText('recorded observations. Sent only',{exact:false}).waitFor();await page.locator('#share-journey').check();
 ensure(await page.locator('#context-preview').getAttribute('open')!==null,'No preview');await page.locator('#question').fill('What have I experienced in Ashenvale?');
 let release;const gate=new Promise(r=>release=r);let posts=0;await page.route('**/local/ask',async route=>{posts++;await gate;await route.fetch();await route.abort('failed');});await page.locator('#ask-submit').evaluate(button=>{for(let i=0;i<5;i++)button.click();});ensure(await page.locator('#ask-submit').isDisabled(),'No pending acknowledgement');release();
 await page.getByRole('log').getByText('Demo data: your recorded journey passes through Ashenvale.',{exact:true}).waitFor();ensure(posts===1&&requests.length===2,'Paid request replayed');ensure(requests[1].journal.length>0&&requests[1].synthetic_journal===true,'Context missing');ensure(!JSON.stringify(requests[1]).includes('demo-installation'),'Character identity sent');ensure(!(await page.locator('#share-journey').isChecked()),'Sharing not reset');
 await page.unroute('**/local/ask');await journey(page);await companion(page);const lastAnswer=await page.locator('#messages article').last().boundingBox(),form=await page.locator('#ask-form').boundingBox();ensure(form.y-(lastAnswer.y+lastAnswer.height)<90,'Sidebar pushes question away from answer');await page.screenshot({path:path.join(root,'docs/companion-personal-ai.png'),fullPage:true});
 await page.emulateMedia({reducedMotion:'reduce'});await page.setViewportSize({width:390,height:844});ensure(!(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth)),'Mobile overflow');await page.screenshot({path:path.join(root,'docs/companion-personal-ai-mobile.png'),fullPage:true});
 const token=await page.locator('meta[name="wisp-token"]').getAttribute('content');const exported=await page.request.get(base+'/local/export',{headers:{'X-Wisp-Token':token}});ensure(!(await exported.text()).includes(access),'Access code leaked into export');
 console.log(JSON.stringify({passed:true,explicitJourneyConsent:true,noImplicitJournalUpload:true,preview:true,syntheticLabel:true,rapidClicks:5,paidCalls:2,lostReplyRecoveredWithoutReplay:true,sharingResets:true,credentialExcludedFromExport:true,mobile:true,reducedMotion:true}));
}finally{await browser?.close();if(child?.exitCode===null){const done=once(child,'exit');child.kill('SIGINT');await done;}backend.close();}
