package companion

import (
	lua "github.com/yuin/gopher-lua"
	"os"
	"testing"
)

func TestNativeLogProbeSingleMarkerAndRestoration(t *testing.T) {
	for _, scenario := range []string{"success", "write_error", "setter_error_after_mutation", "setter_refused", "missing_function", "missing_cvar"} {
		t.Run(scenario, func(t *testing.T) {
			vm := lua.NewState()
			defer vm.Close()
			mock := `
 SCENARIO='` + scenario + `';SlashCmdList={};CVARS={developerLog='0',developerLogWriteToFile='0',developerLogFilterNormal='1'};SETCALLS=0;LOGCALLS=0;TIMERS={}
 function print()end;function time()return 1793960000 end;function GetTime()return 200.125 end
 function GetBuildInfo()return '1.60.1','70790','date',16001 end
 function CreateFrame()return {RegisterEvent=function()end,SetScript=function(_,_,fn)EXIT_HANDLER=fn end}end
 C_Timer={After=function(delay,fn)TIMERS[#TIMERS+1]={delay=delay,fn=fn}end}
 function GetCVar(k)if SCENARIO=='missing_cvar' and k=='developerLogWriteToFile' then return nil end;return CVARS[k]end
 function SetCVar(k,v)
  SETCALLS=SETCALLS+1
  if SCENARIO=='setter_refused' and k=='developerLog' and v=='1' then return end
  CVARS[k]=v
  if SCENARIO=='setter_error_after_mutation' and k=='developerLogWriteToFile' and v=='1' then error('injected setter error')end
 end
 C_Log={LogMessage=function(s)LOGCALLS=LOGCALLS+1;MARKER=s;if SCENARIO=='write_error' then error('injected write error')end end}
 if SCENARIO=='missing_function' then C_Log=nil end
 `
			if err := vm.DoString(mock); err != nil {
				t.Fatal(err)
			}
			b, err := os.ReadFile("../../apps/bridge-lab/native-log-probe.lua")
			if err != nil {
				t.Fatal(err)
			}
			if err = vm.DoString(string(b)); err != nil {
				t.Fatal(err)
			}
			script := `
 assert(SETCALLS==0 and LOGCALLS==0)
 SlashCmdList.WISPNATIVELOG('status');assert(SETCALLS==0 and LOGCALLS==0)
 SlashCmdList.WISPNATIVELOG('test invalid!');assert(SETCALLS==0)
 SlashCmdList.WISPNATIVELOG('test abcd1234')
 local r=WispBridgeLabDB.native_runs[1];assert(r.id=='abcd1234')
 if SCENARIO=='success' then
  assert(LOGCALLS==1 and r.call_returned);assert(CVARS.developerLog=='1');assert(TIMERS[1].delay==60)
  SlashCmdList.WISPNATIVELOG('test abcd1234');assert(LOGCALLS==1)
  TIMERS[1].fn();assert(r.restore_complete)
 elseif SCENARIO=='write_error' then assert(LOGCALLS==1 and r.restore_complete)
 elseif SCENARIO=='setter_error_after_mutation' or SCENARIO=='setter_refused' then assert(LOGCALLS==0 and r.restore_complete)
 else assert(LOGCALLS==0 and SETCALLS==0) end
 assert(CVARS.developerLog=='0' and CVARS.developerLogWriteToFile=='0' and CVARS.developerLogFilterNormal=='1')
 local calls=LOGCALLS
 SlashCmdList.WISPNATIVELOG('test abcd1234');assert(LOGCALLS==calls)
 EXIT_HANDLER();assert(CVARS.developerLog=='0')
 `
			if err = vm.DoString(script); err != nil {
				t.Fatal(err)
			}
		})
	}
}
