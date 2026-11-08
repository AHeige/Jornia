package storage

import (
	"context"
	"fmt"
)

// QuestGuidance serves only exact-build, explicitly verified follow-up claims.
// It is independent of provider imports and never promotes source_checked data.
func (s *Store) QuestGuidance(ctx context.Context, version, build string, ids []int) (Graph, error) {
	g := Graph{GameVersion: version, Entities: []Entity{}, Edges: []Edge{}, Sources: []Source{}}
	if len(ids) == 0 || len(ids) > 32 || build == "" {
		return g, nil
	}
	rows, err := s.Pool.Query(ctx, `SELECT f.quest_id,f.next_quest_id,f.next_quest_title,s.id,s.title,s.uri,s.revision,s.license,s.validation_status,s.provenance,s.expires_at
 FROM quest_followups f JOIN sources s ON s.id=f.source_id
 WHERE f.game_version=$1 AND f.client_build=$2 AND f.quest_id=ANY($3)
 AND s.validation_status='verified' AND s.id NOT LIKE 'foreverdb:%'
 AND s.provenance->>'effective_version'=$2
 AND (s.expires_at IS NULL OR s.expires_at>now())
 ORDER BY f.quest_id,f.next_quest_id,s.id LIMIT 16`, version, build, ids)
	if err != nil {
		return g, err
	}
	defer rows.Close()
	seen := map[string]bool{}
	sources := map[string]bool{}
	for rows.Next() {
		var from, to int
		var title string
		var src Source
		if err = rows.Scan(&from, &to, &title, &src.ID, &src.Title, &src.URI, &src.Revision, &src.License, &src.ValidationStatus, &src.Provenance, &src.ExpiresAt); err != nil {
			return g, err
		}
		a, b := fmt.Sprintf("guidance:quest:%d", from), fmt.Sprintf("guidance:quest:%d", to)
		if !seen[a] {
			g.Entities = append(g.Entities, Entity{ID: a, GameVersion: version, Kind: "quest", Name: fmt.Sprintf("Quest %d", from), SourceID: src.ID})
			g.Anchors = append(g.Anchors, a)
			seen[a] = true
		}
		if !seen[b] {
			g.Entities = append(g.Entities, Entity{ID: b, GameVersion: version, Kind: "quest", Name: title, Description: "Verified recorded follow-up; present eligibility and availability are not established.", SourceID: src.ID})
			seen[b] = true
		}
		g.Edges = append(g.Edges, Edge{ID: fmt.Sprintf("guidance:%d:next:%d:%s", from, to, src.ID), SubjectID: a, Predicate: "next_quest", ObjectID: b, SourceID: src.ID})
		if !sources[src.ID] {
			g.Sources = append(g.Sources, src)
			sources[src.ID] = true
		}
	}
	return g, rows.Err()
}
