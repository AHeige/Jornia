package companion

import (
	"context"
	"encoding/json"
	"github.com/AHeige/Jornia/internal/armory"
	"github.com/AHeige/Jornia/internal/wisp"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func TestCharacterContextExplicitSharingAndIsolation(t *testing.T) {
	ctx := context.Background()
	store, err := Open(filepath.Join(t.TempDir(), "player.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.DB.Close()
	_, err = store.DB.Exec(`INSERT INTO characters(id,name,realm) VALUES('one','Private One','Realm'),('two','Private Two','Realm')`)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	p := armory.Snapshot{Name: "Private", Surname: "One", Region: "eu", Level: 28, Race: "Night Elf", Class: "Warrior", FetchedAt: now, LastLogin: now - 60, ExpiresAt: now + 86400, Image: "https://render.worldofwarcraft.com/private", Items: []armory.Item{{ID: 4705, Name: "Test Pauldrons", Slot: "SHOULDER", Quality: "UNCOMMON"}}}
	raw, _ := json.Marshal(p)
	_, err = store.DB.Exec(`INSERT INTO character_profiles(character_id,body,expires_at) VALUES('one',?,?)`, string(raw), p.ExpiresAt)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := store.ContextPreview(ctx, "one", []string{})
	if err != nil || preview.Character == nil || len(preview.Events) != 0 {
		t.Fatal(preview, err)
	}
	raw, _ = json.Marshal(preview)
	for _, private := range []string{"Private", "render.worldofwarcraft", "Surname", "Realm"} {
		if strings.Contains(string(raw), private) {
			t.Fatal("private identity leaked", string(raw))
		}
	}
	other, err := store.ContextPreview(ctx, "two", []string{})
	if err != nil || other.Character != nil || other.Digest == preview.Digest {
		t.Fatal("cross-character projection", other, err)
	}
	calls := 0
	var got wisp.Request
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		json.NewDecoder(r.Body).Decode(&got)
		json.NewEncoder(w).Encode(wisp.Answer{Status: "unknown", Text: "Unknown"})
	}))
	defer backend.Close()
	server, _ := NewServer(store, backend.URL, "local.test", fstest.MapFS{})
	_, err = server.answer(ctx, Ask{ID: "no-consent", CharacterID: "one", AI: true, Question: "Compare my shoulders with a new item"})
	if err != nil || got.Character != nil {
		t.Fatal("implicit sharing", err, got)
	}
	_, err = server.answer(ctx, Ask{ID: "consent", CharacterID: "one", AI: true, Question: "Describe my saved shoulders", character: preview.Character})
	if err != nil || got.Character == nil || calls != 2 {
		t.Fatal("approved context absent", err, got)
	}
	for _, q := range []string{"what level am i", "what is my class", "my gear", "my character", "min utrustning"} {
		a, handled, err := server.characterAnswer(ctx, "one", q)
		if err != nil || !handled || a.Status != "answered" || a.Route != "local_memory" {
			t.Fatal(q, a, err)
		}
	}
	a, _, _ := server.characterAnswer(ctx, "two", "my gear")
	if a.Status != "unknown" {
		t.Fatal("other character borrowed gear")
	}
	// A changed profile invalidates the reviewed digest before any model call.
	p.Level++
	raw, _ = json.Marshal(p)
	store.DB.Exec(`UPDATE character_profiles SET body=? WHERE character_id='one'`, string(raw))
	body, _ := json.Marshal(Ask{ID: "stale-context-check", CharacterID: "one", AI: true, Question: "My class", ContextDigest: preview.Digest, ContextIDs: []string{}})
	req := httptest.NewRequest("POST", "http://local.test/local/ask", strings.NewReader(string(body)))
	req.Header.Set("Origin", "http://local.test")
	req.Header.Set("X-Wisp-Token", server.Token)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != 409 || calls != 2 {
		t.Fatal("stale profile sent", rec.Code, calls)
	}
}
