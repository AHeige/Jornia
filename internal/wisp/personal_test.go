package wisp

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestIdentityDoesNotGuessRaceOrClassOrRequireWorldSearch(t *testing.T) {
	service := Service{Store: forbiddenWorld{}}
	for _, question := range []string{"Vem är jag?", "Vad är min karaktär?", "Who am I?"} {
		a, e := service.Ask(context.Background(), Request{GameVersion: "forever", Question: question, Snapshot: &GameSnapshot{CapturedAt: time.Now().Unix(), Name: "Cone Thrower", RaceName: "Undead", ClassName: "Mage", Level: 3, Zone: "Tirisfal Glades"}})
		if e != nil || a.Route != "character_context" || !strings.Contains(a.Text, "Undead Mage") || !strings.Contains(a.Text, "Cone Thrower") || len(a.EvidenceIDs) != 1 {
			t.Fatal(e, a)
		}
	}
}
