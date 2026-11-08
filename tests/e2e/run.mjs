import {settings,companion,journey,openImport} from './ui.mjs';
import {spawn} from 'node:child_process';
import {once} from 'node:events';
import fs from 'node:fs/promises';
import path from 'node:path';
import {fileURLToPath} from 'node:url';
import {chromium} from 'playwright';

const root=path.resolve(path.dirname(fileURLToPath(import.meta.url)),'../..');
const data=await fs.mkdtemp(path.join(root,'.cache/e2e-'));
const packet=await fs.readFile(path.join(root,'.cache/addon-export.txt'),'utf8');
let child,browser;
async function start(backend=process.env.WISP_API_URL||'http://127.0.0.1:8090') {
 child=spawn(path.join(root,'.cache/jornia-companion'),['-no-browser','-listen','127.0.0.1:0','-data-dir',data,'-backend',backend],{cwd:root,stdio:['ignore','pipe','pipe']});
 return await new Promise((resolve,reject)=>{
  const timer=setTimeout(()=>reject(new Error('Companion startup timed out')),15000);
  child.once('exit',code=>{clearTimeout(timer);reject(new Error('Companion exited '+code));});
  child.stderr.on('data',buffer=>process.stderr.write(buffer));
  child.stdout.on('data',buffer=>{const match=String(buffer).match(/Jornia: (http:\/\/[^\s]+)/);if(match){clearTimeout(timer);resolve(match[1]);}});
 });
}
async function stop(){if(child&&child.exitCode===null){const done=once(child,'exit');child.kill('SIGINT');await done;}}
function ensure(condition,message){if(!condition)throw new Error(message);}
try {
 const base=await start();browser=await chromium.launch({headless:true});const page=await browser.newPage({viewport:{width:1200,height:900}});
 await page.goto(base);
 await settings(page);await page.locator('#connection-check').click();await page.getByText('World server connected.',{exact:false}).waitFor();
 let demoRelease;const demoGate=new Promise(resolve=>demoRelease=resolve);let demoPosts=0;
 await page.route('**/local/demo',async route=>{demoPosts++;await demoGate;await route.fetch();await route.abort('failed');});
 await page.locator('#demo-start').evaluate(button=>{for(let i=0;i<5;i++)button.click();});
 ensure(await page.locator('#demo-start').isDisabled(),'Demo has no immediate acknowledgement');
 demoRelease();await page.locator('#demo-banner').waitFor({state:'visible'});ensure(demoPosts===1,'Demo activation replayed');await page.unroute('**/local/demo');
 await companion(page);await page.locator('#question').fill('What was I doing last time?');await page.locator('#ask-submit').click();await page.getByText('Demo data — invented history.',{exact:false}).waitFor();
 await settings(page);await page.locator('#demo-start').click();await page.getByText('Demo journal ready.',{exact:false}).waitFor({state:'attached'});await companion(page);ensure(await page.getByRole('log').locator('article').count()===1,'Reactivating demo reset conversations');
 await page.reload();await companion(page);await page.locator('#demo-banner').waitFor({state:'visible'});
 await page.screenshot({path:path.join(root,'docs/companion-demo.png'),fullPage:true});
 await page.emulateMedia({reducedMotion:'reduce'});await page.setViewportSize({width:390,height:844});ensure(!(await page.evaluate(()=>document.documentElement.scrollWidth>window.innerWidth)),'Demo mobile overflow');await page.screenshot({path:path.join(root,'docs/companion-demo-mobile.png'),fullPage:true});await page.setViewportSize({width:1200,height:900});await page.emulateMedia({reducedMotion:'no-preference'});
 await openImport(page);

 await page.locator('#export-data').fill(packet);
 await page.locator('#import-submit').click();
 await page.getByText('2 new entries saved. 0 already recorded.',{exact:true}).waitFor();
 let release;const gate=new Promise(resolve=>{release=resolve});let posts=0;
 await page.route('**/local/ask',async route=>{posts++;await gate;await route.continue();});
 await companion(page);await page.locator('#question').fill('Where is Astranaar?');
 await page.locator('#ask-form').evaluate(form=>{for(let i=0;i<5;i++)form.requestSubmit();});
 ensure(await page.locator('#ask-submit').isDisabled(),'Missing immediate pending state');
 ensure(await page.locator('#pending').isVisible(),'Missing visible pending acknowledgement');
 release();await page.getByText('Astranaar is in Ashenvale.',{exact:true}).waitFor();ensure(posts===1,'Duplicate submission reached local transport');await page.unroute('**/local/ask');
 const ask=async question=>{await companion(page);await page.locator('#question').fill(question);await page.locator('#ask-submit').click();await page.locator('#ask-submit').waitFor({state:'visible'});};
 await ask('What was I doing last time?');await page.getByText('Your last recorded observation',{exact:false}).waitFor();
 ensure(await page.getByRole('log').getByText('I want to talk to "Raene" — no spoilers.',{exact:false}).isVisible(),'Imported note not retrieved');
 await page.reload();await companion(page);await page.getByText('Your last recorded observation',{exact:false}).waitFor();
 await page.screenshot({path:path.join(root,'docs/companion-desktop.png'),fullPage:true});
 await page.setViewportSize({width:390,height:844});ensure(!(await page.evaluate(()=>document.documentElement.scrollWidth>window.innerWidth)),'Mobile overflow');
 await page.screenshot({path:path.join(root,'docs/companion-mobile.png'),fullPage:true});await page.setViewportSize({width:1200,height:900});
 await openImport(page);await page.locator('#export-data').fill(packet);await page.locator('#import-submit').click();
 await page.getByText('0 new entries saved. 2 already recorded.',{exact:true}).waitFor();
 await page.route('**/local/ask',route=>route.fulfill({status:400,contentType:'application/json',body:JSON.stringify({error:'Test: rejected question'})}));
 await ask('Keep my draft');await page.getByRole('alert').filter({hasText:'Test: rejected question'}).waitFor();ensure(await page.locator('#question').inputValue()==='Keep my draft','Draft lost on rejection');await page.unroute('**/local/ask');
 let attempts=0;await page.route('**/local/ask',async route=>{attempts++;await route.fetch();await route.abort('failed');});
 await ask('Where is Mystral Lake?');await page.getByText('Mystral Lake is in Ashenvale.',{exact:true}).waitFor();ensure(attempts===1,'Lost response caused retry');await page.unroute('**/local/ask');
 const alt=JSON.parse(packet.replace(/^WISP1:/,''));alt.character.id='Player-1-alt';alt.character.name='Alt-test';for(const e of alt.events){e.id=e.id.replace('Player-1-test','Player-1-alt');e.zone='Darkshore';if(e.note)e.note='A separate character’s private note.';}
 await openImport(page);
 await page.locator('#export-data').fill('WISP1:'+JSON.stringify(alt));await page.locator('#import-submit').click();await page.getByText('2 new entries saved. 0 already recorded.',{exact:true}).waitFor();
 await ask('What was I doing last time?');await page.getByRole('log').getByText('A separate character’s private note.',{exact:false}).waitFor();ensure(!(await page.getByRole('log').innerText()).includes('Raene'),'Character history leaked');
 await stop();const offline=await start('http://127.0.0.1:1');await page.goto(offline);
 await page.locator('#character').selectOption({label:'TestPlayer · Testrealm'});await companion(page);
 await page.getByRole('log').getByText('I want to talk to "Raene" — no spoilers.',{exact:false}).waitFor();
 await settings(page);await page.locator('#connection-check').click();await page.getByText('The server is unreachable.',{exact:false}).waitFor();
 const before=await page.getByRole('log').locator('article').count();await ask('What was I doing last time?');
 await page.waitForFunction(count=>document.querySelectorAll('#messages article').length>count,before);
 console.log(JSON.stringify({passed:true,demoFiveActivations:5,demoTransportRequests:demoPosts,demoLostReplyRecovered:true,demoLabelSurvivesReload:true,rapidSubmissions:5,actualAskRequests:posts,lostResponseAttempts:attempts,reimportDeduplicated:true,characterIsolation:true,historySurvivesProcessRestart:true,offlineMemory:true,mobileOverflow:false,dataDirectory:data},null,2));
} finally {await browser?.close();await stop();}
