package companion

import (
	"context"
	"fmt"
	"strings"
	"time"

	"wisp/internal/storage"
	"wisp/internal/wisp"
)

// Common personal questions stay useful offline. No implicit profile upload.
func (s *Server) characterAnswer(ctx context.Context, id, question string) (wisp.Answer, bool, error) {
	kind := ""
	switch question {
	case "what level am i", "what is my level", "my level", "vilken level är jag", "vilken nivå är jag", "min nivå":
		kind = "level"
	case "what is my class", "what class am i", "my class", "vilken klass är jag", "min klass":
		kind = "class"
	case "what am i wearing", "what gear do i have", "my equipment", "my gear", "what is my equipment", "min utrustning", "vad har jag för utrustning":
		kind = "gear"
	case "my character", "tell me about my character", "min karaktär", "berätta om min karaktär":
		kind = "character"
	default:
		return wisp.Answer{}, false, nil
	}
	zero := 0.0
	a := wisp.Answer{Status: "unknown", Text: "No recorded data for this question yet. Connect a journal or load this character's Armory profile. For a personal AI question, review and include character context.", GameVersion: "forever", Route: "local_memory", Sources: []storage.Source{}, EvidenceIDs: []string{}, Warnings: []string{}, EstimatedCostEUR: &zero}
	if id == "" {
		return a, true, nil
	}
	if kind == "level" {
		events, err := s.Store.Events(ctx, id)
		if err != nil {
			return a, true, err
		}
		if len(events) > 0 {
			memory, err := s.englishMemory(ctx, id, false)
			return memory, true, err
		}
	}
	profiles, err := s.Store.Profiles(ctx)
	if err != nil {
		return a, true, err
	}
	p, ok := profiles[id]
	if !ok {
		return a, true, nil
	}
	when := time.Unix(p.LastLogin, 0).UTC().Format("2 Jan 2006, 15:04 UTC")
	a.Text = fmt.Sprintf("Your published Armory snapshot (last login %s) records level %d, %s %s.", when, p.Level, p.Race, p.Class)
	if kind == "gear" || kind == "character" {
		if len(p.Items) == 0 {
			if kind == "gear" {
				a.Text = "This Armory snapshot has no equipment data. Missing items do not mean empty equipment slots."
				return a, true, nil
			}
		} else {
			items := []string{}
			for _, item := range p.Items {
				items = append(items, item.Slot+": "+item.Name)
			}
			a.Text += "\n" + strings.Join(items, "\n")
		}
	}
	expires := time.Unix(p.ExpiresAt, 0).UTC()
	a.Status = "answered"
	a.EvidenceIDs = []string{"character:profile"}
	a.Sources = []storage.Source{{ID: "character:profile", Title: "Local Armory snapshot", URI: "wisp-local:character/profile", Revision: when, License: "Private character snapshot", ValidationStatus: "user_imported", ExpiresAt: &expires}}
	a.Warnings = []string{"Dated published profile, not live state. Your active quests, bags and talents are not supplied by this snapshot."}
	return a, true, nil
}
