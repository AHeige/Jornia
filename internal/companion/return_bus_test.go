package companion

import (
	"encoding/binary"
	"fmt"
	lua "github.com/yuin/gopher-lua"
	"hash/adler32"
	"os"
	"strings"
	"testing"
)

func TestReturnPresenceBusRunsInLuaAndRejectsDamage(t *testing.T) {
	vm := lua.NewState()
	defer vm.Close()
	mock := `
 UIParent={};UISpecialFrames={};SlashCmdList={};LABELS={};BYTE=255;CLOCK=true;GUID='Player-test';SOUND=true
 local methods={}
 local meta={__index=function(t,k)return methods[k] or function()end end}
 function CreateFrame() local f=setmetatable({scripts={}},meta);PANEL=f;return f end
 function methods:SetScript(k,fn)self.scripts[k]=fn end
 function methods:CreateFontString()local s=setmetatable({},meta);LABELS[#LABELS+1]=s;return s end
 function methods:CreateTexture()return setmetatable({},meta)end
 function methods:SetText(s)self.text=s end
 function UnitGUID()return GUID end
 function debugprofilestop()return 1 end
 function StopSound()end
 function PlaySoundFile(path)
  if path:find('wispreturn_present',1,true) then return SOUND and true or nil end
  if path:find('AbsentControl',1,true) then return nil end
  if path:find('wispreturn_pulse',1,true) then return CLOCK and true or nil end
  local n=tonumber(path:match('WispOCRInbox(%d+)'))
  if n then return math.floor(BYTE/2^(n-81))%2==1 and true or nil end
 end
 function SEND(value) BYTE=value;CLOCK=not CLOCK;PANEL.scripts.OnUpdate(PANEL,.06);PANEL.scripts.OnUpdate(PANEL,.06) end
 `
	if err := vm.DoString(mock); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("../../apps/bridge-lab/return-byte-bus.lua")
	if err != nil {
		t.Fatal(err)
	}
	if err = vm.DoString(string(data)); err != nil {
		t.Fatal(err)
	}
	send := func(payload string, corrupt bool) {
		t.Helper()
		body := append([]byte{'W', 'P', byte(len(payload))}, []byte(payload)...)
		tail := make([]byte, 4)
		binary.BigEndian.PutUint32(tail, adler32.Checksum(body))
		if corrupt {
			tail[3] ^= 1
		}
		if err := vm.DoString("BYTE=255;CLOCK=true;SlashCmdList.WISPRETURNBUS('start')"); err != nil {
			t.Fatal(err)
		}
		for _, b := range append(body, tail...) {
			if err := vm.DoString(fmt.Sprintf("SEND(%d)", b)); err != nil {
				t.Fatal(err)
			}
		}
		label := vm.GetGlobal("LABELS").(*lua.LTable).RawGetInt(3).(*lua.LTable).RawGetString("text").String()
		want := "WISPDONE"
		if corrupt {
			want = "WISPFAIL"
		}
		if !strings.HasPrefix(label, want) {
			t.Fatalf("receipt=%s want=%s", label, want)
		}
	}
	send("Hi", false)
	send("Again", false)
	send("åäö", false)
	send("bad", true)
	// Lost sound must never be decoded as zero data. Character switch rejects.
	if err = vm.DoString(`BYTE=255;CLOCK=true;SlashCmdList.WISPRETURNBUS('start');SOUND=false;SEND(87);assert(LABELS[3].text:find('WISPREADY',1,true));SOUND=true;GUID='other';PANEL.scripts.OnUpdate(PANEL,.06);assert(LABELS[3].text:find('WISPFAIL',1,true))`); err != nil {
		t.Fatal(err)
	}
}
