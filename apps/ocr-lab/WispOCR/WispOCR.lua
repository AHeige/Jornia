-- First-party finite-session experiment. No chat logging or global input hooks.
local frame = CreateFrame("Frame", "WispOCRPanel", UIParent, "BackdropTemplate")
frame:SetSize(620, 600); frame:SetPoint("CENTER")
frame:SetBackdrop({bgFile="Interface\\Buttons\\WHITE8X8", edgeFile="Interface\\Buttons\\WHITE8X8", edgeSize=1})
frame:SetBackdropColor(.025,.035,.055,1); frame:SetBackdropBorderColor(.7,.48,.19,1)
frame:EnableMouse(true); frame:SetMovable(true); frame:RegisterForDrag("LeftButton")
frame:SetScript("OnDragStart", frame.StartMoving); frame:SetScript("OnDragStop", frame.StopMovingOrSizing)
for _, p in ipairs({"TOPLEFT","TOPRIGHT","BOTTOMLEFT","BOTTOMRIGHT"}) do
    local t=frame:CreateTexture(nil,"OVERLAY"); t:SetSize(6,6); t:SetPoint(p); t:SetColorTexture(1,0,1,1)
end
local function label(y,size,text)
    local s=frame:CreateFontString(nil,"OVERLAY"); s:SetFont("Fonts\\FRIZQT__.TTF",size,"")
    s:SetPoint("TOPLEFT",20,y); s:SetWidth(580); s:SetJustifyH("LEFT"); s:SetTextColor(.94,.94,.92); s:SetText(text or ""); return s
end
label(-20,22,"WISP  /  COMPANION LAB")
label(-54,12,"Enter sends this question to Jornia AI. Images stay on this Mac.")
local state=label(-80,16,"Ready — open the local reader before sending.")
local packet=label(-145,12,"")
packet:SetFont("Fonts\\ARIALN.TTF",12,"");packet:SetHeight(180);packet:SetJustifyV("TOP");packet:SetWordWrap(false)
local answer=label(-320,14,"Your reply will appear here. No journal excerpts are sent in this test.")
answer:SetHeight(160); answer:SetJustifyV("TOP")
local edit=CreateFrame("EditBox",nil,frame,"InputBoxTemplate")
edit:SetSize(575,28); edit:SetPoint("BOTTOMLEFT",24,22); edit:SetAutoFocus(false); edit:SetMaxLetters(160)
edit:SetScript("OnEscapePressed",function(self) self:ClearFocus() end)
local function check(s)
    local sum=0; for i=1,#s do sum=(sum+i*s:byte(i))%65521 end; return sum
end
local share=CreateFrame("CheckButton",nil,frame,"UICheckButtonTemplate")
share:SetSize(22,22);share:SetPoint("TOPLEFT",20,-104);share:SetChecked(false)
local shareLabel=label(-106,12,"Include current level, area and a limited quest snapshot for this question")
shareLabel:ClearAllPoints();shareLabel:SetPoint("TOPLEFT",48,-106);shareLabel:SetWidth(545)
local navigate=CreateFrame("Button",nil,frame,"UIPanelButtonTemplate")
navigate:SetSize(160,24);navigate:SetPoint("BOTTOMLEFT",22,62);navigate:SetText("Show the way");navigate:Hide()
local suggested=nil
navigate:SetScript("OnClick",function()
 if not suggested then return end
 local ok,message=WispNavigation.ShowQuest(suggested.quest,suggested.expires,suggested.guid)
 state:SetText(message);if ok then navigate:Disable() end
end)
local function quote(s)
 return '"'..s:gsub('\\','\\\\'):gsub('"','\\"'):gsub('[%z\1-\31]',function(c) return string.format('\\u%04x',c:byte()) end)..'"'
