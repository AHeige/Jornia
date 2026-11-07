-- Original Wisp addon code. Local event journal. No network bridge, chat collection, input simulation or combat automation.
local frame = CreateFrame("Frame")
local current, lastZone, exportFrame
local function say(text) print("|cffb7d9bbWisp:|r " .. text) end
local function public(value)
    if issecretvalue and issecretvalue(value) then return nil end
    return value
end
local function text(value)
    value = public(value)
    if type(value) ~= "string" then return "" end
    return value:sub(1, 256)
end
local function number(value)
    value = public(value)
    if type(value) ~= "number" or value ~= value then return 0 end
    return math.floor(value)
end
local function quote(value)
    return '"' .. text(value):gsub('[%z\1-\31\\"]', function(c)
        if c == '"' then return '\\"' end
        if c == '\\' then return '\\\\' end
        return string.format('\\u%04x', string.byte(c))
    end) .. '"'
end
local function initialize()
    if WispDB == nil then
        WispDB = { schema = 1, installation_id = tostring(time()) .. "-" .. tostring(math.random(100000,999999)), characters = {} }
    end
    if type(WispDB) ~= "table" or WispDB.schema ~= 1 or type(WispDB.characters) ~= "table" then say("Unsupported journal version. Existing data was preserved."); return false end
    local id = text(UnitGUID("player"))
    if id == "" then return false end
    local c = WispDB.characters[id]
    if not c then c = { id = id, sequence = 0, events = {}, recording = WispDB.consent == "accepted" }; WispDB.characters[id] = c end
    local name,surname=UnitName("player");c.realm=text(GetRealmName());c.name=text(name)
    if type(surname)=="string" and surname~="" and surname~=c.realm then c.name=c.name.." "..text(surname) end
    -- WoW saves this in the actual character directory. It links that directory
    -- to the account journal without guessing from names or numeric realms.
    WispCharacterDB = { schema = 1, installation_id = WispDB.installation_id, character_id = id }
    current = c
    return true
end
local function record(kind, note, questID, observedLevel)
    if not current and not initialize() then return end
    local version, build, _, interface = GetBuildInfo()
    local zone, subzone = text(GetZoneText()), text(GetSubZoneText())
    local level = observedLevel and number(observedLevel) or number(UnitLevel("player"))
    if zone == "" or level < 1 then return end
    if kind == "quest_completed" and number(questID) < 1 then return end
    current.sequence = current.sequence + 1
    local event = {
        id = WispDB.installation_id .. ":" .. current.id .. ":" .. current.sequence,
        kind = kind, observed_at = time(), zone = zone, subzone = subzone,
        level = level, note = text(note), quest_id = number(questID),
        client_version = text(version), client_build = text(build), interface = number(interface), session_id = text(current.session_id)
    }
    if kind == "quest_completed" and WispClientState then
        local q=WispClientState.Quest(questID)
        if q then event.quest_title=q.title;event.objectives=q.objectives end
    end
    table.insert(current.events, event)
    while #current.events > 2000 do table.remove(current.events, 1); current.dropped = (current.dropped or 0) + 1 end
    lastZone = zone .. ":" .. subzone
