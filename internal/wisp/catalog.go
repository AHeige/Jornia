package wisp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"wisp/internal/storage"
)

func (s *Service) catalogAnswer(ctx context.Context, r Request) (Answer, bool, error) {
	q := normalize(r.Question)
	swedish := strings.HasPrefix(q, "vad ") || strings.HasPrefix(q, "vilken ")
	summary := q == "what do you know" || q == "what can you help me with" || q == "what knowledge do you have" || q == "vad vet du" || q == "vad kan du hjälpa mig med" || q == "vilken kunskap har du"
	aliases := map[string]string{"quests": "quests", "quest": "quests", "uppdrag": "quests", "items": "items", "föremål": "items", "gear": "gear", "utrustning": "gear", "zones": "zones", "zoner": "zones", "npcs": "npcs", "npc": "npcs", "classes": "classes", "klasser": "classes", "specs": "specs", "specialiseringar": "specs", "abilities": "abilities", "spells": "abilities", "talents": "talents", "talanger": "talents", "talent trees": "talents", "professions": "professions", "yrken": "professions", "recipes": "recipes", "recept": "recipes", "gathering": "gathering", "dungeons": "dungeons", "raids": "raids", "bosses": "bosses", "bossar": "bosses", "travel": "travel", "resor": "travel", "reputation": "reputation", "pvp": "pvp", "events": "events", "collectibles": "collectibles", "lore": "lore", "builds": "builds", "gear combinations": "gear_combinations", "quest routes": "quest_routes", "uppdragsrutter": "quest_routes", "experiences": "experiences", "upplevelser": "experiences"}
	category := ""
	for _, prefix := range []string{"what do you know about ", "vad vet du om "} {
		if strings.HasPrefix(q, prefix) {
			category = aliases[strings.TrimSpace(strings.TrimPrefix(q, prefix))]
		}
	}
	if !summary && category == "" {
		return Answer{}, false, nil
	}
	reporter, ok := s.Store.(interface {
		Coverage(context.Context, string) (map[string]any, error)
	})
	if !ok || r.GameVersion != "forever" && r.GameVersion != "" {
		return Answer{}, false, nil
	}
	start := time.Now()
	report, err := reporter.Coverage(ctx, "forever")
	if err != nil {
		return Answer{}, true, err
	}
	categories, ok := report["categories"].([]storage.KnowledgeCategory)
	if !ok {
		return Answer{}, false, nil
	}
	text := ""
	if summary {
		available, missing := []string{}, []string{}
		for _, c := range categories {
			if c.Status == "partial" {
				available = append(available, c.Name)
			} else {
				missing = append(missing, c.Name)
			}
		}
		text = "Partial sourced coverage: " + strings.Join(available, ", ") + ".\nNot yet available as dedicated categories or working planners: " + strings.Join(missing, ", ") + ".\nAsk about a named quest, item, NPC or place. Record counts do not establish complete facts."
		if swedish {
			text = "Delvis källbelagt underlag: " + strings.Join(available, ", ") + ".\nSaknas ännu som egna kategorier eller fungerande planering: " + strings.Join(missing, ", ") + ".\nFråga om ett namngivet uppdrag, föremål, NPC eller plats. Antal poster innebär inte fullständiga fakta."
		}
	} else {
		for _, c := range categories {
			if c.ID != category {
				continue
			}
			if c.Derived {
				text = c.Name + ": this planner is not implemented. Catalog entries alone do not establish a valid recommendation."
				if swedish {
					text = c.Name + ": denna planering är inte implementerad. Katalogposter räcker inte för en giltig rekommendation."
				}
			} else if c.Records == 0 {
				text = c.Name + ": no dedicated sourced records are available in this category yet."
				if swedish {
					text = c.Name + ": det finns ännu inga egna källbelagda poster i denna kategori."
				}
			} else {
				text = fmt.Sprintf("%s: %d records and %d active API detail documents. Coverage is partial. Individual fields and relationships have not been measured for completeness; an index entry is not a complete guide.", c.Name, c.Records, c.DetailDocuments)
				if swedish {
					text = fmt.Sprintf("%s: %d poster och %d aktiva API-detaljdokument. Täckningen är ofullständig. Fullständigheten för enskilda fält och relationer är inte uppmätt; en indexpost är inte en komplett guide.", c.Name, c.Records, c.DetailDocuments)
				}
			}
			break
		}
	}
	evidenceKey := category
	if summary {
		evidenceKey = "summary"
	}
	zero := 0.0
	return Answer{Status: "answered", Text: text, GameVersion: "forever", Route: "knowledge_catalog", Sources: []storage.Source{{ID: "wisp:knowledge-catalog", Title: "Wisp knowledge catalog snapshot", URI: "wisp:knowledge-catalog", Revision: time.Now().UTC().Format(time.RFC3339), License: "Wisp operational metadata", ValidationStatus: "operational"}}, EvidenceIDs: []string{"catalog:" + evidenceKey}, Warnings: []string{"Category counts overlap. Missing categories are not filled from model memory. This catalog excludes private player journals."}, EstimatedCostEUR: &zero, LatencyMS: time.Since(start).Milliseconds()}, true, nil
}
