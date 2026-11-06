package companion

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"wisp/internal/wisp"
)

type ContextPreview struct {
	EventIDs  []string           `json:"event_ids"`
	Events    []wisp.Observation `json:"events"`
	Digest    string             `json:"digest"`
	Synthetic bool               `json:"synthetic"`
}

func (s *Store) ContextPreview(ctx context.Context, character string, ids []string) (ContextPreview, error) {
	out := ContextPreview{EventIDs: []string{}, Events: []wisp.Observation{}, Synthetic: character == DemoCharacterID}
	if character == "" {
		return out, nil
	}
	if ids == nil {
		events, err := s.Events(ctx, character)
		if err != nil {
			return out, err
		}
		ids = []string{}
		for _, e := range events {
			if e.Kind != "note" {
				ids = append(ids, e.ID)
			}
			if len(ids) == 8 {
				break
			}
		}
	}
	if len(ids) > 12 {
		return out, errors.New("too many excerpts")
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			return out, errors.New("duplicate excerpt")
		}
		seen[id] = true
		var raw string
		if err := s.DB.QueryRowContext(ctx, `SELECT body FROM events WHERE character_id=? AND id=?`, character, id).Scan(&raw); err != nil {
			return out, errors.New("journal excerpt unavailable")
		}
		var e Event
		if err := json.Unmarshal([]byte(raw), &e); err != nil {
			return out, err
		}
		if e.Kind == "note" {
			return out, errors.New("notes are not shared")
		}
		out.EventIDs = append(out.EventIDs, id)
		out.Events = append(out.Events, wisp.Observation{ID: fmt.Sprintf("memory:%d", len(out.Events)+1), Kind: e.Kind, ObservedAt: e.ObservedAt, Zone: e.Zone, Subzone: e.Subzone, Level: e.Level, QuestID: e.QuestID})
	}
	out.Digest = Fingerprint(struct {
		Character string
		IDs       []string
		Events    []wisp.Observation
		Synthetic bool
	}{character, out.EventIDs, out.Events, out.Synthetic})
	return out, nil
}
