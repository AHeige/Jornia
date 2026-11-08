package companion

import (
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Native registration links the current character before its first SavedVariables
// save. Identity stays local and later journal imports reuse the same alias.
func (s *Server) registerNativeCharacter(w http.ResponseWriter, r *http.Request) {
	if !s.GameContext {
		reply(w, 400, map[string]string{"error": "Native context is not enabled."})
		return
	}
	var req struct {
		ID             string    `json:"mutation_id"`
		InstallationID string    `json:"installation_id"`
		Character      Character `json:"character"`
	}
	if !decode(w, r, &req) {
		return
	}
	if !regexp.MustCompile(`^[A-Za-z0-9_-]{1,100}$`).MatchString(req.InstallationID) || !regexp.MustCompile(`^Player-[A-Za-z0-9-]{1,90}$`).MatchString(req.Character.ID) || req.ID == "" || len(req.ID) > 100 || len(req.Character.Name) > 100 || len(req.Character.Realm) > 100 || !utf8.ValidString(req.Character.Name+req.Character.Realm) || strings.ContainsAny(req.Character.Name+req.Character.Realm, "\x00\r\n") {
		reply(w, 400, map[string]string{"error": "Invalid local character identity."})
		return
	}
	receipt, err := s.Store.importData(r.Context(), req.ID, Export{Schema: 1, InstallationID: req.InstallationID, Character: req.Character, Events: []Event{}}, true)
	if err != nil {
		reply(w, 409, map[string]string{"error": "Could not register this character locally."})
		return
	}
	reply(w, 200, receipt)
}
