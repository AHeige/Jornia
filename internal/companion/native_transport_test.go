package companion

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	lua "github.com/yuin/gopher-lua"
	"hash/adler32"
	"os"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestNativeTransportTwoRepliesAndIdentityRejection(t *testing.T) {
	vm := lua.NewState()
	defer vm.Close()
	mock := `
 UIParent={};C_Timer={After=function()end};GUID='Player-test';NOW=1;CLOCK=true;HEART=true;BYTES=string.rep(string.char(255),8);FRAMES={};REPLIES={}
 local methods={};local meta={__index=function(t,k)return methods[k] or function()end end}
 function CreateFrame()local f=setmetatable({scripts={}},meta);table.insert(FRAMES,f);return f end
 function methods:CreateFontString()return setmetatable({},meta)end;function methods:CreateTexture()return setmetatable({},meta)end
 function methods:SetText(s)self.text=s end;function methods:SetScript(k,f)self.scripts[k]=f end
 function GetTime()return NOW end;function time()return 1794000000 end;function UnitGUID()return GUID end;function UnitLevel()return 2 end;function GetZoneText()return 'Deathknell' end;function GetBuildInfo()return '1.60.1','70790' end
 function StopSound()end;function print()end
 WispJournalUI={Config={},Resolve=function(id,guid,text,isError,metadata)table.insert(REPLIES,{id=id,guid=guid,text=text,error=isError,metadata=metadata})end,SetProgress=function()end}
 function PlaySoundFile(path)
  if path:find('wispreturn_present',1,true)then return true end
  if path:find('AbsentControl',1,true)then return nil end
  if path:find('wispreturn_pulse',1,true)then return CLOCK and true or nil end
  local n=tonumber(path:match('WispOCRInbox(%d+)'))
  if n==65 then return true end;if n==66 then return HEART and true or nil end
  if n then return math.floor(BYTES:byte(math.floor((n-1)/8)+1)/2^((n-1)%8))%2==1 and true or nil end
 end
 C_QuestLog={GetSelectedQuest=function()return 123 end}
 WispClientState={Snapshot=function()return {observed_at=time(),level=2,zone='Deathknell',client_build='70790',quests_available=true,equipment_available=true,professions_available=true,quests={{id=999,title='Other',objectives={}},{id=123,title='Selected',description=string.rep('å',450)..' You will find wolves south.',objective_text='Collect the paws west of the chapel.',waypoint_text='West of the chapel',objectives={{text='Paws',fulfilled=4,required=6}}}},equipment={},professions={}}end}
 function PULSE()NOW=NOW+1;HEART=not HEART;FRAMES[2].scripts.OnUpdate(FRAMES[2],1)end
 function SEND(s)BYTES=s;CLOCK=not CLOCK;WispCompanionTransport.Frame.scripts.OnUpdate(nil,.06);WispCompanionTransport.Frame.scripts.OnUpdate(nil,.06)end
 `
	if err := vm.DoString(mock); err != nil {
		t.Fatal(err)
	}
	code, err := os.ReadFile("../../apps/addon/Jornia/Transport.lua")
	if err != nil {
		t.Fatal(err)
	}
	if err = vm.DoString(string(code)); err != nil {
		t.Fatal(err)
	}
	if err = vm.DoString(`assert(not WispCompanionTransport.Send('Hello',GUID,false));PULSE();PULSE();PULSE()`); err != nil {
		t.Fatal(err)
	}
	for turn := 0; turn < 3; turn++ {
		if err = vm.DoString(`CLOCK=true;BYTES=string.rep(string.char(255),8);PULSE();ID=assert(WispCompanionTransport.Send('Where next?',GUID,true));assert(not WispCompanionTransport.Send('Duplicate',GUID,false))`); err != nil {
			t.Fatal(err)
		}
		line := vm.GetGlobal("WispCompanionTransport").(*lua.LTable).RawGetString("Packet").(*lua.LTable).RawGetString("text").String()
		fields := strings.Fields(strings.Split(line, "\n")[0])
		lines := strings.Split(line, "\n")
		raw, err := hex.DecodeString(strings.ReplaceAll(strings.Join(lines[1:len(lines)-1], ""), " ", ""))
		if err != nil || len(raw) > 640 || !utf8.Valid(raw) {
			t.Fatal("invalid bounded UTF-8 snapshot", err)
		}
		var body map[string]any
		if err = json.Unmarshal(raw, &body); err != nil {
			t.Fatal(err)
		}
		quests := body["s"].([]any)[5].([]any)
		if len(quests) != 1 {
			t.Fatal("selected quest not prioritized")
		}
		quest := quests[0].([]any)
		detail := quest[5].([]any)
		if quest[0].(float64) != 123 || detail[2] != "West of the chapel" || detail[3].(float64) != 1 || !strings.HasSuffix(detail[0].(string), "wolves south.") {
			t.Fatal("directions or truncation lost", quest)
		}

		var nonce uint32
		var binding uint16
		fmt.Sscan(fields[3], &nonce)
		bv, _ := strconv.ParseUint(fields[2], 10, 16)
		binding = uint16(bv)
		id := vm.GetGlobal("ID").String()
		replyID := id
		if turn == 2 {
			replyID = strings.Repeat("0", 24)
		}
		text := []byte("Next: The Damned. Åäö 中文")
		payload := []byte{1, 0}
		payload = append(payload, []byte(replyID)...)
		payload = binary.BigEndian.AppendUint16(payload, uint16(len(text)))
		payload = append(payload, text...)
		payload = append(payload, 1)
		payload = binary.BigEndian.AppendUint16(payload, 2)
		payload = append(payload, make([]byte, 8)...)
		frame := []byte{'W', 'B', 1}
		frame = binary.BigEndian.AppendUint32(frame, nonce)
		frame = binary.BigEndian.AppendUint16(frame, binding)
		frame = binary.BigEndian.AppendUint16(frame, uint16(len(payload)))
		frame = append(frame, payload...)
		frame = binary.BigEndian.AppendUint32(frame, adler32.Checksum(frame))
		for len(frame)%8 != 0 {
			frame = append(frame, 0)
		}
		for i := 0; i < len(frame); i += 8 {
			vm.SetGlobal("BLOCK", lua.LString(frame[i:i+8]))
			if err = vm.DoString(`SEND(BLOCK)`); err != nil {
				t.Fatal(err)
			}
		}
		if err = vm.DoString(`assert(not WispCompanionTransport.IsPending());assert(#REPLIES==` + fmt.Sprint(turn+1) + `)`); err != nil {
			t.Fatal(err)
		}
		r := vm.GetGlobal("REPLIES").(*lua.LTable).RawGetInt(turn + 1).(*lua.LTable)
		if turn < 2 {
			if r.RawGetString("text").String() != string(text) || r.RawGetString("error") != lua.LFalse {
				t.Fatal("turn", turn, "text", r.RawGetString("text"), "error", r.RawGetString("error"))
			}
		} else if r.RawGetString("error") != lua.LTrue {
			t.Fatal("wrong operation was accepted")
		}
	}
}
