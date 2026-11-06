package companion

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"wisp/internal/storage"
	"wisp/internal/wisp"
)

// English local prompts remain offline and retain the original player's note text.
func (s *Server) englishMemory(ctx context.Context, char string, full bool) (wisp.Answer, error) {
	zero := 0.0
	a := wisp.Answer{Status: "unknown", Text: "There is no recorded history for this character yet. Connect a Wisp journal or try the demo.", GameVersion: "forever", Route: "local_memory", Sources: []storage.Source{}, EvidenceIDs: []string{}, Warnings: []string{}, EstimatedCostEUR: &zero}
	j, err := s.Store.Journey(ctx, char)
	if err != nil || len(j.Events) == 0 {
		return a, err
	}
	a.Status = "answered"
	latest := j.Events[0]
	when := time.Unix(latest.ObservedAt, 0).UTC().Format("2 Jan 2006, 15:04 UTC")
	place := func(e Event) string {
		if e.Subzone != "" {
			return e.Zone + " / " + e.Subzone
		}
		return e.Zone
	}
	evidence := []Event{latest}
	if full {
		a.Text = fmt.Sprintf("Your journal records %d entries, %d places and %d completed quest IDs. Last recorded %s.\n", j.Total, j.Places, j.CompletedQuests, when)
		evidence = j.Events
		if len(evidence) > 12 {
			evidence = evidence[:12]
		}
		for i := len(evidence) - 1; i >= 0; i-- {
			e := evidence[i]
			what := "Recorded in " + place(e)
			switch e.Kind {
			case "quest_completed":
				what = fmt.Sprintf("Completed quest #%d in %s", e.QuestID, place(e))
			case "level_up":
				what = fmt.Sprintf("Reached level %d in %s", e.Level, place(e))
			case "note":
				what = "Note in " + place(e) + ": “" + e.Note + "”"
			case "session_start":
				what = "Started a session in " + place(e)
			case "session_end":
				what = "Ended a session in " + place(e)
			}
			a.Text += "\n" + time.Unix(e.ObservedAt, 0).UTC().Format("2 Jan, 15:04") + " · " + what
		}
	} else {
		a.Text = fmt.Sprintf("Your last recorded observation was %s: level %d, in %s.", when, latest.Level, place(latest))
		for _, e := range j.Events {
			if e.Kind == "note" && e.Note != "" {
				a.Text += " Your latest note: “" + e.Note + "”"
				if e.ID != latest.ID {
					evidence = append(evidence, e)
				}
				break
			}
		}
	}
	for _, e := range evidence {
		a.EvidenceIDs = append(a.EvidenceIDs, "memory:"+e.ID)
		a.Sources = append(a.Sources, storage.Source{ID: e.ID, Title: "Local Wisp journal", URI: "wisp-local:event/" + url.PathEscape(e.ID), Revision: e.ClientVersion + " / " + e.ClientBuild, License: "Private player history", ValidationStatus: "user_imported"})
	}
	a.Warnings = []string{"Recorded history, not your live position. Imported files may have been edited."}
	if char == DemoCharacterID {
		a.Text = "Demo data — invented history. " + a.Text
		a.Warnings = []string{"Synthetic events, not observations from WoW or verification of the client."}
		for i := range a.Sources {
			a.Sources[i].Title = "Wisp demo journal · synthetic"
			a.Sources[i].ValidationStatus = "synthetic_demo"
			a.Sources[i].License = "Wisp test data"
		}
	}
	return a, nil
}
