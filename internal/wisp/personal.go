package wisp

import (
	"fmt"
	"github.com/AHeige/Jornia/internal/storage"
	"strings"
	"time"
)

// Basic identity is an observed fact, not something a model should guess from IDs.
// Other phrasings/languages retain the inference path with the same explicit fields.
func personalIdentity(r Request) (Answer, bool) {
	snap := r.Snapshot
	if snap == nil || snap.Name == "" {
		return Answer{}, false
	}
	q := normalizeWords(r.Question)
	swedish := false
	switch q {
	case "vem är jag", "vad är min karaktär", "vad är min char":
		swedish = true
	case "who am i", "what is my character", "what is my char":
	default:
		return Answer{}, false
	}
	identity := strings.TrimSpace(snap.RaceName + " " + snap.ClassName)
	text := fmt.Sprintf("%s · %s · level %d.", snap.Name, identity, snap.Level)
	if snap.Zone != "" {
		if swedish {
			text += " Du är i " + snap.Zone + "."
		} else {
			text += " You are in " + snap.Zone + "."
		}
	}
	expires := time.Unix(snap.CapturedAt+90, 0).UTC()
	return Answer{Status: "answered", Text: text, GameVersion: r.GameVersion, Route: "character_context", EvidenceIDs: []string{"state:snapshot"}, Sources: []storage.Source{{ID: "state:snapshot", Title: "Current addon character observation", URI: "wisp-local:game-snapshot", ValidationStatus: "user_imported", Revision: time.Unix(snap.CapturedAt, 0).UTC().Format(time.RFC3339), ExpiresAt: &expires}}, Warnings: []string{}, Actions: []Action{}}, true
}
