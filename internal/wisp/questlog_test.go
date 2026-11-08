package wisp

import (
	"context"
	"encoding/json"
	"github.com/AHeige/Jornia/internal/ai"
	"github.com/AHeige/Jornia/internal/storage"
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
