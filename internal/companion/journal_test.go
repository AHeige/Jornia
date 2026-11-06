package companion

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	lua "github.com/yuin/gopher-lua"
)

func literal(v any) string {
	switch x := v.(type) {
	case string:
		return strconv.Quote(x)
	case bool:
		if x {
			return "true"
		}
		return "false"
	case nil:
		return "nil"
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case []any:
		parts := []string{}
		for _, item := range x {
			parts = append(parts, literal(item))
		}
		return "{" + strings.Join(parts, ",") + "}"
	case map[string]any:
		keys := []string{}
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := []string{}
		for _, k := range keys {
			parts = append(parts, "["+strconv.Quote(k)+"]="+literal(x[k]))
		}
		return "{" + strings.Join(parts, ",") + "}"
	}
	panic(fmt.Sprintf("unsupported %T", v))
}
func savedFixture(packs ...Export) string {
	chars := map[string]any{}
	installation := ""
	for _, p := range packs {
		installation = p.InstallationID
		b, _ := json.Marshal(p)
		var object map[string]any
		_ = json.Unmarshal(b, &object)
		c := object["character"].(map[string]any)
		c["events"] = object["events"]
		c["recording"] = true
		chars[p.Character.ID] = c
	}
	b, _ := json.Marshal(map[string]any{"schema": 1, "installation_id": installation, "characters": chars})
	var v any
	_ = json.Unmarshal(b, &v)
	return "\n-- Saved by WoW\nWispDB = " + literal(v) + "\n"
}
func TestDataOnlySavedVariables(t *testing.T) {
	p := DemoExport()
	p.InstallationID = "local-test"
	p.Character.ID = "Player-1"
	for i := range p.Events {
		p.Events[i].ID = fmt.Sprintf("local-test:Player-1:%d", i+1)
	}
	raw := savedFixture(p)
	packs, err := ParseSavedVariables(raw)
	if err != nil || len(packs) != 1 || len(packs[0].Events) != 3 {
		t.Fatalf("%v %+v", err, packs)
	}
	for _, bad := range []string{raw + "os.execute('bad')", `WispDB = loadstring('bad')()`, `WispDB = { x = function() end }`, raw[:len(raw)-5], `WispDB = { [1] = 1, [1] = 2 }`, `OtherDB = {}`} {
		if _, err := ParseSavedVariables(bad); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
	r := savedReader{s: `"x\034y\n\195\165"`}
	value, err := r.quoted()
	if err != nil || value != "x\"y\nå" {
		t.Fatalf("escape %q %v", value, err)
	}
}
func TestAutomaticInitialImportPauseRestartAndGraph(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db := filepath.Join(dir, "player.db")
	s, err := Open(db)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { s.DB.Close() }()
	j, _ := NewJournalSync(s)
	root := filepath.Join(dir, "WoW")
	folder := filepath.Join(root, "WTF", "Account", "account", "SavedVariables")
	if err = os.MkdirAll(folder, 0700); err != nil {
		t.Fatal(err)
	}
	p := DemoExport()
	p.InstallationID = "journey-test"
	p.Character.ID = "Player-main"
	p.Character.Name = "Main"
	for i := range p.Events {
		p.Events[i].ID = fmt.Sprintf("journey-test:Player-main:%d", i+1)
		p.Events[i].SessionID = "one-session"
	}
	p.Events[0].Kind = "session_start"
	p.Events[1].Kind = "quest_completed"
	p.Events[1].QuestID = 1020
	alt := p
	alt.Character = Character{ID: "Player-alt", Name: "Alt", Realm: "Realm"}
	alt.Events = append([]Event(nil), p.Events...)
	for i := range alt.Events {
		alt.Events[i].ID = fmt.Sprintf("journey-test:Player-alt:%d", i+1)
		alt.Events[i].Zone = "Darkshore"
		alt.Events[i].Subzone = ""
	}
	file := filepath.Join(folder, "Wisp.lua")
	_ = os.WriteFile(file, []byte(savedFixture(p, alt)), 0600)
	// Nothing is read without explicit first-use permission.
	j.Scan(ctx)
	var n int
	_ = s.DB.QueryRow(`SELECT count(*) FROM events`).Scan(&n)
	if n != 0 {
		t.Fatal("read before consent")
	}
	request := JournalIntent{ID: "journal-enable-1", Root: root, Enabled: true, ConsentVersion: 1}
	if _, err = j.Configure(ctx, request); err != nil {
		t.Fatal(err)
	}
	j.Scan(ctx)
	char := "journey-test:Player-main"
	graph, err := s.Journey(ctx, char)
	if err != nil || graph.Total != 3 || graph.Places != 2 || graph.Sessions != 1 || graph.CompletedQuests != 1 {
		t.Fatalf("graph %+v %v", graph, err)
	}
	// A repeat scan and repeat mutation do not duplicate facts or advance the revision.
	_, _ = j.Configure(ctx, request)
	j.Scan(ctx)
	if j.Status().Config.Revision != 1 {
		t.Fatal("duplicate consent applied")
	}
	if _, err = j.Configure(ctx, JournalIntent{ID: "stale-settings-1", Enabled: false, ExpectedRevision: 0}); err == nil {
		t.Fatal("stale tab overwrote settings")
	}
	if _, err = j.Configure(ctx, JournalIntent{ID: "journal-pause-1", Enabled: false, ExpectedRevision: 1}); err != nil {
		t.Fatal(err)
	}
	p.Events = append(p.Events, Event{ID: "journey-test:Player-main:4", Kind: "level_up", ObservedAt: p.Events[2].ObservedAt + 1, Zone: "Ashenvale", Level: 25, ClientVersion: "1.60.1", Interface: 16001, SessionID: "one-session"})
	_ = os.WriteFile(file, []byte(savedFixture(p, alt)), 0600)
	j.Scan(ctx)
	graph, _ = s.Journey(ctx, char)
	if graph.Total != 3 {
		t.Fatal("paused importer ran")
	}
	_, err = j.Configure(ctx, JournalIntent{ID: "journal-enable-2", Root: root, Enabled: true, ConsentVersion: 1, ExpectedRevision: 2})
	if err != nil {
		t.Fatal(err)
	}
	j.Scan(ctx)
	graph, _ = s.Journey(ctx, char)
	if graph.Total != 4 {
		t.Fatal("new event not imported")
	}
	// Partial/executable files never erase the last good observations.
	_ = os.WriteFile(file, []byte("WispDB = {"), 0600)
	j.Scan(ctx)
	graph, _ = s.Journey(ctx, char)
	if graph.Total != 4 || j.Status().Error == "" {
		t.Fatal("partial file not safely retained")
	}
	_ = os.WriteFile(file, []byte(savedFixture(p, alt)), 0600)
	// Forgetting suppresses resurrection from a file that remains on disk.
	tx, _ := s.DB.Begin()
	_, _ = tx.Exec(`INSERT INTO journey_exclusions(character_id) VALUES(?)`, Fingerprint(char))
	_, _ = tx.Exec(`DELETE FROM operations WHERE character_id=?`, char)
	_, _ = tx.Exec(`DELETE FROM characters WHERE id=?`, char)
	_ = tx.Commit()
	s.DB.Close()
	s, err = Open(db)
	if err != nil {
		t.Fatal(err)
	}
	j, _ = NewJournalSync(s)
	j.Scan(ctx)
	graph, _ = s.Journey(ctx, char)
	var retained int
	_ = s.DB.QueryRow(`SELECT count(*) FROM operations WHERE character_id=?`, char).Scan(&retained)
	if retained != 0 {
		t.Fatal("auto read recreated private deletion receipt")
	}
	if graph.Total != 0 || graph.Places != 0 {
		t.Fatal("forgotten character resurrected")
	}
	other, _ := s.Journey(ctx, "journey-test:Player-alt")
	if other.Total != 3 {
		t.Fatal("character isolation failed")
	}
	if !j.Status().Config.Enabled {
		t.Fatal("consent did not survive restart")
	}
	// An explicit manual import is how the player deliberately restores this character.
	if _, err = s.Import(ctx, "manual-restore-1", p); err != nil {
		t.Fatal(err)
	}
	graph, _ = s.Journey(ctx, char)
	if graph.Total != 4 {
		t.Fatal("manual restore failed")
	}
	server := &Server{Store: s}
	answer, err := server.answer(ctx, Ask{CharacterID: char, Question: "Visa min resa", AI: true})
	if err != nil || answer.Route != "local_memory" || !strings.Contains(answer.Text, "#1020") || strings.Contains(answer.Text, "Darkshore") {
		t.Fatalf("local journey: %+v %v", answer, err)
	}
}
func TestAddonConsentAndAutomaticSessions(t *testing.T) {
	vm := lua.NewState()
	defer vm.Close()
	mock := `
 StaticPopupDialogs={};SlashCmdList={};UIParent={};UISpecialFrames={}; CLOCK=1793872800;ZONE='Ashenvale'
 local frame={};function frame:RegisterEvent()end;function frame:SetScript(name,fn)if name=='OnEvent' then HANDLER=fn end end
 function CreateFrame()return frame end
 function StaticPopup_Show(name) POPUP=name end
 function UnitGUID()return 'Player-one' end;function UnitName()return 'Main' end;function GetRealmName()return 'Realm' end
 function GetBuildInfo()return '1.60.1','70790','date',16001 end;function UnitLevel()return 24 end
 function GetZoneText()return ZONE end;function GetSubZoneText()return '' end;function time()return CLOCK end;function print()end
 `
	if err := vm.DoString(mock); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join("..", "..", "apps", "addon", "Wisp", "Wisp.lua"))
	if err := vm.DoString(string(b)); err != nil {
		t.Fatal(err)
	}
	script := `HANDLER(nil,'PLAYER_LOGIN'); assert(POPUP=='WISP_JOURNAL_CONSENT'); assert(#WispDB.characters['Player-one'].events==0); StaticPopupDialogs.WISP_JOURNAL_CONSENT.OnAccept(); assert(#WispDB.characters['Player-one'].events==1); CLOCK=CLOCK+1;ZONE='Darkshore';HANDLER(nil,'ZONE_CHANGED_NEW_AREA');HANDLER(nil,'QUEST_TURNED_IN',1020);HANDLER(nil,'PLAYER_LOGOUT'); assert(#WispDB.characters['Player-one'].events==4);assert(WispDB.characters['Player-one'].events[4].kind=='session_end');SlashCmdList.WISP('stop');HANDLER(nil,'PLAYER_LEVEL_UP',25);assert(#WispDB.characters['Player-one'].events==4);function UnitGUID()return 'Player-two' end;HANDLER(nil,'PLAYER_LOGIN');assert(#WispDB.characters['Player-two'].events==1)`
	if err := vm.DoString(script); err != nil {
		t.Fatal(err)
	}
}

