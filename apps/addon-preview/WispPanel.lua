-- Original unreleased UI prototype. No network transport or generated answers.
WispJournalUI = {}
local UI = WispJournalUI
local window, chat, journey, chatText, journeyText, status, input, tabs
local character, entries, messages = {}, {}, {}
local function label(parent, text, x, y, width, size, gold)
 local out=parent:CreateFontString(nil,'OVERLAY','GameFontHighlight')
 out:SetPoint('TOPLEFT',x,y);out:SetWidth(width);out:SetJustifyH('LEFT')
 out:SetFont('Fonts\\FRIZQT__.TTF',size or 12)
 out:SetTextColor(gold and .94 or .9,gold and .74 or .93,gold and .44 or .98,1);out:SetText(text)
 return out
end
local function surface(parent,x,y,w,h,r,g,b)
 local f=CreateFrame('Frame',nil,parent,'BackdropTemplate');f:SetPoint('TOPLEFT',x,y);f:SetSize(w,h)
 f:SetBackdrop({bgFile='Interface\\Buttons\\WHITE8X8',edgeFile='Interface\\Buttons\\WHITE8X8',edgeSize=1})
 f:SetBackdropColor(r,g,b,1);f:SetBackdropBorderColor(.23,.31,.46,1);return f
end
local function button(parent,text,x,y,w,callback)
 local b=CreateFrame('Button',nil,parent,'BackdropTemplate');b:SetSize(w,30);b:SetPoint('TOPLEFT',x,y)
 b:SetBackdrop({bgFile='Interface\\Buttons\\WHITE8X8',edgeFile='Interface\\Buttons\\WHITE8X8',edgeSize=1})
 b:SetBackdropColor(.09,.14,.23,1);b:SetBackdropBorderColor(.35,.45,.62,1)
 b.caption=label(b,text,10,-8,w-20,12);b:SetScript('OnClick',callback)
 b:SetScript('OnEnter',function(self)self:SetBackdropBorderColor(.72,.67,.46,1)end)
 b:SetScript('OnLeave',function(self)self:SetBackdropBorderColor(.35,.45,.62,1)end)
 return b
