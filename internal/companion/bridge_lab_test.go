package companion

import (
	lua "github.com/yuin/gopher-lua"
	"os"
	"testing"
)

func TestBridgeLabHasExplicitLocalActionsAndDataOnlyInbox(t *testing.T) {
	vm := lua.NewState()
	defer vm.Close()
	mock := `
 SlashCmdList={};UIParent={};UISpecialFrames={};calls=0;LOADED=false
 local frame={TitleText={SetText=function()end}}
 function frame:RegisterEvent()end
 function frame:SetScript(_,fn) HANDLER=fn end
 function frame:SetSize()end;function frame:SetPoint()end;function frame:Show()end
 function frame:CreateFontString()return {SetPoint=function()end,SetWidth=function()end,SetJustifyH=function()end,SetText=function(_,s)LABEL=s end}end
 function CreateFrame()return frame end
 function print()end;function time()return 1793950000 end;function GetTime()return 100 end
 function GetBuildInfo()return '1.60.1','70790','date',16001 end
 C_AddOns={IsAddOnLoaded=function()return LOADED end,LoadAddOn=function()calls=calls+1;LOADED=true;return true end}
 `
	if err := vm.DoString(mock); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile("../../apps/bridge-lab/WispBridgeLab/WispBridgeLab.lua")
	if err != nil {
		t.Fatal(err)
	}
	if err = vm.DoString(string(b)); err != nil {
		t.Fatal(err)
	}
	script := `
 HANDLER();assert(calls==0)
 SlashCmdList.WISPBRIDGELAB('probe');assert(calls==0);assert(WispBridgeLabDB.probe.capabilities.load_addon=='function')
 WispBridgeLabReceive('zz','61');assert(WispBridgeLabDB.received==nil)
 WispBridgeLabReceive('74657374','48656c6c6f');assert(WispBridgeLabDB.received[1].text=='Hello')
 SlashCmdList.WISPBRIDGELAB('inbox 1');assert(calls==1)
 SlashCmdList.WISPBRIDGELAB('inbox 1');assert(calls==1)
 SlashCmdList.WISPBRIDGELAB('inbox 99');assert(calls==1)
 `
	if err = vm.DoString(script); err != nil {
		t.Fatal(err)
	}
}

