-- First-party finite-session experiment. No chat logging or global input hooks.
local frame = CreateFrame("Frame", "WispOCRPanel", UIParent, "BackdropTemplate")
frame:SetSize(620, 460); frame:SetPoint("CENTER")
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
label(-54,12,"Enter sends this question to Wisp AI. Images stay on this Mac.")
local state=label(-80,16,"Ready — open the local reader before sending.")
local packet=label(-115,16,"")
local answer=label(-225,14,"Your reply will appear here. No journal excerpts are sent in this test.")
answer:SetHeight(170); answer:SetJustifyV("TOP")
local edit=CreateFrame("EditBox",nil,frame,"InputBoxTemplate")
edit:SetSize(575,28); edit:SetPoint("BOTTOMLEFT",24,22); edit:SetAutoFocus(false); edit:SetMaxLetters(160)
edit:SetScript("OnEscapePressed",function(self) self:ClearFocus() end)
local function check(s)
    local sum=0; for i=1,#s do sum=(sum+i*s:byte(i))%65521 end; return sum
end
local used, sequence, pending, accumulated = 0,0,nil,0
local load = C_AddOns and C_AddOns.LoadAddOn or LoadAddOn
local loaded = C_AddOns and C_AddOns.IsAddOnLoaded or IsAddOnLoaded
local function decode(s)
    if type(s)~="string" or #s>48000 or #s%2~=0 or s:find("[^0-9a-f]") then return nil end
    return (s:gsub("..",function(p) return string.char(tonumber(p,16)) end))
end
function WispOCRReceive(id,guid,status,hex)
    if not pending or id~=pending.id or guid~=pending.guid or UnitGUID("player")~=pending.guid then return end
    local text=decode(hex); if not text then return end
    if status=="sending" then
        state:SetText("Question read — sending to Wisp…")
    elseif status=="thinking" then
        state:SetText("Question received — Wisp is thinking…")
    elseif status=="done" or status=="error" then
        state:SetText(status=="done" and string.format("Reply received — %.1f s total",GetTime()-pending.started) or "Could not finish — your question is kept")
        answer:SetText(text:gsub("|","||")); packet:SetText(""); pending=nil
        edit:Enable(); edit:SetFocus()
    end
end
edit:SetScript("OnEnterPressed",function(self)
    if pending then return end
    local q=self:GetText():gsub("^%s+",""):gsub("%s+$",""):gsub("%s+"," ")
    if #q<1 or #q>160 or q:find("[^ -~]") or q:find("|") then
        state:SetText("Lab: use 1–160 plain English characters, without |."); return
    end
    if used>=96 then state:SetText("This experiment's inboxes are exhausted. Draft kept."); return end
    local guid=UnitGUID("player"); if not guid then return end
    sequence=sequence+1
    local id=WispOCRSession..string.format("%010d%06d",time(),sequence)
    local binding=string.format("%05d",check(guid))
    local sum=string.format("%05d",check(id..binding..q))
    pending={id=id,guid=guid,started=GetTime()}; accumulated=0
    -- Stable readable outbox. Restrict text so line wrapping cannot change its bytes.
    packet:SetText("WISPBEGIN "..id.." "..binding.." "..sum.."\n"..q.."\nWISPEND")
    answer:SetText(""); state:SetText("Reading your question locally…"); self:ClearFocus(); self:Disable()
end)
frame:SetScript("OnUpdate",function(_,dt)
    if not pending then return end
    if UnitGUID("player")~=pending.guid then
        pending=nil; packet:SetText(""); state:SetText("Character changed — previous reply will not be shown."); edit:Enable(); return
    end
    if GetTime()-pending.started>55 or used>=96 then
        state:SetText("No reply yet. Check Wisp on your Mac; question kept, no automatic resend.")
        pending=nil; packet:SetText(""); edit:Enable(); return
    end
    accumulated=accumulated+dt; if accumulated<.6 then return end; accumulated=0
    repeat used=used+1 until used>96 or not loaded(string.format("WispOCRInbox%03d",used))
    if used<=96 then load(string.format("WispOCRInbox%03d",used)) end
end)
SLASH_WISPOCR1="/wispocr"
SlashCmdList.WISPOCR=function() if frame:IsShown() then frame:Hide() else frame:Show() end end
frame:Hide()
print("|cffffc466Wisp OCR Lab:|r /wispocr opens the opt-in question reader. No capture until the local reader runs.")
