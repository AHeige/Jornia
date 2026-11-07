package foreverdb

import (
	"encoding/json"
	"testing"
)

func TestIndependentPagesPreserveFilterAndBoundDetails(t *testing.T) {
	var d Document
	if err := json.Unmarshal([]byte(`{"data":[{"id":376}],"meta":{"total":305,"build":"1.60.1.70790"}}`), &d); err != nil {
		t.Fatal(err)
	}
	links := Links("/api/v1/quests?zone=Tirisfal&limit=100", d)
	for _, path := range []string{"/api/v1/quests?limit=100&offset=100&zone=Tirisfal", "/api/v1/quests?limit=100&offset=300&zone=Tirisfal", "/api/v1/quests/376"} {
		if _, ok := links[path]; !ok {
			t.Fatal("missing independent page/detail", path)
		}
	}
	if len(links) != 4 {
		t.Fatal(links)
	}
}

func TestRefreshBudgetDoesNotStarveLongTail(t *testing.T) {
	for path, hours := range map[string]int{"/api/v1/items?limit=100": 24, "/api/v1/classes/mage": 24, "/api/v1/professions/alchemy?limit=100&offset=100": 24, "/api/v1/quests/376": 168, "/api/v1/items/1": 504} {
		if got := int(refreshDelay(path).Hours()); got != hours {
			t.Fatal(path, got)
		}
	}
}
