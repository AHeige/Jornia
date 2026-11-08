-- Local-only view adapter. Uses the existing consented journal without sending data.
local UI=WispJournalUI
if not UI or UI.NativeBindingsInstalled then return end
UI.NativeBindingsInstalled=true
UI.Config={
 runtimeLabel='YOUR LOCAL ADVENTURE JOURNAL',
 railLabel='Local journal\nAI lab is separate',
 disconnectedText='AI chat is not connected to this window yet.',
 footer='Your journey stays on this computer. AI chat is not connected to this window.'
}
local previous=SlashCmdList.WISP
local function personAndEvents()
 local guid=UnitGUID('player');if not guid then return end
 local name,surname=UnitName('player');name=name or 'Character'
 -- This Forever build exposes the surname as UnitName's second result.
 if type(surname)=='string' and surname~='' and surname~=(GetRealmName and GetRealmName() or '') then name=name..' '..surname end
 local db=WispDB
 local c=type(db)=='table' and type(db.characters)=='table' and db.characters[guid]
 local events=type(c)=='table' and type(c.events)=='table' and c.events or {}
 return {id=guid,name=name,level=UnitLevel('player'),zone=GetZoneText() or ''},events
end
local function refresh()
 local person,events=personAndEvents();if person then UI.Show(person,events) end
end
function UI.OpenNative(view)
 refresh()
 if view=='compact' then UI.SetMode('compact')
 elseif view=='chat' then UI.SetMode('full');UI.ShowChat()
 elseif view=='journey' then UI.SetMode('full');UI.ShowJourney()
 else UI.SetMode('full') end
end
UI.OnNote=function(value) if previous then previous('note '..value);refresh() end end
UI.OnPause=function() if previous then previous('stop');refresh() end end
SlashCmdList.WISP=function(message)
 message=message or '';local command=message:match('^%s*(%S*)')
 if command=='' or command=='ui' then if UI.IsShown() then UI.Hide() else UI.OpenNative() end
 elseif command=='compact' or command=='chat' or command=='journey' then UI.OpenNative(command)
 else if previous then previous(message) end;if UI.IsShown() then refresh() end end
end
SLASH_WISPMINI1='/jorniamini';SLASH_WISPMINI2='/wispmini';SlashCmdList.WISPMINI=function()UI.OpenNative('compact')end
-- Update only an already open view. Events must never open a window unexpectedly.
local updates=CreateFrame('Frame')
for _,event in ipairs({'PLAYER_LEVEL_UP','ZONE_CHANGED','ZONE_CHANGED_NEW_AREA','QUEST_TURNED_IN'}) do updates:RegisterEvent(event) end
updates:SetScript('OnEvent',function()
 if not UI.IsShown() then return end
 if C_Timer and C_Timer.After then C_Timer.After(0,function()if UI.IsShown() then refresh() end end)
 else refresh() end
end)
print('|cffffc466Jornia:|r /jornia opens your journey; /jornia compact opens a small resizable chat. Keep Jornia desktop running for connected chat.')
