package companion

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestDirectoryDiscoveryAndJournalIdentity(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"Testspelare", "Second"} {
		if err := os.MkdirAll(filepath.Join(root, "WTF", "Account", "private-account", "Testrealm", name, "SavedVariables"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	store, err := Open(filepath.Join(t.TempDir(), "player.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.DB.Close()
	n, err := store.DiscoverCharacters(context.Background(), root)
	if err != nil || n != 2 {
		t.Fatal(n, err)
	}
	state, _ := store.State(context.Background())
	chars := state["characters"].([]Character)
	if len(chars) != 2 {
		t.Fatal(chars)
	}
	var selected string
	for _, c := range chars {
		if !c.Discovered {
			t.Fatal("missing directory origin")
		}
		if c.Name == "Testspelare" {
			selected = c.ID
		}
		events, _ := store.Events(context.Background(), c.ID)
		if len(events) != 0 {
			t.Fatal("invented journey")
		}
	}
	p := packet()
	receipt, err := store.Import(context.Background(), "discovery-import", p)
	if err != nil || receipt.Status != "completed" {
		t.Fatal(receipt, err)
	}
	events, _ := store.Events(context.Background(), selected)
	if len(events) != 1 {
		t.Fatal("journal not attached to discovered character")
	}
	_, _ = store.DiscoverCharacters(context.Background(), root)
	state, _ = store.State(context.Background())
	if len(state["characters"].([]Character)) != 2 {
		t.Fatal("duplicate roster after journal arrival")
	}
	// Unknown folders aren't treated as characters; no arbitrary settings are parsed.
	_ = os.MkdirAll(filepath.Join(root, "WTF", "Account", "private-account", "Bogus", "Not-a-character"), 0700)
	n, err = store.DiscoverCharacters(context.Background(), root)
	if err != nil || n != 2 {
		t.Fatal("false character", n, err)
	}
}

func TestForgetMappedDirectoryAndJournalDoesNotResurrect(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "WTF", "Account", "account", "Testrealm", "Testspelare", "SavedVariables"), 0700); err != nil {
		t.Fatal(err)
	}
	store, err := Open(filepath.Join(t.TempDir(), "player.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.DB.Close()
	_, _ = store.DiscoverCharacters(context.Background(), root)
	p := packet()
	receipt, err := store.Import(context.Background(), "mapped-import-001", p)
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		CharacterID string `json:"character_id"`
	}
	_ = json.Unmarshal(receipt.Result, &result)
	server, err := NewServer(store, "http://127.0.0.1:1", "127.0.0.1:8787", fstest.MapFS{})
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(map[string]string{"mutation_id": "mapped-forget-001", "character_id": result.CharacterID})
	r := httptest.NewRequest("POST", "http://127.0.0.1:8787/local/forget", bytes.NewReader(payload))
	r.Header.Set("Origin", "http://127.0.0.1:8787")
	r.Header.Set("X-Wisp-Token", server.Token)
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	_, _ = store.DiscoverCharacters(context.Background(), root)
	_, err = store.importData(context.Background(), "mapped-auto-002", p, true)
	if err != nil {
		t.Fatal(err)
	}
	state, err := store.State(context.Background())
	if err != nil || len(state["characters"].([]Character)) != 0 {
		t.Fatal("forgotten mapped identity returned", state, err)
	}
}
