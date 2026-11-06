-- First-party single-marker native log experiment. Inert until explicit test.
-- No chat transmission, AI, memory access, file patching or bulk/padding writes.
local function say(s) print("|cffffc466Wisp Native Test:|r " .. s) end
local clock = GetTimePreciseSec or GetTime
local keys = {"developerLogFilterNormal", "developerLogWriteToFile", "developerLog"}
local get = C_CVar and C_CVar.GetCVar or GetCVar
local set = C_CVar and C_CVar.SetCVar or SetCVar
local active
WispBridgeLabDB = WispBridgeLabDB or {schema=1}
WispBridgeLabDB.native_runs = WispBridgeLabDB.native_runs or {}
local function read(key)
    if type(get) ~= "function" then return nil,"GetCVar unavailable" end
    local ok,value=pcall(get,key)
    if not ok then return nil,tostring(value) end
    if issecretvalue and issecretvalue(value) then return nil,"secret CVar" end
    if type(value) ~= "string" or (value ~= "0" and value ~= "1") then return nil,"missing or non-boolean CVar" end
    return value
end
local function restore(reason)
    local r=active
    if not r then say("Idle; no settings to restore.");return end
    r.restore_reason=reason;r.restore_attempt_epoch=time();r.restore_attempt_uptime=clock()
    r.restored_values={};r.restore_errors={}
    -- Disable logging first if it was originally off, then restore file/filter.
    for i=#keys,1,-1 do
        local key=keys[i]
        if r.changed[key] then
            local ok,err=pcall(set,key,r.original[key])
            local value,readErr=read(key)
            r.restored_values[key]=value or "unreadable"
            if not ok or value~=r.original[key] then r.restore_errors[key]=tostring(err or readErr or "readback mismatch") end
        end
    end
    r.restore_complete=next(r.restore_errors)==nil
    if r.restore_complete then
        r.restored_epoch=time();r.restored_uptime=clock();active=nil
        say("Settings restored and verified. /reload may now save the local receipt.")
    else say("RESTORE INCOMPLETE. Run /wispnative stop to retry; preserve the receipt.") end
end
local frame=CreateFrame("Frame")
frame:RegisterEvent("PLAYER_LOGOUT")
frame:SetScript("OnEvent",function()if active then restore("logout_or_reload") end end)
SLASH_WISPNATIVELOG1="/wispnative"
SlashCmdList.WISPNATIVELOG=function(message)
    if message=="stop" then restore("manual");return end
    if message=="status" then
        say("C_Log.LogMessage="..type(C_Log and C_Log.LogMessage).."; "..(active and "test active" or "idle"))
        for _,key in ipairs(keys)do local value,err=read(key);say(key.."="..tostring(value or err))end
        return
    end
    local id=message:match("^test ([0-9a-f]+)$")
    if not id or #id<8 or #id>32 then say("/wispnative status | test <run-id> | stop");return end
    if active then say("A test is active. No additional marker written.");return end
    for _,old in ipairs(WispBridgeLabDB.native_runs)do
        if old.id==id then say("That run was already attempted; no replay.");return end
    end
    local version,build,_,interface=GetBuildInfo()
    local r={id=id,version=version,build=build,interface=interface,function_type=type(C_Log and C_Log.LogMessage),started_epoch=time(),started_uptime=clock(),original={},changed={}}
    table.insert(WispBridgeLabDB.native_runs,r)
    while #WispBridgeLabDB.native_runs>5 do table.remove(WispBridgeLabDB.native_runs,1)end
    if r.function_type~="function" then r.error="C_Log.LogMessage unavailable";say(r.error.."; no settings changed.");return end
    if type(set)~="function" or not C_Timer or type(C_Timer.After)~="function" then r.error="Required settings/timer API unavailable";say(r.error);return end
    for _,key in ipairs(keys)do
        local value,err=read(key)
        if not value then r.error=key..": "..tostring(err);say("Aborted: "..r.error..". No settings changed.");return end
        r.original[key]=value
    end
    active=r
    C_Timer.After(60,function()if active==r then restore("60_second_timeout") end end)
    local ok,err=pcall(function()
        r.enabled_values={}
        for _,key in ipairs(keys)do
            -- Record before calling: even a setter that mutates then errors
            -- must be included in restoration.
            if r.original[key]~="1" then
                r.changed[key]=true
                set(key,"1")
            end
            local value=read(key);r.enabled_values[key]=value or "unreadable"
            if value~="1" then error("Could not enable "..key) end
        end
        r.sent_uptime=clock();r.sent_epoch=time()
        local marker=string.format("WISPNATIVE1:%s:%.6f:%d",id,r.sent_uptime,r.sent_epoch)
        r.marker=marker
        C_Log.LogMessage(marker) -- Exactly one call; no padding or repeats.
        r.returned_uptime=clock();r.call_returned=true
    end)
    if not ok then r.error=tostring(err);say("Test failed: "..r.error);restore("test_error");return end
    say("ONE marker submitted: "..id.." at uptime "..string.format("%.6f",r.sent_uptime))
    say("Do not reload yet. Settings restore automatically after 60 seconds, or /wispnative stop.")
end
say("Loaded, idle. C_Log.LogMessage="..type(C_Log and C_Log.LogMessage)..". /wispnative status changes nothing.")
