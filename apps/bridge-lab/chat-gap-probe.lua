-- First-party supplementary control: keep logging off across frame boundaries.
-- Explicit sends only, uses the existing run and original logging restoration.
local function say(s)print("|cffffc466Wisp Gap Test:|r "..s)end
local clock=GetTimePreciseSec or GetTime
local pending={}
local frame=CreateFrame("Frame")
frame:RegisterEvent("CHAT_MSG_WHISPER")
frame:RegisterEvent("CHAT_MSG_WHISPER_INFORM")
frame:SetScript("OnEvent",function(_,event,message)
 local entry=pending[message]
 if not entry or entry.run.stopped_uptime then return end
 local record,run=entry.record,entry.run
 if event=="CHAT_MSG_WHISPER_INFORM" then record.sent_echo_uptime=clock();return end
 if record.received_uptime then return end
 record.received_uptime=clock();record.received_epoch=time()
 say("B"..record.sequence.." gap received at "..string.format("%.6f",record.received_uptime))
 C_Timer.After(0.1,function()
  if run.stopped_uptime then return end
  record.toggle_before_uptime=clock();record.toggle_epoch=time()
  LoggingChat(false)
  record.logging_while_off=not not LoggingChat()
  say("Logging off for 250 ms.")
  C_Timer.After(0.25,function()
   if run.stopped_uptime then return end
   LoggingChat(true)
   record.toggle_after_uptime=clock();record.logging_after_toggle=not not LoggingChat()
   say("Gap toggle completed at "..string.format("%.6f",record.toggle_after_uptime))
  end)
 end)
end)
SLASH_WISPCHATGAP1="/wispchatgap"
SlashCmdList.WISPCHATGAP=function()
 local runs=WispBridgeLabDB and WispBridgeLabDB.chat_runs
 local run=runs and runs[#runs]
 if not run or run.stopped_uptime or UnitGUID("player")~=run.character_guid then say("No matching active chat experiment.");return end
 if #run.samples>=12 then say("Sample limit reached.");return end
 local record={run_id=run.id,variant="B",toggle_mode="250ms_gap",sequence=#run.samples+1,sent_uptime=clock(),sent_epoch=time()}
 local payload=string.format("WISPCHAT1:%s:B:%d:%.6f:%d",run.id,record.sequence,record.sent_uptime,record.sent_epoch)
 table.insert(run.samples,record);pending[payload]={record=record,run=run}
 local send=C_ChatInfo and C_ChatInfo.SendChatMessage or SendChatMessage
 local ok,err=pcall(send,payload,"WHISPER",nil,run.target)
 record.send_call_returned=ok
 if not ok then record.error=tostring(err);say("Send failed: "..tostring(err));return end
 say("B"..record.sequence.." gap send requested.")
end
say("Loaded, idle. /wispchatgap sends one explicit supplementary test in the active run.")
