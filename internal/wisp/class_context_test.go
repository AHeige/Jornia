package wisp

import (
	"strings"
	"testing"
)

func TestClassQuestionsUseObservedClassRatherThanQuestAnchors(t *testing.T) {
	for _, q := range []string{"Vilken spell bör jag använda vid levling?", "Which abilities can I learn?", "法术怎么选？", "Quels sorts utiliser ?"} {
		r := Request{Question: q, Snapshot: &GameSnapshot{ClassID: 8, Quests: []GameQuest{{ID: 376, Title: "The Damned"}}}}
		got := snapshotQuestion(r)
		if !classKnowledgeRequest(r) || !strings.Contains(got, "Mage") || strings.Contains(got, "The Damned") {
			t.Fatal(q, got)
		}
	}
	r := Request{Question: "Where do I turn this quest in?", Snapshot: &GameSnapshot{ClassID: 8, Quests: []GameQuest{{ID: 376, Title: "The Damned"}}}}
	if classKnowledgeRequest(r) || !strings.Contains(snapshotQuestion(r), "The Damned") {
		t.Fatal("quest context lost")
	}
	r.Question = "Which spells?"
	r.Snapshot.ClassID = 0
	if classKnowledgeRequest(r) {
		t.Fatal("unknown class inferred")
	}
}
