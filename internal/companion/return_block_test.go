package companion

import (
	"encoding/binary"
	"fmt"
	lua "github.com/yuin/gopher-lua"
	"hash/adler32"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestReturnBlockReceiverProtocolAndFaults(t *testing.T) {
	vm := lua.NewState()
	defer vm.Close()
	mock := `
 -- WoW rejects out-of-range signed integers for %d; host Lua may allow them.
 local format=string.format
 string.format=function(pattern,...)
  if pattern:find('%010d',1,true) then error('Use decimal float formatting for uint32 digests') end
  return format(pattern,...)
 end
 UIParent={};UISpecialFrames={};SlashCmdList={};LABELS={};CLOCK=true;GUID='Player-test';SOUND=true;BYTES=string.rep(string.char(255),8)
 local methods={}
 local meta={__index=function(t,k)return methods[k] or function()end end}
 function CreateFrame(kind,name) local f=setmetatable({scripts={}},meta);if name=='WispReturnBlockPanelV2' then PANEL=f elseif kind=='ScrollingMessageFrame' then RESULT=f end;return f end
 function methods:SetScript(k,fn)self.scripts[k]=fn end
 function methods:CreateFontString()local s=setmetatable({},meta);LABELS[#LABELS+1]=s;return s end
 function methods:CreateTexture()return setmetatable({},meta)end
 function methods:SetText(s)self.text=s end
 function methods:AddMessage(s)self.text=s end;function methods:Clear()self.text='' end
 function UnitGUID()return GUID end;function GetFramerate()return 60 end
 local profiler=0;function debugprofilestop()profiler=profiler+.01;return profiler end
 function StopSound()end
 function PlaySoundFile(path)
  if path:find('wispreturn_present',1,true) then return SOUND and true or nil end
  if path:find('AbsentControl',1,true) then return nil end
  if path:find('wispreturn_pulse',1,true) then return CLOCK and true or nil end
  local n=tonumber(path:match('WispOCRInbox(%d+)'))
  if n then local byte=BYTES:byte(math.floor((n-1)/8)+1);return math.floor(byte/2^((n-1)%8))%2==1 and true or nil end
 end
 function SEND(s) BYTES=s;CLOCK=not CLOCK;PANEL.scripts.OnUpdate(PANEL,.06);PANEL.scripts.OnUpdate(PANEL,.06) end
 `
	if err := vm.DoString(mock); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("../../apps/bridge-lab/return-block-bus.lua")
	if err != nil {
		t.Fatal(err)
	}
	if err = vm.DoString(string(data)); err != nil {
		t.Fatal(err)
	}
	ack := func() string {
		return vm.GetGlobal("LABELS").(*lua.LTable).RawGetInt(3).(*lua.LTable).RawGetString("text").String()
	}
	send := func(payload, fault string) {
		t.Helper()
		if err := vm.DoString("BYTES=string.rep(string.char(255),8);CLOCK=true;SlashCmdList.WISPRETURNBLOCK('start')"); err != nil {
			t.Fatal(err)
		}
		fields := strings.Fields(ack())
		if fields[0] != "WBREADY" {
			t.Fatal(ack())
		}
		nonce, _ := strconv.ParseUint(fields[1], 10, 32)
		binding, _ := strconv.ParseUint(fields[2], 10, 16)
		body := make([]byte, 11)
		copy(body, "WB")
		body[2] = 1
		if fault == "stale" {
			nonce ^= 1
		}
		binary.BigEndian.PutUint32(body[3:7], uint32(nonce))
		binary.BigEndian.PutUint16(body[7:9], uint16(binding))
		binary.BigEndian.PutUint16(body[9:11], uint16(len(payload)))
		body = append(body, []byte(payload)...)
		tail := make([]byte, 4)
		binary.BigEndian.PutUint32(tail, adler32.Checksum(body))
		if fault == "crc" {
			tail[3] ^= 1
		}
		wire := append(body, tail...)
		for len(wire)%8 != 0 {
			wire = append(wire, 0)
		}
		if fault == "padding" {
			wire[len(wire)-1] = 1
		}
		for at := 0; at < len(wire); at += 8 {
			vals := []string{}
			for _, b := range wire[at : at+8] {
				vals = append(vals, strconv.Itoa(int(b)))
			}
			if err := vm.DoString("SEND(string.char(" + strings.Join(vals, ",") + "))"); err != nil {
				t.Fatal(err)
			}
			if strings.HasPrefix(ack(), "WBFAIL") {
				break
			}
		}
		want := "WBDONE"
		if fault != "" {
			want = "WBFAIL"
		}
		if !strings.HasPrefix(ack(), want) {
			t.Fatalf("fault=%s receipt=%s", fault, ack())
		}
		shown := vm.GetGlobal("RESULT").(*lua.LTable).RawGetString("text").String()
		if fault == "" && shown != "Received: "+payload {
			t.Fatalf("wrong shown payload length=%d", len(shown))
		}
		if fault != "" && shown != "" {
			t.Fatal("displayed rejected payload")
		}
	}
	send("Hi", "")
	send("Again", "")
	send("åäö", "")
	send(strings.Repeat("z", 1024), "")
	send("BAD", "crc")
	send("BAD", "padding")
	send("BAD", "stale")
	if err = vm.DoString(`BYTES=string.rep(string.char(255),8);CLOCK=true;SlashCmdList.WISPRETURNBLOCK('start');SOUND=false;SEND(string.rep(string.char(0),8));assert(LABELS[3].text:find('WBREADY',1,true));SOUND=true;GUID='other';PANEL.scripts.OnUpdate(PANEL,.06);assert(LABELS[3].text:find('WBFAIL',1,true))`); err != nil {
		t.Fatal(fmt.Errorf("channel/identity guard: %w", err))
	}
}
