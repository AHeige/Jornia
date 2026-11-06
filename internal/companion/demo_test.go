package companion

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestDemoIsolationReplayAndRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "player.db")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	server, err := NewServer(store, "http://127.0.0.1:1", "127.0.0.1:8787", fstest.MapFS{})
	if err != nil {
		t.Fatal(err)
	}
	real := packet()
	if _, err = store.Import(context.Background(), "real-import-001", real); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("POST", "http://127.0.0.1:8787/local/demo", strings.NewReader(`{"mutation_id":"demo-replay-001"}`))
		req.Header.Set("Origin", "http://127.0.0.1:8787")
		req.Header.Set("X-Wisp-Token", server.Token)
		res := httptest.NewRecorder()
		server.Handler().ServeHTTP(res, req)
		if res.Code != 200 {
			t.Fatal(res.Code, res.Body.String())
		}
	}
	events, err := store.Events(context.Background(), DemoCharacterID)
	if err != nil || len(events) != 3 {
		t.Fatalf("duplicate demo events: %d %v", len(events), err)
	}
	a, err := server.answer(context.Background(), Ask{CharacterID: DemoCharacterID, Question: "Vad gjorde jag sist?"})
	if err != nil || a.Route != "local_memory" || !strings.Contains(a.Text, "Demodata") || a.Sources[0].ValidationStatus != "synthetic_demo" {
		t.Fatalf("unlabelled demo: %+v %v", a, err)
	}
	actual, _ := server.memory(context.Background(), "test-install:Player-1-abc")
	if strings.Contains(actual.Text, "Demodata") || strings.Contains(actual.Text, "påhittad demoanteckning") {
		t.Fatal("demo leaked into imported character")
	}
	store.DB.Close()
	store, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.DB.Close()
	state, err := store.State(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	chars := state["characters"].([]Character)
	if len(chars) != 2 {
		t.Fatal("characters lost on restart")
	}
	for _, c := range chars {
		if c.IsDemo != (c.ID == DemoCharacterID) {
			t.Fatal("demo classification changed")
		}
	}
	raw, _ := json.Marshal(DemoExport())
	req := httptest.NewRequest("POST", "http://127.0.0.1:8787/local/import", strings.NewReader(`{"mutation_id":"spoofed-demo-001","export":`+string(mustJSON(string(raw)))+`}`))
	req.Header.Set("Origin", "http://127.0.0.1:8787")
	req.Header.Set("X-Wisp-Token", server.Token)
	server.Store = store
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != 400 {
		t.Fatal("reserved demo identity accepted via manual import")
	}
}
func mustJSON(value string) []byte { b, _ := json.Marshal(value); return b }

func TestConnectionCheckDoesNotInvokeInference(t *testing.T) {
	calls := 0
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "GET" || r.URL.Path != "/healthz" {
			t.Error("probe invoked non-health endpoint")
		}
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer backend.Close()
	store, err := Open(filepath.Join(t.TempDir(), "player.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.DB.Close()
	server, _ := NewServer(store, backend.URL, "127.0.0.1:8787", fstest.MapFS{})
	req := httptest.NewRequest("GET", "http://127.0.0.1:8787/local/connection", nil)
	req.Header.Set("X-Wisp-Token", server.Token)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if calls != 1 || res.Code != 200 || !strings.Contains(res.Body.String(), `"connected":true`) {
		t.Fatal(res.Code, res.Body.String(), calls)
	}
}
