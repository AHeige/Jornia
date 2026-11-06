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
	"time"
	"wisp/internal/wisp"
)

func TestGameContextExplicitApprovalAndActionFiltering(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "p.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.DB.Close()
	store.DB.Exec(`INSERT INTO characters(id,name,realm) VALUES('one','Private','Realm')`)
	now := time.Now().Unix()
	snap := &wisp.GameSnapshot{CapturedAt: now, Level: 16, Quests: []wisp.GameQuest{{ID: 123, Waypoint: true}}}
	valid := wisp.Action{ID: "navigate:quest:123", Kind: "show_quest_waypoint", QuestID: 123, ExpiresAt: now + 90}
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var got wisp.Request
		json.NewDecoder(r.Body).Decode(&got)
		if got.Snapshot == nil {
			t.Error("missing approved snapshot")
		}
		json.NewEncoder(w).Encode(wisp.Answer{Status: "answered", Text: "Next quest", EvidenceIDs: []string{"state:quest:123"}, Actions: []wisp.Action{valid, {ID: "navigate:quest:999", Kind: "show_quest_waypoint", QuestID: 999, ExpiresAt: now + 90}, {ID: "navigate:quest:123", Kind: "run_script", QuestID: 123, ExpiresAt: now + 90}}})
	}))
	defer backend.Close()
	server, _ := NewServer(store, backend.URL, "local.test", fstest.MapFS{})
	request := Ask{ID: "test-context", CharacterID: "one", Question: "Next?", AI: true, Snapshot: snap}
	post := func() int {
		body, _ := json.Marshal(request)
		r := httptest.NewRequest("POST", "http://local.test/local/ask", strings.NewReader(string(body)))
		r.Header.Set("Origin", "http://local.test")
		r.Header.Set("X-Wisp-Token", server.Token)
		w := httptest.NewRecorder()
		server.Handler().ServeHTTP(w, r)
		return w.Code
	}
	if post() != 400 {
		t.Fatal("lab off accepted")
	}
	server.GameContext = true
	if post() != 400 {
		t.Fatal("unapproved accepted")
	}
	request.SnapshotApproved = true
	snap.CapturedAt = now - 100
	if post() != 409 {
		t.Fatal("expired accepted")
	}
	snap.CapturedAt = now
	answer, err := server.answer(context.Background(), request)
	if err != nil || len(answer.Actions) != 1 || answer.Actions[0] != valid {
		t.Fatal(answer, err)
	}
	request.SnapshotApproved = false
	answer, err = server.answer(context.Background(), request)
	if err != nil || len(answer.Actions) != 0 {
		t.Fatal("unapproved remote action", answer, err)
	}
}
