-- First-party local OCR outbox + reusable checked 8-byte return channel.
-- No chat logging, network calls, game memory access or gameplay automation.
local UI=WispJournalUI
if not UI then return end
WispCompanionTransport={}
local T=WispCompanionTransport
local card=CreateFrame('Frame','WispCompanionTransferCard',UIParent,'BackdropTemplate')
card:SetSize(860,420);card:SetPoint('BOTTOMLEFT',UIParent,'BOTTOMLEFT',20,110);card:SetFrameStrata('FULLSCREEN_DIALOG')
card:SetBackdrop({bgFile='Interface\\Buttons\\WHITE8X8',edgeFile='Interface\\Buttons\\WHITE8X8',edgeSize=1})
card:SetBackdropColor(.025,.035,.055,1);card:SetBackdropBorderColor(.25,.35,.5,1)
for _,point in ipairs({'TOPLEFT','TOPRIGHT','BOTTOMLEFT','BOTTOMRIGHT'})do local t=card:CreateTexture(nil,'OVERLAY');t:SetSize(3,3);t:SetPoint(point);t:SetColorTexture(1,0,1,1)end
local function label(y,size,text)local f=card:CreateFontString(nil,'OVERLAY');f:SetFont('Fonts\\ARIALN.TTF',size,'');f:SetPoint('TOPLEFT',12,y);f:SetWidth(836);f:SetJustifyH('LEFT');f:SetJustifyV('TOP');f:SetText(text or '');return f end
label(-10,14,'Jornia · local message transfer')
local wireText=label(-40,16,'');wireText:SetHeight(316);wireText:SetWordWrap(true)
local transferNote=label(-401,10,'Local OCR · images stay on this Mac · keep this card visible while waiting')
card:Hide();T.Frame=card;T.Packet=wireText
local root='Interface\\AddOns\\'
local control=root..'WispBridgeLab\\wispreturn_present.wav'
local absent=root..'WispBridgeLab\\WispBlockAbsentControl927.wav'
local clock=root..'WispBridgeLab\\wispreturn_pulse.wav'
local thinking=root..'WispOCRInbox065\\WispOCRInbox065_Camelot.toc'
local heartbeat=root..'WispOCRInbox066\\WispOCRInbox066_Camelot.toc'
local aliveAt,heartPhase,heartEdges,heartTime=nil,nil,0,0
local pump=CreateFrame('Frame')
local paths={};for i=1,64 do local n=string.format('WispOCRInbox%03d',i);paths[i]=root..n..'\\'..n..'_Camelot.toc' end
local pending,sequence,elapsed=nil,0,0
local session=tostring(time()):sub(-8)
local function checksum(s)local n=0;for i=1,#s do n=(n+i*s:byte(i))%65521 end;return n end
local function adler(s)local a,b=1,0;for i=1,#s do a=(a+s:byte(i))%65521;b=(b+a)%65521 end;return b*65536+a end
local function u16(s,i)local a,b=s:byte(i,i+1);return a*256+b end
local function u32(s,i)local a,b,c,d=s:byte(i,i+3);return ((a*256+b)*256+c)*256+d end
local function probe(path)local ok,success,handle=pcall(PlaySoundFile,path,'Master');if ok and handle and StopSound then pcall(StopSound,handle)end;if not ok then return nil end;return success and true or false end
pump:SetScript('OnUpdate',function(_,dt)
 heartTime=heartTime+dt;if heartTime<.5 then return end;heartTime=0
 local v=probe(heartbeat);if v~=nil and heartPhase~=nil and v~=heartPhase then aliveAt=GetTime();heartEdges=heartEdges+1 end;heartPhase=v
end)
local function quote(s)return '"'..s:gsub('\\','\\\\'):gsub('"','\\"'):gsub('[%z\1-\31]',function(c)return string.format('\\u%04x',c:byte())end)..'"' end
local function snapshot(q,include)
 local state=include and WispClientState and WispClientState.Snapshot() or nil
 local rows={};local truncated=false;local gear={};local skills={};local missing={}
 if include and not state then return nil,'Current character state is unavailable. Reopen Jornia after the journal is ready.' end
 if state then
  truncated=state.truncated or not state.quests_available
  local ordered={};for _,quest in ipairs(state.quests or {})do ordered[#ordered+1]=quest end
  local selected;if C_QuestLog and C_QuestLog.GetSelectedQuest then local ok,id=pcall(C_QuestLog.GetSelectedQuest);if ok then selected=id end end
  local lower=q:lower()
  table.sort(ordered,function(a,b)
   local function score(quest)
    if quest.title and quest.title~='' and lower:find(quest.title:lower(),1,true)then return 4 end
    if quest.id==selected then return 3 end
    if quest.complete then return 2 end
    return 1
   end
   local x,y=score(a),score(b);if x~=y then return x>y end;return a.id<b.id
  end)
  for i,quest in ipairs(ordered)do
   if i>4 then truncated=true;break end
   local goals={};for j,o in ipairs(quest.objectives or {})do local title=o.text or '';if #title>80 then title='';truncated=true end;goals[#goals+1]='['..j..','..(o.fulfilled or -1)..','..(o.required or -1)..','..(o.finished and '1' or '0')..','..quote(title)..']' end
   rows[#rows+1]='['..quest.id..','..quote(quest.title or '')..','..(quest.complete and '1' or '0')..','..(quest.waypoint and '1' or '0')..',['..table.concat(goals,',')..']]'
  end
  for _,item in ipairs(state.equipment or {})do gear[#gear+1]='['..item.slot..','..item.item_id..']' end
  for _,skill in ipairs(state.professions or {})do if skill.skill_id and skill.skill_id>0 then skills[#skills+1]='['..skill.skill_id..','..(skill.rank or 0)..','..(skill.max_rank or 0)..']'end end
  if not state.quests_available then missing[#missing+1]=quote('quests')end
  if not state.equipment_available then missing[#missing+1]=quote('equipment')end
  if not state.professions_available then missing[#missing+1]=quote('professions')end
  if not state.class_id then missing[#missing+1]=quote('class')end;if not state.race_id then missing[#missing+1]=quote('race')end
 end
 local function body()
  local snap='null'
  if state then snap='['..state.observed_at..','..state.level..','..quote(state.zone)..','..quote(state.client_build)..','..(truncated and '1' or '0')..',['..table.concat(rows,',')..'],{"c":'..(state.class_id or 0)..',"r":'..(state.race_id or 0)..',"e":['..table.concat(gear,',')..'],"p":['..table.concat(skills,',')..'],"m":['..table.concat(missing,',')..']}]'end
  local identity=''
  local current=WispJournalRuntime and WispJournalRuntime.Current()
  if current and WispDB and type(WispDB.installation_id)=='string' then
   identity=',"c":['..quote(WispDB.installation_id)..','..quote(UnitGUID('player'))..','..quote(current.name or 'Character')..','..quote(current.realm or '')..']'
  end
  return '{"q":'..quote(q)..',"s":'..snap..identity..'}'
 end
 -- Protect useful quest objectives before optional gear/profession context.
 while #body()>640 and #gear>0 do table.remove(gear);truncated=true end
 while #body()>640 and #skills>0 do table.remove(skills);truncated=true end
 while #body()>640 and #rows>1 do table.remove(rows);truncated=true end
 if #body()>640 and #rows==1 then return nil,'Open a shorter quest or shorten the question; this context does not fit the current bridge.' end
 return body()
end
local function receipt(kind,value,digest)
 local p=pending;local fields=string.format('%s %s %s %03d %04d %010.0f',kind,p.nonce,p.binding,p.seq,value,digest)
 wireText:SetText(fields..' '..string.format('%05d',checksum(fields)))
end
local function fail(message,code)
 local p=pending;if not p then return end
 receipt('WBFAIL',code,0);if p.kind=='connect'then UI.ResolveConnection(p.id,p.guid,message,true)else UI.Resolve(p.id,p.guid,message,true)end;pending=nil
 C_Timer.After(3,function()if not pending then card:Hide()end end)
end
local function accept(data,phase)
 local p=pending;p.seq=p.seq+1;p.previous=phase;p.candidate=nil;p.phase=nil;p.buffer=p.buffer..data
 local b=p.buffer
 if b:sub(1,2)~='WB' or b:byte(3)~=1 then fail('Unsupported reply frame.',1);return end
 if u32(b,4)~=tonumber(p.nonce)then fail('A stale reply was rejected.',6);return end
 if #b<11 then receipt('WBACK',#b,adler(b));return end
 local length=u16(b,10)
 if u16(b,8)~=tonumber(p.binding) or length>4096 then fail('Reply identity or size rejected.',7);return end
 UI.SetProgress(p.id,p.guid,'Receiving Jornia’s reply…')
 local expected=11+length+4
 if #b<expected then receipt('WBACK',#b,adler(b));return end
 local digest=u32(b,expected-3)
 if adler(b:sub(1,expected-4))~=digest or b:sub(expected+1):find('[^%z]')then fail('The reply checksum did not match.',3);return end
 local payload=b:sub(12,11+length)
 if #payload<39 or payload:byte(1)~=1 or payload:sub(3,26)~=p.id then fail('Reply operation mismatch.',8);return end
 local n=u16(payload,27)
 if #payload~=39+n then fail('Invalid reply envelope.',8);return end
 local text=payload:sub(29,28+n);local at=29+n
 local metadata={foreverdb=payload:byte(at)==1,sources=u16(payload,at+1)}
 local quest,expires=u32(payload,at+3),u32(payload,at+7)
 -- Coordinates never come from a model. Native navigation can be attached separately.
 if quest>0 and expires>time() then metadata.quest=quest;metadata.expires=expires end
 receipt('WBDONE',length,digest)
 if p.kind=='connect'then UI.ResolveConnection(p.id,p.guid,text,payload:byte(2)~=0)else UI.Resolve(p.id,p.guid,text,payload:byte(2)~=0,metadata)end;pending=nil
 C_Timer.After(2,function()if not pending then card:Hide()end end)
end
local function ready(owner)
 if pending then return 'A reply is already pending.' end
 if not aliveAt or GetTime()-aliveAt>5 or heartEdges<2 then return 'Start the local Jornia reader and wait a moment.' end
 if owner~=UnitGUID('player') then return 'The selected character changed.' end
 if probe(control)~=true or probe(absent)~=false or probe(clock)~=true or probe(thinking)~=true then return 'The local return channel is not ready. Start the Jornia reader.' end
 for _,path in ipairs(paths)do if probe(path)~=true then return 'Return files need recovery before another question.' end end
end
local function begin(body,owner,kind)
 local reason=ready(owner);if reason then return nil,reason end
 sequence=sequence+1;local id=session..string.format('%010d%06d',time(),sequence);local binding=string.format('%05d',checksum(owner));local nonce=string.format('%08d',math.random(10000000,99999999))
 pending={id=id,guid=owner,kind=kind,binding=binding,nonce=nonce,seq=0,previous=true,buffer='',started=GetTime()};elapsed=0
 local hex=body:gsub('.',function(c)return string.format('%02x',c:byte())end);local lines={};for i=1,#hex,96 do lines[#lines+1]=hex:sub(i,i+95)end
 local height=math.max(220,math.min(420,80+(#lines+2)*18));card:SetHeight(height);wireText:SetHeight(height-80);transferNote:ClearAllPoints();transferNote:SetPoint('TOPLEFT',12,-height+19)
 local check=checksum(id..binding..nonce..body)
 wireText:SetText('WISPBEGIN3 '..id..' '..binding..' '..nonce..' '..string.format('%05d',check)..'\n'..table.concat(lines,'\n')..'\nWISPEND')
 card:Show();return id
end
function T.Send(q,owner,include)
 if #q<1 or #q>160 then return nil,'Please shorten the question to 160 UTF-8 bytes.' end
 local body,reason=snapshot(q,include);if not body then return nil,reason end;if #body>640 then return nil,'Please shorten the question to include this context.' end
 return begin(body,owner,'ask')
end
local function xor(a,b)local n,place=0,1;for i=1,8 do if a%2~=b%2 then n=n+place end;a=math.floor(a/2);b=math.floor(b/2);place=place*2 end;return n end
function T.Connect(code,owner)
 local reason=ready(owner);if reason then return nil,reason end
 if type(code)~='string' or #code~=64 or code:find('[^0-9a-f]')then return nil,'Use the 64-character invite code from Wisp.' end
 local addons=C_AddOns;if not addons or not addons.LoadAddOn or not addons.IsAddOnLoaded then return nil,'Invite setup is unavailable. Connect through Jornia desktop.' end
 for n=67,80 do local name=string.format('WispOCRInbox%03d',n)
  if not addons.IsAddOnLoaded(name)then
   local ok,loaded=pcall(addons.LoadAddOn,name)
   local pad=WispConnectPads and WispConnectPads[n]
   if WispConnectPads then WispConnectPads[n]=nil end
   if ok and loaded and type(pad)=='table' and type(pad.id)=='string' and #pad.id==16 and type(pad.hex)=='string' and #pad.hex==64 then
    local cipher={};for i=1,64,2 do local a,b=tonumber(code:sub(i,i+1),16),tonumber(pad.hex:sub(i,i+1),16);if not a or not b then return nil,'Invite setup expired. Use Jornia desktop.' end;cipher[#cipher+1]=string.format('%02x',xor(a,b))end
    local body='{"a":{"pad":'..quote(pad.id)..',"cipher":'..quote(table.concat(cipher))..'}}';pad.hex=nil;code=nil
    return begin(body,owner,'connect')
   end
   return nil,'Start or restart the local Jornia reader, then try again. No invite was sent.'
  end
 end
 return nil,'In-game setup slots were used. Connect through Jornia desktop instead.'
end
card:SetScript('OnUpdate',function(_,dt)
 local p=pending;if not p then return end
 if p.guid~=UnitGUID('player')then fail('Character changed; the pending reply will stay in desktop history.',4);return end
 if aliveAt and GetTime()-aliveAt>8 then fail('The local reader stopped. Check desktop history; no automatic resend.',9);return end
 if GetTime()-p.started>120 then fail('No confirmed reply. Check desktop history; no automatic resend.',5);return end
 elapsed=elapsed+dt;if elapsed<.05 then return end;elapsed=0
 if probe(control)~=true or probe(absent)~=false then p.candidate=nil;return end
 local phase=probe(clock)
 if phase==nil or phase==p.previous then
  if p.seq==0 and probe(thinking)==false then UI.SetProgress(p.id,p.guid,'Jornia is thinking…') end
  p.candidate=nil;return
 end
 local bytes={};for j=0,7 do local n=0;for i=1,8 do local bit=probe(paths[j*8+i]);if bit==nil then p.candidate=nil;return end;if bit then n=n+2^(i-1)end end;bytes[j+1]=string.char(n)end
 if probe(clock)~=phase or probe(control)~=true then p.candidate=nil;return end
 local data=table.concat(bytes)
 if p.candidate==data and p.phase==phase then accept(data,phase) else p.candidate=data;p.phase=phase end
end)
UI.OnSend=T.Send
UI.OnConnect=T.Connect
UI.Config.disconnectedText=nil
UI.Config.footer='Questions use local OCR. Images stay here; selected text is sent to Jornia AI.'
T.IsPending=function()return pending~=nil end
T.Stop=function()if pending then fail('The local adapter was replaced. Check desktop history before retrying.',9)end;pump:SetScript('OnUpdate',nil);card:SetScript('OnUpdate',nil);card:Hide()end
print('|cffffc466Jornia:|r Native chat adapter ready. Start the local reader before sending. Images stay local.')
