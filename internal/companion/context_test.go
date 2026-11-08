package companion

import (
	"context"
	"encoding/json"
	"github.com/AHeige/Jornia/internal/storage"
	"github.com/AHeige/Jornia/internal/wisp"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestJourneyContextConsentAndIsolation(t *testing.T) {
	ctx := context.Background()
	store, err := Open(filepath.Join(t.TempDir(), "player.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.DB.Close()
	data := packet()
	data.Events = append(data.Events, Event{ID: "test-install:Player-1-abc:2", Kind: "level_up", ObservedAt: data.Events[0].ObservedAt + 1, Zone: "Ashenvale", Level: 25, ClientVersion: "1.60.1", ClientBuild: "70790", Interface: 16001})
	if _, err = store.Import(ctx, "context-import", data); err != nil {
		t.Fatal(err)
	}
	char := "test-install:Player-1-abc"
	preview, err := store.ContextPreview(ctx, char, nil)
	if err != nil || len(preview.Events) != 1 {
		t.Fatalf("preview %+v %v", preview, err)
	}
	raw, _ := json.Marshal(preview.Events)
	if strings.Contains(string(raw), "Player-") || strings.Contains(string(raw), "Raene") {
		t.Fatal("private IDs or notes shared")
	}
	if _, err = store.ContextPreview(ctx, "another-character", preview.EventIDs); err == nil {
		t.Fatal("cross-character context accepted")
	}
	calls := 0
	var received wisp.Request
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		json.NewDecoder(r.Body).Decode(&received)
		json.NewEncoder(w).Encode(wisp.Answer{Status: "answered", Text: "Svar", Sources: []storage.Source{}, EvidenceIDs: []string{}})
	}))
	defer backend.Close()
	server, err := NewServer(store, backend.URL, "local.test", fstest.MapFS{})
	if err != nil {
		t.Fatal(err)
	}
	req := Ask{ID: "context-ask", CharacterID: char, Question: "Hur går min resa?", AI: true}
	if _, err = server.answer(ctx, req); err != nil {
		t.Fatal(err)
	}
	if len(received.Journal) != 0 {
		t.Fatal("implicit sharing")
	}
	req.journal = preview.Events
	if _, err = server.answer(ctx, req); err != nil || len(received.Journal) != 1 {
		t.Fatalf("explicit context lost %v", err)
	}
	req.journal = nil
	req.ContextIDs = preview.EventIDs
	req.ContextDigest = "wrong"
	body, _ := json.Marshal(req)
	request := httptest.NewRequest("POST", "http://local.test/local/ask", strings.NewReader(string(body)))
	request.Header.Set("Origin", "http://local.test")
	request.Header.Set("X-Wisp-Token", server.Token)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != 409 || calls != 2 {
		t.Fatalf("unreviewed context sent: %d calls %d", response.Code, calls)
	}
}
