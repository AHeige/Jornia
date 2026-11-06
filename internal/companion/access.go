package companion

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

func (s *Server) backendToken() string {
	value := s.access.Load()
	if value == nil {
		return ""
	}
	return value.(string)
}
func (s *Server) loadAccess() {
	var token string
	if s.Store.DB.QueryRow(`SELECT value FROM local_settings WHERE key='access_token'`).Scan(&token) == nil {
		s.access.Store(token)
	}
}

func (s *Server) configureAccess(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID    string `json:"mutation_id"`
		Token string `json:"token"`
	}
	if !decode(w, r, &req) {
		return
	}
	req.Token = strings.TrimSpace(req.Token)
	if len(req.Token) != 64 || strings.Trim(req.Token, "0123456789abcdef") != "" {
		reply(w, 400, map[string]string{"error": "The invite code must contain 64 hexadecimal characters."})
		return
	}
	// Record a fingerprint, never the plaintext credential, in operation receipts.
	fresh, err := s.Store.begin(r.Context(), req.ID, "access", "", Fingerprint(req), map[string]string{"token": "[redacted]"})
	if err != nil {
		reply(w, 409, map[string]string{"error": "Could not save the connection."})
		return
	}
	if !fresh {
		receipt, _ := s.Store.Operation(r.Context(), req.ID)
		reply(w, 200, receipt)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5e9)
	defer cancel()
	check, _ := http.NewRequestWithContext(ctx, "GET", s.Backend+"/healthz", nil)
	check.Header.Set("Authorization", "Bearer "+req.Token)
	response, err := s.Client.Do(check)
	if err == nil {
		defer response.Body.Close()
		if response.StatusCode != 200 {
			err = errors.New("access denied")
		}
	}
	if err != nil {
		_, _ = s.Store.DB.Exec(`UPDATE operations SET status='failed',error='Could not verify the invite code. Check your connection and code.' WHERE id=?`, req.ID)
		receipt, _ := s.Store.Operation(r.Context(), req.ID)
		reply(w, 200, receipt)
		return
	}
	tx, err := s.Store.DB.BeginTx(r.Context(), nil)
	if err != nil {
		reply(w, 503, map[string]string{"error": "Local storage is busy."})
		return
	}
	defer tx.Rollback()
	_, err = tx.Exec(`INSERT INTO local_settings(key,value) VALUES('access_token',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, req.Token)
	if err == nil {
		_, err = tx.Exec(`UPDATE operations SET status='completed',result='{"connected":true}' WHERE id=?`, req.ID)
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		reply(w, 503, map[string]string{"error": "The connection could not be confirmed."})
		return
	}
	s.access.Store(req.Token)
	receipt, _ := s.Store.Operation(r.Context(), req.ID)
	reply(w, 200, receipt)
}

// ExportHistory deliberately excludes local_settings, including access credentials.