func TestSavedVariablesLargeJournalChunksAndProjectionBackfill(t *testing.T) {
	p := DemoExport()
	p.InstallationID = "large-journal"
	p.Character.ID = "Player-many"
	template := p.Events[0]
	p.Events = nil
	for i := 0; i < 205; i++ {
		e := template
		e.ID = fmt.Sprintf("large-journal:Player-many:%d", i+1)
		e.ObservedAt += int64(i)
		e.SessionID = "session-1"
		p.Events = append(p.Events, e)
	}
	packs, err := ParseSavedVariables(savedFixture(p))
	if err != nil || len(packs) != 3 {
		t.Fatalf("chunks %d %v", len(packs), err)
	}
	path := filepath.Join(t.TempDir(), "player.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	for i, pack := range packs {
		if _, err = s.Import(context.Background(), fmt.Sprintf("chunk-import-%d", i), pack); err != nil {
			t.Fatal(err)
		}
	}
	_, _ = s.DB.Exec(`DELETE FROM journey_edges`)
	_, _ = s.DB.Exec(`DELETE FROM journey_nodes`)
	s.DB.Close()
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.DB.Close()
	exported, err := s.ExportHistory(context.Background())
	if err != nil || len(exported["events"].(map[string][]Event)["large-journal:Player-many"]) != 205 {
		t.Fatal("full history export truncated")
	}
	graph, err := s.Journey(context.Background(), "large-journal:Player-many")
	if err != nil || graph.Total != 205 || graph.Places != 1 || graph.Sessions != 1 || len(graph.Events) != 50 {
		t.Fatalf("backfill %+v %v", graph, err)
	}
}