end
local function updateJourney()
 if not journeyText then return end
 local lines={}
 for i=#entries,math.max(1,#entries-4),-1 do
  local e=entries[i]
  local action=e.kind=='quest_completed' and ('Quest #'..tostring(e.quest_id)..' completed') or e.kind=='level_up' and ('Reached level '..tostring(e.level)) or e.kind=='note' and 'A note to remember' or 'A moment in '..(e.subzone or e.zone or '')
  lines[#lines+1]='|cffefbb70'..action..'|r\n'..(e.zone or '')..(e.subzone~='' and e.subzone and (' / '..e.subzone) or '')..' · Level '..tostring(e.level or '?')
  if e.kind=='note' and e.note~='' then lines[#lines+1]=e.note end
  lines[#lines+1]=''
 end
 journeyText:SetText(table.concat(lines,'\n'))
end
local function selectTab(which)
 chat:SetShown(which=='chat');journey:SetShown(which=='journey')
 for name,b in pairs(tabs)do b:SetBackdropColor(name==which and .13 or .06,name==which and .2 or .1,name==which and .34 or .17,1)end
 if which=='journey' then updateJourney()end
end
function UI.AddMessage(role,text)
 local key=character.id or character.name or 'player';messages[key]=messages[key] or {}
 table.insert(messages[key],(role=='user' and '|cffefbb70YOU|r\n' or '|cffefbb70WISP|r\n')..text)
 if chatText then chatText:SetText(table.concat(messages[key],'\n\n'))end
end
function UI.Show(person,events)
 character=person or character;entries=events or entries
 if not window then
  window=surface(UIParent,50,-20,860,500,.035,.055,.09);window:ClearAllPoints();window:SetPoint('CENTER');window:SetFrameStrata('DIALOG')
  window:SetMovable(true);window:EnableMouse(true);window:RegisterForDrag('LeftButton');window:SetScript('OnDragStart',window.StartMoving);window:SetScript('OnDragStop',window.StopMovingOrSizing)
  local rail=surface(window,0,0,190,500,.045,.07,.12)
  label(rail,'WISP.',22,-23,150,30)
  label(rail,'YOUR CHARACTER',22,-88,150,10,true)
  UI.nameLabel=label(rail,'',22,-113,152,21)
  UI.identity=label(rail,'',22,-148,152,11)
  label(rail,'YOUR FIELD JOURNAL',22,-210,155,9,true)
  tabs={journey=button(rail,'Your journey',16,-238,158,function()UI.ShowJourney()end),chat=button(rail,'Talk to Wisp',16,-278,158,function()UI.ShowChat()end)}
  local avatar=rail:CreateTexture(nil,'ARTWORK');avatar:SetPoint('TOPLEFT',20,-350);avatar:SetSize(64,64);avatar:SetTexture('Interface\\AddOns\\Wisp\\Media\\avatar');label(rail,'A little company.',22,-420,150,12)
  label(rail,'Local journal\nPrototype · not released',22,-452,150,9,true)
  label(window,'LUA SIMULATOR · DEMO DATA · AI DISCONNECTED',214,-20,590,9,true)
  button(window,'×',810,-10,28,function()window:Hide()end)
  local banner=surface(window,212,-51,626,110,.06,.1,.18)
  local art=banner:CreateTexture(nil,'BACKGROUND');art:SetAllPoints();art:SetTexture('Interface\\AddOns\\Wisp\\Media\\expedition');art:SetTexCoord(0,1,.25,.52);art:SetVertexColor(.45,.55,.8,1)
  UI.title=label(banner,'Your story so far.',18,-22,540,26)
  label(banner,'Recorded places, moments and milestones.',18,-67,540,12)
  chat=CreateFrame('Frame',nil,window);chat:SetPoint('TOPLEFT',214,-183);chat:SetSize(622,280)
  journey=CreateFrame('Frame',nil,window);journey:SetPoint('TOPLEFT',214,-183);journey:SetSize(622,280)
  chatText=label(chat,'',0,-2,612,13);chatText:SetHeight(190);chatText:SetJustifyV('TOP')
  input=CreateFrame('EditBox',nil,chat,'InputBoxTemplate');input:SetPoint('TOPLEFT',6,-207);input:SetSize(465,30);input:SetFontObject(ChatFontNormal);input:SetAutoFocus(false);input:SetMaxLetters(1000)
  local function send()
   local question=input:GetText() or '';if question==''then return end
   if not UI.OnSend then status:SetText('AI is disconnected. Nothing was sent.');return end
   UI.OnSend(question,character.id);input:SetText('')
  end
  input:SetScript('OnEnterPressed',send);button(chat,'Ask Wisp',490,-207,120,send)
  status=label(chat,'Example conversation. No live AI reply.',0,-253,612,10,true)
  journeyText=label(journey,'',0,-2,612,13);journeyText:SetHeight(205);journeyText:SetJustifyV('TOP')
  local note=CreateFrame('EditBox',nil,journey,'InputBoxTemplate');note:SetPoint('TOPLEFT',6,-210);note:SetSize(355,28);note:SetFontObject(ChatFontNormal);note:SetAutoFocus(false)
  button(journey,'Save note',378,-210,110,function()local value=note:GetText() or '';if value~='' and UI.OnNote then UI.OnNote(value);note:SetText('');updateJourney()end end)
  button(journey,'Pause',502,-210,110,function(self)if UI.OnPause then UI.OnPause();self.caption:SetText('Paused')end end)
  label(window,'Your story stays local. In-game AI connection is still in development.',214,-476,620,9,true)
 end
 UI.nameLabel:SetText(character.name or 'Character');UI.identity:SetText('Level '..tostring(character.level or '?')..'\n'..(character.zone or ''))
 updateJourney();window:Show();selectTab('journey')
 chatText:SetText(table.concat(messages[character.id or character.name or 'player'] or {},'\n\n'))
 return window
end
function UI.ShowJourney()if window then UI.title:SetText('Your story so far.');selectTab('journey')end end
function UI.ShowChat()if window then UI.title:SetText('A moment with Wisp.');selectTab('chat')end end
