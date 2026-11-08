-- Jornia's own 8-byte parallel presence transport. No addon loads per message.
if SlashCmdList.WISPRETURNBLOCK then SlashCmdList.WISPRETURNBLOCK('hide') end
if SlashCmdList.WISPRETURNBUS then SlashCmdList.WISPRETURNBUS('hide') end
if WispOCRPanel then WispOCRPanel:Hide() end
local frame=CreateFrame('Frame','WispReturnBlockPanelV2',UIParent,'BackdropTemplate')
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
frame.title=label(-20,22,'WISP / BLOCK RETURN LAB')
local state=label(-65,15,'Controlled transport — images stay local.')
local ack=label(-115,17,'')
local reply=CreateFrame('ScrollingMessageFrame',nil,frame)
reply:SetPoint('TOPLEFT',20,-215);reply:SetSize(580,150);reply:SetFontObject(GameFontHighlight)
reply:SetJustifyH('LEFT');reply:SetFading(false);reply:SetMaxLines(100);reply:EnableMouseWheel(true)
reply:SetScript('OnMouseWheel',function(self,delta) if delta>0 then self:ScrollUp() else self:ScrollDown() end end)
local metrics=label(-390,12,'')
local root='Interface\\AddOns\\'
local control=root..'WispBridgeLab\\wispreturn_present.wav'
local missing=root..'WispBridgeLab\\WispBlockAbsentControl927.wav'
local clock=root..'WispBridgeLab\\wispreturn_pulse.wav'
local paths={}
for i=1,64 do local n=string.format('WispOCRInbox%03d',i);paths[i]=root..n..'\\'..n..'_Camelot.toc' end
local active,nonce,binding,guid,sequence,buffer,previous,candidate,candidatePhase,elapsed,since
local calls,errors,cpu,maxCPU,ticks,baseFPS,minFPS,fpsTotal,fpsN=0,0,0,0,0,0,0,0,0
local transferStarted=false
local function sum(s) local n=0;for i=1,#s do n=(n+i*s:byte(i))%65521 end;return n end
local function adler(s) local a,b=1,0;for i=1,#s do a=(a+s:byte(i))%65521;b=(b+a)%65521 end;return b*65536+a end
local function u16(s,i) local a,b=s:byte(i,i+1);return a*256+b end
local function u32(s,i) local a,b,c,d=s:byte(i,i+3);return ((a*256+b)*256+c)*256+d end
local function probe(path)
 calls=calls+1
 local ok,success,handle=pcall(PlaySoundFile,path,'Master')
 if ok and handle and StopSound then pcall(StopSound,handle) end
 if not ok then errors=errors+1;return nil end
 return success and true or false
end
local function receipt(kind,value,digest)
 local fields=string.format('%s %s %s %03d %04d %010.0f',kind,nonce,binding,sequence,value,digest)
 ack:SetText(fields..' '..string.format('%05d',sum(fields)))
end
local function fail(text,code) active=false;state:SetText(text);receipt('WBFAIL',code,0) end
local function showMetrics()
 metrics:SetText(string.format('Blocks %d | calls %d | Lua %.1f ms total / %.2f max\nFPS baseline %.0f / min %.0f / mean %.0f | API errors %d',sequence,calls,cpu,maxCPU,baseFPS,minFPS,fpsN>0 and fpsTotal/fpsN or 0,errors))
end
local function start()
 guid=UnitGUID('player');if not guid then return end
 nonce=string.format('%08d',math.random(10000000,99999999));binding=string.format('%05d',sum(guid))
 sequence,buffer,previous,candidate,candidatePhase,elapsed,since=0,'',true,nil,nil,0,0
 calls,errors,cpu,maxCPU,ticks=0,0,0,0,0;transferStarted=false
 baseFPS=GetFramerate and GetFramerate() or 0;minFPS=baseFPS;fpsTotal,fpsN=0,0
 active=false;reply:Clear()
 if probe(control)~=true or probe(missing)~=false or probe(clock)~=true then state:SetText('Channel baseline failed. Restore lab files first.');return end
 for _,path in ipairs(paths) do if probe(path)~=true then state:SetText('Data baseline failed. Restore lab files first.');return end end
 active=true;state:SetText('Ready — waiting for a checked block transfer.')
 receipt('WBREADY',8,0);showMetrics()
