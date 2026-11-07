package foreverdb

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestProjectionKeepsClassicObservationAndTypedChainSeparate(t *testing.T) {
	var d Document
	json.Unmarshal([]byte(`{"data":{"id":376,"name":"The Damned","knownInForever":true,"source":"forever","ends":{"id":1661,"name":"Novice Elreth","type":"npc","source":"seen in WoW Forever","players":5,"location":{"uiMapId":1420,"x":30.9,"y":66.1}},"chain":{"next":[{"questId":380,"name":"Next quest","relation":"Complete first","source":"Classic data (CMaNGOS)"}]}},"meta":{"build":"1.60.1.70790","client":"dev-test"}}`), &d)
	b := Normalize("/api/v1/quests/376", d, time.Now())
	if len(b.Entities) != 3 || len(b.Edges) != 2 || b.Source.URI != "https://foreverdb.net" || b.Source.ValidationStatus != "source_checked" {
		t.Fatal(b)
	}
	joined := ""
	for _, e := range b.Entities {
		joined += e.Description
		if !strings.HasPrefix(e.ID, "foreverdb:1.60.1.70790:") {
			t.Fatal(e)
		}
	}
	for _, word := range []string{"Classic data", "seen in WoW Forever", "30.9", "Complete first", "Alternatives are not all required"} {
		if !strings.Contains(joined, word) {
			t.Fatal(word)
		}
	}
	for _, edge := range b.Edges {
		if strings.Contains(edge.Predicate, "requires") {
			t.Fatal("upgraded relation", edge)
		}
	}
}
func TestRecipeIDsAndNoMediaProjection(t *testing.T) {
	var d Document
	json.Unmarshal([]byte(`{"data":{"slug":"alchemy","name":"Alchemy","recipes":[{"spellId":1,"name":"First","icon":"secret-media"},{"spellId":2,"name":"Second"}]},"meta":{"build":"1.60.1.70790"}}`), &d)
	b := Normalize("/api/v1/professions/alchemy?limit=100", d, time.Now())
	if len(b.Entities) != 3 || b.Entities[1].ID == b.Entities[2].ID {
		t.Fatal(b)
	}
	for _, e := range b.Entities {
		if strings.Contains(e.Description, "secret-media") {
			t.Fatal(e)
		}
	}
}
