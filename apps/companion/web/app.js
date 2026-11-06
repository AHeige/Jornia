'use strict';
const $ = id => document.getElementById(id);
const token = document.querySelector('meta[name="wisp-token"]').content;
let state = {characters:[],events:{},messages:[],operations:[]};
let selected = localStorage.getItem('wisp-character') || '';
let lastReceipt = null;
const id = () => crypto.randomUUID();

// One shared acknowledged-mutation runtime: synchronous single-flight, stable IDs,
// persisted receipts, no automatic replay, and read-only recovery after lost replies.
class Mutations {
  flights = new Map();
  async read(path) {const r = await fetch(path,{headers:{'X-Wisp-Token':token}});if(!r.ok)throw new Error('Could not read local state. Reopen Wisp.');return r.json();}
  async submit(key,path,payload,onPending) {
    if(this.flights.has(key))return this.flights.get(key);
    onPending?.(true);
    const operation = (async()=>{
      const mutation = {...payload,mutation_id:id()};
      sessionStorage.setItem('wisp-pending-'+key,JSON.stringify({id:mutation.mutation_id,key}));
      let receipt;
      try {
        const r = await fetch(path,{method:'POST',headers:{'Content-Type':'application/json','X-Wisp-Token':token},body:JSON.stringify(mutation)});
        const body = await r.json();if(!r.ok){if(r.status<500)sessionStorage.removeItem('wisp-pending-'+key);throw new Error(body.error||'The request could not be confirmed.');}receipt=body;
      } catch(error) {
        if(sessionStorage.getItem('wisp-pending-'+key)) {
          try {receipt=await this.read('/local/operation?id='+encodeURIComponent(mutation.mutation_id));}
          catch {throw new Error('The result is uncertain. Reload to check saved state. This request will not be sent again automatically.');}
        }else throw error;
      }
      const deadline=Date.now()+60000;
      while(receipt.status==='pending') {if(Date.now()>deadline)throw new Error('Still waiting for confirmation. Reload to check saved state; the question will not be sent again.');await new Promise(resolve=>setTimeout(resolve,400));receipt=await this.read('/local/operation?id='+encodeURIComponent(mutation.mutation_id));}
      sessionStorage.removeItem('wisp-pending-'+key);
      if(receipt.status!=='completed')throw new Error(receipt.error||'The result could not be confirmed.');
      return receipt;
    })();
    this.flights.set(key,operation);
    try{return await operation;}finally{if(this.flights.get(key)===operation)this.flights.delete(key);onPending?.(false);}
  }
}
const mutations = new Mutations();
const element = (tag,classes,text) => {const node=document.createElement(tag);node.className=classes||'';if(text!==undefined)node.textContent=text;return node;};
const localTime = value => new Date(value*1000).toLocaleString('en-GB');
function showError(message){$('chat-error').textContent=message;$('chat-error').classList.toggle('hidden',!message);}
function sourceExpiry(answer){return answer.sources?.some(source=>source.expires_at);}
function answerNode(question,answer) {
  const group=element('article','space-y-3');
  const user=element('div','chat chat-end');user.append(element('div','chat-bubble whitespace-pre-wrap',question));
  const ai=element('div','chat chat-start');const content=element('div','max-w-full');
  content.append(element('div','chat-header mb-1 text-sm text-base-content/65',answer.route==='local_memory'?'Wisp · your journal':answer.sources?.some(s=>s.id.startsWith('memory:'))?'Wisp · your journey & world knowledge':'Wisp · world knowledge'));
  content.append(element('div','chat-bubble whitespace-pre-wrap leading-relaxed',answer.text));
  if(answer.sources?.length){const details=element('details','mt-2 text-xs text-base-content/70');details.append(element('summary','cursor-pointer',answer.route==='local_memory'?'View journal evidence':'View sources'));const list=element('ul','mt-2 space-y-2');for(const source of answer.sources){const item=element('li');if(/^https:\/\//.test(source.uri)){const link=element('a','underline',source.title);link.href=source.uri;link.target='_blank';link.rel='noopener noreferrer';item.append(link);}else if(source.uri?.startsWith('wisp-local:event/')){const evidence=element('button','text-link',source.title+' →');evidence.type='button';evidence.addEventListener('click',()=>{activeFilter='all';renderTimeline();setView('journey',true);const eventId=decodeURIComponent(source.uri.slice('wisp-local:event/'.length));const row=[...document.querySelectorAll('[data-event-id]')].find(e=>e.dataset.eventId===eventId);if(row){row.focus();row.scrollIntoView({block:'center'});}});item.append(evidence);}else item.textContent=source.title+' · '+source.revision;list.append(item);}details.append(list);if(answer.warnings?.length)details.append(element('p','mt-2 leading-relaxed',answer.warnings.join(' ')));if(sourceExpiry(answer))details.append(element('p','mt-2 leading-relaxed','Time-limited world source. This answer is removed after its source expires and Wisp is used again. Your journal is unaffected.'));content.append(details);}
  if(answer.model){const meta=element('p','answer-tech mt-2',answer.model+' · '+(answer.estimated_cost_eur==null?'cost unknown':'≈ €'+answer.estimated_cost_eur.toFixed(6)));const evidence=content.querySelector('details');if(evidence)evidence.append(meta);else{const details=element('details');details.append(element('summary','cursor-pointer','Answer details'),meta);content.append(details);}}
  ai.append(content);group.append(user,ai);return group;
}
let currentView='journey', activeFilter='all', currentJourney={events:[],total:0};
const views=['journey','companion','characters','settings'];
function setView(view,focus=false){
 if(!views.includes(view))return;
 currentView=view;
 for(const name of views)$('view-'+name).classList.toggle('hidden',name!==view);
 for(const button of document.querySelectorAll('[data-view]')){if(button.dataset.view===view)button.setAttribute('aria-current','page');else button.removeAttribute('aria-current');}
 $('view-label').textContent=({journey:'Your field journal',companion:'A little company',characters:'Choose your story',settings:'Camp & connection'})[view];
 if(focus){window.scrollTo(0,0);$('main').focus({preventScroll:true});}
}
function chooseCharacter(value){
 sessionStorage.setItem('wisp-draft:'+selected,$('question').value);
 selected=value;lastReceipt=null;activeFilter='all';$('question').value=sessionStorage.getItem('wisp-draft:'+selected)||'';
 showError('');resetContext();render();
 currentJourney={events:[],total:0};renderTimeline();
 refreshJourney().catch(error=>showError(error.message));
}
function renderRoster(){
 const roster=$('character-roster');roster.replaceChildren();
 $('roster-empty').classList.toggle('hidden',state.characters.length>0);
 for(const c of state.characters){
  const latest=(state.events[c.id]||[])[0];const card=element('button','roster-card');card.type='button';card.setAttribute('aria-pressed',String(c.id===selected));card.setAttribute('aria-label','Choose '+c.name+' on '+c.realm);
  const top=element('div','roster-top');top.append(element('span','',c.is_demo?'DEMO CHARACTER':c.discovered?'FOUND IN GAME FILES':'CHARACTER JOURNAL'),element('span','',c.id===selected?'SELECTED':'OPEN →'));card.append(top,element('h2','',c.name),element('p','roster-realm',c.realm),element('p','roster-place',latest?'Level '+latest.level+' / '+(latest.subzone||latest.zone):'No observations yet'));
  card.addEventListener('click',()=>{chooseCharacter(c.id);setView('journey',true);});roster.append(card);
 }
}
function render(){
 const options=state.characters.map(c=>[c.id,(c.is_demo?'Demo · ':'')+c.name+' · '+c.realm]);if(!options.some(([value])=>value===selected)){const recent=[...state.characters].sort((a,b)=>((state.events[b.id]||[])[0]?.observed_at||state.profiles?.[b.id]?.last_login||0)-((state.events[a.id]||[])[0]?.observed_at||state.profiles?.[a.id]?.last_login||0));selected=recent[0]?.id||'';}
 $('character').replaceChildren();for(const [value,label] of options){const option=element('option','',label);option.value=value;$('character').append(option);}if(!options.length){const option=element('option','','No characters yet');option.value='';$('character').append(option);} $('character').value=selected;localStorage.setItem('wisp-character',selected);if(!$('question').value)$('question').value=sessionStorage.getItem('wisp-draft:'+selected)||'';
 const character=state.characters.find(c=>c.id===selected);
 $('demo-banner').classList.toggle('hidden',!character?.is_demo);
 const events=state.events[selected]||[];const latest=events[0];
 $('welcome').classList.toggle('hidden',!!character);$('character-journey').classList.toggle('hidden',!character);
 $('character-context').textContent=latest?'Level '+latest.level+' · '+(latest.subzone||latest.zone)+' · last saved observation':character?.discovered?'Found in WoW files · no recorded adventures yet':'Connect a journal to begin.';
 $('hero-character').textContent=character?.name||'';$('hero-crest').textContent=Array.from(character?.name||'W')[0];$('hero-realm').textContent=(character?.is_demo?'Demo journal / ':'')+(character?.realm||'');
 $('hero-level').textContent=latest?'Level '+latest.level:'Level unknown';$('hero-place').textContent=latest?(latest.subzone||latest.zone):'No places recorded';
 $('last-seen').textContent=latest?'Last recorded '+localTime(latest.observed_at)+' · '+latest.zone+' · not live':'Your first recorded session will appear here.';
 $('chat-character').textContent=character?'With '+character.name+' · '+character.realm:'World questions are welcome, too.';
 $('chat-place').textContent=latest?(latest.subzone||latest.zone):'Your next chapter.';$('chat-memory').textContent=latest?'Last recorded at level '+latest.level+' in '+latest.zone+'. '+localTime(latest.observed_at)+'.':"Connect a character to bring your recorded journey into the conversation.";
 $('backend').textContent='World server: '+state.backend;
 const messages=state.messages.filter(m=>m.character_id===selected).toReversed();$('messages').replaceChildren();for(const m of messages)$('messages').append(answerNode(m.question,m.answer));
 if(lastReceipt?.character_id===selected&&!messages.some(m=>m.id===lastReceipt.id))$('messages').append(answerNode(lastReceipt.question,lastReceipt.answer));
 $('empty').classList.toggle('hidden',messages.length>0||lastReceipt?.character_id===selected);
 $('forget').disabled=!selected;
 const pending=state.operations.find(op=>op.status==='needs_attention'&&op.character_id===selected);if(pending){showError(pending.error);if(!$('question').value)$('question').value=pending.question||'';}
 renderRoster();renderProfile();
}
let refreshVersion=0;
async function refresh(){const version=++refreshVersion;const next=await mutations.read('/local/state');if(version!==refreshVersion)return;state=next;render();await refreshJourney();}
$('question').addEventListener('input',()=>sessionStorage.setItem('wisp-draft:'+selected,$('question').value));
$('character').addEventListener('change',()=>chooseCharacter($('character').value));
$('export-data').addEventListener('input',()=>{
 try{const pack=JSON.parse($('export-data').value.trim().replace(/^WISP1:/,''));$('import-preview').textContent=`${pack.character.name} · ${pack.character.realm} · ${pack.events.length} entries. Stored only on this computer.`;}catch{$('import-preview').textContent='';}
});
$('import-form').addEventListener('submit',async event=>{event.preventDefault();$('import-status').textContent='Saving your journal…';try{
 const receipt=await mutations.submit('import','/local/import',{export:$('export-data').value},busy=>{$('import-submit').disabled=busy;});selected=receipt.result.character_id;await refresh();$('export-data').value='';$('import-preview').textContent='';$('import-status').textContent=`${receipt.result.inserted} new entries saved. ${receipt.result.duplicates} already recorded.`;$('import-panel').open=true;
 }catch(error){$('import-status').textContent=error.message;}});
$('ask-form').addEventListener('submit',async event=>{event.preventDefault();const question=$('question').value.trim();if(!question)return;showError('');const character=selected;
 try{const receipt=await mutations.submit('ask','/local/ask',{character_id:character,question,ai:$('ai').checked,...contextPayload()},busy=>{$('ask-submit').disabled=busy;$('pending').classList.toggle('hidden',!busy);});lastReceipt={id:receipt.id,character_id:character,question,answer:receipt.result};if(selected===character)resetContext();if(selected===character&&$('question').value.trim()===question){$('question').value='';sessionStorage.removeItem('wisp-draft:'+character);}await refresh();$('question').focus();}
 catch(error){if(selected===character)showError(error.message);}
});
for(const button of document.querySelectorAll('[data-question]'))button.addEventListener('click',()=>{$('question').value=button.dataset.question;sessionStorage.setItem('wisp-draft:'+selected,$('question').value);$('question').focus();});
$('download').addEventListener('click',async()=>{try{const data=await mutations.read('/local/export');const url=URL.createObjectURL(new Blob([JSON.stringify(data,null,2)],{type:'application/json'}));const link=element('a');link.href=url;link.download='wisp-history.json';link.click();setTimeout(()=>URL.revokeObjectURL(url),1000);}catch(error){$('privacy-status').textContent=error.message;}});
$('forget').addEventListener('click',async()=>{if(!selected||!confirm('Permanently delete this character’s local journal and conversations? Automatic reimport will be blocked. The original addon file is unaffected. A deliberate manual import allows it again. Export first to keep a copy.'))return;
 try{await mutations.submit('forget','/local/forget',{character_id:selected},busy=>{$('forget').disabled=busy;});lastReceipt=null;selected='';await refresh();$('privacy-status').textContent='This character’s local history has been deleted.';}catch(error){$('privacy-status').textContent=error.message;}

});
(async()=>{try{await refresh();for(const op of state.operations.filter(o=>o.status==='pending')){while((await mutations.read('/local/operation?id='+encodeURIComponent(op.id))).status==='pending')await new Promise(r=>setTimeout(r,500));await refresh();}}catch(error){showError(error.message);}})();

// Acknowledged import through the existing shared mutation runtime; activation never resets history.
$('demo-start').addEventListener('click',async()=>{
 $('demo-status').textContent='Preparing the demo journal…';$('welcome-status').textContent='Preparing your demo journal…';
 try {const receipt=await mutations.submit('demo','/local/demo',{},busy=>{$('demo-start').disabled=busy;});selected=receipt.result.character_id;lastReceipt=null;$('question').value='';showError('');await refresh();$('demo-status').textContent='Demo journal ready. Explore your journey or talk to Wisp.';setView('journey',true);}
 catch(error){$('demo-status').textContent=error.message;$('welcome-status').textContent=error.message;}
});
$('connection-check').addEventListener('click',async()=>{
 $('connection-check').disabled=true;$('connection-status').textContent='Checking the connection…';
 try {const result=await mutations.read('/local/connection');$('connection-status').textContent=result.connected?'World server connected. AI is used when you send an AI question.':'The server is unreachable. Your local journal and memory questions still work.';}
 catch(error){$('connection-status').textContent=error.message;}
 finally{$('connection-check').disabled=false;}
});

// Settings use the same single-flight acknowledged runtime as import and chat.
let journalRevision=0,journalLoaded=false,journalBusy=false;
async function refreshJournal(){
 const status=await mutations.read('/local/journal');const config=status.config;
 if(!journalLoaded){journalRevision=config.revision;$('wow-root').value=config.root||'';$('journal-consent').checked=config.consent_version===2;journalLoaded=true;$('journal-panel').open=config.revision===0;}
 $('journal-pause').disabled=journalBusy||!config.enabled;
 $('journal-status').textContent=(config.enabled?'Journal reading is active. ':'Journal reading is paused. ')+(status.last_import?'Last import: '+localTime(status.last_import)+'. ':'')+(config.enabled?status.characters+' character folders · '+status.files+' Wisp journals. ':'')+(status.error||'');
}
async function configureJournal(enabled){
 const firstImport=state.characters.length===0;
 $('journal-action').textContent=enabled?'Saving your choice…':'Pausing journal reading…';
 try{const receipt=await mutations.submit('journal','/local/journal',{root:$('wow-root').value,enabled,consent_version:enabled?2:0,expected_revision:journalRevision},busy=>{journalBusy=busy;$('journal-enable').disabled=busy;$('journal-pause').disabled=busy;});journalRevision=receipt.result.revision;$('journal-panel').open=false;$('journal-action').textContent=enabled?'Your choice is saved. Existing Wisp journals will be read automatically.':'Reading is paused. Your saved history is kept. Pause in-game recording with /wisp stop.';await refreshJournal();await refresh();if(firstImport&&state.characters.length)setView('characters',true);}
 catch(error){$('journal-action').textContent=error.message;}
}
$('journal-form').addEventListener('submit',event=>{event.preventDefault();if(!$('journal-consent').checked)return;configureJournal(true);});
$('journal-pause').addEventListener('click',()=>configureJournal(false));
function renderTimeline(){
 const journey=currentJourney;
 $('stat-places').textContent=journey.places||0;$('stat-quests').textContent=journey.completed_quests||0;$('stat-events').textContent=journey.total||0;
 $('journey-summary').textContent=journey.total?journey.total+' recorded entries · '+journey.places+' places · '+journey.completed_quests+' completed quest IDs.':'';
 const list=$('journey-events');list.replaceChildren();let day='';
 $('timeline-empty').querySelector('h3').textContent=journey.total?'No entries here yet.':'Your first chapter is waiting.';$('timeline-empty').querySelector('p').textContent=journey.total?'Choose another filter or let your next session add to the story.':'Your character is here. Connect the Wisp addon to start recording a journey.';$('timeline-connect').classList.toggle('hidden',!!journey.total);
 const entries=(journey.events||[]).filter(e=>activeFilter==='all'||(activeFilter==='place'?['zone','snapshot','session_start','session_end'].includes(e.kind):e.kind===activeFilter));
 for(const e of entries){
  const date=new Date(e.observed_at*1000).toLocaleDateString('en-GB',{day:'numeric',month:'long',year:'numeric'});
  const item=element('li');if(date!==day){item.append(element('p','timeline-day',date));day=date;}
  const row=element('article','journal-entry');row.dataset.kind=e.kind;row.dataset.eventId=e.id;row.tabIndex=-1;
  const labels={quest_completed:['◆','Quest #'+e.quest_id+' completed'],level_up:['↑','Reached level '+e.level],note:['✎','A note to remember'],session_start:['◈','A new session'],session_end:['◈','Called it a day'],zone:['◇','Arrived in '+(e.subzone||e.zone)],snapshot:['◇','A moment in '+(e.subzone||e.zone)]};
  const [icon,title]=labels[e.kind]||['◇','Recorded observation'];const glyph=element('span','event-icon',icon);glyph.setAttribute('aria-hidden','true');
  const meta=element('p','entry-meta');meta.append(element('time','',new Date(e.observed_at*1000).toLocaleTimeString('en-GB',{hour:'2-digit',minute:'2-digit'})),element('span','','Level '+e.level));
  row.append(glyph,meta,element('h3','entry-title',title),element('p','entry-place',e.zone+(e.subzone?' / '+e.subzone:'')));
  if(e.note)row.append(element('p','entry-note',e.note));item.append(row);list.append(item);
 }
 $('timeline-empty').classList.toggle('hidden',entries.length>0);
 for(const button of document.querySelectorAll('[data-filter]'))button.setAttribute('aria-pressed',String(button.dataset.filter===activeFilter));
 const trail=$('place-trail');trail.replaceChildren();const places=[];
 for(const e of [...(journey.events||[])].reverse()){
  const name=e.zone+(e.subzone?' / '+e.subzone:'');if(name&&places.at(-1)?.name!==name)places.push({name,level:e.level});
 }
 for(const place of places.slice(-6)){const li=element('li','',place.name);li.append(element('small','','Recorded at level '+place.level));trail.append(li);}
 if(!places.length)trail.append(element('li','','No places recorded yet.'));
}
let journeyGeneration=0;
async function refreshJourney(){
 const version=++journeyGeneration,character=selected;const journey=await mutations.read('/local/journey?character_id='+encodeURIComponent(character));if(selected!==character||version!==journeyGeneration)return;
 currentJourney=journey;renderTimeline();
}
for(const button of document.querySelectorAll('[data-view]'))button.addEventListener('click',event=>{event.preventDefault();setView(button.dataset.view,true);});
for(const button of document.querySelectorAll('[data-setup]'))button.addEventListener('click',()=>{setView('settings',true);$('journal-panel').open=true;$('wow-root').focus();});
for(const button of document.querySelectorAll('[data-demo]'))button.addEventListener('click',()=>{$('demo-start').click();button.disabled=true;const pending=mutations.flights.get('demo');pending?.finally(()=>{button.disabled=false;}).catch(()=>{});});
for(const button of document.querySelectorAll('[data-talk]'))button.addEventListener('click',()=>{setView('companion',true);if(!$('question').value)$('question').value='What was I doing last time?';$('question').focus();});
for(const button of document.querySelectorAll('[data-filter]'))button.addEventListener('click',()=>{activeFilter=button.dataset.filter;renderTimeline();});
(async()=>{try{await refreshJournal();}catch(error){$('journal-status').textContent=error.message;}})();
// Read-only background reconciliation; no automatic inference or mutation replay.
setInterval(async()=>{if(document.hidden)return;try{await refreshJournal();const version=++refreshVersion;const next=await mutations.read('/local/state');if(version!==refreshVersion)return;if(JSON.stringify(next.events)!==JSON.stringify(state.events)||JSON.stringify(next.characters)!==JSON.stringify(state.characters)||JSON.stringify(next.profiles)!==JSON.stringify(state.profiles)){const firstImport=state.characters.length===0;state=next;render();await refreshJourney();if(firstImport&&state.characters.length&&currentView==='settings')setView('characters',true);}}catch{/* Retain the last confirmed journal while offline. */}},3000);

let contextPreview=null,contextCharacter='',contextGeneration=0;
function resetContext(){contextGeneration++;contextPreview=null;contextCharacter='';$('share-journey').checked=false;$('context-preview').classList.add('hidden');$('context-events').replaceChildren();$('context-status').textContent=$('ai').checked&&selected?'Review excerpts before including them in your next question.':'Choose AI and a character to review excerpts.';$('share-journey').disabled=!$('ai').checked||!selected;}
function contextPayload(){
 if(!$('share-journey').checked)return {};
 if(!$('ai').checked||contextCharacter!==selected||!contextPreview)throw Error('Review the journal excerpts before sending.');
 const ids=contextPreview.event_ids;
 if(!ids.length)throw Error('Choose journal excerpts or turn sharing off.');
 // The digest approves the exact displayed set for this question only.
 return {context_ids:ids,context_digest:contextPreview.digest};
}
async function loadContext(){
 const version=++contextGeneration,character=selected;
 if(!$('ai').checked||!character){resetContext();return;}
 $('share-journey').disabled=true;$('context-status').textContent='Reading local journal excerpts…';
 try{const preview=await mutations.read('/local/context?character_id='+encodeURIComponent(character));if(version!==contextGeneration||selected!==character)return;contextPreview=preview;contextCharacter=character;$('context-events').replaceChildren();
 for(let i=0;i<preview.events.length;i++){const e=preview.events[i],label=element('label','flex items-start gap-2');label.append(element('span','',`${localTime(e.observed_at)} · ${({snapshot:'Observation',zone:'Place',level_up:'Level gained',quest_completed:'Quest completed',session_start:'Session started',session_end:'Session ended'})[e.kind]||'Observation'} · level ${e.level} · ${e.zone}${e.subzone?' / '+e.subzone:''}${e.quest_id?' · quest #'+e.quest_id:''}`));$('context-events').append(label);}
 $('context-status').textContent=preview.events.length?`${preview.synthetic?'Demo data. ':''}${preview.events.length} recorded observations. Sent only if you include them for this question.`:'No shareable journal excerpts yet.';$('context-preview').classList.toggle('hidden',!preview.events.length);
 }catch(error){$('context-status').textContent=error.message;}finally{$('share-journey').disabled=false;}
}
$('ai').addEventListener('change',()=>{resetContext();loadContext();});
$('share-journey').addEventListener('change',()=>{if($('share-journey').checked){$('context-preview').open=true;if(!contextPreview||contextCharacter!==selected)loadContext();}});

$('access-form').addEventListener('submit',async event=>{event.preventDefault();$('access-status').textContent='Checking your invite code…';try{await mutations.submit('access','/local/access',{token:$('access-code').value.trim()},busy=>{$('access-save').disabled=busy;});$('access-code').value='';$('access-status').textContent='Wisp is connected. You can now ask AI questions.';$('connection-status').textContent='Invite code and connection verified.';}catch(error){$('access-status').textContent=error.message;}});

// Armory is explicit, dated personal context. It is never included in AI excerpts.
function renderProfile(){
 const profiles=state.profiles||{},profile=profiles[selected],image=$('character-render');
 image.classList.toggle('hidden',!profile?.image);if(profile?.image){if(image.getAttribute('src')!==profile.image)image.src=profile.image;image.alt=profile.name+' '+profile.surname+' — official Blizzard character render';}else image.removeAttribute('src');
 $('portrait-credit').textContent=profile?.image?'Character render © Blizzard · scenery is concept art':'Concept artwork · not your character';
 $('equipment-panel').classList.toggle('hidden',!profile);$('profile-invite').classList.toggle('hidden',!!profile);
 if(profile){
  $('character-context').textContent='Level '+profile.level+' · '+profile.race+' '+profile.class+' · Armory snapshot';
  const latest=(state.events[selected]||[])[0];
  if(!latest){$('hero-level').textContent='Level '+profile.level;$('hero-place').textContent=profile.race+' '+profile.class;$('last-seen').textContent='Blizzard profile · last login '+localTime(profile.last_login)+' · not live';}
  $('equipment-title').textContent=profile.race+' '+profile.class+' / '+profile.items.length+' equipped items';
  $('profile-freshness').textContent='Blizzard Forever / '+profile.region.toUpperCase()+' · fetched '+localTime(profile.fetched_at)+'. Profile last login '+localTime(profile.last_login)+'.';
  const grid=$('equipment-items');grid.replaceChildren();for(const item of profile.items){const entry=element('article','equipment-item');entry.dataset.quality=item.quality;entry.append(element('p','item-slot',item.slot),element('h3','item-name',item.name),element('p','item-id','Item #'+item.id));grid.append(entry);}
  $('profile-warning').textContent=profile.warnings?.join(' ')||'Equipment reflects the published profile and may lag the game.';
 }
 const target=$('profile-target'),previous=target.value;target.replaceChildren();const add=element('option','','Add a character');add.value='';target.append(add);for(const c of state.characters.filter(c=>!c.is_demo)){const option=element('option','',c.name+' / '+c.realm);option.value=c.id;target.append(option);}target.value=[...target.options].some(o=>o.value===previous)?previous:'';
 for(const card of document.querySelectorAll('.roster-card')){const c=state.characters.find(c=>card.getAttribute('aria-label')==='Choose '+c.name+' on '+c.realm);const p=c&&profiles[c.id];if(p?.avatar){const avatar=element('img','roster-avatar');avatar.src=p.avatar;avatar.alt='';avatar.referrerPolicy='no-referrer';card.prepend(avatar);if(!(state.events[c.id]||[]).length)card.querySelector('.roster-place').textContent='Level '+p.level+' / '+p.race+' '+p.class;}}
}
$('character-render').addEventListener('error',()=>{$('character-render').classList.add('hidden');$('portrait-credit').textContent='Blizzard portrait unavailable · scenery is concept art';});
for(const button of document.querySelectorAll('[data-profile-setup]'))button.addEventListener('click',()=>{setView('settings',true);$('profile-panel').open=true;const c=state.characters.find(c=>c.id===selected),p=state.profiles?.[selected];if(c&&!c.is_demo){$('profile-target').value=c.id;const parts=c.name.split(' ');$('profile-name').value=p?.name||parts[0];$('profile-surname').value=p?.surname||parts.slice(1).join(' ');if(p)$('profile-region').value=p.region;}$('profile-name').focus();});
$('profile-form').addEventListener('submit',async event=>{
 event.preventDefault();if(!$('profile-consent').checked)return;
 $('profile-status').textContent='Loading the public character, equipment and portrait…';
 try{const receipt=await mutations.submit('profile','/local/profile',{character_id:$('profile-target').value,name:$('profile-name').value,surname:$('profile-surname').value,region:$('profile-region').value,consent:true},busy=>{$('profile-load').disabled=busy;});sessionStorage.setItem('wisp-draft:'+selected,$('question').value);selected=receipt.result.character_id;lastReceipt=null;$('question').value='';resetContext();await refresh();$('profile-consent').checked=false;$('profile-status').textContent='Character saved. Open their journey to see the portrait and gear.';setView('journey',true);$('equipment-panel').open=true;}catch(error){$('profile-status').textContent=error.message;}
});
(async()=>{try{const found=await mutations.read('/local/installations');$('installation-status').textContent=found.paths.length?'Found '+found.paths.length+' WoW installation'+(found.paths.length===1?'':'s')+'. Confirm the folder below.':'No standard WoW folder detected. Enter your client folder below.';for(const path of found.paths){const option=element('option');option.value=path;$('wow-installations').append(option);}if(found.paths.length===1&&!$('wow-root').value)$('wow-root').value=found.paths[0];}catch{$('installation-status').textContent='Enter your WoW client folder below.';}})();
