-- First-party, explicitly started chat-log experiment. No AI, gameplay input,
-- general chat collection, background sends or automatic start.
local function say(s) print("|cffffc466Wisp Chat Test:|r " .. s) end
local clock = GetTimePreciseSec or GetTime
local frame = CreateFrame("Frame")
local pending = {}
local active
WispBridgeLabDB = WispBridgeLabDB or {schema=1}
WispBridgeLabDB.chat_runs = WispBridgeLabDB.chat_runs or {}
local function stop(reason)
    if not active then say("No test is active.");return end
    LoggingChat(active.original_logging)
    active.stopped_uptime=clock();active.stopped_epoch=time()
    active.stop_reason=reason or "manual"
    say("Stopped. Logging restored to " .. tostring(LoggingChat()) .. ". /reload now saves results.")
    active=nil; pending={}
end
frame:RegisterEvent("CHAT_MSG_WHISPER")
frame:RegisterEvent("CHAT_MSG_WHISPER_INFORM")
frame:RegisterEvent("PLAYER_LOGOUT")
frame:SetScript("OnEvent",function(_,event,message)
    if event=="PLAYER_LOGOUT" then if active then stop("logout") end;return end
    if not active or type(message)~="string" then return end
    local record=pending[message]
    if not record then return end -- Never retain other chat.
    if event=="CHAT_MSG_WHISPER_INFORM" then record.sent_echo_uptime=clock();return end
    if record.received_uptime then return end
    record.received_uptime=clock();record.received_epoch=time()
    say(record.variant .. tostring(record.sequence) .. " received at uptime " .. string.format("%.6f",record.received_uptime))
    if record.variant=="B" then
        C_Timer.After(0.1,function()
            if not active or active.id~=record.run_id then return end
            record.toggle_before_uptime=clock();record.toggle_epoch=time()
            LoggingChat(false);LoggingChat(true)
            record.toggle_after_uptime=clock()
            record.logging_after_toggle=not not LoggingChat()
            say("B toggle completed at " .. string.format("%.6f",record.toggle_after_uptime))
        end)
    end
end)
SLASH_WISPCHATPROBE1="/wispchat"
SlashCmdList.WISPCHATPROBE=function(message)
    local run,target=message:match("^start ([0-9a-f]+) (.+)$")
    if run then
        if active then say("Stop the active test first.");return end
        if #run<8 or #run>32 or #target>100 or target:find("[|\r\n]") then say("Invalid test identity or target.");return end
        local version,build,_,interface=GetBuildInfo()
        active={id=run,target=target,character_guid=UnitGUID("player"),character_name=UnitName("player"),version=version,build=build,interface=interface,original_logging=not not LoggingChat(),started_epoch=time(),started_uptime=clock(),samples={}}
        table.insert(WispBridgeLabDB.chat_runs,active)
        while #WispBridgeLabDB.chat_runs>5 do table.remove(WispBridgeLabDB.chat_runs,1) end
        LoggingChat(true)
        say("Started " .. run .. ". Previously logging=" .. tostring(active.original_logging) .. "; now=" .. tostring(LoggingChat()))
        say("Sends only explicit test payloads to " .. target .. ". Chat logging may also write other chat locally.")
        C_Timer.After(600,function()if active and active.id==run then stop("timeout") end end)
        return
    end
    if message=="stop" then stop();return end
    if message=="status" then say(active and ("Active " .. active.id .. ", logging=" .. tostring(LoggingChat())) or "Idle, logging=" .. tostring(LoggingChat()));return end
    local variant=message:match("^([ab])$")
    if not variant or not active then say("/wispchat start <run-id> <your full name> | a | b | status | stop");return end
    if UnitGUID("player")~=active.character_guid then stop("character changed");return end
    local sequence=#active.samples+1
    if sequence>12 then say("Test sample limit reached; stop first.");return end
    local record={run_id=active.id,variant=variant:upper(),sequence=sequence,sent_uptime=clock(),sent_epoch=time()}
    local payload=string.format("WISPCHAT1:%s:%s:%d:%.6f:%d",active.id,record.variant,sequence,record.sent_uptime,record.sent_epoch)
    pending[payload]=record;table.insert(active.samples,record)
    local send=C_ChatInfo and C_ChatInfo.SendChatMessage or SendChatMessage
    local ok,err=pcall(send,payload,"WHISPER",nil,active.target)
    record.send_call_returned=ok
    if not ok then record.error=tostring(err);say("Send failed: " .. tostring(err));return end
    say(record.variant .. sequence .. " send requested at uptime " .. string.format("%.6f",record.sent_uptime) .. ". Waiting for received event.")
end
say("Loaded, idle. /wispchat status reads the current logging state; start is explicit.")
