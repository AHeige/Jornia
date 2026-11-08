package companion

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestNativeFirstQuestionRegistersWithoutSavingAndReusesJournalIdentity(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "player.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.DB.Close()
	server, err := NewServer(store, "http://127.0.0.1:1", "local.test", fstest.MapFS{})
	if err != nil {
		t.Fatal(err)
	}
	payload := `{"mutation_id":"register-first","installation_id":"test-install","character":{"id":"Player-1-abc","name":"TestPlayer","realm":"Testrealm"}}`
	post := func() *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "http://local.test/local/native-character", strings.NewReader(payload))
		r.Header.Set("Origin", "http://local.test")
		r.Header.Set("X-Wisp-Token", server.Token)
		w := httptest.NewRecorder()
		server.Handler().ServeHTTP(w, r)
		return w
	}
	if w := post(); w.Code != 400 {
		t.Fatal("native context enabled implicitly", w.Code)
	}
	server.GameContext = true
	for i := 0; i < 2; i++ {
		w := post()
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		var receipt Receipt
		if err = json.Unmarshal(w.Body.Bytes(), &receipt); err != nil {
			t.Fatal(err)
		}
	}
	var n int
	store.DB.QueryRow("SELECT count(*) FROM events").Scan(&n)
	if n != 0 {
		t.Fatal("registration invented history")
	}
	data := packet()
	data.Character.ID = "Player-1-abc"
	data.InstallationID = "test-install"
	if _, err = store.Import(context.Background(), "normal-later-import", data); err != nil {
		t.Fatal(err)
	}
	store.DB.QueryRow("SELECT count(*) FROM characters").Scan(&n)
	if n != 1 {
		t.Fatal("later journal split the character", n)
	}
}
