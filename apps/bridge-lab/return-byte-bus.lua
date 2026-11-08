-- Jornia's own bounded presence bus. A single LoD slot installs this receiver;
-- subsequent bytes and messages reuse the SAME file names, never LoadAddOn.
if WispReturnResearchPanel then WispReturnResearchPanel:Hide() end
if SlashCmdList.WISPRETURNRESEARCH then SlashCmdList.WISPRETURNRESEARCH('stop') end
if WispOCRPanel then WispOCRPanel:Hide() end
local frame=CreateFrame('Frame','WispReturnBusPanel',UIParent,'BackdropTemplate')
frame:SetSize(620,460);frame:SetPoint('CENTER')
frame:SetBackdrop({bgFile='Interface\\Buttons\\WHITE8X8',edgeFile='Interface\\Buttons\\WHITE8X8',edgeSize=1})
frame:SetBackdropColor(.025,.035,.055,1);frame:SetBackdropBorderColor(.7,.48,.19,1)
for _,p in ipairs({'TOPLEFT','TOPRIGHT','BOTTOMLEFT','BOTTOMRIGHT'}) do
 local t=frame:CreateTexture(nil,'OVERLAY');t:SetSize(6,6);t:SetPoint(p);t:SetColorTexture(1,0,1,1)
end
local function label(y,size,text)
 local s=frame:CreateFontString(nil,'OVERLAY');s:SetFont('Fonts\\FRIZQT__.TTF',size,'')
 s:SetPoint('TOPLEFT',20,y);s:SetWidth(580);s:SetJustifyH('LEFT');s:SetText(text or '');return s
end
frame.title=label(-20,22,'WISP / REUSABLE RETURN LAB')
local state=label(-65,15,'Controlled bytes only — no AI request or journal upload.')
local ack=label(-115,17,'')
local result=label(-240,16,'')
local metrics=label(-330,13,'')
local root='Interface\\AddOns\\'
local control=root..'WispBridgeLab\\wispreturn_present.wav'
local absent=root..'WispBridgeLab\\WispBusAbsentControl927.wav'
local clock=root..'WispBridgeLab\\wispreturn_pulse.wav'
local paths={}
for i=1,8 do local name=string.format('WispOCRInbox%03d',80+i);paths[i]=root..name..'\\'..name..'_Camelot.toc' end
local active,nonce,binding,guid,sequence,buffer,previous,candidate,candidatePhase,elapsed
local timeSpent,calls,errors=0,0,0
local function sum(text) local n=0;for i=1,#text do n=(n+i*text:byte(i))%65521 end;return n end
local function adler(text)
 local a,b=1,0;for i=1,#text do a=(a+text:byte(i))%65521;b=(b+a)%65521 end;return b*65536+a
end
local function probe(path)
 calls=calls+1
 local ok,success,handle=pcall(PlaySoundFile,path,'Master')
 if ok and handle and StopSound then pcall(StopSound,handle) end
 if not ok then errors=errors+1;return nil end
 return success and true or false
end
local function receipt(kind,value,digest)
 local fields=string.format('%s %s %s %03d %03d %010d',kind,nonce,binding,sequence,value,digest)
 ack:SetText(fields..' '..string.format('%05d',sum(fields)))
end
local function finishError(message,code)
 active=false;state:SetText(message);receipt('WISPFAIL',code,0)
end
local function start()
 guid=UnitGUID('player');if not guid then return end
 nonce=string.format('%08d',math.random(10000000,99999999));binding=string.format('%05d',sum(guid))
 sequence,buffer,candidate,candidatePhase,elapsed,timeSpent,calls,errors=0,'',nil,nil,0,0,0,0
 if probe(control)~=true or probe(absent)~=false then active=false;state:SetText('Channel self-test failed. No bytes accepted.');return end
 previous=probe(clock);if previous~=true then active=false;state:SetText('Clock baseline unavailable. Restore test files first.');return end
 for _,path in ipairs(paths) do if probe(path)~=true then active=false;state:SetText('Data baseline unavailable. Restore test files first.');return end end
 active=true;state:SetText('Ready — waiting for controlled test bytes.');result:SetText('')
 receipt('WISPREADY',0,0)
end
local function accept(value,phase)
 sequence=sequence+1;previous=phase;candidate=nil;candidatePhase=nil
 buffer=buffer..string.char(value)
 state:SetText(string.format('Receiving — %d checked byte frames.',sequence))
 if #buffer<=2 and buffer~=('WP'):sub(1,#buffer) then finishError('Frame header rejected; no text delivered.',1);return end
 if #buffer==3 and value>96 then finishError('Frame length rejected; no text delivered.',2);return end
 if #buffer>=3 then
  local length=buffer:byte(3);local expected=3+length+4
  if #buffer==expected then
   local body=buffer:sub(1,-5);local x1,x2,x3,x4=buffer:byte(-4,-1)
   local transmitted=((x1*256+x2)*256+x3)*256+x4
   if adler(body)~=transmitted then finishError('Checksum failed — no text delivered.',3);return end
   local text=buffer:sub(4,3+length)
   result:SetText('Received: '..text:gsub('|','||'))
   active=false;state:SetText(string.format('Text verified — %d bytes received.',length))
   receipt('WISPDONE',length,transmitted)
   return
  end
 end
 receipt('WISPACK',value,adler(buffer))
end
local since=0
frame:SetScript('OnUpdate',function(_,dt)
 if not active then return end
 elapsed=elapsed+dt;since=since+dt;if since<.05 then return end;since=0
 if UnitGUID('player')~=guid then finishError('Character changed — transfer rejected.',4);return end
 if elapsed>120 then finishError('Test timed out. No automatic restart.',5);return end
 local began=debugprofilestop and debugprofilestop() or 0
 if probe(control)~=true or probe(absent)~=false then
  candidate=nil;candidatePhase=nil;state:SetText('Channel unavailable — waiting without accepting bits.');return
 end
 local phase=probe(clock)
 if phase==nil or phase==previous then candidate=nil;candidatePhase=nil;return end
 local value=0
 for i=1,8 do
  local bit=probe(paths[i]);if bit==nil then candidate=nil;return end
  if bit then value=value+2^(i-1) end
 end
 if probe(clock)~=phase or probe(control)~=true then candidate=nil;candidatePhase=nil;return end
 if candidate==value and candidatePhase==phase then
  accept(value,phase)
 else candidate=value;candidatePhase=phase end
 if debugprofilestop then timeSpent=timeSpent+debugprofilestop()-began end
 metrics:SetText(string.format('Frames: %d | probe calls: %d | sampled work: %.1f ms | API errors: %d',sequence,calls,timeSpent,errors))
end)
SLASH_WISPRETURNBUS1='/wispbus'
SlashCmdList.WISPRETURNBUS=function(message)
 if message=='stop' or message=='hide' then active=false;state:SetText('Stopped. No further reads.');if message=='hide' then frame:Hide() end;return end
 start();frame:Show()
end
table.insert(UISpecialFrames,'WispReturnBusPanel')
WispReturnBusLab={Start=start,Frame=frame,State=state,Ack=ack}
start()
