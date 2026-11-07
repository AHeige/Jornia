package companion

import (
	"context"
	"encoding/json"
	lua "github.com/yuin/gopher-lua"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"wisp/internal/wisp"
)

func TestLiveStateQuestProgressConsentAndFreshQuestionCapture(t *testing.T) {
	vm := lua.NewState()
	defer vm.Close()
	mock := `
 FRAMES={};SlashCmdList={};NOW=1794000000;LEVEL=15;PROGRESS=0;HASQUEST=true;ITEM=100
 local methods={};local meta={__index=function(t,k)return methods[k]or function()end end}
 function CreateFrame()local f=setmetatable({scripts={}},meta);table.insert(FRAMES,f);return f end
 function methods:SetScript(k,v)self.scripts[k]=v end
 function print()end;function time()return NOW end;function UnitGUID()return 'Player-test' end;function UnitName()return 'Test','Hero' end;function GetRealmName()return 'Testrealm' end
 function GetBuildInfo()return '1.60.1','70790','date',16001 end;function UnitLevel()return LEVEL end
 function GetZoneText()return 'Tirisfal Glades' end;function GetSubZoneText()return 'Deathknell' end
 function GetInventoryItemID(_,slot)if slot==16 then return ITEM end end
 C_QuestLog={GetNumQuestLogEntries=function()return HASQUEST and 1 or 0 end,GetInfo=function()return {questID=376,title='The Damned'}end,IsComplete=function()return false end,GetQuestObjectives=function()return {{text='Scavenger Paw',type='item',numFulfilled=PROGRESS,numRequired=6,finished=false}}end}
 function FIRE(event,arg)for _,f in ipairs(FRAMES)do if f.scripts.OnEvent then f.scripts.OnEvent(f,event,arg)end end end
 `
	if err := vm.DoString(mock); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{"Wisp.lua", "State.lua"} {
		b, err := os.ReadFile("../../apps/addon/Wisp/" + file)
		if err != nil {
			t.Fatal(err)
		}
		if err = vm.DoString(string(b)); err != nil {
			t.Fatal(err)
		}
	}
	if err := vm.DoString(`FIRE('PLAYER_LOGIN');assert(WispClientState.Current.level==15);assert(not WispDB.characters['Player-test'].current_state);SlashCmdList.WISP('start');local c=WispDB.characters['Player-test'];assert(c.current_state.level==15);local n=#c.events;FIRE('QUEST_LOG_UPDATE');assert(#c.events==n);PROGRESS=2;FIRE('QUEST_LOG_UPDATE');assert(#c.events==n+1 and c.events[#c.events].kind=='quest_progress');FIRE('QUEST_LOG_UPDATE');assert(#c.events==n+1);LEVEL=16;local snap=WispClientState.Snapshot();assert(snap.level==16 and snap.quests[1].objectives[1].fulfilled==2);ITEM=101;FIRE('PLAYER_EQUIPMENT_CHANGED',16);assert(c.events[#c.events].kind=='equipment_changed');HASQUEST=false;FIRE('QUEST_LOG_UPDATE');assert(c.events[#c.events].kind=='quest_removed');local saved=c.current_state;SlashCmdList.WISP('stop');LEVEL=17;WispClientState.Snapshot();assert(c.current_state==saved and c.current_state.level==16);assert(WispClientState.Current.level==17)`); err != nil {
		t.Fatal(err)
	}
}
func TestCheckpointImportIsDatedIdempotentAndCharacterScoped(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "player.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.DB.Close()
	ctx := context.Background()
	p := DemoExport()
	p.InstallationID = "state-test"
	p.Character.ID = "Player-state"
	for i := range p.Events {
		p.Events[i].ID = p.InstallationID + ":" + p.Character.ID + ":" + string(rune('a'+i))
	}
	p.CurrentState = &LocalState{Schema: 1, ObservedAt: time.Now().Unix(), Revision: 2, Level: 16, Zone: "Tirisfal Glades", ClientBuild: "70790", Interface: 16001, Quests: []wisp.GameQuest{{ID: 376, Title: "The Damned"}}, Professions: []ProfessionState{}, Equipment: []EquippedItem{{Slot: 16, ItemID: 100}}}
	p.Events = p.Events[:1]
	p.Events[0].Kind = "quest_progress"
	p.Events[0].QuestID = 376
	p.Events[0].QuestTitle = "The Damned"
	packs, svErr := ParseSavedVariables(savedFixture(p))
	if svErr != nil || len(packs) != 1 || packs[0].CurrentState == nil {
		t.Fatal("saved checkpoint lost", svErr)
	}
	raw, _ := json.Marshal(p)
	parsed, err := ParseExport(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	r, err := s.Import(ctx, "state-import-1", parsed)
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		CharacterID string `json:"character_id"`
	}
	json.Unmarshal(r.Result, &result)
	if _, err = s.Import(ctx, "state-import-2", parsed); err != nil {
		t.Fatal(err)
	}
	j, err := s.Journey(ctx, result.CharacterID)
	if err != nil || j.Total != 1 || j.CompletedQuests != 0 || j.SavedState.Level != 16 {
		t.Fatal(j, err)
	}
	parsed.CurrentState.Revision = 1
	parsed.CurrentState.Level = 15
	if _, err = s.Import(ctx, "state-import-old", parsed); err != nil {
		t.Fatal(err)
	}
	state, _ := s.SavedState(ctx, result.CharacterID)
	if state.Level != 16 {
		t.Fatal("older checkpoint overwrote newer state")
	}
	if _, err = s.SavedState(ctx, "another-character"); err == nil {
		t.Fatal("state leaked across characters")
	}
	// The saved checkpoint must never be silently promoted to live AI context.
	preview, err := s.ContextPreview(ctx, result.CharacterID, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(preview)
	if strings.Contains(string(b), "game_snapshot") {
		t.Fatal("saved state promoted to live")
	}
}
