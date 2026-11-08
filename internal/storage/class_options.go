package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"sort"
	"strings"
)

// ClassOptions selects a small source-labelled training reference from an already
// cached class detail. It does not inspect a spellbook or infer an optimal rotation.
func (s *Store) ClassOptions(ctx context.Context, version, class string, level int) (Graph, error) {
	g := Graph{GameVersion: version, Entities: []Entity{}, Edges: []Edge{}, Sources: []Source{}}
	slug := strings.ToLower(class)
	if s.ExcludeForeverDB || version != "forever" || level < 1 || level > 60 || !map[string]bool{"warrior": true, "paladin": true, "hunter": true, "rogue": true, "priest": true, "shaman": true, "mage": true, "warlock": true, "druid": true}[slug] {
		return g, nil
	}
	var raw []byte
	var src Source
	var build string
	err := s.Pool.QueryRow(ctx, `SELECT d.payload,st.value,s.id,s.title,s.uri,s.revision,s.license,s.validation_status,s.provenance,s.expires_at
 FROM api_ingest_state st JOIN api_ingest_documents d ON d.namespace='foreverdb-'||st.value AND d.path='/api/v1/classes/'||$1 AND d.expires_at>now()
 JOIN entities e ON e.id='foreverdb:'||st.value||':classes:'||coalesce(d.payload#>>'{data,id}',d.payload#>>'{data,slug}',$1) AND e.game_version='forever'
 JOIN sources s ON s.id=e.source_id
 WHERE st.key='foreverdb:build' AND s.provenance->>'effective_version'=st.value
 AND s.validation_status IN ('source_checked','verified') AND (s.expires_at IS NULL OR s.expires_at>now())`, slug).Scan(&raw, &build, &src.ID, &src.Title, &src.URI, &src.Revision, &src.License, &src.ValidationStatus, &src.Provenance, &src.ExpiresAt)
	if err == pgx.ErrNoRows {
		return g, nil
	}
	if err != nil {
		return g, err
	}
	rows, err := trainingOptions(raw, level)
	if err != nil {
		return g, err
	}
	for _, r := range rows {
		encoded, _ := json.Marshal(r)
		g.Entities = append(g.Entities, Entity{ID: fmt.Sprintf("class-option:%s:%s:%d", build, slug, r.SpellID), GameVersion: version, Kind: "ability", Name: r.Name, Description: "Conservative rank-labelled trainer subset; unranked effects omitted. Source-listed reference, not proof this character has learned it or that it is an optimal rotation. " + string(encoded), SourceID: src.ID})
		g.Anchors = append(g.Anchors, g.Entities[len(g.Entities)-1].ID)
	}
	if len(rows) > 0 {
		g.Sources = append(g.Sources, src)
	}
	return g, nil
}

type trainingOption struct {
	Name    string                     `json:"name"`
	Rank    string                     `json:"rank"`
	Level   int                        `json:"trainer_level"`
	SpellID int                        `json:"spell_id"`
	Timing  string                     `json:"timing"`
	Origin  map[string]json.RawMessage `json:"origin"`
}

func trainingOptions(raw []byte, level int) ([]trainingOption, error) {
	var doc struct {
		Data struct {
			Abilities []json.RawMessage `json:"abilities"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	current, next := []trainingOption{}, []trainingOption{}
	nextLevel := 61
	for _, row := range doc.Data.Abilities {
		var a struct {
			Name    string `json:"name"`
			Learned string `json:"learned"`
			Ranks   []struct {
				Rank    string `json:"rank"`
				Level   int    `json:"level"`
				SpellID int    `json:"spellId"`
			} `json:"ranks"`
		}
		if json.Unmarshal(row, &a) != nil || a.Learned != "trainer" || a.Name == "" {
			continue
		}
		var fields map[string]json.RawMessage
		_ = json.Unmarshal(row, &fields)
		// Keep origin/availability labels; known historical or unavailable rows are
		// excluded rather than treating their presence in a client dump as availability.
		var unavailable bool
		for _, key := range []string{"seasonOfDiscoveryLeftover", "betaEvent"} {
			var v bool
			_ = json.Unmarshal(fields[key], &v)
			unavailable = unavailable || v
		}
		if v, ok := fields["announced"]; ok && string(v) == "false" {
			unavailable = true
		}
		lower := strings.ToLower(a.Name)
		if unavailable || strings.Contains(lower, "[old]") || strings.Contains(lower, "internal") {
			continue
		}
		origin := map[string]json.RawMessage{}
		for _, key := range []string{"learned", "source", "levelSource", "knownInForever", "newInForever", "liveClient", "speculation", "speculationReasons"} {
			if v, ok := fields[key]; ok {
				origin[key] = v
			}
		}
		var best *trainingOption
		for _, rank := range a.Ranks {
			if rank.Level < 1 || rank.Level > 60 || rank.SpellID < 1 || strings.TrimSpace(rank.Rank) == "" {
				continue
			}
			option := trainingOption{Name: a.Name, Rank: rank.Rank, Level: rank.Level, SpellID: rank.SpellID, Origin: origin}
			if rank.Level <= level {
				option.Timing = "listed_at_or_below_current_level"
				if best == nil || option.Level > best.Level {
					copy := option
					best = &copy
				}
			} else {
				option.Timing = "next_listed_training_level"
				if rank.Level < nextLevel {
					nextLevel = rank.Level
					next = nil
				}
				if rank.Level == nextLevel {
					next = append(next, option)
				}
			}
		}
		if best != nil {
			current = append(current, *best)
		}
	}
	order := func(rows []trainingOption) {
		sort.Slice(rows, func(i, j int) bool {
			if rows[i].Level != rows[j].Level {
				return rows[i].Level < rows[j].Level
			}
			return rows[i].Name < rows[j].Name
		})
	}
	order(current)
	order(next)
	// Reserve space for the next trainer visit even for higher-level characters.
	if len(current) > 12 {
		current = current[:12]
	}
	if len(next) > 4 {
		next = next[:4]
	}
	return append(current, next...), nil
}