end
WispJournalRuntime={
 Current=function() if not current then initialize() end;return current end,
 Record=function(kind,questID,level,details)
  if not current or not current.recording or WispDB.consent~="accepted" then return end
  record(kind,nil,questID,level)
  if details then local e=current.events[#current.events];for k,v in pairs(details)do e[k]=v end end
 end
}
local function export()
    if not current and not initialize() then say("Log in to a character first."); return end
    if #current.events == 0 then record("snapshot") end
    local parts = {}
    for index=math.max(1, #current.events-99),#current.events do
        local e=current.events[index]
        table.insert(parts, '{"id":' .. quote(e.id) .. ',"kind":' .. quote(e.kind) .. ',"observed_at":' .. e.observed_at ..
            ',"zone":' .. quote(e.zone) .. ',"subzone":' .. quote(e.subzone) .. ',"level":' .. e.level ..
            ',"note":' .. quote(e.note) .. ',"quest_id":' .. e.quest_id .. ',"client_version":' .. quote(e.client_version) ..
            ',"client_build":' .. quote(e.client_build) .. ',"interface":' .. e.interface .. ',"session_id":' .. quote(e.session_id) .. '}')
    end
    local payload = 'WISP1:{"schema":1,"installation_id":' .. quote(WispDB.installation_id) .. ',"character":{"id":' .. quote(current.id) ..
        ',"name":' .. quote(current.name) .. ',"realm":' .. quote(current.realm) .. '},"events":[' .. table.concat(parts, ',') .. ']}'
    if not exportFrame then
        exportFrame = CreateFrame("Frame", "WispExportFrame", UIParent, "BasicFrameTemplateWithInset")
        exportFrame:SetSize(560, 330); exportFrame:SetPoint("CENTER"); exportFrame:SetMovable(true); exportFrame:EnableMouse(true)
        exportFrame:RegisterForDrag("LeftButton"); exportFrame:SetScript("OnDragStart", exportFrame.StartMoving); exportFrame:SetScript("OnDragStop", exportFrame.StopMovingOrSizing)
        exportFrame.TitleText:SetText("Wisp — local adventure export")
        local help = exportFrame:CreateFontString(nil,"OVERLAY","GameFontHighlightSmall")
        help:SetPoint("TOPLEFT",18,-38); help:SetWidth(520); help:SetJustifyH("LEFT")
        help:SetText("This contains only this character's Wisp journal. Nothing is sent automatically.\nCtrl+A, Ctrl+C; then paste into Import in the Wisp companion.")
        local scroll = CreateFrame("ScrollFrame",nil,exportFrame,"UIPanelScrollFrameTemplate")
        scroll:SetPoint("TOPLEFT",18,-84);scroll:SetPoint("BOTTOMRIGHT",-34,18)
        local edit = CreateFrame("EditBox",nil,scroll)
        edit:SetMultiLine(true);edit:SetFontObject(ChatFontNormal);edit:SetWidth(490);edit:SetAutoFocus(false)
        edit:SetScript("OnEscapePressed",function() exportFrame:Hide() end)
        scroll:SetScrollChild(edit); exportFrame.edit = edit
        table.insert(UISpecialFrames, "WispExportFrame")
    end
    exportFrame:Show();exportFrame.edit:SetText(payload);exportFrame.edit:SetFocus();exportFrame.edit:HighlightText()
end
local function beginSession()
    current.session_id = WispDB.installation_id .. ":" .. current.id .. ":session:" .. tostring(time()) .. ":" .. tostring(current.sequence + 1)
    record("session_start")
end
local function acceptJournal()
    WispDB.consent = "accepted"; current.recording = true; beginSession()
    if WispClientState then WispClientState.RecordingChanged() end
    say("Journal active. Your locations, level-ups, completed quest IDs and notes stay local. /wisp stop pauses this character. The companion reads saved files after /reload or logout.")
end
if StaticPopupDialogs then
    StaticPopupDialogs.WISP_JOURNAL_CONSENT = {
        text = "Wisp can automatically keep a local adventure journal for your characters: sessions, areas, levels, quest progress, equipped item IDs, professions and your notes. No chat is collected or uploaded. The companion can read Wisp's saved file after /reload or logout. You can pause with /wisp stop. Start recording?",
        button1 = "Start journal", button2 = "Not now", timeout = 0, whileDead = true, hideOnEscape = true,
        OnAccept = function() acceptJournal() end,
        OnCancel = function() WispDB.consent = "declined"; current.recording = false end
    }
end
SLASH_WISP1 = "/wisp"
SlashCmdList.WISP = function(message)
    if not current and not initialize() then return end
    local command, rest = message:match("^(%S*)%s*(.-)$")
    if command == "start" then acceptJournal()
    elseif command == "stop" then current.recording = false;say("Journal paused. Existing entries remain on this computer.")
    elseif command == "mark" then record("snapshot");say("Current area recorded locally.")
    elseif command == "note" and rest ~= "" then record("note",rest);say("Note saved locally.")
    elseif command == "export" then export()
    elseif command == "clear" then current.events = {};say("This character's addon journal cleared. Already imported companion data is separate.")
    else
        local version, build, _, interface = GetBuildInfo()
        say("/wisp start | stop | mark | note <text> | export | clear")
        say("Journal: " .. (current.recording and "on" or "paused") .. ". Client " .. text(version) .. " build " .. text(build) .. " interface " .. number(interface))
    end
end
frame:RegisterEvent("PLAYER_LOGIN");frame:RegisterEvent("PLAYER_LOGOUT");frame:RegisterEvent("ZONE_CHANGED_NEW_AREA");frame:RegisterEvent("ZONE_CHANGED");frame:RegisterEvent("PLAYER_LEVEL_UP");frame:RegisterEvent("QUEST_TURNED_IN")
frame:SetScript("OnEvent",function(_,event,arg1)
    if event == "PLAYER_LOGIN" then
        initialize()
        if not current then return end
        if WispDB.consent == nil then
            if StaticPopup_Show then StaticPopup_Show("WISP_JOURNAL_CONSENT") else say("Use /wisp start to enable the local journal.") end
        elseif WispDB.consent == "accepted" and current.recording then beginSession() end
        return
    end
    if not current or not current.recording or WispDB.consent ~= "accepted" then return end
    if event == "PLAYER_LOGOUT" then record("session_end"); return end
    if event == "QUEST_TURNED_IN" then record("quest_completed",nil,arg1)
    elseif event == "PLAYER_LEVEL_UP" then record("level_up",nil,nil,arg1)
    elseif text(GetZoneText()) .. ":" .. text(GetSubZoneText()) ~= lastZone then record("zone") end
end)
