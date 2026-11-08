package companion

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/AHeige/Jornia/internal/wisp"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	_ "modernc.org/sqlite"
)

type Character struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Realm      string `json:"realm"`
	IsDemo     bool   `json:"is_demo,omitempty"`
	Discovered bool   `json:"discovered,omitempty"`
}
type Event struct {
	QuestTitle    string               `json:"quest_title,omitempty"`
	Objectives    []wisp.GameObjective `json:"objectives,omitempty"`
	ItemID        int                  `json:"item_id,omitempty"`
	Slot          int                  `json:"slot,omitempty"`
	ID            string               `json:"id"`
	Kind          string               `json:"kind"`
	ObservedAt    int64                `json:"observed_at"`
	Zone          string               `json:"zone"`
	Subzone       string               `json:"subzone"`
	Level         int                  `json:"level"`
	Note          string               `json:"note"`
	QuestID       int                  `json:"quest_id"`
	ClientVersion string               `json:"client_version"`
	ClientBuild   string               `json:"client_build"`
	Interface     int                  `json:"interface"`
	SessionID     string               `json:"session_id,omitempty"`
}
type Export struct {
	CurrentState   *LocalState `json:"current_state,omitempty"`
	Schema         int         `json:"schema"`
	InstallationID string      `json:"installation_id"`
	Character      Character   `json:"character"`
	Events         []Event     `json:"events"`
}
type Receipt struct {
	Question    string          `json:"question,omitempty"`
	CharacterID string          `json:"character_id,omitempty"`
	ID          string          `json:"id"`
	Status      string          `json:"status"`
	Result      json.RawMessage `json:"result,omitempty"`
	Error       string          `json:"error,omitempty"`
}
type Message struct {
	ID          string          `json:"id"`
	CharacterID string          `json:"character_id"`
	Question    string          `json:"question"`
	Answer      json.RawMessage `json:"answer"`
	CreatedAt   string          `json:"created_at"`
}
type Store struct{ DB *sql.DB }

