-- Jornia's own read-only candidate probe. No external code, logging or AI calls.
local panel=CreateFrame('Frame','WispReturnResearchPanel',UIParent,'BasicFrameTemplateWithInset')
panel:SetSize(620,310);panel:SetPoint('TOP',0,-60)
panel.TitleText:SetText('Jornia return-channel research — read-only probe')
local label=panel:CreateFontString(nil,'OVERLAY','GameFontHighlight')
label:SetPoint('TOPLEFT',18,-38);label:SetWidth(580);label:SetJustifyH('LEFT')
local paths={
 {'empty control','Interface\\AddOns\\WispBridgeLab\\wispreturn_present.wav'},
 {'empty pulse','Interface\\AddOns\\WispBridgeLab\\wispreturn_pulse.wav'},
 {'silent WAV','Interface\\AddOns\\WispBridgeLab\\wispreturn_silent.wav'},
 {'known Lua','Interface\\AddOns\\WispBridgeInbox02\\Inbox.lua'},
 {'known TOC','Interface\\AddOns\\WispBridgeInbox02\\WispBridgeInbox02_Camelot.toc'},
 {'missing Lua','Interface\\AddOns\\WispBridgeInbox02\\WispDefinitelyAbsent7b72.lua'},
 {'missing WAV','Interface\\AddOns\\WispBridgeInbox02\\WispDefinitelyAbsent7b72.wav'},
}
local elapsed,ticks,changes,last=0,0,0,nil
local history={}
local function sample()
 local lines={'Sound setting unchanged: '..tostring(GetCVar and GetCVar('Sound_EnableAllSound'))}
 local bits={}
 for _,p in ipairs(paths) do
  local ok,success,handle=pcall(PlaySoundFile,p[2],'Master')
  if ok and handle and StopSound then pcall(StopSound,handle) end
  local value=ok and tostring(success) or 'API error'
  table.insert(lines,p[1]..': '..value);table.insert(bits,value)
 end
 local signature=table.concat(bits,',')
 if last and signature~=last then changes=changes+1;table.insert(history,string.format('%.2fs: %s',elapsed,signature));if #history>5 then table.remove(history,1) end end
 last=signature;ticks=ticks+1
 table.insert(lines,'Samples: '..ticks..' | changes: '..changes..' | auto-stop after 30 s')
 table.insert(lines,table.concat(history,'\n'));label:SetText(table.concat(lines,'\n'))
end
local function tick(self,dt)
 elapsed=elapsed+dt
 if elapsed>=30 then self:SetScript('OnUpdate',nil);return end
 if elapsed>=ticks*.25 then sample() end
end
local function start() elapsed,ticks,changes,last=0,0,0,nil;history={};sample();panel:SetScript('OnUpdate',tick);panel:Show() end
SLASH_WISPRETURNRESEARCH1='/wispreturn'
SlashCmdList.WISPRETURNRESEARCH=function(message) if message=='stop' then panel:SetScript('OnUpdate',nil) else start() end end
start()
table.insert(UISpecialFrames,'WispReturnResearchPanel')
print('Jornia return research: read-only file-presence probe running for 30 seconds. No files or settings changed by this Lua module.')
