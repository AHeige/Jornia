-- First-party navigation adapter. No model coordinates, movement, quest acceptance or code execution.
WispNavigation = {}
local function finite(n) return type(n)=='number' and n==n and n~=math.huge and n~=-math.huge end
local function waypoint(id)
 if not C_QuestLog or not C_QuestLog.GetNextWaypoint then return false end
 local ok,map,x,y=pcall(C_QuestLog.GetNextWaypoint,id)
 if ok and finite(map) and map>0 and map%1==0 and finite(x) and x>=0 and x<=1 and finite(y) and y>=0 and y<=1 then return map,x,y end
end
function WispNavigation.HasWaypoint(id) return waypoint(id)~=nil end
function WispNavigation.ShowQuest(id,expires,guid)
 if guid~=UnitGUID('player') or type(expires)~='number' or expires<=time() then return false,'This suggestion expired. Ask again.' end
 if type(id)~='number' or id<1 or id%1~=0 or not C_QuestLog or not C_QuestLog.IsOnQuest or not C_QuestLog.IsOnQuest(id) then return false,'This quest is no longer active on this character.' end
 if InCombatLockdown and InCombatLockdown() then return false,'Try showing the waypoint after combat.' end
 local map,x,y=waypoint(id)
 if not map then return false,'The game has no waypoint for this quest right now.' end
 if not C_Map or not C_Map.CanSetUserWaypointOnMap or not C_Map.SetUserWaypoint or not C_Map.GetUserWaypoint or not UiMapPoint or not UiMapPoint.CreateFromCoordinates then return false,'Native waypoint support is unavailable in this client.' end
 local allowed,can=pcall(C_Map.CanSetUserWaypointOnMap,map)
 if not allowed or not can then return false,'This map does not support native waypoints.' end
 local made,point=pcall(UiMapPoint.CreateFromCoordinates,map,x,y)
 if not made then return false,'The game could not create this waypoint.' end
 local placed=pcall(C_Map.SetUserWaypoint,point)
 local read,actual=pcall(C_Map.GetUserWaypoint)
 if not placed or not read or not actual or actual.uiMapID~=map or not actual.position then return false,'The waypoint was not confirmed by the game.' end
 local ax,ay=actual.position:GetXY()
 if not finite(ax) or not finite(ay) or math.abs(ax-x)>.0001 or math.abs(ay-y)>.0001 then return false,'The waypoint was not confirmed by the game.' end
 if C_SuperTrack and C_SuperTrack.SetSuperTrackedUserWaypoint then pcall(C_SuperTrack.SetSuperTrackedUserWaypoint,true) end
 return true,'Waypoint set from the current quest marker.'
end
