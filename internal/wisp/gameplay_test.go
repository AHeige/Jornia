package wisp

import (
	"context"
	"testing"
	"time"
	"wisp/internal/ai"
	"wisp/internal/storage"
)

func TestGamePlanUsesOnlyFreshObservedQuests(t *testing.T) {
	s := &GameSnapshot{CapturedAt: time.Now().Unix(), Level: 16, Zone: "灰谷", ClientBuild: "70790", Quests: []GameQuest{{ID: 2, Title: "À faire", Waypoint: true}, {ID: 1, Title: "Terminé", Complete: true, Waypoint: true}, {ID: 3}}}
	calls := 0
	plan, _, err := BuildGamePlan(context.Background(), s, func(context.Context) (storage.Graph, error) { calls++; return storage.Graph{}, nil })
	if err != nil || calls != 1 || len(plan.Trace) != 4 || plan.Candidates[0].QuestID != 1 {
		t.Fatal(plan, err, calls)
	}
	if a := plan.Action("quest:1", s, []string{"state:quest:1"}); a == nil || a.QuestID != 1 {
		t.Fatal(a)
	}
	for _, c := range []struct {
		id string
		e  []string
	}{{"quest:99", []string{"state:quest:99"}}, {"quest:1", []string{"state:quest:2"}}, {"quest:3", []string{"state:quest:3"}}} {
		if plan.Action(c.id, s, c.e) != nil {
			t.Fatal("forged/unsupported action", c)
		}
	}
	s.CapturedAt -= 100
	if ValidateSnapshot(s) == nil || plan.Action("quest:1", s, []string{"state:quest:1"}) != nil {
		t.Fatal("stale snapshot accepted")
	}
	s.CapturedAt = time.Now().Unix()
	s.Quests = append(s.Quests, s.Quests[0])
	if ValidateSnapshot(s) == nil {
		t.Fatal("duplicate quest accepted")
	}
}

type gameModel struct{ action, text, status string }

func (m gameModel) Generate(_ context.Context, _ string, _ string) (ai.Completion, error) {
	return ai.Completion{Model: "test", Result: ai.Result{Status: m.status, Text: m.text, ActionID: m.action, EvidenceIDs: func() []string {
		if m.status == "answered" {
			return []string{"state:quest:123"}
		}
		return nil
	}()}}, nil
}
func TestModelCannotInventActionAndUnknownKeepsLanguage(t *testing.T) {
	snap := &GameSnapshot{CapturedAt: time.Now().Unix(), Level: 16, Quests: []GameQuest{{ID: 123, Waypoint: true}}}
	for _, id := range []string{"quest:123", "quest:999", "RunScript()"} {
		svc := Service{Store: noWorld{}, Generator: gameModel{action: id, text: "Commencez par cette quête.", status: "answered"}}
		a, err := svc.Ask(context.Background(), Request{Question: "Que faire maintenant ?", Mode: "generate", GameVersion: "forever", Snapshot: snap})
		if err != nil {
			t.Fatal(err)
		}
		if id == "quest:123" {
			if a.Status != "answered" || len(a.Actions) != 1 || len(a.Sources) != 1 {
				t.Fatal(a)
			}
		} else if a.Status != "unknown" || len(a.Actions) != 0 {
			t.Fatal("invented action", a)
		}
	}
	svc := Service{Store: noWorld{}, Generator: gameModel{text: "この情報はまだありません。", status: "unknown"}}
	a, err := svc.Ask(context.Background(), Request{Question: "どこ？", Mode: "generate", GameVersion: "forever"})
	if err != nil || a.Text != "この情報はまだありません。" {
		t.Fatal(a, err)
	}
}
