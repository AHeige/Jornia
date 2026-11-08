package wisp

import (
	"context"
	"github.com/AHeige/Jornia/internal/ai"
	"github.com/AHeige/Jornia/internal/storage"
	"strings"
	"testing"
)

type noWorld struct{}

func (noWorld) Graph(context.Context, string, bool) (storage.Graph, error) {
	return storage.Graph{}, nil
}

type journalModel struct {
	knowledge string
	evidence  string
}

func (m *journalModel) Generate(_ context.Context, _ string, k string) (ai.Completion, error) {
	m.knowledge = k
	return ai.Completion{Model: "test", Result: ai.Result{Status: "answered", Text: "Senast sparade nivå är 16.", EvidenceIDs: []string{m.evidence}}}, nil
}
func TestJournalOnlyAIAndEvidence(t *testing.T) {
	event := Observation{ID: "memory:1", Kind: "level_up", ObservedAt: 1700000000, Zone: "Ashenvale", Level: 16}
	model := &journalModel{evidence: "memory:1"}
	s := Service{Store: noWorld{}, Generator: model}
	answer, err := s.Ask(context.Background(), Request{Question: "Vad minns du av resan?", GameVersion: "forever", Mode: "generate", Journal: []Observation{event}, SyntheticJournal: true})
	if err != nil || answer.Status != "answered" || answer.Route != "berget" || len(answer.Sources) != 1 || !strings.Contains(model.knowledge, `"journal_is_live":false`) {
		t.Fatalf("%+v %v %s", answer, err, model.knowledge)
	}
	model.evidence = "memory:2"
	answer, err = s.Ask(context.Background(), Request{Question: "Vad minns du?", Mode: "generate", Journal: []Observation{event}})
	if err != nil || answer.Status != "unknown" {
		t.Fatalf("Invented evidence accepted: %+v %v", answer, err)
	}
	event.ID = "Player-sensitive"
	if ValidateJournal([]Observation{event}) == nil {
		t.Fatal("identity accepted")
	}
}
