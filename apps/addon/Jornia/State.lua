-- Owned live character state. Only public addon APIs; no chat or other players.
-- Lua memory is live; current_state/events reach disk only when WoW saves.
WispClientState={}
local S=WispClientState
local previous,guid,pending,revision=nil,nil,false,0
local function call(fn,...)if type(fn)~='function'then return nil end;local ok,a,b,c,d,e,f,g,h=pcall(fn,...);if ok then return a,b,c,d,e,f,g,h end end
local function visible(v)if issecretvalue and issecretvalue(v)then return nil end;return v end
local function num(v,max)v=visible(v);if type(v)=='number' and v==v and v>=0 and v<=(max or 2147483647)then return math.floor(v)end end
local function txt(v,n)v=visible(v);if type(v)~='string'then return '' end;n=n or 160;if #v<=n then return v end;while n>0 and v:byte(n+1)>=128 and v:byte(n+1)<=191 do n=n-1 end;return v:sub(1,n)end
local function character()return WispJournalRuntime and WispJournalRuntime.Current()end
local function canRecord(c)return c and c.recording and WispDB and WispDB.consent=='accepted'end
local function same(a,b)
 if type(a)~=type(b)then return false end;if type(a)~='table'then return a==b end
 for k,v in pairs(a)do if not same(v,b[k])then return false end end;for k in pairs(b)do if a[k]==nil then return false end end;return true
