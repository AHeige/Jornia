package companion

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// WriteOCRSession explicitly grants the local experiment access to this running
// companion. Never enabled by default, served over HTTP, or written to logs.
func (s *Server) WriteOCRSession(path string) error {
	identities := map[string]string{}
	rows, err := s.Store.DB.Query(`SELECT alias,character_id FROM journal_aliases`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var alias, id string
		if err = rows.Scan(&alias, &id); err != nil {
			return err
		}
		if n := strings.LastIndex(alias, ":Player-"); n >= 0 {
			guid := alias[n+1:]
			if old, ok := identities[guid]; ok && old != id {
				return fmt.Errorf("ambiguous OCR identity")
			}
			identities[guid] = id
		}
	}
	if err = rows.Err(); err != nil {
		return err
	}
	data, err := json.Marshal(map[string]any{"base_url": "http://" + s.Host, "token": s.Token, "identities": identities})
	if err != nil {
		return err
	}
	// Exclusive create refuses symlinks and old session credentials.
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, err = file.Write(data)
	closeErr := file.Close()
	if err != nil {
		return err
	}
	return closeErr
}