end
local function snapshot(q)
 local rows={};local truncated=false
 if share:GetChecked() then
  local count=C_QuestLog and C_QuestLog.GetNumQuestLogEntries and C_QuestLog.GetNumQuestLogEntries() or 0
  for i=1,math.min(count,200) do
   local info=C_QuestLog.GetInfo(i)
   if info and not info.isHeader and type(info.questID)=='number' then
    if #rows>=4 then truncated=true;break end
    local title=info.title or '';if #title>80 then title='' end
    local complete=C_QuestLog.IsComplete and C_QuestLog.IsComplete(info.questID) and 1 or 0
    local waypoint=WispNavigation and WispNavigation.HasWaypoint(info.questID) and 1 or 0
    rows[#rows+1]='['..info.questID..','..quote(title)..','..complete..','..waypoint..']'
   end
  end
  if count>200 then truncated=true end
 end
 local zone=GetZoneText and GetZoneText() or '';if #zone>80 then zone='' end
 local _,build=GetBuildInfo()
 local function body()
  local current=share:GetChecked() and '['..time()..','..UnitLevel('player')..','..quote(zone)..','..quote(tostring(build))..','..(truncated and '1' or '0')..',['..table.concat(rows,',')..']]' or 'null'
  return '{"q":'..quote(q)..',"s":'..current..'}'
 end
 while #body()>400 and #rows>0 do table.remove(rows);truncated=true end
 return body()
end
local used, sequence, pending, accumulated = 0,0,nil,0
local load = C_AddOns and C_AddOns.LoadAddOn or LoadAddOn
local loaded = C_AddOns and C_AddOns.IsAddOnLoaded or IsAddOnLoaded
local function decode(s)
    if type(s)~="string" or #s>48000 or #s%2~=0 or s:find("[^0-9a-f]") then return nil end
    return (s:gsub("..",function(p) return string.char(tonumber(p,16)) end))
end
function WispOCRReceive(id,guid,status,hex,questID,actionExpires)
    if not pending or id~=pending.id or guid~=pending.guid or UnitGUID("player")~=pending.guid then return end
    local text=decode(hex); if not text then return end
    if status=="sending" then
        state:SetText("Question read — sending to Jornia…")
    elseif status=="thinking" then
        state:SetText("Question received — Jornia is thinking…")
    elseif status=="done" or status=="error" then
        state:SetText(status=="done" and string.format("Reply received — %.1f s total",GetTime()-pending.started) or "Could not finish — your question is kept")
        answer:SetText(text:gsub("|","||")); packet:SetText("")
        if status=="done" and type(questID)=="number" and questID>0 and questID%1==0 and type(actionExpires)=="number" and actionExpires>time() and actionExpires<=time()+90 then
         suggested={quest=questID,expires=actionExpires,guid=guid};navigate:Enable();navigate:Show()
        end
        pending=nil;share:SetChecked(false);share:Enable()
        edit:Enable(); edit:SetFocus()
    end
end
edit:SetScript("OnEnterPressed",function(self)
    if pending then return end
    local q=self:GetText():gsub("^%s+",""):gsub("%s+$",""):gsub("%s+"," ")
    if #q<1 or #q>160 then
        state:SetText("Please shorten your message. All languages are accepted."); return
    end
    if used>=96 then state:SetText("This experiment's inboxes are exhausted. Draft kept."); return end
    local guid=UnitGUID("player"); if not guid then return end
    sequence=sequence+1
    local id=WispOCRSession..string.format("%010d%06d",time(),sequence)
    local binding=string.format("%05d",check(guid))
    local wire=snapshot(q)
    if #wire>400 then state:SetText("Please shorten your message.");return end
    local sum=string.format("%05d",check(id..binding..wire))
    pending={id=id,guid=guid,started=GetTime()}; accumulated=0
    share:SetChecked(false);share:Disable()
    -- ASCII hex transports UTF-8 without requiring OCR to recognize the user's script.
    local hex=wire:gsub('.',function(c) return string.format('%02x',c:byte()) end)
    local lines={};for i=1,#hex,72 do lines[#lines+1]=hex:sub(i,i+71) end
    packet:SetText("WISPBEGIN2 "..id.." "..binding.." "..sum.."\n"..table.concat(lines,'\n').."\nWISPEND")
    suggested=nil;navigate:Hide()
    answer:SetText(""); state:SetText("Reading your question locally…"); self:ClearFocus(); self:Disable()
end)
frame:SetScript("OnUpdate",function(_,dt)
    if suggested and (suggested.guid~=UnitGUID("player") or suggested.expires<=time()) then suggested=nil;navigate:Hide() end
    if not pending then return end
    if UnitGUID("player")~=pending.guid then
        pending=nil;share:SetChecked(false);share:Enable();suggested=nil;navigate:Hide(); packet:SetText(""); state:SetText("Character changed — previous reply will not be shown."); edit:Enable(); return
    end
    if GetTime()-pending.started>55 or used>=96 then
        state:SetText("No reply yet. Check Jornia on your Mac; question kept, no automatic resend.")
        pending=nil;share:SetChecked(false);share:Enable(); packet:SetText(""); edit:Enable(); return
    end
    accumulated=accumulated+dt; if accumulated<.6 then return end; accumulated=0
    repeat used=used+1 until used>96 or not loaded(string.format("WispOCRInbox%03d",used))
    if used<=96 then load(string.format("WispOCRInbox%03d",used)) end
end)
SLASH_WISPOCR1="/wispocr"
SlashCmdList.WISPOCR=function() if frame:IsShown() then frame:Hide() else frame:Show() end end
frame:Hide()
print("|cffffc466Wisp OCR Lab:|r /wispocr opens the opt-in question reader. No capture until the local reader runs.")

WispOCRControls={Frame=frame,Edit=edit,Share=share,Navigate=navigate,Packet=packet,State=state}
