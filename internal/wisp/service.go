package wisp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"wisp/internal/ai"
	"wisp/internal/harness"
	"wisp/internal/storage"
)

var ErrUnavailable = errors.New("live inference is not configured")
var ErrRateLimited = errors.New("local inference limit reached; try again in a minute")

type Request struct {
	Question         string            `json:"question"`
	GameVersion      string            `json:"game_version"`
	Mode             string            `json:"mode"`
	Spoilers         bool              `json:"spoilers"`
	Journal          []Observation     `json:"journal,omitempty"`
	SyntheticJournal bool              `json:"synthetic_journal,omitempty"`
	Character        *CharacterContext `json:"character,omitempty"`
	Snapshot         *GameSnapshot     `json:"snapshot,omitempty"`
}
type Answer struct {
	Status           string           `json:"status"`
	Text             string           `json:"text"`
	GameVersion      string           `json:"game_version"`
	Route            string           `json:"route"`
	Sources          []storage.Source `json:"sources"`
	EvidenceIDs      []string         `json:"evidence_ids"`
	Warnings         []string         `json:"warnings"`
	Model            *string          `json:"model"`
	Usage            *ai.Usage        `json:"usage"`
	EstimatedCostEUR *float64         `json:"estimated_cost_eur"`
	LatencyMS        int64            `json:"latency_ms"`
	Actions          []Action         `json:"actions,omitempty"`
	ToolTrace        []harness.Event  `json:"tool_trace,omitempty"`
}
type Knowledge interface {
	Graph(context.Context, string, bool) (storage.Graph, error)
}
type Service struct {
	Store                   Knowledge
	Generator               ai.Generator
	InputPrice, OutputPrice *float64
	mu                      sync.Mutex
	window                  time.Time
	calls                   int
	active                  int
}

func (s *Service) allow() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if time.Since(s.window) >= time.Minute {
		s.window = time.Now()
		s.calls = 0
	}
	if s.calls >= 6 || s.active >= 2 {
		return false
	}
	s.calls++
	s.active++
	return true
}
func (s *Service) release()     { s.mu.Lock(); defer s.mu.Unlock(); s.active-- }
func normalize(s string) string { return strings.Trim(strings.ToLower(strings.TrimSpace(s)), " ?.!") }

var where = regexp.MustCompile(`(?i)^(?:where is|var ligger|var finns) (.+?)\??$`)
var about = regexp.MustCompile(`(?i)^(?:tell me about|what is|berätta om|vad är) (.+?)\??$`)
var places = regexp.MustCompile(`(?i)^(?:what places are in|vilka platser finns i) (.+?)\??$`)
var giver = regexp.MustCompile(`(?i)^(?:who gives|vem ger) (.+?)\??$`)
var quests = regexp.MustCompile(`(?i)^(?:what quests are in|vilka uppdrag finns i) (.+?)\??$`)
var recipient = regexp.MustCompile(`(?i)^(?:who do i turn in|vem lämnar jag in) (.+?) (?:to|hos)\??$`)

var nextQuest = regexp.MustCompile(`(?i)^(?:what comes after|vilket uppdrag kommer efter|vad kommer efter) (.+?)\??$`)
var previousQuest = regexp.MustCompile(`(?i)^(?:what comes before|vilket uppdrag kommer före|vad kommer före) (.+?)\??$`)

