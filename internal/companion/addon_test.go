package companion

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	lua "github.com/yuin/gopher-lua"
)

func TestAddonExportRunsInLua51(t *testing.T) {
	vm := lua.NewState()
	defer vm.Close()
	mock := `
 local methods = {}
 local meta = {__index=function(t,k) if k=='TitleText' then local child=setmetatable({},getmetatable(t));rawset(t,k,child);return child end; return function(self,...) if k=='SetText' then LAST_TEXT=(...) elseif k=='CreateFontString' then return setmetatable({},getmetatable(self)) end end end}
 function CreateFrame() return setmetatable({},meta) end
 UIParent={}; UISpecialFrames={}; SlashCmdList={}
 function UnitGUID() return 'Player-1-test' end
 function UnitName() return 'TestPlayer' end
 function GetRealmName() return 'Testrealm' end
 function GetBuildInfo() return '1.60.1','70790','Nov 2026',16001 end
 function UnitLevel() return 24 end
 function GetZoneText() return 'Ashenvale' end
 function GetSubZoneText() return 'Astranaar' end
 function time() return 1793872800 end
 function print() end
 `
	if err := vm.DoString(mock); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join("..", "..", "apps", "addon", "Jornia", "Wisp.lua")
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if err = vm.DoString(string(b)); err != nil {
		t.Fatal(err)
	}
	if err = vm.DoString(`SlashCmdList.WISP('mark'); SlashCmdList.WISP('note I want to talk to "Raene" — no spoilers.'); SlashCmdList.WISP('export')`); err != nil {
		t.Fatal(err)
	}
	raw := vm.GetGlobal("LAST_TEXT").String()
	if output := os.Getenv("WISP_TEST_EXPORT_PATH"); output != "" {
		if err = os.WriteFile(output, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
	}
	packet, err := ParseExport(raw)
	if err != nil {
		t.Fatalf("export not importable: %v\n%s", err, raw)
	}
	if len(packet.Events) != 2 || packet.Events[0].Kind != "snapshot" || !strings.Contains(packet.Events[1].Note, `"Raene"`) {
		t.Fatalf("unexpected packet %+v", packet)
	}
}
