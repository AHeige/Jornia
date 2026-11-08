-- Optional QR laboratory display, sharing the existing checked chat protocol.
-- Static while read: no animation, flashing, new window or automatic AI replay.
local T,UI,QR=WispCompanionTransport,WispJournalUI,JorniaQR
if not T or not UI or not QR then return end
if T.IsPending() then error('Wait for the current operation before replacing the QR display')end
local slots={};T.QRSlots=slots;local card=T.Frame
for _,region in ipairs({card:GetRegions()})do region:Hide()end
T.QRInitialized=nil
local function render(text)
 local body=text:match('^WISPBEGIN4 [^\n]+\n([%s0-9a-f]+)\nWISPEND$')
 if body then
  local raw=body:gsub('%s',''):gsub('..',function(pair)return string.char(tonumber(pair,16))end)
  text='JORNIAQR1\n'..text:match('^WISPBEGIN4 ([^\n]+)')..'\n'..raw
 end
 local matrix,n=QR.Encode(text)
 local parent=UI.Controls and UI.Controls.Chat
 assert(parent,'Open Jornia chat before sending')
 card:SetParent(parent);card:ClearAllPoints();card:SetPoint('TOPRIGHT',parent,'TOPRIGHT',0,0)
 card:SetFrameStrata('DIALOG');card:SetFrameLevel(parent:GetFrameLevel()+5)
 -- Integral physical module pitch avoids blurred subpixel edges at a given UI scale.
 local scale=card:GetEffectiveScale();local pitch=math.max(1,math.floor(110*scale/(n+8)))/scale
 local side=(n+8)*pitch;card:SetSize(side,side)
 card:SetBackdrop(nil)
 if T.Packet then T.Packet:Hide()end
 if not T.QRInitialized then
  for _,region in ipairs({card:GetRegions()})do region:Hide()end
  local bg=card:CreateTexture(nil,'BACKGROUND');bg:SetAllPoints();bg:SetColorTexture(1,1,1,1);T.QRInitialized=true
 end
 local used=0
 for y=0,n-1 do local x=0
  while x<n do
   if matrix[y][x]then
    local start=x;repeat x=x+1 until x>=n or not matrix[y][x]
    used=used+1;local t=slots[used];if not t then t=card:CreateTexture(nil,'ARTWORK');t:SetColorTexture(0,0,0,1);slots[used]=t end
    t:ClearAllPoints();t:SetPoint('TOPLEFT',(start+4)*pitch,-(y+4)*pitch);t:SetSize((x-start)*pitch,pitch);t:Show()
   else x=x+1 end
  end
 end
 for i=used+1,#slots do slots[i]:Hide()end
end
T.MaxBodyBytes=523
T.Display=render
local reserved=math.max(110,89/card:GetEffectiveScale())+8
UI.Config.transferWidth=reserved
if UI.SetTransferWidth then UI.SetTransferWidth(reserved)
elseif UI.Controls then
 -- Compatibility with the already loaded laboratory view; no reload needed.
 local function reserve()UI.Controls.Messages:SetWidth(UI.Controls.Chat:GetWidth()-reserved)end
 reserve();if not T.QRReserveHook then UI.Controls.Window:HookScript('OnUpdate',reserve);T.QRReserveHook=true end
end
UI.Config.footer='Selected text is sent to Jornia AI. QR images stay on this computer.'
