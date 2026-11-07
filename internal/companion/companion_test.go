package companion

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"
)

func packet() Export {
	return Export{Schema: 1, InstallationID: "test-install", Character: Character{ID: "Player-1-abc", Name: "TestPlayer", Realm: "Testrealm"}, Events: []Event{{ID: "test-install:Player-1-abc:1", Kind: "note", ObservedAt: time.Now().Add(-time.Minute).Unix(), Zone: "Ashenvale", Subzone: "Astranaar", Level: 24, Note: "I wanted to talk to Raene.", ClientVersion: "1.60.1", ClientBuild: "70790", Interface: 16001}}}
}
func TestImportIntegrityAndRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "player.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	data := packet()
	b, _ := json.Marshal(data)
	parsed, err := ParseExport("WISP1:" + string(b))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"import-001", "import-001", "import-002"} {
		r, err := s.Import(ctx, id, parsed)
		if err != nil || r.Status != "completed" {
			t.Fatalf("import %+v %v", r, err)
		}
	}
	events, err := s.Events(ctx, "test-install:Player-1-abc")
	if err != nil || len(events) != 1 {
		t.Fatalf("duplicate events: %d %v", len(events), err)
	}
	parsed.Events[0].Note = "Changed"
	if _, err = s.Import(ctx, "import-003", parsed); err != ErrConflict {
		t.Fatalf("tampering accepted: %v", err)
	}
	if _, err = s.begin(ctx, "ask-pending", "ask", "test-install:Player-1-abc", "hash"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.begin(ctx, "ask-concurrent", "ask", "test-install:Player-1-abc", "other"); err == nil {
		t.Fatal("concurrent ask accepted")
	}
	s.DB.Close()
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.DB.Close()
	receipt, err := s.Operation(ctx, "ask-pending")
	if err != nil || receipt.Status != "needs_attention" {
		t.Fatalf("pending request replayed or lost: %+v %v", receipt, err)
	}
	events, _ = s.Events(ctx, "test-install:Player-1-abc")
	if len(events) != 1 || events[0].Note != "I wanted to talk to Raene." {
		t.Fatal("history changed across restart")
	}
}
func TestRequestBoundaryAndExactlyOnce(t *testing.T) {
	var count atomic.Int32
	release := make(chan struct{})
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		var raw map[string]any
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			t.Error(err)
		}
		if len(raw) != 4 {
			t.Errorf("private context leaked: %+v", raw)
		}
		if _, ok := raw["character_id"]; ok {
			t.Error("character id leaked")
		}
		<-release
		fmt.Fprint(w, `{"status":"answered","text":"Astranaar is in Ashenvale.","game_version":"forever","route":"deterministic","sources":[],"evidence_ids":[],"warnings":[],"estimated_cost_eur":0}`)
	}))
	defer backend.Close()
	store, err := Open(filepath.Join(t.TempDir(), "player.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.DB.Close()
	data := packet()
	receipt, err := store.Import(context.Background(), "import-local", data)
	if err != nil {
		t.Fatal(err)
	}
	_ = receipt
	server, err := NewServer(store, backend.URL, "127.0.0.1:8787", fstest.MapFS{"index.html": {Data: []byte("__WISP_TOKEN__")}})
	if err != nil {
		t.Fatal(err)
	}
	handler := server.Handler()
	send := func(path, body, origin, token string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest("POST", "http://127.0.0.1:8787"+path, strings.NewReader(body))
		req.Header.Set("Origin", origin)
		req.Header.Set("X-Wisp-Token", token)
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		return res
	}
	body := `{"mutation_id":"ask-world-001","character_id":"test-install:Player-1-abc","question":"Where is Astranaar?","ai":false}`
	if res := send("/local/ask", body, "https://evil.example", server.Token); res.Code != 403 {
		t.Fatal("cross origin accepted")
	}
	if res := send("/local/ask", body, "http://127.0.0.1:8787", ""); res.Code != 403 {
		t.Fatal("missing token accepted")
	}
	for i := 0; i < 5; i++ {
		if res := send("/local/ask", body, "http://127.0.0.1:8787", server.Token); res.Code != 202 {
			t.Fatal(res.Code, res.Body.String())
		}
	}
	close(release)
	deadline := time.Now().Add(3 * time.Second)
	for {
		r, _ := store.Operation(context.Background(), "ask-world-001")
		if r.Status == "completed" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("never completed %+v", r)
		}
		time.Sleep(5 * time.Millisecond)
	}
	if count.Load() != 1 {
		t.Fatalf("duplicate inference: %d", count.Load())
	}
	if res := send("/local/ask", strings.Replace(body, "Where is Astranaar?", "Changed?", 1), "http://127.0.0.1:8787", server.Token); res.Code != 409 {
		t.Fatal("mutation payload conflict accepted")
	}
	a, err := server.memory(context.Background(), "test-install:Player-1-abc")
	if err != nil || a.Status != "answered" || !strings.Contains(a.Text, "Raene") || count.Load() != 1 {
		t.Fatalf("local memory failed: %+v %v", a, err)
	}
}
func TestRejectUnsupportedExport(t *testing.T) {
	for _, change := range []func(*Export){func(e *Export) { e.Events[0].Interface = 120105 }, func(e *Export) { e.Schema = 2 }, func(e *Export) { e.Events[0].ID = "collision" }, func(e *Export) { e.Events[0].Kind = "guild_chat" }, func(e *Export) { e.Events[0].ObservedAt = time.Now().Add(time.Hour).Unix() }} {
		data := packet()
		change(&data)
		b, _ := json.Marshal(data)
		if _, err := ParseExport(string(b)); err == nil {
			t.Fatal("invalid export accepted")
		}
	}
}

func TestForgetKeepsOtherCharacterAndClearsRawReceipt(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "player.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.DB.Close()
	data := packet()
	if _, err = store.Import(context.Background(), "import-first", data); err != nil {
		t.Fatal(err)
	}
	alt := packet()
	alt.Character.ID = "Player-1-other"
	alt.Events[0].ID = "test-install:Player-1-other:1"
	if _, err = store.Import(context.Background(), "import-other", alt); err != nil {
		t.Fatal(err)
	}
	server, err := NewServer(store, "http://127.0.0.1:1", "127.0.0.1:8787", fstest.MapFS{})
	if err != nil {
		t.Fatal(err)
	}
	body := `{"mutation_id":"forget-first","character_id":"test-install:Player-1-abc"}`
	for i := 0; i < 2; i++ {
		r := httptest.NewRequest("POST", "http://127.0.0.1:8787/local/forget", strings.NewReader(body))
		r.Header.Set("Origin", "http://127.0.0.1:8787")
		r.Header.Set("X-Wisp-Token", server.Token)
		w := httptest.NewRecorder()
		server.Handler().ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	events, _ := store.Events(context.Background(), "test-install:Player-1-abc")
	if len(events) != 0 {
		t.Fatal("forgotten events remain")
	}
	events, _ = store.Events(context.Background(), "test-install:Player-1-other")
	if len(events) != 1 {
		t.Fatal("other character was changed")
	}
	var char, request string
	if err = store.DB.QueryRow(`SELECT character_id,request_json FROM operations WHERE id='forget-first'`).Scan(&char, &request); err != nil || char != "" || request != "{}" {
		t.Fatalf("raw deleted identity remains: %q %q %v", char, request, err)
	}
}
