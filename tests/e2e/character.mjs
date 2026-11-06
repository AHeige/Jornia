import {settings,companion} from './ui.mjs';
import {spawn} from 'node:child_process';
import {once} from 'node:events';
import fs from 'node:fs/promises';
import path from 'node:path';
import http from 'node:http';
import {fileURLToPath} from 'node:url';
import {chromium} from 'playwright';
const root=path.resolve(path.dirname(fileURLToPath(import.meta.url)),'../..');
const data=await fs.mkdtemp(path.join(root,'.cache/character-'));let child,browser;const asks=[];const access='c'.repeat(64);
const backend=http.createServer(async(req,res)=>{res.setHeader('content-type','application/json');if(req.headers.authorization!=='Bearer '+access){res.writeHead(401);res.end('{}');return;}if(req.url==='/healthz'){res.end('{"status":"ok"}');return;}let body='';for await(const p of req)body+=p;const q=JSON.parse(body);const now=Math.floor(Date.now()/1000);
 if(req.url==='/api/wisp/character'){res.end(JSON.stringify({name:q.name,surname:q.surname,region:q.region,level:28,race:'Night Elf',class:'Warrior',fetched_at:now,last_login:now-3600,expires_at:now+86400,items:[{id:4705,slot:'SHOULDER',name:'Synthetic Pauldrons',quality:'UNCOMMON'}],warnings:['Synthetic test profile.']}));return;}
 asks.push(q);res.end(JSON.stringify({status:'answered',text:q.character?'Your selected dated snapshot lists Synthetic Pauldrons.':'No personal context was sent.',route:'berget',model:'test-model',game_version:'forever',evidence_ids:q.character?['character:profile']:[],sources:q.character?[{id:'character:profile',title:'Selected local Armory snapshot',uri:'wisp-local:character/profile',revision:'test',validation_status:'user_imported'}]:[],warnings:['Synthetic browser test.']}));});backend.listen(0,'127.0.0.1');await once(backend,'listening');
function ensure(x,m){if(!x)throw Error(m);}
try{
 child=spawn(process.env.WISP_TEST_BINARY||path.join(root,'.cache/wisp-companion'),['-no-browser','-listen','127.0.0.1:0','-data-dir',data,'-backend','http://127.0.0.1:'+backend.address().port],{stdio:['ignore','pipe','pipe']});
 const base=await new Promise((resolve,reject)=>{child.stdout.on('data',x=>{const m=String(x).match(/Wisp: (http:\/\/[^\s]+)/);if(m)resolve(m[1]);});child.once('exit',code=>reject(Error('exit '+code)));});
 browser=await chromium.launch({headless:true});const page=await browser.newPage({viewport:{width:1280,height:1000}});await page.goto(base);await settings(page);
 await page.locator('#access-code').fill(access);await page.locator('#access-save').click();await page.getByText('Wisp is connected.',{exact:false}).waitFor();
 await page.locator('#profile-name').fill('Aster');await page.locator('#profile-surname').fill('Test');await page.locator('#profile-consent').check();await page.locator('#profile-load').click();await page.getByText('Character saved.',{exact:false}).waitFor({state:'attached'});await companion(page);
 await page.locator('#ai').check();await page.getByText('Dated class, race, level',{exact:false}).waitFor();
 await page.locator('#question').fill('Compare my saved equipment');await page.locator('#ask-submit').click();await page.getByRole('log').getByText('No personal context was sent.',{exact:true}).waitFor();ensure(!asks[0].character,'Implicit profile sharing');
 await page.locator('#share-journey').check();await page.locator('#context-preview').waitFor({state:'visible'});ensure((await page.locator('#context-events').innerText()).includes('Synthetic Pauldrons'),'Gear missing from preview');ensure((await page.locator('#context-events').innerText()).includes('not live'),'Freshness missing');
 await page.screenshot({path:path.join(data,'preview-desktop.png'),fullPage:true});
 await page.locator('#question').fill('Describe the shoulders in my dated profile');await page.locator('#ask-submit').click();await page.getByRole('log').getByText('Your selected dated snapshot lists Synthetic Pauldrons.',{exact:true}).waitFor();
 ensure(asks.length===2&&asks[1].character.items[0].id===4705,'Approved profile missing');ensure(!JSON.stringify(asks[1]).includes('Aster')&&!JSON.stringify(asks[1]).includes('armory:'),'Identity uploaded');ensure(!await page.locator('#share-journey').isChecked(),'Consent not reset');
 // Offline personal question must not issue another backend request.
 await page.locator('#ai').uncheck();await page.locator('#question').fill('My gear');await page.locator('#ask-submit').click();await page.getByRole('log').getByText('SHOULDER: Synthetic Pauldrons',{exact:false}).waitFor();ensure(asks.length===2,'Local gear question used backend');
 await page.emulateMedia({reducedMotion:'reduce'});await page.setViewportSize({width:390,height:844});await page.locator('#ai').check();await page.locator('#share-journey').check();await page.getByText('Dated class, race, level',{exact:false}).waitFor();ensure(!(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth)),'Mobile overflow');await page.screenshot({path:path.join(data,'preview-mobile.png'),fullPage:true});
 console.log(JSON.stringify({passed:true,profileOnlyAI:true,noImplicitUpload:true,exactGearPreview:true,noNameOrGUIDUpload:true,sharingResets:true,offlineGear:true,mobile:true,reducedMotion:true,screenshots:data}));
}finally{await browser?.close();if(child?.exitCode===null){const done=once(child,'exit');child.kill('SIGINT');await done;}backend.close();}
