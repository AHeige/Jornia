package httpapi

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestPublicMediaAllowlist(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "media"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{"media/character-preview.jpg", "media/expedition.jpg", "media/wisp.png", "download.css", "media/forest.jpg", "media/addon-preview.jpg", "media/journey-preview.jpg", "media/ingame-chat-preview.jpg", "media/ingame-journey-preview.jpg"} {
		if err := os.WriteFile(filepath.Join(dir, file), []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "private.txt"), []byte("not public"), 0600); err != nil {
		t.Fatal(err)
	}
	handler := Public(nil, nil, dir)
	for _, test := range []struct {
		path   string
		status int
	}{
		{"/media/character-preview.jpg", 200}, {"/media/expedition.jpg", 200}, {"/media/wisp.png", 200}, {"/media/wisp-original.png", 404}, {"/media/ingame-chat-preview.jpg", 200}, {"/media/ingame-journey-preview.jpg", 200}, {"/download.css", 200}, {"/media/forest.jpg", 200}, {"/media/addon-preview.jpg", 200}, {"/media/journey-preview.jpg", 200},
		{"/private.txt", 404}, {"/media/private.txt", 404}, {"/media/../private.txt", 404}, {"/media/forest-original.png", 404},
	} {
		request := httptest.NewRequest("GET", test.path, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != test.status {
			t.Errorf("%s got %d, want %d", test.path, response.Code, test.status)
		}
	}
}
