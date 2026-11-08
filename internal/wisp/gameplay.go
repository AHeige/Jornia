package wisp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/AHeige/Jornia/internal/harness"
	"github.com/AHeige/Jornia/internal/storage"
	"strings"
	"time"
	"unicode/utf8"
)

type GameObjective struct {
	Index     int    `json:"index,omitempty"`
	Text      string `json:"text,omitempty"`
	Kind      string `json:"kind,omitempty"`
	Fulfilled *int   `json:"fulfilled,omitempty"`
	Required  *int   `json:"required,omitempty"`
	Finished  bool   `json:"finished"`
}
type GameQuest struct {
	Description   string          `json:"description,omitempty"`
	ObjectiveText string          `json:"objective_text,omitempty"`
	WaypointText  string          `json:"waypoint_text,omitempty"`
	TextTruncated bool            `json:"text_truncated,omitempty"`
	Objectives    []GameObjective `json:"objectives,omitempty"`
	ID            int             `json:"id"`
	Title         string          `json:"title,omitempty"`
	Complete      bool            `json:"complete"`
	Waypoint      bool            `json:"waypoint"`
}

// A per-question, explicitly approved observation. Absence is unknown, not an empty quest log.
type GameItem struct {
	Slot   int `json:"slot"`
	ItemID int `json:"item_id"`
}
type GameProfession struct {
	SkillID int `json:"skill_id"`
	Rank    int `json:"rank"`
	MaxRank int `json:"max_rank"`
}
type CompletedQuest struct {
	ID             int      `json:"id"`
	Title          string   `json:"title"`
	ObservedAt     int64    `json:"observed_at"`
	Confirmation   string   `json:"confirmation"`
	ObjectiveTexts []string `json:"objective_texts"`
}
type GameSnapshot struct {
	CompletedQuests []CompletedQuest `json:"completed_quests,omitempty"`
	ClassID         int              `json:"class_id,omitempty"`
	RaceID          int              `json:"race_id,omitempty"`
	Equipment       []GameItem       `json:"equipment,omitempty"`
	Professions     []GameProfession `json:"professions,omitempty"`
	Missing         []string         `json:"missing,omitempty"`
	CapturedAt      int64            `json:"captured_at"`
	Level           int              `json:"level"`
	Zone            string           `json:"zone"`
	ClientBuild     string           `json:"client_build"`
	Quests          []GameQuest      `json:"quests"`
	Truncated       bool             `json:"truncated"`
}
type Action struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	QuestID   int    `json:"quest_id"`
	ExpiresAt int64  `json:"expires_at"`
}
type ObjectiveRemainder struct {
	Index     int    `json:"index"`
	Text      string `json:"text,omitempty"`
	Remaining *int   `json:"remaining,omitempty"`
	Complete  bool   `json:"complete"`
}

type Candidate struct {
	Remaining   []ObjectiveRemainder `json:"objectives"`
	Guidance    string               `json:"guidance"`
	ID          string               `json:"id"`
	QuestID     int                  `json:"quest_id"`
	Title       string               `json:"title"`
	Goal        string               `json:"goal"`
	EvidenceID  string               `json:"evidence_id"`
	CanNavigate bool                 `json:"can_navigate"`
}
type GamePlan struct {
	Candidates []Candidate     `json:"candidates"`
	Trace      []harness.Event `json:"trace"`
}

func ValidateSnapshot(s *GameSnapshot) error {
	if s == nil {
		return nil
	}
	now := time.Now().Unix()
	if s.CapturedAt > now+5 || s.CapturedAt < now-90 || s.Level < 1 || s.Level > 100 || len(s.Zone) > 160 || !utf8.ValidString(s.Zone) || len(s.ClientBuild) > 30 || len(s.Quests) > 32 {
		return errors.New("invalid or stale game snapshot")
	}
	if len(s.Equipment) > 19 || len(s.Professions) > 12 || len(s.Missing) > 8 || s.ClassID < 0 || s.ClassID > 100 || s.RaceID < 0 || s.RaceID > 100 {
		return errors.New("invalid character snapshot")
	}
	slots := map[int]bool{}
	for _, i := range s.Equipment {
		if i.Slot < 1 || i.Slot > 19 || slots[i.Slot] || i.ItemID < 1 || i.ItemID > 2147483647 {
			return errors.New("invalid current equipment")
		}
		slots[i.Slot] = true
	}
	for _, p := range s.Professions {
		if p.SkillID < 1 || p.SkillID > 2147483647 || p.Rank < 0 || p.MaxRank < 0 || p.Rank > p.MaxRank || p.MaxRank > 10000 {
			return errors.New("invalid current profession")
		}
	}
	for _, field := range s.Missing {
		switch field {
		case "quests", "equipment", "professions", "class", "race":
		default:
			return errors.New("unknown missing state field")
		}
	}
	seen := map[int]bool{}
	for _, q := range s.Quests {
		if q.ID < 1 || q.ID > 2147483647 || seen[q.ID] || len(q.Title) > 200 || !utf8.ValidString(q.Title) {
			return errors.New("invalid quest observation")
		}
		if err := ValidateQuestText(q); err != nil {
			return err
		}
		if len(q.Objectives) > 8 {
			return errors.New("too many quest objectives")
		}
		for _, o := range q.Objectives {
			if o.Index < 0 || o.Index > 32 || len(o.Text) > 160 || len(o.Kind) > 30 || !utf8.ValidString(o.Text+o.Kind) || o.Fulfilled != nil && (*o.Fulfilled < 0 || *o.Fulfilled > 2147483647) || o.Required != nil && (*o.Required < 0 || *o.Required > 2147483647) {
				return errors.New("invalid quest objective")
			}
		}
		seen[q.ID] = true
	}
	if len(s.CompletedQuests) > 8 {
		return errors.New("too many completed quests")
	}
	for _, q := range s.CompletedQuests {
		if q.ID < 1 || q.ID > 2147483647 || seen[q.ID] || len(q.Title) > 100 || !utf8.ValidString(q.Title) || q.ObservedAt < 1 || q.ObservedAt > s.CapturedAt+5 || (q.Confirmation != "client_flag" && q.Confirmation != "turn_in_event") || len(q.ObjectiveTexts) > 4 {
			return errors.New("invalid completed quest evidence")
		}
		for _, v := range q.ObjectiveTexts {
			if len(v) > 80 || !utf8.ValidString(v) {
				return errors.New("invalid completed objective text")
			}
		}
		seen[q.ID] = true
	}
	return nil
}

