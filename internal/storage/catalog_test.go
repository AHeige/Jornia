package storage

import "testing"

func TestCatalogDoesNotConfuseRecordCountsWithReadiness(t *testing.T) {
	cats := CategoryCatalog(map[string]int{"item": 100000, "item_set": 5, "class": 9, "quest": 7000}, map[string]int{"item": 10, "quest": 3})
	if len(cats) != 25 {
		t.Fatal(len(cats))
	}
	seen := map[string]bool{}
	for _, c := range cats {
		if seen[c.ID] || len(c.Required) == 0 || len(c.NextSources) == 0 || c.FieldCoverage != "not_measured" {
			t.Fatal(c)
		}
		seen[c.ID] = true
		if c.Status == "complete" || c.Status == "ready" {
			t.Fatal("counts falsely imply complete facts", c)
		}
		switch c.ID {
		case "gear":
			if c.Records != 100005 || c.DetailDocuments != 10 || c.Status != "partial" {
				t.Fatal(c)
			}
		case "quests":
			if c.DetailDocuments != 3 {
				t.Fatal(c)
			}
		case "talents", "lore", "experiences":
			if c.Status != "missing" {
				t.Fatal(c)
			}
		case "builds", "gear_combinations", "quest_routes":
			if c.Status != "not_implemented" {
				t.Fatal(c)
			}
		}
	}
}