func (s *Service) Ask(ctx context.Context, r Request) (Answer, error) {
	swedish := regexp.MustCompile(`(?i)^(var |vem |vilka |vad |berätta |hur )`).MatchString(strings.TrimSpace(r.Question))
	if err := ValidateJournal(r.Journal); err != nil {
		return Answer{}, err
	}
	if err := ValidateCharacter(r.Character); err != nil {
		return Answer{}, err
	}
	if r.Character != nil && r.GameVersion != "" && r.GameVersion != "forever" {
		return Answer{}, errors.New("character context belongs to Forever")
	}
	if err := ValidateSnapshot(r.Snapshot); err != nil {
		return Answer{}, err
	}
	if r.Snapshot != nil && r.GameVersion != "forever" && r.GameVersion != "" {
		return Answer{}, errors.New("game snapshot belongs to Forever")
	}
	if answer, handled, err := s.catalogAnswer(ctx, r); handled {
		return answer, err
	}
	start := time.Now()
	zero := 0.0
	a := Answer{Status: "unknown", Text: "I don't have verified knowledge for that question and game version yet.", GameVersion: r.GameVersion, Route: "deterministic", Sources: []storage.Source{}, EvidenceIDs: []string{}, Warnings: []string{}, EstimatedCostEUR: &zero}
	if swedish {
		a.Text = "Jag saknar källbelagda uppgifter för den frågan och spelversionen."
	}
	plan, g, err := BuildGamePlan(ctx, r.Snapshot, func(ctx context.Context) (storage.Graph, error) {
		if retriever, ok := s.Store.(interface {
			Retrieve(context.Context, string, bool, string) (storage.Graph, error)
		}); ok {
			return retriever.Retrieve(ctx, r.GameVersion, r.Spoilers, snapshotQuestion(r))
		}
		return s.Store.Graph(ctx, r.GameVersion, r.Spoilers)
	})
	a.ToolTrace = plan.Trace
	if err != nil {
		return a, err
	}
	if len(g.Entities) == 0 && len(r.Journal) == 0 && r.Character == nil && r.Snapshot == nil && r.Mode != "generate" {
		a.LatencyMS = time.Since(start).Milliseconds()
		return a, nil
	}
	isFixture := strings.HasPrefix(r.GameVersion, "fixture-")
	if isFixture {
		a.Warnings = append(a.Warnings, "Development fixture only; Forever compatibility is unverified.")
	}
	if !isFixture {
		for _, source := range g.Sources {
			if source.ValidationStatus == "source_checked" {
				a.Warnings = append(a.Warnings, "Sources checked for Forever; API or community evidence does not independently verify the current in-game state.")
				break
			}
		}
	}
	entities := map[string]storage.Entity{}
	for _, e := range g.Entities {
		entities[e.ID] = e
	}
	facts := map[string]bool{}
	if r.Snapshot != nil {
		facts["state:snapshot"] = true
		for _, q := range r.Snapshot.Quests {
			facts[fmt.Sprintf("state:quest:%d", q.ID)] = true
		}
	}
	if r.Character != nil {
		facts[r.Character.EvidenceID] = true
	}
	for _, e := range r.Journal {
		facts[e.ID] = true
	}
	for _, e := range g.Entities {
		facts["entity:"+e.ID] = true
	}
	for _, e := range g.Edges {
		facts["edge:"+e.ID] = true
	}
	finish := func() {
		if snap := r.Snapshot; snap != nil {
			for _, id := range a.EvidenceIDs {
				if id == "state:snapshot" || strings.HasPrefix(id, "state:quest:") {
					expires := time.Unix(snap.CapturedAt+90, 0).UTC()
					a.Sources = append(a.Sources, storage.Source{ID: id, Title: "Approved per-question addon observation", URI: "wisp-local:game-snapshot", Revision: time.Unix(snap.CapturedAt, 0).UTC().Format(time.RFC3339), License: "Private player context", ValidationStatus: "user_imported", ExpiresAt: &expires})
				}
			}
			a.Warnings = append(a.Warnings, "The game snapshot is user-supplied, time-limited and may contain only part of the quest log. Navigation is rechecked in the game on click.")
		}

		if c := r.Character; c != nil {
			for _, id := range a.EvidenceIDs {
				if id == c.EvidenceID {
					expires := time.Unix(c.ExpiresAt, 0).UTC()
					a.Sources = append(a.Sources, storage.Source{ID: id, Title: "Selected local Armory snapshot", URI: "wisp-local:character/profile", Revision: time.Unix(c.LastLogin, 0).UTC().Format(time.RFC3339), License: "Private dated character context", ValidationStatus: "user_imported", ExpiresAt: &expires})
				}
			}
			a.Warnings = append(a.Warnings, "Character equipment and level are from a dated published profile, not live game state.")
		}
		for _, e := range r.Journal {
			for _, id := range a.EvidenceIDs {
				if id == e.ID {
					a.Sources = append(a.Sources, storage.Source{ID: e.ID, Title: "Selected local journal excerpt", URI: "wisp-local:excerpt/" + e.ID, Revision: time.Unix(e.ObservedAt, 0).UTC().Format(time.RFC3339), License: "Private player history", ValidationStatus: "user_imported"})
				}
			}
		}
		if len(r.Journal) > 0 {
			a.Warnings = append(a.Warnings, "Journal excerpts are saved observations, not current game state.")
			if r.SyntheticJournal {
				a.Warnings = append(a.Warnings, "The demo journal is synthetic and does not describe real game events.")
			}
		}

		sourceIDs := map[string]bool{}
		for _, id := range a.EvidenceIDs {
			for _, e := range g.Entities {
				if id == "entity:"+e.ID {
					sourceIDs[e.SourceID] = true
				}
			}
			for _, e := range g.Edges {
				if id == "edge:"+e.ID {
					sourceIDs[e.SourceID] = true
				}
			}
		}
		for _, src := range g.Sources {
			if sourceIDs[src.ID] {
				a.Sources = append(a.Sources, src)
			}
		}
		a.LatencyMS = time.Since(start).Milliseconds()
	}
	if r.Mode != "generate" {
		q := strings.TrimSpace(r.Question)
		for _, rule := range []struct {
			pattern *regexp.Regexp
			kind    string
		}{{where, "where"}, {about, "about"}, {places, "places"}, {giver, "giver"}, {quests, "quests"}, {recipient, "recipient"}, {nextQuest, "next"}, {previousQuest, "previous"}} {
			m := rule.pattern.FindStringSubmatch(q)
			if len(m) == 0 {
				continue
			}
			for _, e := range g.Entities {
				if normalize(m[1]) != normalize(e.Name) && normalize(m[1]) != e.ID {
					continue
				}
				if (rule.kind == "giver" || rule.kind == "recipient" || rule.kind == "next" || rule.kind == "previous") && e.Kind != "quest" {
					continue
				}
				switch rule.kind {
				case "next", "previous":
					for _, edge := range g.Edges {
						if edge.Predicate != "next_quest" {
							continue
						}
						targetID := ""
						if rule.kind == "next" && edge.SubjectID == e.ID {
							targetID = edge.ObjectID
						}
						if rule.kind == "previous" && edge.ObjectID == e.ID {
							targetID = edge.SubjectID
						}
						if targetID == "" {
							continue
						}
						target := entities[targetID]
						direction := "after"
						if rule.kind == "previous" {
							direction = "before"
						}
						a.Text = target.Name + " is recorded " + direction + " " + e.Name + " in this quest chain. Check availability in-game."
						if swedish {
							direction = "efter"
							if rule.kind == "previous" {
								direction = "före"
							}
							a.Text = target.Name + " listas " + direction + " " + e.Name + " i uppdragskedjan. Kontrollera tillgängligheten i spelet."
						}
						a.EvidenceIDs = []string{"entity:" + e.ID, "entity:" + target.ID, "edge:" + edge.ID}
						a.Status = "answered"
						break
					}

				case "about":
					a.Text = e.Description
					a.EvidenceIDs = []string{"entity:" + e.ID}
					a.Status = "answered"
				case "where":
					for _, edge := range g.Edges {
						if edge.SubjectID == e.ID && edge.Predicate == "located_in" {
							parent := entities[edge.ObjectID]
							a.Text = e.Name + " is in " + parent.Name + "."
							if swedish {
								a.Text = e.Name + " ligger i " + parent.Name + "."
							}
							a.EvidenceIDs = []string{"entity:" + e.ID, "entity:" + parent.ID, "edge:" + edge.ID}
							a.Status = "answered"
							break
						}
					}
				case "giver", "recipient":
					predicate, label := "offered_by", " is offered by "
					if rule.kind == "recipient" {
						predicate, label = "turn_in_to", " is turned in to "
					}
					for _, edge := range g.Edges {
						if edge.SubjectID == e.ID && edge.Predicate == predicate {
							target := entities[edge.ObjectID]
							a.Text = e.Name + label + target.Name + "."
							if swedish {
								label = " ges av "
								if rule.kind == "recipient" {
									label = " lämnas in hos "
								}
								a.Text = e.Name + label + target.Name + "."
							}
							a.EvidenceIDs = []string{"entity:" + e.ID, "entity:" + target.ID, "edge:" + edge.ID}
							a.Status = "answered"
							break
						}
					}
				case "places", "quests":
					names := []string{}
					ids := []string{"entity:" + e.ID}
					if rule.kind == "quests" {
						for _, edge := range g.Edges {
							child := entities[edge.SubjectID]
							if edge.ObjectID == e.ID && edge.Predicate == "quest_in" && child.Kind == "quest" {
								names = append(names, child.Name)
								ids = append(ids, "entity:"+child.ID, "edge:"+edge.ID)
							}
						}
					} else {
						// Include nested locations such as a dungeon on a coast, but never NPCs or quests.
						included := map[string]bool{e.ID: true}
						changed := true
						for changed {
							changed = false
							for _, edge := range g.Edges {
								child := entities[edge.SubjectID]
								if included[edge.ObjectID] && !included[child.ID] && edge.Predicate == "located_in" && isPlace(child.Kind) {
									included[child.ID] = true
									changed = true
									names = append(names, child.Name)
									ids = append(ids, "entity:"+child.ID, "edge:"+edge.ID)
								}
							}
						}
					}
					if len(names) > 0 {
						sort.Strings(names)
						a.Text = "Known places in " + e.Name + ": " + strings.Join(names, ", ") + "."
						if swedish {
							a.Text = "Kända platser i " + e.Name + ": " + strings.Join(names, ", ") + "."
						}
						if rule.kind == "quests" {
							a.Text = "Known quest entries in " + e.Name + " (not a complete list): " + strings.Join(names, ", ") + "."
							if swedish {
								a.Text = "Kända uppdrag i " + e.Name + " (inte en fullständig lista): " + strings.Join(names, ", ") + "."
							}
						}
						a.EvidenceIDs = ids
						a.Status = "answered"
					}

				}
				// A name can exist in several separately sourced records. A catalog
				// entry without the requested relationship must not hide curated facts.
				if a.Status == "answered" {
					break
				}
			}
			break
		}
		if a.Status == "answered" && isFixture {
			a.Text = "In this development fixture: " + a.Text
		}
		finish()
		return a, nil
	}
	// Retrieve only named entities and their one-hop neighborhood. No fuzzy guesses.
	selected := map[string]bool{}
	for _, id := range g.Anchors {
		selected[id] = true
	}
	question := " " + normalizeWords(snapshotQuestion(r)) + " "
	for _, e := range g.Entities {
		if strings.Contains(question, " "+normalizeWords(e.Name)+" ") {
			selected[e.ID] = true
		}
	}
	if len(selected) == 0 && len(r.Journal) == 0 && r.Character == nil && r.Snapshot == nil && r.Mode != "generate" {
		finish()
		return a, nil
	}
	matched := map[string]bool{}
	for id := range selected {
		matched[id] = true
	}
	contextGraph := storage.Graph{GameVersion: g.GameVersion, Entities: []storage.Entity{}, Edges: []storage.Edge{}, Sources: g.Sources}
	for _, e := range g.Edges {
		if matched[e.SubjectID] || matched[e.ObjectID] {
			selected[e.SubjectID] = true
			selected[e.ObjectID] = true
			contextGraph.Edges = append(contextGraph.Edges, e)
		}
	}
	allowed := map[string]bool{}
	if r.Snapshot != nil {
		allowed["state:snapshot"] = true
		for _, q := range r.Snapshot.Quests {
			allowed[fmt.Sprintf("state:quest:%d", q.ID)] = true
		}
	}
	if r.Character != nil {
		allowed[r.Character.EvidenceID] = true
	}
	for _, e := range r.Journal {
		allowed[e.ID] = true
	}
	for _, e := range g.Entities {
		if selected[e.ID] {
			contextGraph.Entities = append(contextGraph.Entities, e)
			allowed["entity:"+e.ID] = true
		}
	}
	// Broad class graphs can contain hundreds of skills. Keep a deterministic
	// bounded subset, prioritizing SQL/name anchors, before spending model tokens.
	encodedWorld, _ := json.Marshal(contextGraph.Entities)
	if len(encodedWorld) > 24000 {
		candidates := append([]storage.Entity(nil), contextGraph.Entities...)
		sort.SliceStable(candidates, func(i, j int) bool { return matched[candidates[i].ID] && !matched[candidates[j].ID] })
		kept := map[string]bool{}
		contextGraph.Entities = []storage.Entity{}
		budget := 0
		for _, e := range candidates {
			raw, _ := json.Marshal(e)
			if budget+len(raw) > 24000 {
				delete(allowed, "entity:"+e.ID)
				continue
			}
			budget += len(raw)
			kept[e.ID] = true
			contextGraph.Entities = append(contextGraph.Entities, e)
		}
		edges := []storage.Edge{}
		for _, e := range contextGraph.Edges {
			if kept[e.SubjectID] && kept[e.ObjectID] {
				edges = append(edges, e)
			}
		}
		contextGraph.Edges = edges
		a.Warnings = append(a.Warnings, "A bounded selection of world facts was retrieved; this is not the complete class, item or quest catalog.")
	}
	for _, e := range contextGraph.Edges {
		allowed["edge:"+e.ID] = true
	}
	if s.Generator == nil {
		return a, ErrUnavailable
	}
	if !s.allow() {
		return a, ErrRateLimited
	}
	defer s.release()
	// Send only relevant source status, not full audit/license metadata, to inference.
	sourceIDs := map[string]bool{}
	for _, e := range contextGraph.Entities {
		sourceIDs[e.SourceID] = true
	}
	for _, e := range contextGraph.Edges {
		sourceIDs[e.SourceID] = true
	}
	sourceRefs := []map[string]string{}
	for _, src := range g.Sources {
		if sourceIDs[src.ID] {
			sourceRefs = append(sourceRefs, map[string]string{"id": src.ID, "validation_status": src.ValidationStatus})
		}
	}
	b, _ := json.Marshal(map[string]any{"game_version": contextGraph.GameVersion, "entities": contextGraph.Entities, "edges": contextGraph.Edges, "sources": sourceRefs, "journal": r.Journal, "character": r.Character, "synthetic_journal": r.SyntheticJournal, "journal_is_live": false, "character_is_live": false, "game_snapshot": r.Snapshot, "next_step_options": plan.Candidates})
	if len(b) > 40000 && (r.Character != nil || len(r.Journal) > 0 || r.Snapshot != nil) {
		// Broad world matches must not prevent a supported personal-only answer.
		// Strip both world facts and their allowed citations before inference.
		allowed = map[string]bool{}
		if r.Snapshot != nil {
			allowed["state:snapshot"] = true
			for _, q := range r.Snapshot.Quests {
				allowed[fmt.Sprintf("state:quest:%d", q.ID)] = true
			}
		}
		for _, e := range r.Journal {
			allowed[e.ID] = true
		}
		if r.Character != nil {
			allowed[r.Character.EvidenceID] = true
		}
		b, _ = json.Marshal(map[string]any{"game_version": r.GameVersion, "entities": []storage.Entity{}, "edges": []storage.Edge{}, "sources": []any{}, "journal": r.Journal, "character": r.Character, "synthetic_journal": r.SyntheticJournal, "journal_is_live": false, "character_is_live": false, "game_snapshot": r.Snapshot, "next_step_options": plan.Candidates})
		a.Warnings = append(a.Warnings, "World matches exceeded the context budget. This answer can use only your selected dated character context; name a specific world resource for a combined answer.")
	}
	if len(b) > 40000 {
		a.Warnings = append(a.Warnings, "Retrieved context exceeded the model budget; narrow the question. No model request was sent.")
		finish()
		return a, nil
	}
	completion, err := s.Generator.Generate(ctx, r.Question, string(b))
	if err != nil {
		return a, err
	}
	a.Route = "berget"
	a.Model = &completion.Model
	a.Usage = completion.Usage
	a.EstimatedCostEUR = nil
	if completion.Usage != nil && s.InputPrice != nil && s.OutputPrice != nil {
		cost := (float64(completion.Usage.PromptTokens)**s.InputPrice + float64(completion.Usage.CompletionTokens)**s.OutputPrice) / 1e6
		a.EstimatedCostEUR = &cost
	}
	result := completion.Result
	valid := (result.Status == "answered" || result.Status == "unknown") && strings.TrimSpace(result.Text) != "" && len(result.Text) <= 4000
	if result.Status == "answered" && len(result.EvidenceIDs) == 0 {
		valid = false
	}
	for _, id := range result.EvidenceIDs {
		if !allowed[id] || !facts[id] {
			valid = false
		}
	}
	if result.Status == "unknown" && len(result.EvidenceIDs) > 0 {
		valid = false
	}
	// A single cited, observed quest is sufficient to offer optional navigation.
	// This avoids making UI availability depend on the model remembering an optional field.
	if result.Status == "answered" && result.ActionID == "" {
		choices := []string{}
		for _, candidate := range plan.Candidates {
			if plan.Action(candidate.ID, r.Snapshot, result.EvidenceIDs) != nil {
				choices = append(choices, candidate.ID)
			}
		}
		if len(choices) == 1 {
			result.ActionID = choices[0]
		}
	}
	if result.ActionID != "" {
		if result.Status != "answered" || plan.Action(result.ActionID, r.Snapshot, result.EvidenceIDs) == nil {
			valid = false
		}
	}
	if !valid {
		a.Warnings = append(a.Warnings, "Model answer failed evidence validation; answer withheld.")
		finish()
		return a, nil
	}
	a.Status = result.Status
	if action := plan.Action(result.ActionID, r.Snapshot, result.EvidenceIDs); action != nil {
		a.Actions = []Action{*action}
	}
	if a.Status == "answered" || a.Status == "unknown" {
		a.Text = result.Text
		a.EvidenceIDs = result.EvidenceIDs
		if isFixture {
			a.Text = "Development fixture (Forever unverified): " + a.Text
		}
	}
	finish()
	return a, nil
}

var nonWord = regexp.MustCompile(`[^\pL\pN]+`)

func normalizeWords(s string) string {
	return strings.TrimSpace(nonWord.ReplaceAllString(strings.ToLower(s), " "))
}

func isPlace(kind string) bool {
	switch kind {
	case "continent", "zone", "settlement", "location", "dungeon":
		return true
	}
	return false
}
