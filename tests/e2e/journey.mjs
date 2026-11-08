import {settings,companion,journey,openImport} from './ui.mjs';
import {spawn} from 'node:child_process';
import {once} from 'node:events';
import fs from 'node:fs/promises';
import path from 'node:path';
import {fileURLToPath} from 'node:url';
import {chromium} from 'playwright';
const root=path.resolve(path.dirname(fileURLToPath(import.meta.url)),'../..');
const data=await fs.mkdtemp(path.join(root,'.cache/journey-'));
const wow=path.join(data,'WoW');const folder=path.join(wow,'WTF','Account','test','SavedVariables');await fs.mkdir(folder,{recursive:true});
const packet=JSON.parse((await fs.readFile(path.join(root,'.cache/addon-export.txt'),'utf8')).replace(/^WISP1:/,''));
const other=structuredClone(packet);other.character.id='Player-alt';other.character.name='Alt';other.events=other.events.map(e=>({...e,id:e.id.replace(packet.character.id,other.character.id),zone:'Darkshore',subzone:'',note:'Separate journey'}));
function lua(v){if(v===null)return 'nil';if(typeof v==='string')return JSON.stringify(v).replace(/\\u([0-9a-f]{4})/gi,(_,hex)=>String.fromCharCode(parseInt(hex,16)));if(Array.isArray(v))return '{'+v.map(lua).join(',')+'}';if(typeof v==='object')return '{'+Object.entries(v).map(([k,x])=>'['+lua(k)+']='+lua(x)).join(',')+'}';return String(v);}
const file=path.join(folder,'Wisp.lua');
async function save(){await fs.writeFile(file,'WispDB = '+lua({schema:1,installation_id:packet.installation_id,characters:{[packet.character.id]:{...packet.character,events:packet.events},[other.character.id]:{...other.character,events:other.events}}}));}
await save();let child,browser;
async function start(){child=spawn(path.join(root,'.cache/jornia-companion'),['-no-browser','-listen','127.0.0.1:0','-data-dir',data,'-backend','http://127.0.0.1:1'],{cwd:root,stdio:['ignore','pipe','pipe']});return new Promise((resolve,reject)=>{const timer=setTimeout(()=>reject(Error('startup')),15000);child.stderr.on('data',x=>process.stderr.write(x));child.stdout.on('data',x=>{const m=String(x).match(/Jornia: (http:\/\/[^\s]+)/);if(m){clearTimeout(timer);resolve(m[1]);}});child.once('exit',code=>reject(Error('exit '+code)));});}
async function stop(){if(child&&child.exitCode===null){const done=once(child,'exit');child.kill('SIGINT');await done;}}
function ensure(value,message){if(!value)throw Error(message);}
try{
 const base=await start();browser=await chromium.launch({headless:true});const page=await browser.newPage({viewport:{width:1200,height:1050}});await page.goto(base);await settings(page);
 ensure(await page.locator('#journal-panel').getAttribute('open')!==null,'First use not visible');ensure(!(await page.locator('#journal-consent').isChecked()),'Consent preselected');
 await page.waitForTimeout(3200);ensure(await page.locator('#character option').count()===1&&(await page.locator('#character').innerText()).includes('No characters'),'Imported before consent');
 await page.locator('#wow-root').fill(wow);await page.locator('#journal-consent').check();
 let release;const gate=new Promise(r=>release=r);let posts=0;await page.route('**/local/journal',async route=>{if(route.request().method()!=='POST'){await route.continue();return;}posts++;await gate;await route.fetch();await route.abort('failed');});
 await page.locator('#journal-enable').evaluate(button=>{for(let i=0;i<5;i++)button.click();});ensure(await page.locator('#journal-enable').isDisabled(),'No immediate acknowledgement');release();
 await page.getByText('Your choice is saved.',{exact:false}).waitFor({state:'attached'});await page.unroute('**/local/journal');ensure(posts===1,'Repeated consent requests');
 await page.getByRole('option',{name:'Alt · '+other.character.realm,exact:true}).waitFor({state:'attached'});await page.locator('#character').selectOption({label:packet.character.name+' · '+packet.character.realm});
 await journey(page);await page.getByText('2 recorded entries',{exact:false}).waitFor();
 await companion(page);await page.locator('#question').fill('Show my journey');await page.locator('#ask-submit').click();await page.getByRole('log').getByText('Your journal records',{exact:false}).waitFor();ensure(!(await page.getByRole('log').innerText()).includes('Separate journey'),'History crossed characters');
 await journey(page);const last=packet.events.at(-1);packet.events.push({...last,id:packet.installation_id+':'+packet.character.id+':3',kind:'level_up',observed_at:last.observed_at+1,level:25,note:''});await save();await page.getByText('3 recorded entries',{exact:false}).waitFor({timeout:15000});
 await settings(page);if(await page.locator('#journal-panel').getAttribute('open')===null)await page.locator('#journal-panel > summary').click();await page.locator('#journal-pause').click();await page.locator('#journal-action').filter({hasText:'Reading is paused.'}).waitFor();packet.events.push({...last,id:packet.installation_id+':'+packet.character.id+':4',kind:'zone',observed_at:last.observed_at+2,zone:'Felwood',subzone:'',level:25,note:''});await save();await page.waitForTimeout(3500);ensure(!(await page.locator('#journey-summary').innerText()).includes('4 recorded entries'),'Paused importer read file');
 await page.reload();await settings(page);await page.getByText('Journal reading is paused.',{exact:false}).waitFor();await settings(page);if(await page.locator('#journal-panel').getAttribute('open')===null)await page.locator('#journal-panel > summary').click();await page.locator('#journal-consent').check();await page.locator('#journal-enable').click();await journey(page);await page.getByText('4 recorded entries',{exact:false}).waitFor({timeout:15000});
 await page.screenshot({path:path.join(root,'docs/companion-journey.png'),fullPage:true});
 await page.emulateMedia({reducedMotion:'reduce'});await page.setViewportSize({width:390,height:844});ensure(!(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth)),'Mobile overflow');await page.screenshot({path:path.join(root,'docs/companion-journey-mobile.png'),fullPage:true});
 await stop();const restarted=await start();await page.goto(restarted);await page.locator('#character').selectOption({label:packet.character.name+' · '+packet.character.realm});await journey(page);await page.getByText('4 recorded entries',{exact:false}).waitFor();await companion(page);await page.locator('#question').fill('Show my journey');await page.locator('#ask-submit').click();await page.getByRole('log').getByText('4 entries',{exact:false}).waitFor();
 console.log(JSON.stringify({passed:true,initialConsent:true,initialMultiCharacterImport:true,rapidSettingsClicks:5,settingsRequests:posts,lostSettingsReplyRecovered:true,automaticFileUpdate:true,pause:true,consentAndGraphSurviveRestart:true,journeyAnswersOffline:true,characterIsolation:true,mobileOverflow:false,dataDirectory:data},null,2));
}finally{await browser?.close();await stop();}
