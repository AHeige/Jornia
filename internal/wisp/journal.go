package wisp

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// Observation is an explicitly selected journal excerpt, never a live snapshot.
// IDs are request-local references; character identity remains on the desktop.
type Observation struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	ObservedAt int64  `json:"observed_at"`
	Zone       string `json:"zone"`
	Subzone    string `json:"subzone,omitempty"`
	Level      int    `json:"level"`
	QuestID    int    `json:"quest_id,omitempty"`
}

func ValidateJournal(events []Observation) error {
	if len(events) > 12 {
		return errors.New("at most 12 journal excerpts")
	}
	for i, e := range events {
		if e.ID != fmt.Sprintf("memory:%d", i+1) || e.ObservedAt <= 0 || e.Level < 1 || e.Level > 100 || e.QuestID < 0 || len(e.Zone) > 160 || len(e.Subzone) > 160 || !utf8.ValidString(e.Zone+e.Subzone) {
			return errors.New("invalid journal excerpt")
		}
		switch e.Kind {
		case "snapshot", "zone", "level_up", "quest_completed", "session_start", "session_end":
		default:
			return errors.New("unsupported journal event")
		}
	}
	return nil
}

func retrievalQuestion(r Request) string {
	q := r.Question
	// Add the latest saved location for context-dependent questions. It remains history.
	if len(r.Journal) > 0 {
		q += " " + strings.TrimSpace(r.Journal[0].Zone)
	}
	return q
}
