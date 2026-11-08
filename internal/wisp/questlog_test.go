package wisp

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/AHeige/Jornia/internal/ai"
	"github.com/AHeige/Jornia/internal/storage"
	"strings"
	"testing"
	"time"
)

type forbiddenWorld struct{}

func (forbiddenWorld) Graph(context.Context, string, bool) (storage.Graph, error) {
	panic("questlog mode queried world database")
}
func (forbiddenWorld) Coverage(context.Context, string) (map[string]any, error) {
	panic("questlog mode queried catalog")
}

type questlogModel struct {
	t     *testing.T
	calls int
}

func (m *questlogModel) Generate(_ context.Context, _ string, body string) (ai.Completion, error) {
	m.calls++
	var context struct {
		QuestlogOnly bool             `json:"questlog_only"`
		Entities     []storage.Entity `json:"entities"`
		Snapshot     *GameSnapshot    `json:"game_snapshot"`
		Options      []Candidate      `json:"next_step_options"`
	}
	if err := json.Unmarshal([]byte(body), &context); err != nil {
		m.t.Fatal(err)
	}
	if !context.QuestlogOnly || len(context.Entities) != 0 || context.Snapshot.Level != 16 || len(context.Options) != 2 {
		m.t.Fatalf("wrong model context: %+v", context)
	}
	if context.Options[0].QuestID != 376 || context.Options[0].Remaining[0].Remaining == nil || *context.Options[0].Remaining[0].Remaining != 2 {
		m.t.Fatal("missing assessed progress", context.Options)
	}
	return ai.Completion{Model: "test", Result: ai.Result{Status: "answered", Text: "Collect two more test paws.", EvidenceIDs: []string{"state:quest:376"}}}, nil
}
func TestQuestlogModeNeedsNoWorldOrProvider(t *testing.T) {
	fulfilled, required := 4, 6
	snapshot := &GameSnapshot{CapturedAt: time.Now().Unix(), Level: 16, Zone: "Deathknell", ClientBuild: "70790", Quests: []GameQuest{{ID: 376, Title: "Observed quest", Objectives: []GameObjective{{Index: 1, Text: "Test paws", Fulfilled: &fulfilled, Required: &required}}}, {ID: 3901, Title: "Details unavailable"}}}
	model := &questlogModel{t: t}
	svc := Service{Store: forbiddenWorld{}, Generator: model, QuestLogOnly: true}
	answer, err := svc.Ask(context.Background(), Request{Question: "What do I do now?", Mode: "generate", GameVersion: "forever", Snapshot: snapshot})
	if err != nil || answer.Status != "answered" || model.calls != 1 || len(answer.Sources) != 1 || answer.Sources[0].ID != "state:quest:376" || len(answer.Actions) != 0 {
		t.Fatal(answer, err)
	}
	snapshot.CapturedAt -= 100
	if _, err = svc.Ask(context.Background(), Request{Question: "Next?", Mode: "generate", GameVersion: "forever", Snapshot: snapshot}); err == nil || model.calls != 1 {
		t.Fatal("stale state reached model")
	}
}

type guidanceStore struct{ err error }

func (g guidanceStore) Graph(context.Context, string, bool) (storage.Graph, error) {
	panic("broad world retrieval")
}
func (g guidanceStore) QuestGuidance(_ context.Context, version, build string, ids []int) (storage.Graph, error) {
	if g.err != nil {
		return storage.Graph{}, g.err
	}
	return storage.Graph{GameVersion: version, Anchors: []string{"guidance:quest:376"}, Entities: []storage.Entity{{ID: "guidance:quest:376", Kind: "quest", Name: "Quest 376", SourceID: "review"}, {ID: "guidance:quest:999", Kind: "quest", Name: "Synthetic Follow-up", SourceID: "review"}}, Edges: []storage.Edge{{ID: "next:376:999", SubjectID: "guidance:quest:376", Predicate: "next_quest", ObjectID: "guidance:quest:999", SourceID: "review"}}, Sources: []storage.Source{{ID: "review", ValidationStatus: "verified", Title: "Synthetic reviewed chain", URI: "https://example.test"}}}, nil
}

type guidanceModel struct {
	t        *testing.T
	degraded bool
}

func (m guidanceModel) Generate(_ context.Context, _ string, body string) (ai.Completion, error) {
	var value struct {
		Snapshot *GameSnapshot   `json:"game_snapshot"`
		Edges    []storage.Edge  `json:"edges"`
		Allowed  map[string]bool `json:"allowed_evidence_ids"`
	}
	if err := json.Unmarshal([]byte(body), &value); err != nil {
		m.t.Fatal(err)
	}
	if value.Snapshot.Quests[0].Description != "Go west of the chapel." {
		m.t.Fatal("observed directions lost")
	}
	result := ai.Result{Status: "answered", Text: "Go west of the chapel for the remaining paws.", EvidenceIDs: []string{"state:quest:376"}}
	if !m.degraded {
		if len(value.Edges) != 1 || !value.Allowed["edge:next:376:999"] {
			m.t.Fatal("verified continuation missing")
		}
		result.Text += " The recorded follow-up is Synthetic Follow-up; availability is not confirmed."
		result.EvidenceIDs = append(result.EvidenceIDs, "edge:next:376:999", "entity:guidance:quest:999")
	} else if len(value.Edges) != 0 {
		m.t.Fatal("unavailable store leaked evidence")
	}
	return ai.Completion{Model: "test", Result: result}, nil
}
func TestRichGuidanceUsesObservedDirectionsAndSurvivesMissingFollowups(t *testing.T) {
	for _, degraded := range []bool{false, true} {
		var failure error
		if degraded {
			failure = fmt.Errorf("storage unavailable")
		}
		svc := Service{Store: guidanceStore{err: failure}, Generator: guidanceModel{t: t, degraded: degraded}, QuestLogOnly: true, QuestGuidance: true}
		snap := &GameSnapshot{CapturedAt: time.Now().Unix(), Level: 2, ClientBuild: "70790", Quests: []GameQuest{{ID: 376, Title: "The Damned", Description: "Go west of the chapel."}}}
		answer, err := svc.Ask(context.Background(), Request{Question: "Where now and what follows?", Mode: "generate", GameVersion: "forever", Snapshot: snap})
		if err != nil || answer.Status != "answered" || len(answer.Actions) != 0 {
			t.Fatal(answer, err)
		}
		if degraded && !strings.Contains(strings.Join(answer.Warnings, " "), "lookup is unavailable") {
			t.Fatal("missing degraded guidance warning")
		}
	}
}

type identityWorld struct {
	forbiddenWorld
	ids []int
}

func (w *identityWorld) RetrieveQuestContext(_ context.Context, _ string, _ bool, _ string, ids []int) (storage.Graph, error) {
	w.ids = ids
	return storage.Graph{GameVersion: "forever"}, nil
}
func TestCurrentAndTurnedInQuestsReachIdentityRetrieval(t *testing.T) {
	store := &identityWorld{}
	service := Service{Store: store}
	now := time.Now().Unix()
	_, err := service.Ask(context.Background(), Request{GameVersion: "forever", Question: "Vad gör jag nu?", Snapshot: &GameSnapshot{CapturedAt: now, Level: 3, Quests: []GameQuest{{ID: 3901, Title: "Localized title"}}, CompletedQuests: []CompletedQuest{{ID: 376, ObservedAt: now, Confirmation: "turn_in_event"}}}})
	if err != nil || len(store.ids) != 2 || store.ids[0] != 3901 || store.ids[1] != 376 {
		t.Fatal(err, store.ids)
	}
}
