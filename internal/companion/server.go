package companion

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/AHeige/Jornia/internal/storage"
	"github.com/AHeige/Jornia/internal/wisp"
)

type Server struct {
	Store       *Store
	Backend     string
	Client      *http.Client
	Host, Token string
	Assets      fs.FS
	Journal     *JournalSync
	GameContext bool
	access      atomic.Value
}
type Ask struct {
	ID               string             `json:"mutation_id"`
	CharacterID      string             `json:"character_id"`
	Question         string             `json:"question"`
	AI               bool               `json:"ai"`
	ContextIDs       []string           `json:"context_ids,omitempty"`
	ContextDigest    string             `json:"context_digest,omitempty"`
	Snapshot         *wisp.GameSnapshot `json:"snapshot,omitempty"`
	SnapshotApproved bool               `json:"snapshot_approved,omitempty"`
	journal          []wisp.Observation
	character        *wisp.CharacterContext
	synthetic        bool
}

func NewServer(store *Store, backend, host string, assets fs.FS) (*Server, error) {
	u, err := url.Parse(backend)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" && u.Path != "/" {
		return nil, errors.New("backend must be an origin URL")
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "127.0.0.1" || u.Hostname() == "localhost" || u.Hostname() == "::1")) {
		return nil, errors.New("backend requires HTTPS or loopback HTTP")
	}
	token := make([]byte, 32)
	if _, err = rand.Read(token); err != nil {
		return nil, err
	}
	server := &Server{Store: store, Backend: strings.TrimRight(backend, "/"), Host: host, Token: hex.EncodeToString(token), Assets: assets, Client: &http.Client{Timeout: 40 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
	server.loadAccess()
	return server, nil
}
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /local/native-character", s.registerNativeCharacter)
	mux.HandleFunc("POST /local/access", s.configureAccess)
	mux.HandleFunc("POST /local/profile", s.configureProfile)
	mux.HandleFunc("GET /local/installations", func(w http.ResponseWriter, r *http.Request) {
		reply(w, 200, map[string]any{"paths": DetectInstallations()})
	})
	mux.HandleFunc("GET /local/context", func(w http.ResponseWriter, r *http.Request) {
		preview, err := s.Store.ContextPreview(r.Context(), r.URL.Query().Get("character_id"), nil)
		if err != nil {
			reply(w, 400, map[string]string{"error": "Could not read journal excerpts."})
			return
		}
		reply(w, 200, preview)
	})
	mux.HandleFunc("GET /local/journal", func(w http.ResponseWriter, r *http.Request) {
		if s.Journal == nil {
			reply(w, 200, JournalStatus{})
			return
		}
		reply(w, 200, s.Journal.Status())
	})
	mux.HandleFunc("POST /local/journal", func(w http.ResponseWriter, r *http.Request) {
		var req JournalIntent
		if !decode(w, r, &req) {
			return
		}
		if s.Journal == nil {
			reply(w, 503, map[string]string{"error": "Journal reading is unavailable."})
			return
		}
		receipt, err := s.Journal.Configure(r.Context(), req)
		if err != nil {
			reply(w, 409, map[string]string{"error": err.Error()})
			return
		}
		reply(w, 200, receipt)
	})
	mux.HandleFunc("GET /local/journey", func(w http.ResponseWriter, r *http.Request) {
		journey, err := s.Store.Journey(r.Context(), r.URL.Query().Get("character_id"))
		if err != nil {
			reply(w, 500, map[string]string{"error": "Could not read this journey."})
			return
		}
		reply(w, 200, journey)
	})
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		b, err := fs.ReadFile(s.Assets, "index.html")
		if err != nil {
			reply(w, 500, map[string]string{"error": "interface unavailable"})
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, strings.ReplaceAll(string(b), "__WISP_TOKEN__", s.Token))
	})
	mux.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(s.Assets))))
	mux.HandleFunc("GET /local/state", func(w http.ResponseWriter, r *http.Request) {
		state, err := s.Store.State(r.Context())
		if err != nil {
			reply(w, 500, map[string]string{"error": "Could not read local history."})
			return
		}
		state["backend"] = s.Backend
		reply(w, 200, state)
	})
	mux.HandleFunc("GET /local/connection", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
		defer cancel()
		request, _ := http.NewRequestWithContext(ctx, "GET", s.Backend+"/healthz", nil)
		if s.backendToken() != "" {
			request.Header.Set("Authorization", "Bearer "+s.backendToken())
		}
		response, err := s.Client.Do(request)
		connected := false
		if err == nil {
			defer response.Body.Close()
			var result struct {
				Status string `json:"status"`
			}
			connected = response.StatusCode == 200 && json.NewDecoder(io.LimitReader(response.Body, 1024)).Decode(&result) == nil && result.Status == "ok"
		}
		reply(w, 200, map[string]any{"connected": connected, "backend": s.Backend})
	})
	mux.HandleFunc("POST /local/demo", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID string `json:"mutation_id"`
		}
		if !decode(w, r, &req) {
			return
		}
		receipt, err := s.Store.Import(r.Context(), req.ID, DemoExport())
		if err != nil {
			reply(w, 409, map[string]string{"error": "The demo journal could not be saved."})
			return
		}
		reply(w, 200, receipt)
	})
	mux.HandleFunc("GET /local/operation", func(w http.ResponseWriter, r *http.Request) {
		receipt, err := s.Store.Operation(r.Context(), r.URL.Query().Get("id"))
		if err != nil {
			reply(w, 404, map[string]string{"error": "Operation not found."})
			return
		}
		reply(w, 200, receipt)
	})
	mux.HandleFunc("POST /local/import", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     string `json:"mutation_id"`
			Export string `json:"export"`
		}
		if !decode(w, r, &req) {
			return
		}
		data, err := ParseExport(req.Export)
		if err != nil {
			reply(w, 400, map[string]string{"error": err.Error()})
			return
		}
		if data.InstallationID == demoInstallation {
			reply(w, 400, map[string]string{"error": "This identity is reserved for the demo. Use Explore demo character."})
			return
		}
		receipt, err := s.Store.Import(r.Context(), req.ID, data)
		if err != nil {
			code := 500
			if errors.Is(err, ErrConflict) || errors.Is(err, ErrBusy) {
				code = 409
			}
			reply(w, code, map[string]string{"error": "Import not saved. An existing entry may have different contents."})
			return
		}
		reply(w, 200, receipt)
	})
	mux.HandleFunc("POST /local/ask", func(w http.ResponseWriter, r *http.Request) {
		var req Ask
		if !decode(w, r, &req) {
			return
		}
		req.Question = strings.TrimSpace(req.Question)
		if len(req.Question) == 0 || len(req.Question) > 1000 {
			reply(w, 400, map[string]string{"error": "Write a question of no more than 1000 bytes."})
			return
		}
		if req.CharacterID != "" {
			var id string
			if s.Store.DB.QueryRowContext(r.Context(), `SELECT id FROM characters WHERE id=?`, req.CharacterID).Scan(&id) != nil {
				reply(w, 400, map[string]string{"error": "Choose a local character."})
				return
			}
		}
		if req.Snapshot != nil {
			if !s.GameContext || !req.SnapshotApproved || !req.AI || req.CharacterID == "" {
				reply(w, 400, map[string]string{"error": "Current quest context requires an explicitly enabled lab and per-question approval."})
				return
			}
			if err := wisp.ValidateSnapshot(req.Snapshot); err != nil {
				reply(w, 409, map[string]string{"error": "The current quest observation is invalid or expired. Capture a new question in-game."})
				return
			}
		}
		if len(req.ContextIDs) > 0 || req.ContextDigest != "" {
			if !req.AI || req.CharacterID == "" {
				reply(w, 400, map[string]string{"error": "Journal excerpts require a selected character and AI."})
				return
			}
			preview, err := s.Store.ContextPreview(r.Context(), req.CharacterID, req.ContextIDs)
			if err != nil || preview.Digest != req.ContextDigest {
				reply(w, 409, map[string]string{"error": "Granska journalutdragen igen innan du skickar."})
				return
			}
			req.journal = preview.Events
			req.character = preview.Character
			req.synthetic = preview.Synthetic
		}
		fresh, err := s.Store.begin(r.Context(), req.ID, "ask", req.CharacterID, Fingerprint(req), req)
		if err != nil {
			code := 500
			if errors.Is(err, ErrConflict) || errors.Is(err, ErrBusy) {
				code = 409
			}
			reply(w, code, map[string]string{"error": "The question could not be accepted. Check whether another question is still running."})
			return
		}
		if fresh {
			// Own the operation beyond the browser connection. A lost response must not resend paid inference.
			go s.runAsk(req)
		}
		receipt, err := s.Store.Operation(r.Context(), req.ID)
		if err != nil {
			reply(w, 500, map[string]string{"error": "Could not read the question status."})
			return
		}
		reply(w, 202, receipt)
	})
	mux.HandleFunc("GET /local/export", func(w http.ResponseWriter, r *http.Request) {
		state, err := s.Store.ExportHistory(r.Context())
		if err != nil {
			reply(w, 500, map[string]string{"error": "Could not export local history."})
			return
		}
		w.Header().Set("Content-Disposition", `attachment; filename="wisp-history.json"`)
		reply(w, 200, state)
	})
	mux.HandleFunc("POST /local/forget", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID          string `json:"mutation_id"`
			CharacterID string `json:"character_id"`
		}
		if !decode(w, r, &req) {
			return
		}
		fresh, err := s.Store.begin(r.Context(), req.ID, "forget", req.CharacterID, Fingerprint(req), req)
		if err != nil {
			reply(w, 409, map[string]string{"error": "Could not start deletion."})
			return
		}
		if !fresh {
			receipt, err := s.Store.Operation(r.Context(), req.ID)
			if err != nil {
				reply(w, 500, map[string]string{"error": "Could not read saved status."})
				return
			}
			reply(w, 200, receipt)
			return
		}
		tx, err := s.Store.DB.BeginTx(r.Context(), nil)
		if err != nil {
			reply(w, 500, map[string]string{"error": "The local database is busy."})
			return
		}
		defer tx.Rollback()
		fail := func(message string) {
			_ = tx.Rollback()
			_, _ = s.Store.DB.ExecContext(r.Context(), `UPDATE operations SET status='failed',error=? WHERE id=?`, message, req.ID)
			reply(w, 409, map[string]string{"error": message})
		}
		var count int
		err = tx.QueryRowContext(r.Context(), `SELECT count(*) FROM operations WHERE character_id=? AND kind='ask' AND status='pending'`, req.CharacterID).Scan(&count)
		if err != nil || count > 0 {
			fail("Wait for pending questions to finish.")
			return
		}
		if _, err = tx.ExecContext(r.Context(), `DELETE FROM operations WHERE character_id=? AND id<>?`, req.CharacterID, req.ID); err == nil {
			_, err = tx.ExecContext(r.Context(), `INSERT INTO journey_exclusions(character_id) VALUES(?) ON CONFLICT DO NOTHING`, Fingerprint(req.CharacterID))
		}
		if err == nil {
			rows, e := tx.QueryContext(r.Context(), `SELECT folder_id FROM character_files WHERE character_id=? UNION SELECT alias FROM journal_aliases WHERE character_id=?`, req.CharacterID, req.CharacterID)
			if e != nil {
				err = e
			} else {
				aliases := []string{}
				for rows.Next() {
					var v string
					if e = rows.Scan(&v); e != nil {
						break
					}
					aliases = append(aliases, v)
				}
				rows.Close()
				err = e
				for _, v := range aliases {
					if err != nil {
						break
					}
					_, err = tx.ExecContext(r.Context(), `INSERT INTO journey_exclusions(character_id) VALUES(?) ON CONFLICT DO NOTHING`, Fingerprint(v))
				}
			}
		}
		if err == nil {
			_, err = tx.ExecContext(r.Context(), `DELETE FROM characters WHERE id=?`, req.CharacterID)
		}
		if err != nil {
			fail("Could not delete the history.")
			return
		}
		if _, err = tx.ExecContext(r.Context(), `UPDATE operations SET status='completed',result='{"deleted":true}',character_id='',request_json='{}' WHERE id=?`, req.ID); err != nil {
			fail("Deletion could not be confirmed.")
			return
		}
		if err = tx.Commit(); err != nil {
			reply(w, 500, map[string]string{"error": "Deletion could not be confirmed."})
			return
		}
		receipt, _ := s.Store.Operation(r.Context(), req.ID)
		reply(w, 200, receipt)
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; connect-src 'self'; img-src 'self' data: https://render.worldofwarcraft.com; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'")
		if r.Host != s.Host {
			reply(w, 403, map[string]string{"error": "Host not allowed"})
			return
		}
		origin := r.Header.Get("Origin")
		if origin != "" && origin != "http://"+s.Host {
			reply(w, 403, map[string]string{"error": "Origin not allowed"})
			return
		}
		if strings.HasPrefix(r.URL.Path, "/local/") {
			if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Wisp-Token")), []byte(s.Token)) != 1 {
				reply(w, 403, map[string]string{"error": "Reopen the Jornia window."})
				return
			}
			if r.Method == "POST" && origin != "http://"+s.Host {
				reply(w, 403, map[string]string{"error": "Origin required"})
				return
			}
		}
		mux.ServeHTTP(w, r)
	})
}
func (s *Server) runAsk(req Ask) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	answer, err := s.answer(ctx, req)
	if err != nil {
		status := "failed"
		message := "Jornia could not return an answer. Your draft is kept; no reply has been invented."
		if errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "transport") {
			status = "needs_attention"
			message = "The result is uncertain. This request will not be sent again automatically. Your draft is kept."
		}
		_, _ = s.Store.DB.ExecContext(context.Background(), `UPDATE operations SET status=?,error=? WHERE id=?`, status, message, req.ID)
		return
	}
	raw, err := json.Marshal(answer)
	if err != nil {
		return
	}
	tx, err := s.Store.DB.BeginTx(context.Background(), nil)
	if err != nil {
		return
	}
	defer tx.Rollback()
	if req.CharacterID != "" {
		_, err = tx.Exec(`INSERT INTO messages(id,character_id,question,answer,created_at) VALUES(?,?,?,?,?)`, req.ID, req.CharacterID, req.Question, string(raw), time.Now().UTC().Format(time.RFC3339Nano))
		if err != nil {
			return
		}
	}
	if _, err = tx.Exec(`UPDATE operations SET status='completed',result=? WHERE id=?`, string(raw), req.ID); err != nil {
		return
	}
	_ = tx.Commit()
}
func (s *Server) answer(ctx context.Context, req Ask) (wisp.Answer, error) {
	q := strings.ToLower(strings.Trim(req.Question, " ?.!"))
	if !(req.AI && (len(req.journal) > 0 || req.character != nil || req.Snapshot != nil)) {
		if answer, handled, err := s.characterAnswer(ctx, req.CharacterID, q); handled {
			return answer, err
		}
		switch q {
		case "what was i doing last time", "what did i do last", "where am i":
			return s.englishMemory(ctx, req.CharacterID, false)
		case "my journey", "show my journey":
			return s.englishMemory(ctx, req.CharacterID, true)
		case "vad gjorde jag sist", "vad höll vi på med sist", "var är jag", "vad minns du om mig":
			return s.memory(ctx, req.CharacterID)
		case "min resa", "visa min resa", "berätta om min resa", "vilka platser har jag besökt", "vad har jag upplevt", "vilka uppdrag har jag klarat":
			return s.journeyAnswer(ctx, req.CharacterID)
		}
	}
	// Journal excerpts require per-question approval. Structured character identity,
	// notes and conversation history stay local; approved quest text is game-provided text.
	mode := "auto"
	if req.AI {
		mode = "generate"
	}
	body, _ := json.Marshal(wisp.Request{Question: req.Question, GameVersion: "forever", Mode: mode, Journal: req.journal, Character: req.character, Snapshot: req.Snapshot, SyntheticJournal: req.synthetic})
	request, err := http.NewRequestWithContext(ctx, "POST", s.Backend+"/api/wisp/ask", bytes.NewReader(body))
	if err != nil {
		return wisp.Answer{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Wisp-Request-ID", req.ID)
	if s.backendToken() != "" {
		request.Header.Set("Authorization", "Bearer "+s.backendToken())
	}
	response, err := s.Client.Do(request)
	if err != nil {
		return wisp.Answer{}, errors.New("transport failed")
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return wisp.Answer{}, fmt.Errorf("backend HTTP %d", response.StatusCode)
	}
	var out wisp.Answer
	if err = json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&out); err != nil {
		return out, errors.New("invalid backend response")
	}
	if out.Status != "answered" && out.Status != "unknown" {
		return out, errors.New("invalid backend status")
	}
	// A remote action is data. Retain only an action tied to this approved, fresh quest observation.
	actions := []wisp.Action{}
	if req.Snapshot != nil && req.SnapshotApproved && wisp.ValidateSnapshot(req.Snapshot) == nil {
		for _, action := range out.Actions {
			for _, q := range req.Snapshot.Quests {
				if action.Kind == "show_quest_waypoint" && action.ID == fmt.Sprintf("navigate:quest:%d", q.ID) && action.QuestID == q.ID && q.Waypoint && action.ExpiresAt > time.Now().Unix() && action.ExpiresAt <= req.Snapshot.CapturedAt+90 {
					for _, e := range out.EvidenceIDs {
						if e == fmt.Sprintf("state:quest:%d", q.ID) {
							actions = append(actions, action)
						}
					}
				}
			}
		}
	}
	out.Actions = actions
	// Replace request-local memory citations with local-only journal references.
	for i := range out.Sources {
		for j, e := range req.journal {
			if out.Sources[i].ID == e.ID && j < len(req.ContextIDs) {
				out.Sources[i].URI = "wisp-local:event/" + url.PathEscape(req.ContextIDs[j])
			}
		}
	}
	return out, nil
}

// memory renders the retained Swedish-language response path. English prompts use englishMemory.
func (s *Server) memory(ctx context.Context, char string) (wisp.Answer, error) {
	zero := 0.0
	a := wisp.Answer{Status: "unknown", Text: "Jag har ingen importerad historik för den här karaktären. Använd /wisp export i spelet och importera paketet här.", GameVersion: "forever", Route: "local_memory", Sources: []storage.Source{}, EvidenceIDs: []string{}, Warnings: []string{}, EstimatedCostEUR: &zero}
	if char == "" {
		return a, nil
	}
	events, err := s.Store.Events(ctx, char)
	if err != nil {
		return a, err
	}
	if len(events) == 0 {
		return a, nil
	}
	e := events[0]
	when := time.Unix(e.ObservedAt, 0).UTC().Format("2006-01-02 15:04 UTC")
	a.Status = "answered"
	a.Text = fmt.Sprintf("Senaste importerade observationen är från %s: nivå %d, i %s", when, e.Level, e.Zone)
	if e.Subzone != "" {
		a.Text += " — " + e.Subzone
	}
	a.Text += "."
	evidence := []Event{e}
	for _, candidate := range events {
		if candidate.Kind == "note" && candidate.Note != "" {
			a.Text += " Din senaste anteckning: ”" + candidate.Note + "”."
			if candidate.ID != e.ID {
				evidence = append(evidence, candidate)
			}
			break
		}
	}
	for _, ev := range evidence {
		a.EvidenceIDs = append(a.EvidenceIDs, "memory:"+ev.ID)
		a.Sources = append(a.Sources, storage.Source{ID: ev.ID, Title: "Importerad Wisp-journal", URI: "wisp-local:event/" + url.PathEscape(ev.ID), Revision: ev.ClientVersion + " / " + ev.ClientBuild, License: "Privat spelarhistorik", ValidationStatus: "user_imported"})
	}
	a.Warnings = []string{"Detta är importerad historik, inte din aktuella plats i spelet. Exportens innehåll kan ha ändrats av användaren."}
	if char == DemoCharacterID {
		a.Text = "Demodata — påhittad spelhistorik. " + a.Text
		a.Warnings = []string{"Syntetiska testhändelser. Detta är inte observationer från WoW eller verifiering av klienten."}
		for i := range a.Sources {
			a.Sources[i].Title = "Jornia demojournal · syntetisk"
			a.Sources[i].ValidationStatus = "synthetic_demo"
			a.Sources[i].License = "Jornia testdata"
		}
	}
	return a, nil
}
func decode(w http.ResponseWriter, r *http.Request, out any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 180000)
	defer r.Body.Close()
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if d.Decode(out) != nil || d.Decode(new(any)) != io.EOF {
		reply(w, 400, map[string]string{"error": "Invalid request."})
		return false
	}
	return true
}
func reply(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
