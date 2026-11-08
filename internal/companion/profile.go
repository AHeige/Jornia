package companion

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/AHeige/Jornia/internal/armory"
)

func (s *Store) initProfiles() error {
	_, err := s.DB.Exec(`CREATE TABLE IF NOT EXISTS character_profiles(character_id TEXT PRIMARY KEY REFERENCES characters(id) ON DELETE CASCADE,body TEXT NOT NULL,expires_at INTEGER NOT NULL);DELETE FROM character_profiles WHERE expires_at<unixepoch();CREATE UNIQUE INDEX IF NOT EXISTS operations_single_profile ON operations(character_id) WHERE status='pending' AND kind='profile';`)
	return err
}
func (s *Store) Profiles(ctx context.Context) (map[string]armory.Snapshot, error) {
	if _, err := s.DB.ExecContext(ctx, `DELETE FROM character_profiles WHERE expires_at<=?`, time.Now().Unix()); err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT character_id,body FROM character_profiles`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]armory.Snapshot{}
	for rows.Next() {
		var id, raw string
		if err = rows.Scan(&id, &raw); err != nil {
			return nil, err
		}
		var p armory.Snapshot
		if json.Unmarshal([]byte(raw), &p) != nil {
			return nil, errors.New("invalid saved profile")
		}
		out[id] = p
	}
	return out, rows.Err()
}
func (s *Server) configureProfile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID          string `json:"mutation_id"`
		CharacterID string `json:"character_id"`
		Name        string `json:"name"`
		Surname     string `json:"surname"`
		Region      string `json:"region"`
		Consent     bool   `json:"consent"`
	}
	if !decode(w, r, &req) {
		return
	}
	lookup := armory.Lookup{Name: strings.TrimSpace(req.Name), Surname: strings.TrimSpace(req.Surname), Region: req.Region}
	if !req.Consent || !lookup.Valid() {
		reply(w, 400, map[string]string{"error": "Enter the full name and allow a public Blizzard profile lookup first."})
		return
	}
	if s.backendToken() == "" {
		reply(w, 400, map[string]string{"error": "Connect your Jornia invite code in Settings before loading Armory."})
		return
	}
	characterID := req.CharacterID
	if characterID != "" {
		var name string
		if s.Store.DB.QueryRowContext(r.Context(), `SELECT name FROM characters WHERE id=?`, characterID).Scan(&name) != nil || characterID == DemoCharacterID {
			reply(w, 400, map[string]string{"error": "Choose a real local character or add a new one."})
			return
		}
		if !strings.EqualFold(name, lookup.Name) && !strings.EqualFold(name, lookup.Name+" "+lookup.Surname) {
			reply(w, 400, map[string]string{"error": "The name does not match the selected character. Choose Add a character to create a separate profile."})
			return
		}
	} else {
		characterID = "armory:" + lookup.Region + ":" + strings.ToLower(lookup.Name+"-"+lookup.Surname)
	}
	fresh, err := s.Store.begin(r.Context(), req.ID, "profile", characterID, Fingerprint(req))
	if err != nil {
		reply(w, 409, map[string]string{"error": "Another profile request conflicts with this operation."})
		return
	}
	if !fresh {
		receipt, _ := s.Store.Operation(r.Context(), req.ID)
		reply(w, 200, receipt)
		return
	}
	fail := func(message string) {
		_, _ = s.Store.DB.Exec(`UPDATE operations SET status='failed',error=? WHERE id=?`, message, req.ID)
		receipt, _ := s.Store.Operation(r.Context(), req.ID)
		reply(w, 200, receipt)
	}
	raw, _ := json.Marshal(lookup)
	ctx, cancel := context.WithTimeout(r.Context(), 32*time.Second)
	defer cancel()
	remote, _ := http.NewRequestWithContext(ctx, "POST", s.Backend+"/api/wisp/character", bytes.NewReader(raw))
	remote.Header.Set("Authorization", "Bearer "+s.backendToken())
	remote.Header.Set("X-Wisp-Request-ID", req.ID)
	remote.Header.Set("Content-Type", "application/json")
	response, err := s.Client.Do(remote)
	if err != nil {
		fail("Armory result is uncertain. No request will be replayed automatically; your existing journal is unchanged.")
		return
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 128*1024+1))
	if err != nil || len(body) > 128*1024 {
		fail("Armory response could not be read.")
		return
	}
	if response.StatusCode != 200 {
		message := "Armory is unavailable. Check the invite code, full name and region."
		var e struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(body, &e) == nil && e.Error != "" {
			message = e.Error
		}
		fail(message)
		return
	}
	var snapshot armory.Snapshot
	if json.Unmarshal(body, &snapshot) != nil || !strings.EqualFold(snapshot.Name, lookup.Name) || !strings.EqualFold(snapshot.Surname, lookup.Surname) || snapshot.Region != lookup.Region || snapshot.ExpiresAt <= time.Now().Unix() || snapshot.ExpiresAt > time.Now().Add(28*24*time.Hour+time.Minute).Unix() {
		fail("Armory returned an invalid character snapshot.")
		return
	}
	tx, err := s.Store.DB.BeginTx(ctx, nil)
	if err != nil {
		fail("Could not save the character snapshot.")
		return
	}
	defer tx.Rollback()
	if req.CharacterID == "" {
		_, err = tx.ExecContext(ctx, `INSERT INTO characters(id,name,realm) VALUES(?,?,?) ON CONFLICT(id) DO NOTHING`, characterID, lookup.Name+" "+lookup.Surname, strings.ToUpper(lookup.Region)+" · Forever")
	}
	if err == nil {
		_, err = tx.ExecContext(ctx, `INSERT INTO character_profiles(character_id,body,expires_at) VALUES(?,?,?) ON CONFLICT(character_id) DO UPDATE SET body=excluded.body,expires_at=excluded.expires_at WHERE excluded.expires_at>=character_profiles.expires_at`, characterID, string(body), snapshot.ExpiresAt)
	}
	result, _ := json.Marshal(map[string]string{"character_id": characterID})
	if err == nil {
		_, err = tx.ExecContext(ctx, `UPDATE operations SET status='completed',result=? WHERE id=?`, string(result), req.ID)
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		_ = tx.Rollback()
		fail("The snapshot could not be saved. Your journal was preserved.")
		return
	}
	receipt, _ := s.Store.Operation(ctx, req.ID)
	reply(w, 200, receipt)
}