end
-- Read only the player's selected quest. Never change selection to scrape other quests.
local function questText(id)
 local selected=call(C_QuestLog and C_QuestLog.GetSelectedQuest)
 if not selected then
  local index=call(GetQuestLogSelection);local info=index and call(C_QuestLog and C_QuestLog.GetInfo,index)
  selected=type(info)=='table' and info.questID or nil
 end
 local description,objective='',''
 if selected==id then description,objective=call(GetQuestLogQuestText) end
 description=visible(description);objective=visible(objective)
 local waypoint=visible(call(C_QuestLog and C_QuestLog.GetNextWaypointText,id))
 return txt(description,1024),txt(objective,512),txt(waypoint,240),(type(description)=='string' and #description>1024)or(type(objective)=='string' and #objective>512)or(type(waypoint)=='string' and #waypoint>240)
end
local function quests()
 local rows={};local truncated=false
 local count=num(call(C_QuestLog and C_QuestLog.GetNumQuestLogEntries),1000)
 if not count then return rows,true,false end
 for i=1,count do local q=call(C_QuestLog and C_QuestLog.GetInfo,i)
  if type(q)~='table'then truncated=true end
  if type(q)=='table' and not q.isHeader and num(q.questID) and q.questID>0 then
   if #rows>=50 then truncated=true;break end
   local objectives={};local raw=call(C_QuestLog and C_QuestLog.GetQuestObjectives,q.questID)
   if type(raw)=='table'then for index,o in ipairs(raw)do if index>8 then truncated=true;break end
    if type(o)=='table'then objectives[#objectives+1]={text=txt(o.text,160),kind=txt(o.type,30),fulfilled=num(o.numFulfilled),required=num(o.numRequired),finished=visible(o.finished)==true}end
   end end
   local description,objectiveText,waypointText,textTruncated=questText(q.questID)
   rows[#rows+1]={description=description,objective_text=objectiveText,waypoint_text=waypointText,text_truncated=textTruncated,id=q.questID,title=txt(q.title,200),complete=call(C_QuestLog and C_QuestLog.IsComplete,q.questID)==true,waypoint=WispNavigation and WispNavigation.HasWaypoint(q.questID)or false,objectives=objectives}
  end
 end
 table.sort(rows,function(a,b)return a.id<b.id end);return rows,truncated,not truncated
end
local function make()
 local id=call(UnitGUID,'player');if type(id)~='string' or id==''then return end
 local _,build,_,interface=call(GetBuildInfo);local _,_,class=call(UnitClass,'player');local _,_,race=call(UnitRace,'player')
 local state={schema=1,observed_at=time(),revision=revision+1,level=num(call(UnitLevel,'player'),100)or 0,zone=txt(call(GetZoneText)),subzone=txt(call(GetSubZoneText)),client_build=txt(build,30),interface=num(interface),class_id=num(class),race_id=num(race),xp=num(call(UnitXP,'player')),xp_max=num(call(UnitXPMax,'player')),dead=call(UnitIsDeadOrGhost,'player')==true,equipment={},professions={}}
 state.quests,state.truncated,state.quests_available=quests()
 state.equipment_available=type(GetInventoryItemID)=='function'
 state.professions_available=type(GetProfessions)=='function' and type(GetProfessionInfo)=='function'
 for slot=1,19 do local item=num(call(GetInventoryItemID,'player',slot));if item and item>0 then state.equipment[#state.equipment+1]={slot=slot,item_id=item}end end
 if type(GetProfessions)=='function'then local indices={call(GetProfessions)};for _,index in ipairs(indices)do if num(index)then local name,_,rank,maxRank,_,_,skill=call(GetProfessionInfo,index);state.professions[#state.professions+1]={name=txt(name,100),skill_id=num(skill),rank=num(rank),max_rank=num(maxRank)}end end end
 return state,id
end
local function record(kind,q,item,slot)
 if WispJournalRuntime then WispJournalRuntime.Record(kind,q and q.id,nil,{quest_title=q and q.title,objectives=q and q.objectives,item_id=item,slot=slot})end
end
function S.Refresh(reason,slot)
 local state,id=make();if not state then return end
 local c=character();local recording=canRecord(c)
 if id~=guid then previous=nil;guid=id;revision=c and c.current_state and num(c.current_state.revision)or 0;state.revision=revision+1 end
 state.session_id=c and c.session_id or '';state.dropped_events=c and c.dropped or 0
 if recording and previous then
  if state.quests_available and previous.quests_available then
  local old,new={},{};for _,q in ipairs(previous.quests)do old[q.id]=q end;for _,q in ipairs(state.quests)do new[q.id]=q;local before=old[q.id]
   if not before then record('quest_accepted',q)
   elseif not same(before.objectives,q.objectives) or before.complete~=q.complete then record('quest_progress',q)end
  end
  for _,q in ipairs(previous.quests)do if not new[q.id] and reason~='QUEST_TURNED_IN'then record('quest_removed',q)end end
  end
  if reason=='PLAYER_EQUIPMENT_CHANGED' and not same(previous.equipment,state.equipment)then local item=call(GetInventoryItemID,'player',slot);record('equipment_changed',nil,num(item),num(slot,19))end
  if state.dead~=previous.dead then record(state.dead and 'death' or 'resurrected')end
 end
 revision=state.revision;previous=state;S.Current=state;S.CharacterGUID=id
 if recording then c.current_state=state end
 return state
end
function S.Snapshot()
 -- Re-read at send time; do not send a cached SavedVariables/Armory state as live.
 return S.Refresh('question')
end
function S.Quest(id)for _,q in ipairs(previous and previous.quests or {})do if q.id==id then return q end end end
function S.RecordingChanged()previous=nil;S.Refresh('recording_changed')end
local f=CreateFrame('Frame');S.Frame=f
for _,event in ipairs({'PLAYER_LOGIN','PLAYER_LOGOUT','PLAYER_LEVEL_UP','PLAYER_XP_UPDATE','ZONE_CHANGED','ZONE_CHANGED_NEW_AREA','QUEST_ACCEPTED','QUEST_REMOVED','QUEST_TURNED_IN','QUEST_LOG_UPDATE','PLAYER_EQUIPMENT_CHANGED','SKILL_LINES_CHANGED','PLAYER_DEAD','PLAYER_ALIVE','PLAYER_UNGHOST'})do f:RegisterEvent(event)end
f:SetScript('OnEvent',function(_,event,arg)
 if event=='PLAYER_LOGOUT' or event=='QUEST_TURNED_IN' or event=='PLAYER_EQUIPMENT_CHANGED'then S.Refresh(event,arg);return end
 if pending then return end;pending=true
 local function update()pending=false;S.Refresh(event,arg)end
 if C_Timer and C_Timer.After then C_Timer.After(.25,update)else update()end
end)
