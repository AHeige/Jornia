-- In-chat arcane sigil around the existing checked QR transport.
-- Modules and quiet zone stay static; only the outer rim breathes while waiting.
local T,UI,QR=WispCompanionTransport,WispJournalUI,JorniaQR
if not T or not UI or not QR then return end
if T.IsPending() then error('Wait for the current operation before replacing the QR display')end
if T.StopSigil then T.StopSigil()end
local slots={};T.QRSlots=slots;local card=T.Frame
local reserved=math.max(110,89/card:GetEffectiveScale())+32
local sigil,emblem,caption,halo,attached
local themes={arcane={.48,.44,.95},grove={.25,.72,.55},ember={.93,.50,.27}}
local theme=UI.Config.sigilTheme or (WispDB and WispDB.sigil_theme) or 'arcane'
if not themes[theme]then theme='arcane'end
local reduced=UI.Config.reduceMotion or (WispDB and WispDB.sigil_reduce_motion) or false
local ornaments={}
function UI.SetSigilTheme(value)
 if not themes[value]then return false end
 theme=value;UI.Config.sigilTheme=value;if WispDB then WispDB.sigil_theme=value end;return true
end
function UI.SetSigilReducedMotion(value)
 reduced=value and true or false;UI.Config.reduceMotion=reduced;if WispDB then WispDB.sigil_reduce_motion=reduced end
end
local function makeSigil(parent)
 sigil=CreateFrame('Frame',nil,parent,'BackdropTemplate');UI.Sigil=sigil;sigil:SetSize(reserved-8,reserved+16);sigil:SetPoint('TOPRIGHT',parent,'TOPRIGHT',0,0)
 sigil:SetFrameLevel(parent:GetFrameLevel()+4)
 sigil:SetBackdrop({bgFile='Interface\\Buttons\\WHITE8X8',edgeFile='Interface\\Buttons\\WHITE8X8',edgeSize=1})
 sigil:SetBackdropColor(.025,.033,.065,1)
 halo=CreateFrame('Frame',nil,sigil,'BackdropTemplate');halo:SetPoint('TOPLEFT',3,-3);halo:SetPoint('BOTTOMRIGHT',-3,3)
 halo:SetBackdrop({edgeFile='Interface\\Buttons\\WHITE8X8',edgeSize=1})
 for _,point in ipairs({'TOPLEFT','TOPRIGHT','BOTTOMLEFT','BOTTOMRIGHT'})do
  for _,size in ipairs({{16,2},{2,16}})do
   local t=sigil:CreateTexture(nil,'OVERLAY');t:SetSize(size[1],size[2]);t:SetPoint(point,sigil,point,0,0);ornaments[#ornaments+1]=t
  end
 end
 emblem=CreateFrame('Frame',nil,sigil);emblem:SetSize(70,70);emblem:SetPoint('CENTER',0,9)
 for i,size in ipairs({52,46,28,22,8})do
  local t=emblem:CreateTexture(nil,'ARTWORK',nil,i);t:SetSize(size,size);t:SetPoint('CENTER');t:SetRotation(math.pi/4)
  if i%2==0 then t:SetColorTexture(.025,.033,.065,1)else ornaments[#ornaments+1]=t end
 end
 local name=emblem:CreateFontString(nil,'OVERLAY');name:SetFont('Fonts\\FRIZQT__.TTF',10,'');name:SetPoint('TOP',emblem,'BOTTOM',0,-7);name:SetText('J O R N I A');name:SetTextColor(.78,.82,.94,1)
 caption=sigil:CreateFontString(nil,'OVERLAY');caption:SetFont('Fonts\\FRIZQT__.TTF',10,'');caption:SetPoint('BOTTOM',0,8);caption:SetWidth(reserved-20);caption:SetTextColor(.76,.79,.91,1)
 sigil:EnableMouse(true)
 sigil:SetScript('OnMouseUp',function(_,button)
  if button~='RightButton'then return end
  if IsShiftKeyDown and IsShiftKeyDown()then UI.SetSigilReducedMotion(not reduced)
  else UI.SetSigilTheme(({arcane='grove',grove='ember',ember='arcane'})[theme])end
  local refresh=sigil:GetScript('OnEnter');if refresh then refresh()end
 end)
 sigil:SetScript('OnEnter',function()if GameTooltip then GameTooltip:SetOwner(sigil,'ANCHOR_LEFT');GameTooltip:SetText(theme:sub(1,1):upper()..theme:sub(2)..' sigil');GameTooltip:AddLine('Right-click to change theme.',.8,.83,.9);GameTooltip:AddLine(reduced and 'Motion reduced.' or 'Gentle motion enabled.',.8,.83,.9);GameTooltip:AddLine('Shift-right-click to toggle motion.',.8,.83,.9);GameTooltip:Show()end end)
 sigil:SetScript('OnLeave',function()if GameTooltip then GameTooltip:Hide()end end)
 attached=parent
end
local observer=CreateFrame('Frame');local elapsed=0
observer:SetScript('OnUpdate',function(_,dt)
 elapsed=elapsed+dt;if elapsed<.08 then return end;elapsed=0
 local parent=UI.Controls and UI.Controls.Chat;if not parent then return end
 if UI.GetMode()=='compact' and UI.Controls.Window:GetHeight()<reserved+204 then UI.Controls.Window:SetHeight(reserved+204)end
 if not sigil then makeSigil(parent)elseif attached~=parent then sigil:SetParent(parent);sigil:ClearAllPoints();sigil:SetPoint('TOPRIGHT',parent,'TOPRIGHT',0,0);attached=parent end
 local phase=UI.Activity or 'idle';local active=T.IsPending()
 local colors=phase=='error' and {.92,.61,.30}or themes[theme]
 local strength=active and .72 or .40
 if active and not reduced then strength=.64+.12*math.sin(GetTime()*2.4)end
 sigil:SetBackdropBorderColor(colors[1],colors[2],colors[3],strength)
 halo:SetBackdropBorderColor(colors[1],colors[2],colors[3],strength*.35)
 for _,texture in ipairs(ornaments)do texture:SetColorTexture(colors[1],colors[2],colors[3],.9)end
 emblem:SetShown(not card:IsShown())
 local captions={idle='A little company.',sending='Sending…',thinking='Thinking…',receiving='Reply arriving…',ready='Reply received',error='Try again'}
 caption:SetText(captions[phase]or captions.idle)
end)
T.StopSigil=function()observer:SetScript('OnUpdate',nil);if sigil then sigil:Hide()end end
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
 if not sigil then makeSigil(parent)end
 card:SetParent(parent);card:ClearAllPoints();card:SetPoint('CENTER',sigil,'CENTER',0,9)
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
UI.Config.transferWidth=reserved
UI.Config.transferHeight=reserved+16
if UI.SetTransferWidth then UI.SetTransferWidth(reserved)
elseif UI.Controls then
 -- Compatibility with the already loaded laboratory view; no reload needed.
 local function reserve()UI.Controls.Messages:SetWidth(UI.Controls.Chat:GetWidth()-reserved)end
 reserve();if not T.QRReserveHook then UI.Controls.Window:HookScript('OnUpdate',reserve);T.QRReserveHook=true end
end
UI.Config.footer='Selected text is sent to Jornia AI. QR images stay on this computer.'
