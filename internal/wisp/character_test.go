package wisp

import (
	"context"
	"fmt"
	"github.com/AHeige/Jornia/internal/storage"
	"strings"
	"testing"
	"time"
)

func TestDatedCharacterOnlyEvidenceAndBounds(t *testing.T) {
	now := time.Now().Unix()
	c := &CharacterContext{EvidenceID: "character:profile", Level: 28, Race: "Night Elf", Class: "Warrior", FetchedAt: now, LastLogin: now - 3600, ExpiresAt: now + 86400, Items: []CharacterItem{{ID: 4705, Name: "Lambent Scale Pauldrons", Slot: "SHOULDER", Quality: "UNCOMMON"}}}
	if err := ValidateCharacter(c); err != nil {
		t.Fatal(err)
	}
	m := &journalModel{evidence: "character:profile"}
	s := Service{Store: noWorld{}, Generator: m}
	if _, err := s.Ask(context.Background(), Request{Question: "My gear?", GameVersion: "retail", Mode: "generate", Character: c}); err == nil {
		t.Fatal("Forever profile used for another game version")
	}
	a, err := s.Ask(context.Background(), Request{Question: "What did my saved profile equip?", GameVersion: "forever", Mode: "generate", Character: c})
	if err != nil || a.Status != "answered" || len(a.Sources) != 1 || a.Sources[0].ExpiresAt == nil || !strings.Contains(m.knowledge, `"character_is_live":false`) {
		t.Fatal(a, err, m.knowledge)
	}
	m.evidence = "memory:1"
	a, err = s.Ask(context.Background(), Request{Question: "My gear?", Mode: "generate", Character: c})
	if err != nil || a.Status != "unknown" {
		t.Fatal("fabricated citation accepted", a, err)
	}
	for _, mutate := range []func(*CharacterContext){func(x *CharacterContext) { x.ExpiresAt = now - 1 }, func(x *CharacterContext) { x.LastLogin = now + 1000 }, func(x *CharacterContext) { x.EvidenceID = "Player-private" }, func(x *CharacterContext) { x.Items = append(x.Items, x.Items[0]) }} {
		x := *c
		x.Items = append([]CharacterItem{}, c.Items...)
		mutate(&x)
		if ValidateCharacter(&x) == nil {
			t.Fatal("invalid snapshot accepted", x)
		}
	}
}

type wideWorld struct{}

func (wideWorld) Graph(context.Context, string, bool) (storage.Graph, error) {
	g := storage.Graph{GameVersion: "forever"}
	for i := 0; i < 100; i++ {
		g.Entities = append(g.Entities, storage.Entity{ID: fmt.Sprintf("wide-%d", i), Name: "Weapon", Description: strings.Repeat("wide world data ", 100)})
	}
	return g, nil
}
func TestPersonalContextSurvivesBroadWorldMatches(t *testing.T) {
	now := time.Now().Unix()
	c := &CharacterContext{EvidenceID: "character:profile", Level: 16, Class: "Warlock", FetchedAt: now, LastLogin: now - 60, ExpiresAt: now + 3600}
	model := &journalModel{evidence: "character:profile"}
	s := Service{Store: wideWorld{}, Generator: model}
	a, err := s.Ask(context.Background(), Request{Question: "Which weapon is in my profile?", Mode: "generate", Character: c})
	if err != nil || a.Status != "answered" || len(model.knowledge) > 40000 {
		t.Fatal(a, err, len(model.knowledge))
	}
	model.evidence = "entity:wide-99"
	a, err = s.Ask(context.Background(), Request{Question: "Which weapon is in my profile?", Mode: "generate", Character: c})
	if err != nil || a.Status != "unknown" {
		t.Fatal("omitted world fact citation accepted", a, err)
	}
}
