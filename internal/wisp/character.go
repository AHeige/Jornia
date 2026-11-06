package wisp

import (
	"errors"
	"time"
	"unicode/utf8"
)

// CharacterContext is an explicitly reviewed, dated local Armory projection.
// No name, GUID, realm, image URL, credentials or private notes leave the client.
type CharacterContext struct {
	EvidenceID string          `json:"evidence_id"`
	Level      int             `json:"level"`
	Race       string          `json:"race"`
	Class      string          `json:"class"`
	FetchedAt  int64           `json:"fetched_at"`
	LastLogin  int64           `json:"last_login"`
	ExpiresAt  int64           `json:"expires_at"`
	Items      []CharacterItem `json:"items"`
}

type CharacterItem struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Slot    string `json:"slot"`
	Quality string `json:"quality"`
}

func ValidateCharacter(c *CharacterContext) error {
	if c == nil {
		return nil
	}
	now := time.Now().Unix()
	validText := func(s string, n int) bool { return len(s) <= n && utf8.ValidString(s) }
	if c.EvidenceID != "character:profile" || c.Level < 1 || c.Level > 100 || c.FetchedAt <= 0 || c.FetchedAt > now+60 || c.LastLogin <= 0 || c.LastLogin > c.FetchedAt+60 || c.ExpiresAt <= now || c.ExpiresAt > c.FetchedAt+28*86400 || !validText(c.Race, 80) || !validText(c.Class, 80) || len(c.Items) > 24 {
		return errors.New("invalid or expired character snapshot")
	}
	seen := map[string]bool{}
	for _, item := range c.Items {
		if item.ID < 1 || item.ID > 2147483647 || item.Name == "" || item.Slot == "" || seen[item.Slot] || !validText(item.Name, 160) || !validText(item.Slot, 40) || !validText(item.Quality, 40) {
			return errors.New("invalid character equipment")
		}
		seen[item.Slot] = true
	}
	return nil
}
