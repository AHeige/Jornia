package companion

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"
	"wisp/internal/storage"
	"wisp/internal/wisp"
)

func TestExpiredWorldCacheDoesNotDeletePersonalJourney(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "player.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.DB.Close()
	ctx := context.Background()
	_, err = s.Import(ctx, "demo-expiry-setup", DemoExport())
	if err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-time.Hour)
	answer, _ := json.Marshal(wisp.Answer{Status: "answered", Text: "Expired external world data", Sources: []storage.Source{{ID: "api-expired", ExpiresAt: &old}}})
	for _, id := range []string{"expired-message", "personal-message"} {
		text := string(answer)
		if id == "personal-message" {
			text = `{"text":"Local personal answer","sources":[]}`
		}
		if _, err = s.DB.Exec(`INSERT INTO messages(id,character_id,question,answer,created_at) VALUES(?,?,?,?,?)`, id, DemoCharacterID, "Question", text, time.Now().Format(time.RFC3339)); err != nil {
			t.Fatal(err)
		}
	}
	_, err = s.DB.Exec(`INSERT INTO operations(id,kind,character_id,payload_hash,status,result,created_at) VALUES('expired-operation','ask',?,'hash','completed',?,?)`, DemoCharacterID, string(answer), time.Now().Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.State(ctx); err != nil {
		t.Fatal(err)
	}
	var count int
	_ = s.DB.QueryRow(`SELECT count(*) FROM messages`).Scan(&count)
	if count != 1 {
		t.Fatal("wrong messages removed")
	}
	_ = s.DB.QueryRow(`SELECT count(*) FROM events`).Scan(&count)
	if count != 3 {
		t.Fatal("personal observations deleted")
	}
	_ = s.DB.QueryRow(`SELECT count(*) FROM operations WHERE id='expired-operation'`).Scan(&count)
	if count != 0 {
		t.Fatal("expired source retained in receipt")
	}
}
