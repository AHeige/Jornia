package companion

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"time"
	"wisp/internal/storage"
	"wisp/internal/wisp"
)

// The graph is a rebuildable local projection. Original observations remain authoritative.
func (s *Store) initJourney() error {
	_, err := s.DB.Exec(`CREATE TABLE IF NOT EXISTS journey_nodes(character_id TEXT NOT NULL REFERENCES characters(id) ON DELETE CASCADE,id TEXT NOT NULL,kind TEXT NOT NULL,label TEXT NOT NULL,PRIMARY KEY(character_id,id));
 CREATE TABLE IF NOT EXISTS journey_edges(character_id TEXT NOT NULL REFERENCES characters(id) ON DELETE CASCADE,source TEXT NOT NULL,kind TEXT NOT NULL,target TEXT NOT NULL,PRIMARY KEY(character_id,source,kind,target));
 CREATE INDEX IF NOT EXISTS journey_node_kind ON journey_nodes(character_id,kind);
 CREATE INDEX IF NOT EXISTS journey_edge_target ON journey_edges(character_id,target,kind);
 CREATE TABLE IF NOT EXISTS journey_exclusions(character_id TEXT PRIMARY KEY);
 CREATE TABLE IF NOT EXISTS local_settings(key TEXT PRIMARY KEY,value TEXT NOT NULL);`)
	if err != nil {
		return err
	}
	rows, err := s.DB.Query(`SELECT e.character_id,e.body FROM events e WHERE NOT EXISTS(SELECT 1 FROM journey_nodes n WHERE n.character_id=e.character_id AND n.id='event:'||e.id)`)
	if err != nil {
		return err
	}
	type entry struct {
		char string
		e    Event
	}
	var entries []entry
	for rows.Next() {
		var x entry
		var raw string
		if err = rows.Scan(&x.char, &raw); err != nil {
			rows.Close()
			return err
		}
		if err = json.Unmarshal([]byte(raw), &x.e); err != nil {
			rows.Close()
			return err
		}
		entries = append(entries, x)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, x := range entries {
		if err = projectEvent(context.Background(), tx, x.char, x.e); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func projectEvent(ctx context.Context, tx *sql.Tx, char string, e Event) error {
	node := func(id, kind, label string) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO journey_nodes(character_id,id,kind,label) VALUES(?,?,?,?) ON CONFLICT DO NOTHING`, char, id, kind, label)
		return err
	}
	edge := func(from, kind, to string) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO journey_edges(character_id,source,kind,target) VALUES(?,?,?,?) ON CONFLICT DO NOTHING`, char, from, kind, to)
		return err
	}
	session := "session:" + e.SessionID
	sessionLabel := "Spelsession"
	if e.SessionID == "" {
		session = "observations:" + time.Unix(e.ObservedAt, 0).UTC().Format("2006-01-02")
		sessionLabel = "Importerade observationer · " + time.Unix(e.ObservedAt, 0).UTC().Format("2006-01-02")
	}
	ev := "event:" + e.ID
	place := "place:" + Fingerprint([]string{e.Zone, e.Subzone})
	label := e.Zone
	if e.Subzone != "" {
		label += " · " + e.Subzone
	}
	for _, n := range [][3]string{{"character", "character", char}, {session, "session", sessionLabel}, {ev, "event", e.Kind}, {place, "place", label}} {
		if err := node(n[0], n[1], n[2]); err != nil {
			return err
		}
	}
	for _, r := range [][3]string{{"character", "has_session", session}, {session, "has_event", ev}, {ev, "at_place", place}} {
		if err := edge(r[0], r[1], r[2]); err != nil {
			return err
		}
	}
	if e.Kind == "quest_completed" && e.QuestID > 0 {
		quest := fmt.Sprintf("quest:%d", e.QuestID)
		if err := node(quest, "quest", fmt.Sprintf("Uppdrag #%d", e.QuestID)); err != nil {
			return err
		}
		return edge(ev, "completed_quest", quest)
	}
	return nil
}

type Journey struct {
	Events          []Event `json:"events"`
	Total           int     `json:"total"`
	Places          int     `json:"places"`
	Sessions        int     `json:"sessions"`
	CompletedQuests int     `json:"completed_quests"`
	LastObserved    int64   `json:"last_observed"`
}

func (s *Store) Journey(ctx context.Context, char string) (Journey, error) {
	j := Journey{Events: []Event{}}
	if err := s.DB.QueryRowContext(ctx, `SELECT count(*),coalesce(max(observed_at),0) FROM events WHERE character_id=?`, char).Scan(&j.Total, &j.LastObserved); err != nil {
		return j, err
	}
	for _, q := range []struct {
		kind string
		dest *int
	}{{"place", &j.Places}, {"session", &j.Sessions}, {"quest", &j.CompletedQuests}} {
		if err := s.DB.QueryRowContext(ctx, `SELECT count(*) FROM journey_nodes WHERE character_id=? AND kind=?`, char, q.kind).Scan(q.dest); err != nil {
			return j, err
		}
	}
	events, err := s.Events(ctx, char)
	if len(events) > 50 {
		events = events[:50]
	}
	j.Events = events
	return j, err
}

func (s *Server) journeyAnswer(ctx context.Context, char string) (wisp.Answer, error) {
	a, err := s.memory(ctx, char)
	if err != nil || a.Status != "answered" {
		return a, err
	}
	j, err := s.Store.Journey(ctx, char)
	if err != nil {
		return a, err
	}
	a.Text = fmt.Sprintf("Din sparade resa omfattar %d observationer, %d platser och %d olika inlämnade quest-ID:n. Senast observerat %s.\n", j.Total, j.Places, j.CompletedQuests, time.Unix(j.LastObserved, 0).UTC().Format("2006-01-02 15:04 UTC"))
	events := j.Events
	if len(events) > 12 {
		events = events[:12]
	}
	a.Sources = []storage.Source{}
	a.EvidenceIDs = []string{}
	if j.Total > len(events) {
		a.Text += "Här är de senaste observationerna i tidsordning:\n"
	}
	for i := len(events) - 1; i >= 0; i-- {
		e := events[i]
		what := e.Zone
		if e.Subzone != "" {
			what += " — " + e.Subzone
		}
		switch e.Kind {
		case "quest_completed":
			what = fmt.Sprintf("Lämnade in uppdrag #%d i %s", e.QuestID, what)
		case "level_up":
			what = fmt.Sprintf("Nådde nivå %d i %s", e.Level, what)
		case "note":
			what = "Anteckning i " + what + ": ”" + e.Note + "”"
		case "session_start":
			what = "Började en spelsession i " + what
		case "session_end":
			what = "Avslutade en spelsession i " + what
		default:
			what = "Observerades i " + what
		}
		a.Text += "\n" + time.Unix(e.ObservedAt, 0).UTC().Format("01-02 15:04 UTC") + " · " + what
		a.EvidenceIDs = append(a.EvidenceIDs, "memory:"+e.ID)
		a.Sources = append(a.Sources, storage.Source{ID: e.ID, Title: "Lokal Wisp-observation", URI: "wisp-local:event/" + url.PathEscape(e.ID), Revision: e.ClientVersion + " / " + e.ClientBuild, License: "Privat spelarhistorik", ValidationStatus: "user_imported"})
	}
	if char == DemoCharacterID {
		a.Text = "Demodata — påhittad spelhistorik. " + a.Text
		for i := range a.Sources {
			a.Sources[i].Title = "Wisp demojournal · syntetisk"
			a.Sources[i].ValidationStatus = "synthetic_demo"
			a.Sources[i].License = "Wisp testdata"
		}
	}
	return a, nil
}

// Export is not restricted to the timeline's display window.
func (s *Store) ExportHistory(ctx context.Context) (map[string]any, error) {
	if err := s.pruneExpiredAnswers(ctx); err != nil {
		return nil, err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	characters := []Character{}
	events := map[string][]Event{}
	messages := []Message{}
	rows, err := tx.QueryContext(ctx, `SELECT id,name,realm FROM characters ORDER BY id`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var c Character
		if err = rows.Scan(&c.ID, &c.Name, &c.Realm); err != nil {
			rows.Close()
			return nil, err
		}
		c.IsDemo = c.ID == DemoCharacterID
		characters = append(characters, c)
		events[c.ID] = []Event{}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	rows, err = tx.QueryContext(ctx, `SELECT character_id,body FROM events ORDER BY observed_at,id`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var char, body string
		var e Event
		if err = rows.Scan(&char, &body); err != nil {
			rows.Close()
			return nil, err
		}
		if err = json.Unmarshal([]byte(body), &e); err != nil {
			rows.Close()
			return nil, err
		}
		events[char] = append(events[char], e)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	rows, err = tx.QueryContext(ctx, `SELECT id,character_id,question,answer,created_at FROM messages ORDER BY created_at,id`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var m Message
		var answer string
		if err = rows.Scan(&m.ID, &m.CharacterID, &m.Question, &answer, &m.CreatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		m.Answer = json.RawMessage(answer)
		messages = append(messages, m)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	profiles, err := s.Profiles(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{"schema": 1, "characters": characters, "events": events, "messages": messages, "profiles": profiles}, nil
}