var ErrConflict = errors.New("same identity has different content")
var ErrBusy = errors.New("a question is already running for this character")

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{DB: db}
	_, err = db.Exec(`PRAGMA secure_delete=ON; PRAGMA locking_mode=EXCLUSIVE; PRAGMA foreign_keys=ON; PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000;
 CREATE TABLE IF NOT EXISTS characters(id TEXT PRIMARY KEY,name TEXT NOT NULL,realm TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS character_checkpoints(character_id TEXT PRIMARY KEY REFERENCES characters(id) ON DELETE CASCADE,observed_at INTEGER NOT NULL,revision INTEGER NOT NULL,body TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS events(id TEXT PRIMARY KEY,character_id TEXT NOT NULL REFERENCES characters(id) ON DELETE CASCADE,observed_at INTEGER NOT NULL,body TEXT NOT NULL);
 CREATE INDEX IF NOT EXISTS events_character ON events(character_id,observed_at);
 CREATE TABLE IF NOT EXISTS operations(id TEXT PRIMARY KEY,kind TEXT NOT NULL,character_id TEXT NOT NULL,payload_hash TEXT NOT NULL,status TEXT NOT NULL,result TEXT,error TEXT NOT NULL DEFAULT '',created_at TEXT NOT NULL,request_json TEXT NOT NULL DEFAULT '{}');
 CREATE TABLE IF NOT EXISTS messages(id TEXT PRIMARY KEY,character_id TEXT NOT NULL REFERENCES characters(id) ON DELETE CASCADE,question TEXT NOT NULL,answer TEXT NOT NULL,created_at TEXT NOT NULL);
 CREATE UNIQUE INDEX IF NOT EXISTS operations_single_ask ON operations(character_id) WHERE status='pending' AND kind='ask';
 UPDATE operations SET status='needs_attention',error='Jornia closed before the result was confirmed. The question will not be sent again automatically.' WHERE status='pending';`)
	if err != nil {
		db.Close()
		return nil, err
	}
	if err = s.initProfiles(); err != nil {
		db.Close()
		return nil, err
	}
	if err = s.initDiscovery(); err != nil {
		db.Close()
		return nil, err
	}
	if err = s.initJourney(); err != nil {
		db.Close()
		return nil, err
	}
	if err = s.pruneExpiredAnswers(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	// Add the request journal to existing prototype databases without destroying history.
	columns, err := db.Query(`PRAGMA table_info(operations)`)
	if err != nil {
		db.Close()
		return nil, err
	}
	found := false
	for columns.Next() {
		var cid, notnull, pk int
		var name, kind string
		var value any
		if err = columns.Scan(&cid, &name, &kind, &notnull, &value, &pk); err != nil {
			columns.Close()
			db.Close()
			return nil, err
		}
		if name == "request_json" {
			found = true
		}
	}
	columns.Close()
	if !found {
		if _, err = db.Exec(`ALTER TABLE operations ADD COLUMN request_json TEXT NOT NULL DEFAULT '{}'`); err != nil {
			db.Close()
			return nil, err
		}
	}
	_ = os.Chmod(path, 0600)
	return s, nil
}
func ParseExport(raw string) (Export, error) {
	var out Export
	raw = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(raw), "WISP1:"))
	if len(raw) > 150000 || !utf8.ValidString(raw) {
		return out, errors.New("the export is too large or has invalid text encoding")
	}
	d := json.NewDecoder(strings.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&out); err != nil {
		return out, errors.New("invalid Jornia export packet")
	}
	// Unmarshal additionally rejects trailing objects/content.
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return out, errors.New("invalid Jornia export packet")
	}
	if out.Schema != 1 || out.InstallationID == "" || len(out.InstallationID) > 100 || out.Character.ID == "" || len(out.Character.ID) > 100 || len(out.Character.Name) > 256 || len(out.Character.Realm) > 256 || len(out.Events) == 0 || len(out.Events) > 100 {
		return out, errors.New("unsupported export version or contents")
	}
	if err := validateLocalState(out.CurrentState); err != nil {
		return out, err
	}
	seen := map[string]bool{}
	for _, e := range out.Events {
		if e.ID == "" || len(e.ID) > 256 || seen[e.ID] || !strings.HasPrefix(e.ID, out.InstallationID+":"+out.Character.ID+":") {
			return out, errors.New("invalid or repeated event identity")
		}
		seen[e.ID] = true
		if e.ObservedAt < 946684800 || e.ObservedAt > time.Now().Add(5*time.Minute).Unix() || e.Level < 0 || e.Level > 100 || e.QuestID < 0 || e.QuestID > 10000000 || e.Interface != 16001 || !strings.HasPrefix(e.ClientVersion, "1.60.") {
			return out, errors.New("unsupported client or event values; this alpha requires Forever 1.60.x / 16001")
		}
		if len(e.Zone) > 1024 || len(e.Subzone) > 1024 || len(e.Note) > 1024 || len(e.ClientBuild) > 100 || len(e.SessionID) > 256 {
			return out, errors.New("event text is too long")
		}
		if len(e.QuestTitle) > 200 || !utf8.ValidString(e.QuestTitle) || e.ItemID < 0 || e.Slot < 0 || e.Slot > 19 || validateObjectives(e.Objectives) != nil {
			return out, errors.New("invalid event detail")
		}
		switch e.Kind {
		case "snapshot", "zone", "level_up", "quest_completed", "note", "session_start", "session_end", "quest_accepted", "quest_removed", "quest_progress", "equipment_changed", "death", "resurrected":
		default:
			return out, errors.New("unknown event type")
		}
	}
	return out, nil
}
func Fingerprint(value any) string {
	b, _ := json.Marshal(value)
	return fmt.Sprintf("%x", sha256.Sum256(b))
}
func (s *Store) Operation(ctx context.Context, id string) (Receipt, error) {
	var r Receipt
	var raw sql.NullString
	var request string
	err := s.DB.QueryRowContext(ctx, `SELECT id,status,result,error,request_json,character_id FROM operations WHERE id=?`, id).Scan(&r.ID, &r.Status, &raw, &r.Error, &request, &r.CharacterID)
	var intent struct {
		Question string `json:"question"`
	}
	_ = json.Unmarshal([]byte(request), &intent)
	r.Question = intent.Question
	if raw.Valid {
		r.Result = json.RawMessage(raw.String)
	}
	return r, err
}
func (s *Store) begin(ctx context.Context, id, kind, char, hash string, request ...any) (bool, error) {
	if len(id) < 8 || len(id) > 100 {
		return false, errors.New("invalid mutation identity")
	}
	payload := "{}"
	if len(request) > 0 {
		b, err := json.Marshal(request[0])
		if err != nil {
			return false, err
		}
		payload = string(b)
	}
	result, err := s.DB.ExecContext(ctx, `INSERT INTO operations(id,kind,character_id,payload_hash,status,created_at,request_json) VALUES(?,?,?,?,'pending',?,?) ON CONFLICT(id) DO NOTHING`, id, kind, char, hash, time.Now().UTC().Format(time.RFC3339Nano), payload)
	if err != nil {
		if strings.Contains(err.Error(), "operations.character_id") {
			return false, ErrBusy
		}
		return false, err
	}
	n, _ := result.RowsAffected()
	if n == 1 {
		return true, nil
	}
	var old string
	if err = s.DB.QueryRowContext(ctx, `SELECT payload_hash FROM operations WHERE id=?`, id).Scan(&old); err != nil {
		return false, err
	}
	if old != hash {
		return false, ErrConflict
	}
	return false, nil
}
func (s *Store) Import(ctx context.Context, id string, data Export) (Receipt, error) {
	return s.importData(ctx, id, data, false)
}
func (s *Store) importData(ctx context.Context, id string, data Export, automatic bool) (Receipt, error) {
	alias := data.InstallationID + ":" + data.Character.ID
	charID, err := s.journalCharacter(ctx, alias, data.Character.Name, data.Character.Realm)
	if err != nil {
		return Receipt{}, err
	}
	fresh, err := s.begin(ctx, id, "import", charID, Fingerprint(data))
	if err != nil {
		return Receipt{}, err
	}
	if !fresh {
		return s.Operation(ctx, id)
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Receipt{}, err
	}
	defer tx.Rollback()
	fail := func(err error) (Receipt, error) {
		_ = tx.Rollback()
		_, _ = s.DB.ExecContext(ctx, `UPDATE operations SET status='failed',error='Import not saved: conflicting entries or a database error.' WHERE id=?`, id)
		return Receipt{}, err
	}
	if automatic {
		var n int
		if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM journey_exclusions WHERE character_id=?`, Fingerprint(charID)).Scan(&n); err != nil {
			return fail(err)
		}
		if n > 0 {
			// A skipped file read must not recreate even a receipt containing the forgotten ID.
			_, err = tx.ExecContext(ctx, `DELETE FROM operations WHERE id=?`, id)
			if err != nil {
				return fail(err)
			}
			if err = tx.Commit(); err != nil {
				return Receipt{}, err
			}
			return Receipt{ID: id, Status: "completed", Result: json.RawMessage(`{"excluded":true}`)}, nil
		}
	} else {
		if _, err = tx.ExecContext(ctx, `DELETE FROM journey_exclusions WHERE character_id=?`, Fingerprint(charID)); err != nil {
			return fail(err)
		}
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO characters(id,name,realm) VALUES(?,?,?) ON CONFLICT(id) DO UPDATE SET name=CASE WHEN EXISTS(SELECT 1 FROM character_files WHERE character_id=characters.id) THEN characters.name ELSE excluded.name END,realm=excluded.realm`, charID, data.Character.Name, data.Character.Realm)
	if err != nil {
		return fail(err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO journal_aliases(alias,character_id) VALUES(?,?) ON CONFLICT(alias) DO NOTHING`, alias, charID); err != nil {
		return fail(err)
	}
	if data.CurrentState != nil {
		state := data.CurrentState
		body, _ := json.Marshal(state)
		if _, err = tx.ExecContext(ctx, `INSERT INTO character_checkpoints(character_id,observed_at,revision,body) VALUES(?,?,?,?) ON CONFLICT(character_id) DO UPDATE SET observed_at=excluded.observed_at,revision=excluded.revision,body=excluded.body WHERE excluded.observed_at>character_checkpoints.observed_at OR (excluded.observed_at=character_checkpoints.observed_at AND excluded.revision>character_checkpoints.revision)`, charID, state.ObservedAt, state.Revision, string(body)); err != nil {
			return fail(err)
		}
	}
	inserted := 0
	for _, e := range data.Events {
		b, _ := json.Marshal(e)
		var old, char string
		err = tx.QueryRowContext(ctx, `SELECT body,character_id FROM events WHERE id=?`, e.ID).Scan(&old, &char)
		if err == nil {
			if old != string(b) || char != charID {
				return fail(ErrConflict)
			}
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return fail(err)
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO events(id,character_id,observed_at,body) VALUES(?,?,?,?)`, e.ID, charID, e.ObservedAt, string(b)); err != nil {
			return fail(err)
		}
		if err = projectEvent(ctx, tx, charID, e); err != nil {
			return fail(err)
		}
		inserted++
	}
	result, _ := json.Marshal(map[string]any{"character_id": charID, "inserted": inserted, "duplicates": len(data.Events) - inserted})
	if _, err = tx.ExecContext(ctx, `UPDATE operations SET status='completed',result=? WHERE id=?`, string(result), id); err != nil {
		return fail(err)
	}
	if err = tx.Commit(); err != nil {
		return Receipt{}, err
	}
	return s.Operation(ctx, id)
}
func (s *Store) Events(ctx context.Context, char string) ([]Event, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT body FROM events WHERE character_id=? ORDER BY observed_at DESC,CAST(substr(id,length(rtrim(id,'0123456789'))+1) AS INTEGER) DESC,id DESC LIMIT 200`, char)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Event{}
	for rows.Next() {
		var raw string
		if err = rows.Scan(&raw); err != nil {
			return nil, err
		}
		var e Event
		if err = json.Unmarshal([]byte(raw), &e); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].ObservedAt != out[j].ObservedAt {
			return out[i].ObservedAt > out[j].ObservedAt
		}
		sequence := func(id string) int64 {
			value, _ := strconv.ParseInt(id[strings.LastIndex(id, ":")+1:], 10, 64)
			return value
		}
		return sequence(out[i].ID) > sequence(out[j].ID)
	})
	return out, rows.Err()
}
func (s *Store) State(ctx context.Context) (map[string]any, error) {
	if err := s.pruneExpiredAnswers(ctx); err != nil {
		return nil, err
	}
	chars := []Character{}
	rows, err := s.DB.QueryContext(ctx, `SELECT id,name,realm,EXISTS(SELECT 1 FROM character_files f WHERE f.character_id=characters.id) FROM characters ORDER BY name,id`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var c Character
		if err = rows.Scan(&c.ID, &c.Name, &c.Realm, &c.Discovered); err != nil {
			rows.Close()
			return nil, err
		}
		c.IsDemo = c.ID == DemoCharacterID
		chars = append(chars, c)
	}
	rows.Close()
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	events := map[string][]Event{}
	for _, c := range chars {
		list, err := s.Events(ctx, c.ID)
		if err != nil {
			return nil, err
		}
		events[c.ID] = list
	}
	messages := []Message{}
	rows, err = s.DB.QueryContext(ctx, `SELECT id,character_id,question,answer,created_at FROM messages ORDER BY created_at DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var m Message
		var raw string
		if err = rows.Scan(&m.ID, &m.CharacterID, &m.Question, &raw, &m.CreatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		m.Answer = json.RawMessage(raw)
		messages = append(messages, m)
	}
	rows.Close()
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	pending := []Receipt{}
	rows, err = s.DB.QueryContext(ctx, `SELECT id,status,error,request_json,character_id FROM operations WHERE status IN ('pending','needs_attention') ORDER BY created_at DESC LIMIT 20`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var r Receipt
		var request string
		if err = rows.Scan(&r.ID, &r.Status, &r.Error, &request, &r.CharacterID); err != nil {
			rows.Close()
			return nil, err
		}
		var intent struct {
			Question string `json:"question"`
		}
		_ = json.Unmarshal([]byte(request), &intent)
		r.Question = intent.Question
		pending = append(pending, r)
	}
	rows.Close()
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	profiles, err := s.Profiles(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{"characters": chars, "events": events, "messages": messages, "operations": pending, "profiles": profiles}, nil
}