end
local function accept(data,phase)
 sequence=sequence+1;previous=phase;candidate=nil;candidatePhase=nil;buffer=buffer..data
 if buffer:sub(1,2)~='WB' or buffer:byte(3)~=1 then fail('Unsupported frame — no text delivered.',1);return end
 if u32(buffer,4)~=tonumber(nonce) then fail('Stale transfer rejected — no text delivered.',6);return end
 if #buffer>=11 then
  if u16(buffer,8)~=tonumber(binding) then fail('Character binding rejected.',7);return end
  local length=u16(buffer,10);if length>1024 then fail('Payload length rejected.',2);return end
  local expected=11+length+4
  state:SetText(string.format('Receiving — %d / %d payload bytes.',math.min(math.max(#buffer-11,0),length),length))
  if #buffer>=expected then
   local body=buffer:sub(1,expected-4);local digest=u32(buffer,expected-3)
   if buffer:sub(expected+1):find('[^%z]') or adler(body)~=digest then fail('Checksum failed — no text delivered.',3);return end
   reply:AddMessage('Received: '..buffer:sub(12,11+length):gsub('|','||'));reply:ScrollToTop()
   active=false;state:SetText(string.format('Verified — %d payload bytes, %d blocks.',length,sequence));receipt('WBDONE',length,digest);return
  end
 else state:SetText('Receiving and validating the frame header…') end
 receipt('WBACK',#buffer,adler(buffer))
end
local function tick(dt)
 elapsed=elapsed+dt;since=since+dt;if since<.05 then return end;since=0
 if UnitGUID('player')~=guid then fail('Character changed — transfer stopped.',4);return end
 if elapsed>120 then fail('Transfer timed out. No automatic restart.',5);return end
 if probe(control)~=true or probe(missing)~=false then candidate=nil;candidatePhase=nil;state:SetText('Channel unavailable — no data accepted.');return end
 local phase=probe(clock)
 if phase==nil or phase==previous then candidate=nil;candidatePhase=nil;return end
 transferStarted=true
 local bytes={}
 for j=0,7 do
  local value=0
  for i=1,8 do local bit=probe(paths[j*8+i]);if bit==nil then candidate=nil;candidatePhase=nil;return end;if bit then value=value+2^(i-1) end end
  bytes[j+1]=string.char(value)
 end
 if probe(clock)~=phase or probe(control)~=true then candidate=nil;candidatePhase=nil;return end
 local data=table.concat(bytes)
 if candidate==data and candidatePhase==phase then accept(data,phase) else candidate=data;candidatePhase=phase end
end
local uiElapsed=0
frame:SetScript('OnUpdate',function(_,dt)
 if not active then return end
 local began=debugprofilestop and debugprofilestop() or 0
 tick(dt)
 if debugprofilestop then local spent=debugprofilestop()-began;cpu=cpu+spent;maxCPU=math.max(maxCPU,spent);ticks=ticks+1 end
 if transferStarted and GetFramerate then local fps=GetFramerate();minFPS=math.min(minFPS,fps);fpsTotal=fpsTotal+fps;fpsN=fpsN+1 end
 uiElapsed=uiElapsed+dt;if uiElapsed>=.25 or not active then uiElapsed=0;showMetrics() end
end)
SLASH_WISPRETURNBLOCK1='/wispblock'
SlashCmdList.WISPRETURNBLOCK=function(message)
 if message=='stop' or message=='hide' then active=false;state:SetText('Stopped. No further reads.');if message=='hide' then frame:Hide() end;return end
 start();frame:Show()
end
table.insert(UISpecialFrames,'WispReturnBlockPanelV2')
WispReturnBlockLab={Frame=frame,Start=start,State=state,Ack=ack}
start()