// ValidateQuestText bounds private observed game text; it is never shared world knowledge.
func ValidateQuestText(q GameQuest) error {
	if len(q.Description) > 1024 || len(q.ObjectiveText) > 512 || len(q.WaypointText) > 240 || !utf8.ValidString(q.Description+q.ObjectiveText+q.WaypointText) {
		return errors.New("invalid observed quest text")
	}
	return nil
}
func BuildGamePlan(ctx context.Context, s *GameSnapshot, fetch func(context.Context) (storage.Graph, error)) (GamePlan, storage.Graph, error) {
	var g storage.Graph
	plan := GamePlan{Candidates: []Candidate{}, Trace: []harness.Event{}}
	runtime := harness.New()
	register := func(name, description string, fn func(context.Context, json.RawMessage) (json.RawMessage, error)) {
		if err := runtime.Register(harness.Tool{Spec: harness.Spec{Name: name, Description: description}, Run: fn}); err != nil {
			panic(err)
		}
	}
	register("read_game_context", "Validate the approved per-question addon observation", func(_ context.Context, _ json.RawMessage) (json.RawMessage, error) {
		if err := ValidateSnapshot(s); err != nil {
			return nil, err
		}
		return json.Marshal(map[string]any{"available": s != nil})
	})
	register("find_active_quest_options", "Construct options only from quests present in this observation", func(_ context.Context, _ json.RawMessage) (json.RawMessage, error) {
		if s != nil {
			for _, ready := range []bool{true, false} {
				for _, q := range s.Quests {
					if q.Complete != ready {
						continue
					}
					goal := "continue_objectives"
					if ready {
						goal = "check_turn_in"
					}
					candidate := Candidate{ID: fmt.Sprintf("quest:%d", q.ID), QuestID: q.ID, Title: q.Title, Goal: goal, EvidenceID: fmt.Sprintf("state:quest:%d", q.ID), CanNavigate: q.Waypoint, Remaining: []ObjectiveRemainder{}, Guidance: "Observed objectives. Location and eligibility are not supplied."}
					for i, objective := range q.Objectives {
						progress := ObjectiveRemainder{Index: i + 1, Text: objective.Text, Complete: objective.Finished}
						if objective.Fulfilled != nil && objective.Required != nil {
							remaining := *objective.Required - *objective.Fulfilled
							if remaining < 0 {
								remaining = 0
							}
							progress.Remaining = &remaining
						}
						candidate.Remaining = append(candidate.Remaining, progress)
					}
					if q.Complete {
						candidate.Guidance = "The client marks this quest complete; its turn-in marker can be checked in the game. Recipient and position are not supplied."
					} else if len(q.Objectives) == 0 {
						candidate.Guidance = "Objective details are unavailable until the player opens this quest."
					}
					plan.Candidates = append(plan.Candidates, candidate)
				}
			}
		}
		return json.Marshal(plan.Candidates)
	})
	register("retrieve_world_evidence", "Retrieve bounded versioned world facts from the Jornia store", func(ctx context.Context, _ json.RawMessage) (json.RawMessage, error) {
		var err error
		g, err = fetch(ctx)
		if err != nil {
			return nil, err
		}
		return json.Marshal(map[string]int{"entities": len(g.Entities), "relationships": len(g.Edges)})
	})
	register("validate_navigation_options", "Check freshness and limit proposals before presentation", func(_ context.Context, _ json.RawMessage) (json.RawMessage, error) {
		if err := ValidateSnapshot(s); err != nil {
			return nil, err
		}
		return json.Marshal(map[string]any{"requires_click": true, "coordinates_from_model": false})
	})
	calls := []harness.Call{}
	for _, name := range []string{"read_game_context", "retrieve_world_evidence", "find_active_quest_options", "validate_navigation_options"} {
		calls = append(calls, harness.Call{ID: name, Tool: name, Input: json.RawMessage(`{}`)})
	}
	result, err := runtime.Run(ctx, calls)
	plan.Trace = result.Trace
	return plan, g, err
}
func (p GamePlan) Action(choice string, s *GameSnapshot, evidence []string) *Action {
	if s == nil || ValidateSnapshot(s) != nil || choice == "" {
		return nil
	}
	for _, c := range p.Candidates {
		if c.ID != choice || !c.CanNavigate {
			continue
		}
		for _, id := range evidence {
			if id == c.EvidenceID {
				return &Action{ID: "navigate:" + c.ID, Kind: "show_quest_waypoint", QuestID: c.QuestID, ExpiresAt: s.CapturedAt + 90}
			}
		}
	}
	return nil
}
func snapshotQuestion(r Request) string {
	q := retrievalQuestion(r)
	if r.Snapshot != nil {
		q += " " + r.Snapshot.Zone
		for _, done := range r.Snapshot.CompletedQuests {
			q += " " + done.Title
		}
		for _, quest := range r.Snapshot.Quests {
			if quest.Title != "" {
				q += " " + strings.TrimSpace(quest.Title)
			}
		}
	}
	return q
}