func TestChatProbeIsOptInMeasuresReceiveAndRestoresLogging(t *testing.T) {
	for _, original := range []string{"false", "true"} {
		t.Run(original, func(t *testing.T) {
			vm := lua.NewState()
			defer vm.Close()
			mock := `
 SlashCmdList={};LOGGING=` + original + `;LOGCALLS=0;SENDS=0;NOW=123.456;TIMERS={}
 function print()end;function time()return 1793960000 end;function GetTime()return NOW end
 function UnitGUID()return 'Player-test' end;function UnitName()return 'Cone' end
 function GetBuildInfo()return '1.60.1','70790','date',16001 end
 function LoggingChat(value)if value~=nil then LOGGING=value;LOGCALLS=LOGCALLS+1 end;return LOGGING end
 function CreateFrame()return {RegisterEvent=function()end,SetScript=function(_,_,fn)CHAT_HANDLER=fn end}end
 C_Timer={After=function(delay,fn)TIMERS[#TIMERS+1]={delay=delay,fn=fn} end}
 C_ChatInfo={SendChatMessage=function(payload,kind,lang,target)SENDS=SENDS+1;PAYLOAD=payload;assert(kind=='WHISPER');assert(target=='Cone Thrower')end}
 `
			if err := vm.DoString(mock); err != nil {
				t.Fatal(err)
			}
			b, err := os.ReadFile("../../apps/bridge-lab/chat-probe.lua")
			if err != nil {
				t.Fatal(err)
			}
			if err = vm.DoString(string(b)); err != nil {
				t.Fatal(err)
			}
			script := `
 assert(SENDS==0 and LOGCALLS==0)
 SlashCmdList.WISPCHATPROBE('status');assert(LOGCALLS==0)
 SlashCmdList.WISPCHATPROBE('a');assert(SENDS==0)
 SlashCmdList.WISPCHATPROBE('start abcd1234 Cone Thrower');assert(LOGGING)
 SlashCmdList.WISPCHATPROBE('a');assert(SENDS==1)
 CHAT_HANDLER(nil,'CHAT_MSG_WHISPER','someone else');assert(#WispBridgeLabDB.chat_runs[1].samples==1)
 CHAT_HANDLER(nil,'CHAT_MSG_WHISPER',PAYLOAD);assert(#TIMERS==1)
 SlashCmdList.WISPCHATPROBE('b');assert(SENDS==2)
 CHAT_HANDLER(nil,'CHAT_MSG_WHISPER_INFORM',PAYLOAD);assert(#TIMERS==1)
 CHAT_HANDLER(nil,'CHAT_MSG_WHISPER',PAYLOAD);assert(#TIMERS==2);assert(TIMERS[2].delay==0.1)
 TIMERS[2].fn();assert(LOGGING)
 CHAT_HANDLER(nil,'CHAT_MSG_WHISPER',PAYLOAD);assert(#TIMERS==2)
 SlashCmdList.WISPCHATPROBE('stop');assert(LOGGING==` + original + `)
 TIMERS[2].fn();assert(LOGGING==` + original + `)
 SlashCmdList.WISPCHATPROBE('a');assert(SENDS==2)
 `
			if err = vm.DoString(script); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestChatGapControlSpansFramesAndHonoursStop(t *testing.T) {
	vm := lua.NewState()
	defer vm.Close()
	mock := `
 SlashCmdList={};LOGGING=true;NOW=100;TIMERS={};SENDS=0
 RUN={id='abcd1234',target='Cone Thrower',character_guid='Player-test',samples={}}
 WispBridgeLabDB={chat_runs={RUN}}
 function print()end;function time()return 1793960000 end;function GetTime()return NOW end
 function UnitGUID()return 'Player-test' end
 function LoggingChat(v)if v~=nil then LOGGING=v end;return LOGGING end
 function CreateFrame()return {RegisterEvent=function()end,SetScript=function(_,_,fn)GAP_HANDLER=fn end}end
 C_Timer={After=function(delay,fn)TIMERS[#TIMERS+1]={delay=delay,fn=fn}end}
 C_ChatInfo={SendChatMessage=function(payload,kind,lang,target)SENDS=SENDS+1;PAYLOAD=payload;assert(kind=='WHISPER');assert(target=='Cone Thrower')end}
 `
	if err := vm.DoString(mock); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile("../../apps/bridge-lab/chat-gap-probe.lua")
	if err != nil {
		t.Fatal(err)
	}
	if err = vm.DoString(string(b)); err != nil {
		t.Fatal(err)
	}
	script := `
 assert(SENDS==0);SlashCmdList.WISPCHATGAP();assert(SENDS==1)
 GAP_HANDLER(nil,'CHAT_MSG_WHISPER_INFORM',PAYLOAD);assert(#TIMERS==0)
 GAP_HANDLER(nil,'CHAT_MSG_WHISPER',PAYLOAD);assert(#TIMERS==1);assert(TIMERS[1].delay==0.1)
 TIMERS[1].fn();assert(not LOGGING);assert(#TIMERS==2);assert(TIMERS[2].delay==0.25)
 TIMERS[2].fn();assert(LOGGING);assert(RUN.samples[1].toggle_mode=='250ms_gap')
 RUN.stopped_uptime=101;LOGGING=false;TIMERS[2].fn();assert(not LOGGING)
 SlashCmdList.WISPCHATGAP();assert(SENDS==1)
 `
	if err = vm.DoString(script); err != nil {
		t.Fatal(err)
	}
}
