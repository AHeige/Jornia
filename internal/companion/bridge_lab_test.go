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
