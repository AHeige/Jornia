package companion

import (
	"context"
	"encoding/json"
	"github.com/AHeige/Jornia/internal/armory"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func TestProfileReceiptScopeExpiryAndExport(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "player.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.DB.Close()
	calls := 0
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "Bearer private-test-token" {
			t.Error("missing access")
		}
		now := time.Now().Unix()
		_ = json.NewEncoder(w).Encode(armory.Snapshot{Name: "Kungen", Surname: "Forever", Region: "eu", Level: 28, Class: "Warrior", FetchedAt: now, ExpiresAt: now + 28*86400, Items: []armory.Item{{ID: 4705, Name: "Shoulders"}}})
	}))
	defer backend.Close()
	server, err := NewServer(store, backend.URL, "127.0.0.1:8787", fstest.MapFS{})
	if err != nil {
		t.Fatal(err)
	}
	server.access.Store("private-test-token")
	payload := `{"mutation_id":"profile-test-001","character_id":"","name":"Kungen","surname":"Forever","region":"eu","consent":true}`
	for i := 0; i < 2; i++ {
		r := httptest.NewRequest("POST", "http://127.0.0.1:8787/local/profile", strings.NewReader(payload))
		w := httptest.NewRecorder()
		server.configureProfile(w, r)
		if w.Code != 200 || !strings.Contains(w.Body.String(), "completed") {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	if calls != 1 {
		t.Fatal("profile lookup replayed", calls)
	}
	state, err := store.State(context.Background())
	if err != nil || len(state["profiles"].(map[string]armory.Snapshot)) != 1 {
		t.Fatal("profile missing", err)
	}
	export, err := store.ExportHistory(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(export)
	if strings.Contains(string(raw), "private-test-token") || !strings.Contains(string(raw), "Kungen") {
		t.Fatal("invalid export")
	}
	_, _ = store.DB.Exec(`UPDATE character_profiles SET expires_at=1`)
	profiles, err := store.Profiles(context.Background())
	if err != nil || len(profiles) != 0 {
		t.Fatal("expired profile retained")
	}
}
func osMkdirCharacter(root string) error {
	return os.MkdirAll(filepath.Join(root, "WTF", "Account", "account", "Realm", "Player", "SavedVariables"), 0700)
}
func TestDiscoveryRequiresExpandedConsent(t *testing.T) {
	root := t.TempDir()
	if err := osMkdirCharacter(root); err != nil {
		t.Fatal(err)
	}
	store, err := Open(filepath.Join(t.TempDir(), "player.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.DB.Close()
	sync, err := NewJournalSync(store)
	if err != nil {
		t.Fatal(err)
	}
	_, err = sync.Configure(context.Background(), JournalIntent{ID: "old-consent-001", Root: root, Enabled: true, ConsentVersion: 1, ExpectedRevision: 0})
	if err != nil {
		t.Fatal(err)
	}
	sync.Scan(context.Background())
	state, _ := store.State(context.Background())
	if len(state["characters"].([]Character)) != 0 {
		t.Fatal("expanded old permission")
	}
	_, err = sync.Configure(context.Background(), JournalIntent{ID: "new-consent-002", Root: root, Enabled: true, ConsentVersion: 2, ExpectedRevision: 1})
	if err != nil {
		t.Fatal(err)
	}
	sync.Scan(context.Background())
	state, _ = store.State(context.Background())
	if len(state["characters"].([]Character)) != 1 {
		t.Fatal("new consent did not discover")
	}
}
