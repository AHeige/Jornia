package armory

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestForeverIdentityMediaAndEquipment(t *testing.T) {
	for _, bad := range []bool{false, true} {
		calls := 0
		client := &Client{ClientID: "test", ClientSecret: "not-a-real-secret", HTTP: &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
			calls++
			raw := ""
			switch r.URL.Host {
			case "oauth.battle.net":
				raw = `{"access_token":"test-token","expires_in":3600}`
			case "eu.api.blizzard.com":
				if r.URL.Query().Get("namespace") != "profile-forever-eu" || r.Header.Get("Authorization") != "Bearer test-token" {
					t.Fatal("invalid source")
				}
				self := r.URL.String()
				if bad {
					self = strings.ReplaceAll(self, "profile-forever-eu", "profile-classic-eu")
				}
				if strings.HasSuffix(r.URL.Path, "/equipment") {
					raw = `{"character":{"id":1},"equipped_items":[{"item":{"id":4705},"slot":{"name":"Shoulders"},"quality":{"type":"UNCOMMON"},"name":"Lambent Scale Pauldrons"}]}`
				} else if strings.HasSuffix(r.URL.Path, "/character-media") {
					raw = `{"character":{"id":1},"assets":[{"key":"main-raw","value":"https://attacker.example/character.png"},{"key":"avatar","value":"https://render.worldofwarcraft.com/forever-eu/character/1/avatar.jpg"}]}`
				} else {
					raw = `{"id":1,"name":"Kungen","surname":"Forever","level":28,"race":{"name":"Night Elf"},"character_class":{"name":"Warrior"},"last_login_timestamp":1793914815000}`
				}
				raw = strings.TrimSuffix(raw, "}") + `,"_links":{"self":{"href":"` + self + `"}}}`
			default:
				t.Fatal("unexpected upstream", r.URL.Host)
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(raw)), Header: http.Header{}}, nil
		})}}
		s, err := client.Fetch(context.Background(), Lookup{Name: "Kungen", Surname: "Forever", Region: "eu"})
		if bad {
			if err == nil {
				t.Fatal("classic fallback accepted")
			}
			continue
		}
		if err != nil || s.Level != 28 || s.Class != "Warrior" || len(s.Items) != 1 || s.Items[0].ID != 4705 || s.Image != "" || s.Avatar == "" || calls != 4 {
			t.Fatalf("snapshot: %+v %v calls=%d", s, err, calls)
		}
		if s.ExpiresAt-s.FetchedAt != 28*86400 {
			t.Fatal("invalid retention")
		}
	}
}
func TestRejectInvalidLookupBeforeNetwork(t *testing.T) {
	c := &Client{}
	for _, q := range []Lookup{{Name: "../bad", Surname: "Forever", Region: "eu"}, {Name: "Kungen", Surname: "Forever", Region: "retail"}} {
		if _, err := c.Fetch(context.Background(), q); err == nil {
			t.Fatal("invalid lookup accepted")
		}
	}
}
