package companion

import (
	"context"
	"encoding/json"
	"errors"
	"time"
	"unicode/utf8"
	"wisp/internal/wisp"
)

type EquippedItem struct {
	Slot   int `json:"slot"`
	ItemID int `json:"item_id"`
}
type ProfessionState struct {
	Name    string `json:"name"`
	SkillID int    `json:"skill_id,omitempty"`
	Rank    int    `json:"rank,omitempty"`
	MaxRank int    `json:"max_rank,omitempty"`
}

// Saved state is a dated local checkpoint. Only a fresh per-question addon
// snapshot can be supplied to the backend as game_snapshot.
type LocalState struct {
	Schema               int               `json:"schema"`
	ObservedAt           int64             `json:"observed_at"`
	Revision             int64             `json:"revision"`
	SessionID            string            `json:"session_id"`
	Level                int               `json:"level"`
	Zone                 string            `json:"zone"`
	Subzone              string            `json:"subzone"`
	ClientBuild          string            `json:"client_build"`
	Interface            int               `json:"interface"`
	ClassID              int               `json:"class_id,omitempty"`
	RaceID               int               `json:"race_id,omitempty"`
	XP                   int64             `json:"xp,omitempty"`
	XPMax                int64             `json:"xp_max,omitempty"`
	Dead                 bool              `json:"dead"`
	Quests               []wisp.GameQuest  `json:"quests"`
	Equipment            []EquippedItem    `json:"equipment"`
	Professions          []ProfessionState `json:"professions"`
	Truncated            bool              `json:"truncated"`
	QuestsAvailable      bool              `json:"quests_available"`
	EquipmentAvailable   bool              `json:"equipment_available"`
	ProfessionsAvailable bool              `json:"professions_available"`
	DroppedEvents        int               `json:"dropped_events"`
}

func validateLocalState(s *LocalState) error {
	if s == nil {
		return nil
	}
	if s.Schema != 1 || s.ObservedAt < 946684800 || s.ObservedAt > time.Now().Add(5*time.Minute).Unix() || s.Revision < 1 || s.Interface != 16001 || s.Level < 1 || s.Level > 100 || len(s.Zone) > 160 || len(s.Subzone) > 160 || len(s.ClientBuild) > 30 || len(s.SessionID) > 256 || !utf8.ValidString(s.Zone+s.Subzone+s.SessionID) || len(s.Quests) > 50 || len(s.Equipment) > 19 || len(s.Professions) > 12 || s.XP < 0 || s.XPMax < 0 || s.DroppedEvents < 0 {
		return errors.New("invalid local state checkpoint")
	}
	seen := map[int]bool{}
	for _, q := range s.Quests {
		if seen[q.ID] || q.ID < 1 || q.ID > 2147483647 || len(q.Title) > 200 || !utf8.ValidString(q.Title) || validateObjectives(q.Objectives) != nil {
			return errors.New("invalid local quest state")
		}
		seen[q.ID] = true
	}
	seen = map[int]bool{}
	for _, i := range s.Equipment {
		if i.Slot < 1 || i.Slot > 19 || seen[i.Slot] || i.ItemID < 1 || i.ItemID > 2147483647 {
			return errors.New("invalid equipment state")
		}
		seen[i.Slot] = true
	}
	for _, p := range s.Professions {
		if len(p.Name) > 100 || !utf8.ValidString(p.Name) || p.SkillID < 0 || p.Rank < 0 || p.MaxRank < 0 || p.Rank > p.MaxRank {
			return errors.New("invalid profession state")
		}
	}
	return nil
}
func validateObjectives(rows []wisp.GameObjective) error {
	if len(rows) > 8 {
		return errors.New("too many objective observations")
	}
	for _, o := range rows {
		if o.Index < 0 || o.Index > 32 || len(o.Text) > 160 || len(o.Kind) > 30 || !utf8.ValidString(o.Text+o.Kind) || o.Fulfilled != nil && (*o.Fulfilled < 0 || *o.Fulfilled > 2147483647) || o.Required != nil && (*o.Required < 0 || *o.Required > 2147483647) {
			return errors.New("invalid objective observation")
		}
	}
	return nil
}
func (s *Store) SavedState(ctx context.Context, char string) (*LocalState, error) {
	var raw string
	err := s.DB.QueryRowContext(ctx, `SELECT body FROM character_checkpoints WHERE character_id=?`, char).Scan(&raw)
	if err != nil {
		return nil, err
	}
	var state LocalState
	if err = json.Unmarshal([]byte(raw), &state); err != nil {
		return nil, err
	}
	return &state, nil
}
