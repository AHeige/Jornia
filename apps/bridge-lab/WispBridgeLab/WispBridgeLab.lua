-- Research only. Never sends chat, enables logging, opens a socket or invokes a
-- shell. Four preinstalled load-on-demand slots test *inbound* file freshness.
local function say(s) print("|cffffc466Wisp Lab:|r " .. s) end
local panel
local function show(s)
    if not panel then
        panel = CreateFrame("Frame", "WispBridgeLabPanel", UIParent, "BasicFrameTemplateWithInset")
        panel:SetSize(540, 280); panel:SetPoint("CENTER")
        panel.TitleText:SetText("Wisp Bridge Lab — not a live AI connection")
        local label = panel:CreateFontString(nil, "OVERLAY", "GameFontHighlight")
        label:SetPoint("TOPLEFT", 20, -50); label:SetWidth(490); label:SetJustifyH("LEFT")
        panel.label = label; table.insert(UISpecialFrames, "WispBridgeLabPanel")
    end
    panel.label:SetText(s); panel:Show()
end
local function unhex(s)
    if type(s) ~= "string" or #s > 24000 or #s % 2 ~= 0 or s:find("[^0-9a-f]") then return nil end
    return (s:gsub("..", function(pair) return string.char(tonumber(pair, 16)) end))
end
function WispBridgeLabReceive(requestHex, textHex)
    local request, text = unhex(requestHex), unhex(textHex)
    if not request or not text then say("Invalid inbox data rejected."); return end
    WispBridgeLabDB.received = WispBridgeLabDB.received or {}
    table.insert(WispBridgeLabDB.received, {request_id=request, text=text, received_at=time(), uptime=GetTime()})
    while #WispBridgeLabDB.received > 12 do table.remove(WispBridgeLabDB.received,1) end
    show("FILE-INBOX EXPERIMENT\n\nRequest: " .. request .. "\n\n" .. text .. "\n\nThis is locally supplied test text, not an AI answer. Each slot loads once per UI lifetime.")
    say("Received inbox " .. request .. ".")
end
local function probe()
    local version, build, _, interface = GetBuildInfo()
    local APIs = {
        io_open = io and io.open, os_execute = os and os.execute,
        socket = socket, require = require, loadfile = loadfile, dofile = dofile,
        load_addon = C_AddOns and C_AddOns.LoadAddOn or LoadAddOn,
        logging_chat = LoggingChat, logging_combat = LoggingCombat,
        console_exec = ConsoleExec, debug_log = DebugLog,
    }
    local capabilities = {}
    for _, key in ipairs({"io_open","os_execute","socket","require","loadfile","dofile","load_addon","logging_chat","logging_combat","console_exec","debug_log"}) do
        capabilities[key] = type(APIs[key]); say(key .. ": " .. capabilities[key])
    end
    WispBridgeLabDB.probe = {version=version,build=build,interface=interface,at=time(),capabilities=capabilities}
    say("Probe stored locally. /reload saves the report; no API above was called.")
end
local f = CreateFrame("Frame")
f:RegisterEvent("PLAYER_LOGIN")
f:SetScript("OnEvent", function()
    WispBridgeLabDB = WispBridgeLabDB or {schema=1}
    say("Installed, idle. /wispbridge probe or /wispbridge inbox 1. No data transport is enabled.")
end)
SLASH_WISPBRIDGELAB1 = "/wispbridge"
SlashCmdList.WISPBRIDGELAB = function(message)
    if message == "probe" then probe(); return end
    local slot = tonumber((message:match("^inbox ([1-4])$")))
    if slot then
        local name = string.format("WispBridgeInbox%02d",slot)
        local loaded = C_AddOns and C_AddOns.IsAddOnLoaded or IsAddOnLoaded
        local load = C_AddOns and C_AddOns.LoadAddOn or LoadAddOn
        if not load or not loaded then say("LoadAddOn API unavailable.");return end
        if loaded(name) then say("Slot already consumed. Use another slot or /reload.");return end
        local ok, reason = load(name)
        say(ok and "Slot loaded." or ("Slot failed: " .. tostring(reason)))
        return
    end
    say("/wispbridge probe | inbox 1..4. Research only, no live bridge.")
end
