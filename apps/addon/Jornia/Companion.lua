-- First-party companion view. Transport adapters supply the send/resolve hooks.
WispJournalUI = {}
local UI = WispJournalUI
UI.Config = {}
UI.SettingsOwnsContext = true
UI.ConnectionState = "unknown"
UI.ClearsAcceptedInput = true
local window,rail,chat,journey,chatText,journeyText,status,input,tabs,sendButton,modeButton,closeButton,resize,footer,connectButton
local character,entries,messages,drafts,contextSharing={},{},{},{},{}
local mode,tab='full','journey'
local sizes={full={860,540},compact={420,340}}
local pending,layoutBusy,lastW,lastH
local function key() return character.id or character.name or 'player' end
local function safe(s) return tostring(s or ''):gsub('|','||') end
local function label(parent,text,x,y,width,size,gold)
 local out=parent:CreateFontString(nil,'OVERLAY','GameFontHighlight')
 out:SetPoint('TOPLEFT',x,y);out:SetWidth(width);out:SetJustifyH('LEFT')
 out:SetFont('Fonts\\FRIZQT__.TTF',size or 12,'')
 out:SetTextColor(gold and .94 or .9,gold and .74 or .93,gold and .44 or .98,1);out:SetText(text)
 return out
end
local function place(f,x,y,w,h)
 f:ClearAllPoints();f:SetPoint('TOPLEFT',x,y);f:SetSize(w,h)
end
local function surface(parent,x,y,w,h,r,g,b)
 local f=CreateFrame('Frame',nil,parent,'BackdropTemplate');place(f,x,y,w,h)
 f:SetBackdrop({bgFile='Interface\\Buttons\\WHITE8X8',edgeFile='Interface\\Buttons\\WHITE8X8',edgeSize=1})
 f:SetBackdropColor(r,g,b,1);f:SetBackdropBorderColor(.23,.31,.46,1);return f
end
local function button(parent,text,x,y,w,callback)
 local b=CreateFrame('Button',nil,parent,'BackdropTemplate');place(b,x,y,w,30)
 b:SetBackdrop({bgFile='Interface\\Buttons\\WHITE8X8',edgeFile='Interface\\Buttons\\WHITE8X8',edgeSize=1})
 b:SetBackdropColor(.09,.14,.23,1);b:SetBackdropBorderColor(.35,.45,.62,1)
 b.caption=label(b,text,10,-8,w-20,12);b:SetScript('OnClick',callback)
 b:SetScript('OnEnter',function(self)self:SetBackdropBorderColor(.72,.67,.46,1)end)
 b:SetScript('OnLeave',function(self)self:SetBackdropBorderColor(.35,.45,.62,1)end)
 return b
end
local function renderMessages()
 if not chatText then return end
 chatText:Clear()
 for _,message in ipairs(messages[key()] or {}) do chatText:AddMessage(message) end
