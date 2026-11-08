-- Local-only view adapter. Uses the existing consented journal without sending data.
local UI=WispJournalUI
if not UI or UI.NativeBindingsInstalled then return end
UI.NativeBindingsInstalled=true
UI.Config={
 runtimeLabel='YOUR LOCAL ADVENTURE JOURNAL',
 railLabel='Your journey\nYour companion',
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
 local share;if type(c)=='table' and type(c.share_current_context)=='boolean'then share=c.share_current_context end
 local events=type(c)=='table' and type(c.events)=='table' and c.events or {}
 return {id=guid,name=name,level=UnitLevel('player'),zone=GetZoneText() or '',share_current_context=share},events
end
local function refresh(live)
 local person,events=personAndEvents();if person then
  if live and UI.Update then UI.Update(person,events)else UI.Show(person,events)end
 end
end
function UI.OpenNative(view)
 refresh()
 if view=='compact' then UI.SetMode('compact')
 elseif view=='chat' then UI.SetMode('full');UI.ShowChat()
 elseif view=='journey' then UI.SetMode('full');UI.ShowJourney()
 else UI.SetMode('full') end
end
UI.OnContextPreference=function(owner,value)
 local c=WispDB and WispDB.characters and WispDB.characters[owner]
 if c and owner==UnitGUID('player')then c.share_current_context=value end
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
local updates=CreateFrame('Frame');UI.NativeUpdates=updates;local queued=false
for _,event in ipairs({'PLAYER_LEVEL_UP','ZONE_CHANGED','ZONE_CHANGED_INDOORS','ZONE_CHANGED_NEW_AREA','QUEST_TURNED_IN','QUEST_ACCEPTED','QUEST_REMOVED','QUEST_LOG_UPDATE','PLAYER_EQUIPMENT_CHANGED','SKILL_LINES_CHANGED','PLAYER_DEAD','PLAYER_ALIVE','PLAYER_UNGHOST'}) do updates:RegisterEvent(event) end
updates:SetScript('OnEvent',function()
 if not UI.IsShown()or queued then return end;queued=true
 local function update()queued=false;if UI.IsShown()then refresh(true)end end
 if C_Timer and C_Timer.After then C_Timer.After(.35,update)else update()end
end)
print('|cffffc466Jornia:|r /jornia opens your journey; /jornia compact opens a small resizable chat. Keep Jornia desktop running for connected chat.')
