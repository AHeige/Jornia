// Developer-only adapter for srrmke/wow-addons-sandbox. Never included in the client.
import './style.css';
import {Runtime} from './runtime';
import {Renderer} from './render';
import source from '../../../apps/addon/Wisp/Wisp.lua?raw';
import toc from '../../../apps/addon/Wisp/Wisp_Camelot.toc?raw';
import panel from '../../../apps/addon-preview/WispPanel.lua?raw';
import forest from '../../../apps/companion/web/media/expedition.jpg?url';
import avatar from '../../../apps/companion/web/media/wisp.png?url';
const errors:string[]=[];
const stage=document.getElementById('stage')!;
Object.assign(stage.style,{position:'relative',width:'960px',height:'540px',borderRadius:'0',transform:'none',background:'#0b1324'});
Object.assign(document.body.style,{margin:'0',height:'540px',background:'#0b1324',overflow:'hidden'});
const runtime=new Runtime({log:(kind,text)=>{if(kind==='error')errors.push(text)},requestReload:()=>{}});
const renderer=new Renderer(stage,{mouse:(...args)=>runtime.mouse(...args),edit:(...args)=>runtime.edit(...args),cursor:(x,y)=>runtime.mouseMove(x,y)},(path)=>path.toLowerCase().includes('wisp\\media\\expedition')?forest:path.toLowerCase().includes('wisp\\media\\avatar')?avatar:undefined);
// The simulator needs explicit screen coordinates for native-style move/resize.
stage.addEventListener('pointermove',event=>{const rect=stage.getBoundingClientRect();runtime.mouseMove(event.clientX-rect.left,540-(event.clientY-rect.top));});
const mock=`
function GetZoneText() return 'Ashenvale' end
function GetSubZoneText() return 'Astranaar' end
function UnitGUID() return 'Player-Wisp-Preview' end
function UnitName() return 'Demo traveller' end
function GetRealmName() return 'Preview' end
function UnitLevel() return 24 end
function time() return 1793880000 end
`;
await runtime.boot([{name:'Wisp',meta:{Interface:'16001',Title:'Wisp',SavedVariables:'WispDB'},files:['PreviewWorld.lua','Wisp.lua','WispPanel.lua'],text:new Map([['previewworld.lua',mock],['wisp.lua',source],['wisppanel.lua',panel],['wisp_camelot.toc',toc]]),images:new Map(),dependencies:[]}],{w:960,h:540,sv:{},client:'forever'});
const scenario=new URLSearchParams(location.search).get('scenario')||'export';
if(scenario==='export'||scenario==='chat'||scenario==='journey'||scenario==='compact'){
 runtime.slash('/run StaticPopupDialogs.WISP_JOURNAL_CONSENT.OnAccept(); StaticPopup_Hide()');
 runtime.fire('QUEST_TURNED_IN',1020);
 if(scenario==='export')runtime.slash('/wisp export');
}
if(scenario==='chat'||scenario==='journey'||scenario==='compact'){
 runtime.slash(`/run local c=WispDB.characters['Player-Wisp-Preview']; WispJournalUI.OnPause=function() SlashCmdList.WISP('stop') end; WispJournalUI.OnNote=function(text) SlashCmdList.WISP('note '..text) end; WispJournalUI.Show({id='Player-Wisp-Preview',name='Demo traveller',level=24,zone='Ashenvale'},c.events); WispJournalUI.AddMessage('user','What was I doing in Ashenvale?'); WispJournalUI.AddMessage('assistant','Example reply: You visited Astranaar and turned in quest #1020. Once the AI connection is ready, Wisp can use these journal events in its answers.')`);
 if(scenario==='journey')runtime.slash('/run WispJournalUI.ShowJourney()');else runtime.slash('/run WispJournalUI.ShowChat()');
 if(scenario==='compact')runtime.slash("/run WispJournalUI.SetMode('compact')");
}
runtime.slash('/run if Minimap then Minimap:Hide() end');
function render(){runtime.tick(1/60);const snapshot=runtime.snapshot();if(snapshot)renderer.update(snapshot);requestAnimationFrame(render)}
render();
document.documentElement.dataset.ready='true';
document.documentElement.dataset.source='apps/addon/Wisp/Wisp.lua';
document.getElementById('preview-errors')!.textContent=JSON.stringify(errors);
// Read-only evidence for the screenshot test; UI comes entirely from executed Lua.
(window as any).wispPreviewEvidence={errors,source,scenario};