end
function UI.JourneyLines(events)
 local lines={}
 for i=#events,math.max(1,#events-5),-1 do
  local e=events[i];local quest=e.quest_title and e.quest_title~='' and e.quest_title or ('Quest #'..tostring(e.quest_id or '?'))
  local actions={quest_completed=quest..' · turned in',quest_accepted=quest..' · accepted',quest_progress=quest..' · progress',quest_removed=quest..' · left quest log',level_up='Reached level '..tostring(e.level or '?'),equipment_changed='Equipment changed',death='Fell in battle',resurrected='Back on your feet',note='A note to remember',session_start='Back in Azeroth',snapshot='Your journey began'}
  local action=actions[e.kind]or('Arrived in '..(e.subzone~='' and e.subzone or e.zone or 'Azeroth'))
  lines[#lines+1]='|cffefbb70'..safe(action)..'|r\n'..safe(e.zone)..(e.subzone and e.subzone~='' and (' / '..safe(e.subzone)) or '')..' · Level '..tostring(e.level or '?')
  if e.kind=='note' and e.note and e.note~=''then lines[#lines+1]=safe(e.note)
  elseif e.kind=='quest_progress' and e.objectives and e.objectives[1]then lines[#lines+1]=safe(e.objectives[1].text or '')end
  lines[#lines+1]=''
 end
 return table.concat(lines,'\n')
end
local function updateJourney()
 if journeyText then journeyText:SetText(UI.JourneyLines(entries))end
end
function UI.Update(person,events)
 if not window or not window:IsShown()then return false end
 if person.id~=key()then UI.Show(person,events);return true end
 character=person;entries=events or entries
 UI.nameLabel:SetText(safe(person.name));UI.identity:SetText('Level '..tostring(person.level)..'\n'..safe(person.zone))
 UI.compactIdentity:SetText(safe(person.name)..' · Lv '..tostring(person.level))
 updateJourney();return true
end
function UI.UpdateAIStatus()
 local transport=WispCompanionTransport
 local available=transport and transport.IsReaderAvailable and transport.IsReaderAvailable()
 local text=available==false and 'Desktop reader offline · start Jornia desktop' or UI.ConnectionState=='connected' and 'AI ready · last response confirmed' or UI.ConnectionState=='attention' and 'AI needs attention · check Jornia desktop' or 'AI connection not checked · send a question to try'
 if UI.aiStatus then UI.aiStatus:SetText(text)end
 if connectButton then connectButton.caption:SetText('Settings')end
end
local function selectTab(which)
 tab=which;chat:SetShown(mode=='compact' or which=='chat');journey:SetShown(mode=='full' and which=='journey')
 for name,b in pairs(tabs) do b:SetBackdropColor(name==which and .13 or .06,name==which and .2 or .1,name==which and .34 or .17,1) end
 if which=='journey' then updateJourney() end
 if UI.title then UI.title:SetText(which=='journey' and 'Your story so far.' or 'Talk to Jornia') end
end
local function updatePending()
 if not input then return end
 local waiting=pending and pending.character==key()
 -- One shared transport operation, independent of window size or selected character.
 if UI.ShareContext then UI.ShareContext:SetEnabled(not pending) end
 if pending then
  if input.Disable then input:Disable() end;input:EnableMouse(false);input:ClearFocus();sendButton:Disable()
 else
  if input.Enable then input:Enable() end;input:EnableMouse(true);sendButton:Enable()
 end
 if waiting then status:SetText(pending.connection and 'Checking invite code…' or pending.progress or 'Sending your message…')
 elseif pending then status:SetText('A reply is pending for another character.')
 elseif not UI.OnSend then status:SetText(UI.Config.disconnectedText or 'Preview · AI disconnected. Nothing is sent.')
 else status:SetText('Enter to send · your current character only') end
end
local function layout()
 if layoutBusy or not window then return end;layoutBusy=true
 local w,h=window:GetWidth(),window:GetHeight()
 local minW,minH=mode=='compact' and 340 or 760,mode=='compact' and math.max(260,(UI.Config.transferHeight or 0)+160)or 500
 w=math.max(minW,math.min(w,mode=='compact' and 700 or 1100));h=math.max(minH,math.min(h,760))
 if window:GetWidth()~=w or window:GetHeight()~=h then window:SetSize(w,h) end
 sizes[mode]={w,h};lastW,lastH=w,h
 local compact=mode=='compact';rail:SetShown(not compact);UI.title:SetShown(not compact);footer:SetShown(not compact)
 UI.compactTitle:SetShown(compact);UI.compactIdentity:SetShown(compact);UI.previewLabel:SetShown(not compact)
 place(closeButton,w-42,-10,28,30);place(modeButton,w-160,-10,108,30)
 if UI.DragHandle then UI.DragHandle:SetWidth(w-176) end
 modeButton.caption:SetText(compact and 'Open journal' or 'Compact chat')
 if compact then
  place(chat,18,-70,w-36,h-84)
 else
  rail:SetHeight(h)
  place(chat,214,-94,w-236,h-131);place(journey,214,-94,w-236,h-131)
  UI.title:SetWidth(w-250);UI.previewLabel:SetWidth(w-420)
  footer:ClearAllPoints();footer:SetPoint('BOTTOMLEFT',214,12);footer:SetWidth(w-250)
 end
 if connectButton then place(connectButton,compact and w-120 or 20,compact and -42 or -308,100,28);UI.compactIdentity:SetWidth(compact and w-160 or 280) end
 local cw,ch=chat:GetWidth(),chat:GetHeight()
 place(chatText,0,0,cw-(UI.Config.transferWidth or 0),math.max(40,ch-(UI.SettingsOwnsContext and 76 or 104)))
 if UI.ShareContext and not UI.SettingsOwnsContext then place(UI.ShareContext,0,-ch+88,22,22);UI.shareLabel:ClearAllPoints();UI.shareLabel:SetPoint('TOPLEFT',28,-ch+84);UI.shareLabel:SetWidth(cw-28) end
 place(input,6,-ch+63,cw-116,30);place(sendButton,cw-98,-ch+63,98,30)
 status:ClearAllPoints();status:SetPoint('TOPLEFT',0,-ch+22);status:SetWidth(cw)
 local jw,jh=journey:GetWidth(),journey:GetHeight()
 journeyText:SetWidth(jw);journeyText:SetHeight(jh-54)
 place(UI.note,6,-jh+34,jw-248,28);place(UI.noteButton,jw-224,-jh+34,110,30);place(UI.pauseButton,jw-106,-jh+34,106,30)
 if window.SetResizeBounds then window:SetResizeBounds(minW,minH,compact and 700 or 1100,760) end
 resize:ClearAllPoints();resize:SetPoint('BOTTOMRIGHT',-3,3)
 selectTab(tab);layoutBusy=false
end
function UI.SetTransferWidth(width)
 UI.Config.transferWidth=width;layout()
end
function UI.ShowConnection()
 if not window then return end
 if not UI.connectionWindow then
  local box=surface(UIParent,0,0,480,335,.035,.055,.09);box:ClearAllPoints();box:SetPoint('CENTER');box:SetFrameStrata('FULLSCREEN_DIALOG');box:EnableMouse(true)
  label(box,'Settings',20,-18,430,21,true)
  label(box,'Your character',20,-54,430,12,true)
  UI.ShareContext:SetParent(box);place(UI.ShareContext,16,-76,22,22);UI.ShareContext:Show()
  UI.shareLabel:SetParent(box);UI.shareLabel:ClearAllPoints();UI.shareLabel:SetPoint('TOPLEFT',44,-80);UI.shareLabel:SetWidth(410);UI.shareLabel:SetText('Use character and recent journey context');UI.shareLabel:Show()
  label(box,'Shares your name, race, class, level and quests when you send.\nJourney questions include up to three recorded moments.',20,-110,430,11)
  UI.aiStatus=label(box,'',20,-156,430,11,true)
  label(box,'Invite code · only needed to connect or replace access',20,-182,430,11)
  local edit=CreateFrame('EditBox',nil,box,'InputBoxTemplate');place(edit,26,-206,424,30);edit:SetFontObject(ChatFontNormal);edit:SetAutoFocus(false);edit:SetMaxLetters(64);edit:SetPassword(true)
  local result=label(box,'',20,-246,430,11,true)
  local submit
  local function activate()
   if pending then result:SetText('Wait for the current operation to finish.');return end
   local code=edit:GetText()or '';edit:SetText('');edit:ClearFocus()
   if not UI.OnConnect then result:SetText('Start the local Jornia reader first.');return end
   local ok,id,reason=pcall(UI.OnConnect,code,key());code=nil
   if not ok or type(id)~='string' then result:SetText(type(reason)=='string'and safe(reason)or'Connection was not accepted. Nothing was saved.');return end
   pending={id=id,character=key(),connection=true};submit:Disable();result:SetText('Checking invite code…');updatePending()
  end
  submit=button(box,'Connect AI',238,-285,108,activate)
  button(box,'Close',358,-285,98,function()edit:SetText('');edit:ClearFocus();box:Hide()end)
  edit:SetScript('OnEnterPressed',activate);edit:SetScript('OnEscapePressed',function()edit:SetText('');edit:ClearFocus();box:Hide()end)
  box:SetScript('OnHide',function()edit:SetText('');edit:ClearFocus()end)
  UI.connectionWindow=box;UI.connectionInput=edit;UI.connectionStatus=result;UI.connectionSubmit=submit
 end
 UI.UpdateAIStatus();UI.connectionWindow:Show();UI.connectionInput:ClearFocus()
end
function UI.ResolveConnection(id,owner,text,isError)
 if not pending or not pending.connection or pending.id~=id or pending.character~=owner then return false end
 pending=nil;updatePending()
 if UI.connectionStatus then UI.connectionStatus:SetText(safe(text));UI.connectionSubmit:Enable() end
 UI.ConnectionState=isError and 'attention' or 'connected';UI.UpdateAIStatus()
 return true
end
function UI.SetMode(value)
 if value~='full' and value~='compact' then return false end
 mode=value
 if window then window:StopMovingOrSizing();window:SetSize(unpack(sizes[mode]));layout() end
 return true
end
function UI.GetMode() return mode end
function UI.AddMessage(role,text,owner)
 owner=owner or key();messages[owner]=messages[owner] or {}
 table.insert(messages[owner],(role=='user' and '|cffefbb70YOU|r\n' or '|cffefbb70JORNIA|r\n')..safe(text))
 if owner==key() then renderMessages() end
end
-- Preserve the current conversation during an idle first-party UI upgrade.
function UI.RestoreSession(lines,draft)
 if pending or not input then return false end
 messages[key()]={}
 for i=math.max(1,#lines-199),#lines do if type(lines[i])=='string'then table.insert(messages[key()],lines[i])end end
 drafts[key()]=draft or '';input:SetText(drafts[key()]);renderMessages();updatePending();return true
end
function UI.SetProgress(requestID,owner,text,phase)
 if pending and pending.id==requestID and pending.character==owner then pending.progress=text;UI.Activity=phase or UI.Activity;if owner==key() then status:SetText(safe(text))end end
end
function UI.Resolve(requestID,owner,text,isError,metadata)
 if not pending or pending.id~=requestID or pending.character~=owner then return false end
 local sentDraft=pending.draft
 UI.Activity=isError and 'error' or 'ready'
 UI.ConnectionState=isError and 'attention' or 'connected';UI.UpdateAIStatus()
 pending=nil
 if isError and sentDraft then
  drafts[owner]=sentDraft
  if owner==key() then input:SetText(sentDraft) end
 end
 if not isError then
  UI.AddMessage('assistant',text,owner);drafts[owner]=''
  if metadata and metadata.foreverdb then
   table.insert(messages[owner],'|cffdcb675|Hwispsource:foreverdb|hData from ForeverDB.net — https://foreverdb.net|h|r');if owner==key() then renderMessages() end
  end
  if owner==key() then input:SetText('') end
 end
 updatePending()
 if isError and owner==key() then status:SetText(text and text~='' and safe(text) or 'No confirmed reply. Draft kept; no automatic resend.')
 elseif owner==key() and metadata then status:SetText('Reply received · '..tostring(metadata.sources or 0)..' sources · details in desktop') end
 return true
end
function UI.Show(person,events)
 if input then drafts[key()]=input:GetText() or '' end
 character=person or character;entries=events or entries
 if not window then
  window=surface(UIParent,50,-20,860,540,.035,.055,.09);window:ClearAllPoints();window:SetPoint('CENTER');window:SetFrameStrata('DIALOG')
  window:SetBackdropColor(.035,.055,.09,.86)
  window:SetMovable(true);window:SetResizable(true);window:SetClampedToScreen(true);window:EnableMouse(true);window:RegisterForDrag('LeftButton')
  window:SetScript('OnDragStart',window.StartMoving);window:SetScript('OnDragStop',window.StopMovingOrSizing)
  window:SetScript('OnHide',window.StopMovingOrSizing)
  window:SetScript('OnSizeChanged',layout)
  -- Also handles renderers/clients that do not emit OnSizeChanged during a drag.
  window:SetScript('OnUpdate',function()if window:GetWidth()~=lastW or window:GetHeight()~=lastH then layout() end end)
  rail=surface(window,0,0,190,540,.045,.07,.12)
  label(rail,'JORNIA.',22,-23,150,30);label(rail,'YOUR CHARACTER',22,-88,150,10,true)
  UI.nameLabel=label(rail,'',22,-113,152,21);UI.identity=label(rail,'',22,-148,152,11)
  label(rail,'YOUR FIELD JOURNAL',22,-210,155,9,true)
  tabs={journey=button(rail,'Your journey',16,-238,158,function()UI.ShowJourney()end),chat=button(rail,'Talk to Jornia',16,-278,158,function()UI.ShowChat()end)}
  local avatar=rail:CreateTexture(nil,'ARTWORK');avatar:SetPoint('TOPLEFT',20,-350);avatar:SetSize(64,64);avatar:SetTexture('Interface\\AddOns\\Wisp\\Media\\avatar')
  label(rail,'A little company.',22,-420,150,12);label(rail,UI.Config.railLabel or 'Local journal\nPreview · not released',22,-452,150,9,true)
  UI.previewLabel=label(window,UI.Config.runtimeLabel or 'LUA PREVIEW · NOT A LIVE GAME CONNECTION',214,-20,420,9,true)
  UI.compactTitle=label(window,'JORNIA',18,-16,100,19);UI.compactIdentity=label(window,'',18,-43,280,11,true)
  modeButton=button(window,'Compact chat',700,-10,108,function()UI.SetMode(mode=='full' and 'compact' or 'full')end)
  closeButton=button(window,'×',810,-10,28,function()window:Hide()end)
  connectButton=button(window,'Settings',20,-308,100,function()UI.ShowConnection()end)
  closeButton.caption:ClearAllPoints();closeButton.caption:SetPoint('CENTER');closeButton.caption:SetWidth(26);closeButton.caption:SetJustifyH('CENTER')
  UI.title=label(window,'Your story so far.',214,-56,600,20)
  chat=CreateFrame('Frame',nil,window);journey=CreateFrame('Frame',nil,window)
  chatText=CreateFrame('ScrollingMessageFrame',nil,chat);chatText:SetFont('Fonts\\FRIZQT__.TTF',13,'');chatText:SetJustifyH('LEFT');chatText:SetFading(false);chatText:SetMaxLines(200);chatText:EnableMouseWheel(true)
  if chatText.SetHyperlinksEnabled then chatText:SetHyperlinksEnabled(true) end;chatText:EnableMouse(true)
  chatText:SetScript('OnHyperlinkClick',function(_,link)if link=='wispsource:foreverdb' then
   if not UI.sourceWindow then
    local box=surface(UIParent,0,0,420,150,.035,.055,.09);box:ClearAllPoints();box:SetPoint('CENTER');box:SetFrameStrata('FULLSCREEN_DIALOG')
    label(box,'Data from ForeverDB.net',18,-16,370,18,true);label(box,'Copy this link to open the source in your browser.',18,-48,380,12)
    local edit=CreateFrame('EditBox',nil,box,'InputBoxTemplate');place(edit,24,-76,370,28);edit:SetFontObject(ChatFontNormal);edit:SetAutoFocus(false);edit:SetText('https://foreverdb.net')
    button(box,'Close',298,-112,96,function()box:Hide()end);UI.sourceWindow=box;UI.sourceEdit=edit
   end
   UI.sourceWindow:Show();UI.sourceEdit:SetFocus();UI.sourceEdit:HighlightText()
  end end)
  chatText:SetScript('OnMouseWheel',function(self,delta)if delta>0 then if self.ScrollUp then self:ScrollUp() end elseif self.ScrollDown then self:ScrollDown() end end)
  input=CreateFrame('EditBox',nil,chat,'InputBoxTemplate');input:SetFontObject(ChatFontNormal);input:SetAutoFocus(false);input:SetMaxLetters(1000)
  input:SetScript('OnEscapePressed',function(self)self:ClearFocus()end)
  input:SetScript('OnTextChanged',function(self)drafts[key()]=self:GetText() or '' end)
  local function send()
   if pending then return end
   local question=input:GetText() or '';if not question:find('%S') then return end
   if not UI.OnSend then status:SetText('AI is disconnected. Nothing was sent.');return end
   -- Sender must acknowledge acceptance with an operation ID. Nil/error preserves the draft.
   local ok,id,reason=pcall(UI.OnSend,question,key(),UI.ShareContext and UI.ShareContext:GetChecked() or false)
   if not ok or type(id)~='string' or id=='' then status:SetText(type(reason)=='string' and safe(reason) or 'No acceptance confirmed. Draft kept; check Jornia.');return end
   pending={id=id,character=key(),draft=question};UI.Activity='sending';UI.AddMessage('user',question);drafts[key()]='';input:SetText('');input:ClearFocus();updatePending()
  end
  input:SetScript('OnEnterPressed',send);sendButton=button(chat,'Ask Jornia',490,-207,98,send)
  status=label(chat,'',0,-253,612,10,true)
  UI.ShareContext=CreateFrame('CheckButton',nil,chat,'UICheckButtonTemplate');UI.ShareContext:SetChecked(true)
  UI.ShareContext:SetScript('OnClick',function(self)contextSharing[key()]=self:GetChecked() and true or false;if UI.OnContextPreference then UI.OnContextPreference(key(),contextSharing[key()])end end)
  UI.shareLabel=label(chat,'Use character and recent journey context',28,-232,580,11)
  if UI.SettingsOwnsContext then UI.ShareContext:Hide();UI.shareLabel:Hide()end
  UI.ShareContext:SetScript('OnEnter',function(self)if GameTooltip then GameTooltip:SetOwner(self,'ANCHOR_RIGHT');GameTooltip:SetText('Current character context');GameTooltip:AddLine('Reads level, area, class/race IDs, gear IDs, profession ranks, a limited quest/progress snapshot, and recent confirmed turn-ins when you send.',1,1,1,true);GameTooltip:AddLine('Selected text goes through Jornia to Berget. Images stay local. Missing fields remain unknown.',1,1,1,true);GameTooltip:Show()end end)
  UI.ShareContext:SetScript('OnLeave',function()if GameTooltip then GameTooltip:Hide()end end)

  journeyText=label(journey,'',0,-2,612,13);journeyText:SetJustifyV('TOP')
  UI.note=CreateFrame('EditBox',nil,journey,'InputBoxTemplate');UI.note:SetFontObject(ChatFontNormal);UI.note:SetAutoFocus(false)
  UI.noteButton=button(journey,'Save note',378,-210,110,function()local value=UI.note:GetText() or '';if value~='' and UI.OnNote then UI.OnNote(value);UI.note:SetText('');updateJourney()end end)
  UI.pauseButton=button(journey,'Pause',502,-210,106,function(self)if UI.OnPause then UI.OnPause();self.caption:SetText('Paused')end end)
  footer=label(window,UI.Config.footer or 'Your story stays local. In-game AI connection is still in development.',214,-506,620,9,true)
  resize=CreateFrame('Button',nil,window);resize:SetSize(22,22);resize:EnableMouse(true)
  label(resize,'//',4,-4,18,14,true)
  resize:SetScript('OnMouseDown',function(_,which)if which=='LeftButton' then window:StartSizing('BOTTOMRIGHT') end end)
  resize:SetScript('OnMouseUp',function()window:StopMovingOrSizing();layout()end)
  UI.Controls={Window=window,Input=input,Resize=resize,Mode=modeButton,Connect=connectButton,Status=status,Chat=chat,Journey=journey,Messages=chatText,Send=sendButton,JourneyText=journeyText}
  -- A dedicated title hit region also works where native drag events do not bubble.
  local drag=CreateFrame('Frame',nil,window);drag:SetPoint('TOPLEFT');drag:SetSize(684,44)
  drag:SetFrameLevel(window:GetFrameLevel()+10);drag:EnableMouse(true)
  drag:SetScript('OnMouseDown',function(_,which)if which=='LeftButton' then window:StartMoving() end end)
  drag:SetScript('OnMouseUp',function()window:StopMovingOrSizing()end)
  UI.DragHandle=drag

 end
 UI.nameLabel:SetText(safe(character.name or 'Character'));UI.identity:SetText('Level '..tostring(character.level or '?')..'\n'..safe(character.zone))
 UI.compactIdentity:SetText(safe(character.name or 'Character')..' · Lv '..tostring(character.level or '?'))
 if contextSharing[key()]==nil then contextSharing[key()]=character.share_current_context~=false end
 UI.ShareContext:SetChecked(contextSharing[key()]);input:SetText(drafts[key()] or '');updateJourney();renderMessages();updatePending();layout();window:Show()
 return window
end
function UI.ShowJourney()if window then if mode=='compact' then UI.SetMode('full') end;selectTab('journey')end end
function UI.ShowChat()if window then selectTab('chat')end end

function UI.IsShown() return window and window:IsShown() or false end
function UI.Hide() if window then window:Hide() end end
