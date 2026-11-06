package wisp

import (
	"context"
	"strings"
	"testing"
	"wisp/internal/storage"
)

type catalogWorld struct{ noWorld }

func (catalogWorld) Coverage(context.Context, string) (map[string]any, error) {
	return map[string]any{"categories": storage.CategoryCatalog(map[string]int{"quest": 7, "item": 1000}, map[string]int{"quest": 2})}, nil
}
func TestCatalogQueriesExposeGapsWithoutInference(t *testing.T) {
	model := &journalModel{}
	s := Service{Store: catalogWorld{}, Generator: model}
	for _, q := range []string{"Vad kan du hjälpa mig med?", "What do you know?", "Vad vet du om quests?", "What do you know about items?", "Vad vet du om talanger?", "What do you know about lore?", "What do you know about quest routes?", "Vad vet du om upplevelser?"} {
		a, err := s.Ask(context.Background(), Request{Question: q, GameVersion: "forever", Mode: "generate"})
		if err != nil || a.Status != "answered" || a.Route != "knowledge_catalog" || a.Model != nil || len(a.Sources) != 1 || a.EstimatedCostEUR == nil || *a.EstimatedCostEUR != 0 {
			t.Fatal(q, a, err)
		}
		if q == "Vad vet du om quests?" && (!strings.Contains(a.Text, "7 poster") || !strings.Contains(a.Text, "2 aktiva") || !strings.Contains(a.Text, "ofullständig")) {
			t.Fatal(a)
		}
	}
	if model.knowledge != "" {
		t.Fatal("catalog spent a model call")
	}
	a, handled, err := s.catalogAnswer(context.Background(), Request{Question: "What do you know about Astranaar?", GameVersion: "forever"})
	if handled || err != nil {
		t.Fatal("named world entity intercepted", a, err)
	}
	_, handled, _ = s.catalogAnswer(context.Background(), Request{Question: "What do you know?", GameVersion: "retail"})
	if handled {
		t.Fatal("Forever catalog served for Retail")
	}
}
