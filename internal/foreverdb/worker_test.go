package foreverdb

import (
	"encoding/json"
	"net/url"
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

func TestStarterZonePriorityFlowsThroughPagesAndDetails(t *testing.T) {
	for _, zone := range StarterZones {
		path := "/api/v1/quests?" + url.Values{"zone": {zone}, "limit": {"100"}}.Encode()
		if !AllowedPath(path) {
			t.Fatal(path)
		}
		d := Document{Data: []any{map[string]any{"id": float64(376)}}}
		d.Meta.Total = 205
		links := Links(path, d)
		if len(links) != 3 {
			t.Fatal(zone, links)
		}
		for next, priority := range links {
			want := -9
			if starterCatalog(next) {
				want = -10
			}
			if priority != want {
				t.Fatal(next, priority, want)
			}
		}
	}
	ordinary := Links("/api/v1/quests?zone=Ashenvale&limit=100", Document{Data: []any{map[string]any{"id": float64(376)}}})
	if ordinary["/api/v1/quests/376"] != 12 {
		t.Fatal(ordinary)
	}
}
