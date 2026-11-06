package companion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	marker := filepath.Join(root, "WTF", "Account", "private-account", "Testrealm", "Testspelare", "SavedVariables", "Wisp.lua")
	if err := os.WriteFile(marker, []byte(identityFixture(p)), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DiscoverCharacters(context.Background(), root); err != nil {
		t.Fatal(err)
	}
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
	marker := filepath.Join(root, "WTF", "Account", "account", "Testrealm", "Testspelare", "SavedVariables", "Wisp.lua")
	if err := os.WriteFile(marker, []byte(identityFixture(p)), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DiscoverCharacters(context.Background(), root); err != nil {
		t.Fatal(err)
	}
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

func identityFixture(p Export) string {
	return fmt.Sprintf(`WispCharacterDB = {schema=1,installation_id=%q,character_id=%q}`, p.InstallationID, p.Character.ID)
}

func TestFolderIdentityWithSurnameAndNumericRealm(t *testing.T) {
	for _, journalFirst := range []bool{false, true} {
		t.Run(fmt.Sprint(journalFirst), func(t *testing.T) {
			ctx := context.Background()
			root := t.TempDir()
			folder := filepath.Join(root, "WTF", "Account", "account", "93", "Sch-Nice", "SavedVariables")
			if err := os.MkdirAll(folder, 0700); err != nil {
				t.Fatal(err)
			}
			s, err := Open(filepath.Join(t.TempDir(), "player.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer s.DB.Close()
			j, _ := NewJournalSync(s)
			_, err = j.Configure(ctx, JournalIntent{ID: "identity-enable", Root: root, Enabled: true, ConsentVersion: 2})
			if err != nil {
				t.Fatal(err)
			}
			j.Scan(ctx)
			before, _ := s.State(ctx)
			preliminary := before["characters"].([]Character)[0].ID
			p := packet()
			p.Character.Name = "Sch"
			p.Character.Realm = "CAMWOW EU5 Realm 1 PvE"
			account := filepath.Join(root, "WTF", "Account", "account", "SavedVariables")
			if err = os.MkdirAll(account, 0700); err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(filepath.Join(account, "Wisp.lua"), []byte(savedFixture(p)), 0600); err != nil {
				t.Fatal(err)
			}
			if journalFirst {
				j.Scan(ctx)
			}
			if err = os.WriteFile(filepath.Join(folder, "Wisp.lua"), []byte(identityFixture(p)), 0600); err != nil {
				t.Fatal(err)
			}
			j.Scan(ctx)
			j.Scan(ctx)
			state, err := s.State(ctx)
			if err != nil {
				t.Fatal(err)
			}
			chars := state["characters"].([]Character)
			if len(chars) != 1 || chars[0].Name != "Sch-Nice" {
				t.Fatal("duplicate or lost surname", chars)
			}
			if !journalFirst && chars[0].ID != preliminary {
				t.Fatal("changed preliminary identity", chars)
			}
			events, _ := s.Events(ctx, chars[0].ID)
			if len(events) != len(p.Events) {
				t.Fatal("missing or duplicated events", events)
			}
			if j.Status().Error != "" {
				t.Fatal(j.Status().Error)
			}
			// Restart discovery remains stable; reusing the directory for a different
			// GUID creates a separate journal rather than contaminating the old one.
			p.Character.ID = "Player-different"
			if err = os.WriteFile(filepath.Join(folder, "Wisp.lua"), []byte(identityFixture(p)), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err = s.DiscoverCharacters(ctx, root); err != nil {
				t.Fatal(err)
			}
			events, _ = s.Events(ctx, chars[0].ID)
			if len(events) != 1 {
				t.Fatal("old history changed")
			}
		})
	}
}

func TestIdentityMarkerRejectsExecutableAndMalformedData(t *testing.T) {
	for _, raw := range []string{`WispCharacterDB = os.execute("bad")`, `WispCharacterDB={schema=1,installation_id="x",character_id="Player-x"};print(1)`, `WispCharacterDB={schema=1,installation_id="x",character_id="missing-guid"}`} {
		if _, err := parseCharacterIdentity(raw); err == nil {
			t.Fatal("accepted invalid marker", raw)
		}
	}
}

func TestSameNamesAcrossAccountsAreNotIdentity(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	for _, account := range []string{"one", "two"} {
		if err := os.MkdirAll(filepath.Join(root, "WTF", "Account", account, "93", "Sch-Nice", "SavedVariables"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	s, err := Open(filepath.Join(t.TempDir(), "player.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.DB.Close()
	if n, err := s.DiscoverCharacters(ctx, root); err != nil || n != 2 {
		t.Fatal(n, err)
	}
	state, _ := s.State(ctx)
	if len(state["characters"].([]Character)) != 2 {
		t.Fatal("same-name folders merged")
	}
	p := packet()
	p.Character.Name = "Sch-Nice"
	p.Character.Realm = "93"
	if _, err := s.Import(ctx, "without-marker-import", p); err != nil {
		t.Fatal(err)
	}
	state, _ = s.State(ctx)
	if len(state["characters"].([]Character)) != 3 {
		t.Fatal("name-only journal merge")
	}
	marker := filepath.Join(root, "WTF", "Account", "two", "93", "Sch-Nice", "SavedVariables", "Wisp.lua")
	if err := os.WriteFile(marker, []byte(identityFixture(p)), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DiscoverCharacters(ctx, root); err != nil {
		t.Fatal(err)
	}
	state, _ = s.State(ctx)
	if len(state["characters"].([]Character)) != 2 {
		t.Fatal("marker didn't resolve exact folder")
	}
}
