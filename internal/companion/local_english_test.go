package companion

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestEnglishMemoryPromptsStayLocalAndScoped(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "player.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.DB.Close()
	server, err := NewServer(store, "http://127.0.0.1:1", "127.0.0.1:8787", fstest.MapFS{})
	if err != nil {
		t.Fatal(err)
	}
	p := packet()
	if _, err = store.Import(context.Background(), "english-import", p); err != nil {
		t.Fatal(err)
	}
	for _, question := range []string{"What was I doing last time?", "My journey"} {
		a, err := server.answer(context.Background(), Ask{CharacterID: "test-install:Player-1-abc", Question: question})
		if err != nil || a.Route != "local_memory" || a.Status != "answered" || !strings.Contains(a.Text, "Your ") || !strings.Contains(a.Text, p.Events[0].Note) || len(a.Sources) == 0 {
			t.Fatalf("invalid local answer: %+v %v", a, err)
		}
		empty, err := server.answer(context.Background(), Ask{CharacterID: "not-this-character", Question: question})
		if err != nil || empty.Status != "unknown" || strings.Contains(empty.Text, p.Events[0].Note) {
			t.Fatalf("scope leak: %+v %v", empty, err)
		}
	}
	if _, err = store.Import(context.Background(), "english-demo", DemoExport()); err != nil {
		t.Fatal(err)
	}
	a, err := server.answer(context.Background(), Ask{CharacterID: DemoCharacterID, Question: "My journey"})
	if err != nil || !strings.HasPrefix(a.Text, "Demo data") || a.Sources[0].ValidationStatus != "synthetic_demo" {
		t.Fatalf("unlabelled demo: %+v %v", a, err)
	}
}
