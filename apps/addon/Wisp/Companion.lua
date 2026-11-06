-- First-party companion UI. Not yet bundled or connected to the live OCR transport.
WispJournalUI = {}
local UI = WispJournalUI
local window,rail,banner,chat,journey,chatText,journeyText,status,input,tabs,sendButton,modeButton,closeButton,resize,footer
local character,entries,messages,drafts={},{},{},{}
local mode,tab='full','journey'
local sizes={full={860,540},compact={420,340}}
local pending,layoutBusy,lastW,lastH
local function key() return character.id or character.name or 'player' end
local function safe(s) return tostring(s or ''):gsub('|','||') end
local function label(parent,text,x,y,width,size,gold)
 local out=parent:CreateFontString(nil,'OVERLAY','GameFontHighlight')
 out:SetPoint('TOPLEFT',x,y);out:SetWidth(width);out:SetJustifyH('LEFT')
 out:SetFont('Fonts\\FRIZQT__.TTF',size or 12)
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
local function updateJourney()
 if not journeyText then return end
 local lines={}
 for i=#entries,math.max(1,#entries-4),-1 do
  local e=entries[i]
  local action=e.kind=='quest_completed' and ('Quest #'..tostring(e.quest_id)..' completed') or e.kind=='level_up' and ('Reached level '..tostring(e.level)) or e.kind=='note' and 'A note to remember' or 'A moment in '..(e.subzone or e.zone or '')
  lines[#lines+1]='|cffefbb70'..safe(action)..'|r\n'..safe(e.zone)..(e.subzone and e.subzone~='' and (' / '..safe(e.subzone)) or '')..' · Level '..tostring(e.level or '?')
  if e.kind=='note' and e.note~='' then lines[#lines+1]=safe(e.note) end
  lines[#lines+1]=''
 end
 journeyText:SetText(table.concat(lines,'\n'))
end
local function selectTab(which)
 tab=which;chat:SetShown(mode=='compact' or which=='chat');journey:SetShown(mode=='full' and which=='journey')
 for name,b in pairs(tabs) do b:SetBackdropColor(name==which and .13 or .06,name==which and .2 or .1,name==which and .34 or .17,1) end
 if which=='journey' then updateJourney() end
 if UI.title then UI.title:SetText(which=='journey' and 'Your story so far.' or 'A moment with Wisp.') end
end
local function updatePending()
 if not input then return end
 local waiting=pending and pending.character==key()
 -- One shared transport operation, independent of window size or selected character.
 if pending then
  if input.Disable then input:Disable() end;input:EnableMouse(false);input:ClearFocus();sendButton:Disable()
 else
  if input.Enable then input:Enable() end;input:EnableMouse(true);sendButton:Enable()
 end
 if waiting then status:SetText('Wisp is thinking…')
 elseif pending then status:SetText('A reply is pending for another character.')
 elseif not UI.OnSend then status:SetText('Preview · AI disconnected. Nothing is sent.')
 else status:SetText('Enter to send · your current character only') end
end
local function layout()
 if layoutBusy or not window then return end;layoutBusy=true
 local w,h=window:GetWidth(),window:GetHeight()
 local minW,minH=mode=='compact' and 340 or 760,mode=='compact' and 260 or 500
 w=math.max(minW,math.min(w,mode=='compact' and 700 or 1100));h=math.max(minH,math.min(h,760))
 if window:GetWidth()~=w or window:GetHeight()~=h then window:SetSize(w,h) end
 sizes[mode]={w,h};lastW,lastH=w,h
 local compact=mode=='compact';rail:SetShown(not compact);banner:SetShown(not compact);footer:SetShown(not compact)
 UI.compactTitle:SetShown(compact);UI.compactIdentity:SetShown(compact);UI.previewLabel:SetShown(not compact)
 place(closeButton,w-42,-10,28,30);place(modeButton,w-160,-10,108,30)
 modeButton.caption:SetText(compact and 'Open journal' or 'Compact chat')
 if compact then
  place(chat,18,-70,w-36,h-84)
 else
  rail:SetHeight(h);place(banner,212,-51,w-234,110)
  place(chat,214,-183,w-236,h-220);place(journey,214,-183,w-236,h-220)
  UI.title:SetWidth(w-290);UI.bannerCopy:SetWidth(w-290);UI.previewLabel:SetWidth(w-420)
  footer:ClearAllPoints();footer:SetPoint('BOTTOMLEFT',214,12);footer:SetWidth(w-250)
 end
 local cw,ch=chat:GetWidth(),chat:GetHeight()
 place(chatText,0,0,cw,math.max(80,ch-80))
 place(input,6,-ch+63,cw-116,30);place(sendButton,cw-98,-ch+63,98,30)
 status:ClearAllPoints();status:SetPoint('TOPLEFT',0,-ch+22);status:SetWidth(cw)
 local jw,jh=journey:GetWidth(),journey:GetHeight()
 journeyText:SetWidth(jw);journeyText:SetHeight(jh-54)
 place(UI.note,6,-jh+34,jw-248,28);place(UI.noteButton,jw-224,-jh+34,110,30);place(UI.pauseButton,jw-106,-jh+34,106,30)
 if window.SetResizeBounds then window:SetResizeBounds(minW,minH,compact and 700 or 1100,760) end
 resize:ClearAllPoints();resize:SetPoint('BOTTOMRIGHT',-3,3)
 selectTab(tab);layoutBusy=false
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
 table.insert(messages[owner],(role=='user' and '|cffefbb70YOU|r\n' or '|cffefbb70WISP|r\n')..safe(text))
 if owner==key() then renderMessages() end
end
function UI.Resolve(requestID,owner,text,isError)
 if not pending or pending.id~=requestID or pending.character~=owner then return false end
 pending=nil
 if not isError then
  UI.AddMessage('assistant',text,owner);drafts[owner]=''
  if owner==key() then input:SetText('') end
 end
 updatePending()
 if isError and owner==key() then status:SetText('No confirmed reply. Draft kept; no automatic resend.') end
 return true
end
function UI.Show(person,events)
 if input then drafts[key()]=input:GetText() or '' end
 character=person or character;entries=events or entries
 if not window then
  window=surface(UIParent,50,-20,860,540,.035,.055,.09);window:ClearAllPoints();window:SetPoint('CENTER');window:SetFrameStrata('DIALOG')
  window:SetMovable(true);window:SetResizable(true);window:SetClampedToScreen(true);window:EnableMouse(true);window:RegisterForDrag('LeftButton')
  window:SetScript('OnDragStart',window.StartMoving);window:SetScript('OnDragStop',window.StopMovingOrSizing)
  window:SetScript('OnHide',window.StopMovingOrSizing)
  window:SetScript('OnSizeChanged',layout)
  -- Also handles renderers/clients that do not emit OnSizeChanged during a drag.
  window:SetScript('OnUpdate',function()if window:GetWidth()~=lastW or window:GetHeight()~=lastH then layout() end end)
  rail=surface(window,0,0,190,540,.045,.07,.12)
  label(rail,'WISP.',22,-23,150,30);label(rail,'YOUR CHARACTER',22,-88,150,10,true)
  UI.nameLabel=label(rail,'',22,-113,152,21);UI.identity=label(rail,'',22,-148,152,11)
  label(rail,'YOUR FIELD JOURNAL',22,-210,155,9,true)
  tabs={journey=button(rail,'Your journey',16,-238,158,function()UI.ShowJourney()end),chat=button(rail,'Talk to Wisp',16,-278,158,function()UI.ShowChat()end)}
  local avatar=rail:CreateTexture(nil,'ARTWORK');avatar:SetPoint('TOPLEFT',20,-350);avatar:SetSize(64,64);avatar:SetTexture('Interface\\AddOns\\Wisp\\Media\\avatar')
  label(rail,'A little company.',22,-420,150,12);label(rail,'Local journal\nPreview · not released',22,-452,150,9,true)
  UI.previewLabel=label(window,'LUA PREVIEW · NOT A LIVE GAME CONNECTION',214,-20,420,9,true)
  UI.compactTitle=label(window,'WISP',18,-16,100,19);UI.compactIdentity=label(window,'',18,-43,280,11,true)
  modeButton=button(window,'Compact chat',700,-10,108,function()UI.SetMode(mode=='full' and 'compact' or 'full')end)
  closeButton=button(window,'×',810,-10,28,function()window:Hide()end)
  banner=surface(window,212,-51,626,110,.06,.1,.18)
  local art=banner:CreateTexture(nil,'BACKGROUND');art:SetAllPoints();art:SetTexture('Interface\\AddOns\\Wisp\\Media\\expedition');art:SetTexCoord(0,1,.25,.52);art:SetVertexColor(.45,.55,.8,1)
  UI.title=label(banner,'Your story so far.',18,-22,540,26);UI.bannerCopy=label(banner,'Recorded places, moments and milestones.',18,-67,540,12)
  chat=CreateFrame('Frame',nil,window);journey=CreateFrame('Frame',nil,window)
  chatText=CreateFrame('ScrollingMessageFrame',nil,chat);chatText:SetFont('Fonts\\FRIZQT__.TTF',13);chatText:SetJustifyH('LEFT');chatText:SetFading(false);chatText:SetMaxLines(200);chatText:EnableMouseWheel(true)
  chatText:SetScript('OnMouseWheel',function(self,delta)if delta>0 then if self.ScrollUp then self:ScrollUp() end elseif self.ScrollDown then self:ScrollDown() end end)
  input=CreateFrame('EditBox',nil,chat,'InputBoxTemplate');input:SetFontObject(ChatFontNormal);input:SetAutoFocus(false);input:SetMaxLetters(1000)
  input:SetScript('OnEscapePressed',function(self)self:ClearFocus()end)
  input:SetScript('OnTextChanged',function(self)drafts[key()]=self:GetText() or '' end)
  local function send()
   if pending then return end
   local question=input:GetText() or '';if not question:find('%S') then return end
   if not UI.OnSend then status:SetText('AI is disconnected. Nothing was sent.');return end
   -- Sender must acknowledge acceptance with an operation ID. Nil/error preserves the draft.
   local ok,id=pcall(UI.OnSend,question,key())
   if not ok or type(id)~='string' or id=='' then status:SetText('No acceptance confirmed. Draft kept; check Wisp.');return end
   pending={id=id,character=key()};UI.AddMessage('user',question);input:ClearFocus();updatePending()
  end
  input:SetScript('OnEnterPressed',send);sendButton=button(chat,'Ask Wisp',490,-207,98,send)
  status=label(chat,'',0,-253,612,10,true)
  journeyText=label(journey,'',0,-2,612,13);journeyText:SetJustifyV('TOP')
  UI.note=CreateFrame('EditBox',nil,journey,'InputBoxTemplate');UI.note:SetFontObject(ChatFontNormal);UI.note:SetAutoFocus(false)
  UI.noteButton=button(journey,'Save note',378,-210,110,function()local value=UI.note:GetText() or '';if value~='' and UI.OnNote then UI.OnNote(value);UI.note:SetText('');updateJourney()end end)
  UI.pauseButton=button(journey,'Pause',502,-210,106,function(self)if UI.OnPause then UI.OnPause();self.caption:SetText('Paused')end end)
  footer=label(window,'Your story stays local. In-game AI connection is still in development.',214,-506,620,9,true)
  resize=CreateFrame('Button',nil,window);resize:SetSize(22,22);resize:EnableMouse(true)
  label(resize,'//',4,-4,18,14,true)
  resize:SetScript('OnMouseDown',function(_,which)if which=='LeftButton' then window:StartSizing('BOTTOMRIGHT') end end)
  resize:SetScript('OnMouseUp',function()window:StopMovingOrSizing();layout()end)
  UI.Controls={Window=window,Input=input,Resize=resize,Mode=modeButton,Status=status,Chat=chat,Journey=journey,Messages=chatText}
 end
 UI.nameLabel:SetText(safe(character.name or 'Character'));UI.identity:SetText('Level '..tostring(character.level or '?')..'\n'..safe(character.zone))
 UI.compactIdentity:SetText(safe(character.name or 'Character')..' · Lv '..tostring(character.level or '?'))
 input:SetText(drafts[key()] or '');updateJourney();renderMessages();updatePending();layout();window:Show()
 return window
end
function UI.ShowJourney()if window then if mode=='compact' then UI.SetMode('full') end;selectTab('journey')end end
function UI.ShowChat()if window then selectTab('chat')end end
