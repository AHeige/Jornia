package ingestion

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type transport func(*http.Request) (*http.Response, error)

func (f transport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func response(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}
func TestAuthorizedForeverAndHostBoundaries(t *testing.T) {
	calls := 0
	c := Client{Region: "eu", ClientID: "test-id", ClientSecret: "test-secret", HTTP: &http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Host == "oauth.battle.net" {
			if r.Method != "POST" || r.URL.RawQuery != "" || r.Header.Get("Authorization") == "" {
				t.Fatal("OAuth secret placement")
			}
			return response(200, `{"access_token":"ephemeral-token","expires_in":3600}`), nil
		}
		if r.URL.Host != "eu.api.blizzard.com" || r.Header.Get("Authorization") != "Bearer ephemeral-token" || r.URL.Query().Get("namespace") != "static-forever-eu" {
			t.Fatal("unexpected API request")
		}
		return response(200, `{"_links":{"self":{"href":"https://eu.api.blizzard.com/data/wow/playable-class/index?namespace=static-1.60.1_70399-forever-eu"}},"classes":[{"id":1,"name":"Warrior","key":{"href":"https://eu.api.blizzard.com/data/wow/playable-class/1?namespace=static-1.60.1_70399-forever-eu"}}]}`), nil
	})}}
	d, err := c.Fetch(context.Background(), "/data/wow/playable-class/index", "static-forever-eu")
	if err != nil || d.Namespace != "static-1.60.1_70399-forever-eu" || calls != 2 {
		t.Fatalf("%+v %v %d", d, err, calls)
	}
	if links := c.Links(d); len(links) != 2 {
		t.Fatalf("index links %+v", links)
	}
	for _, path := range []string{"https://evil.example/data/wow/item/1", "/profile/wow/character/foo", "/data/wow/media/item/1", "/data/wow/quest/../profile", "/data/wow/item/1?access_token=bad", "/data/wow/item/1?namespace=static-eu"} {
		if _, err = c.Fetch(context.Background(), path, "static-forever-eu"); err == nil {
			t.Fatalf("accepted %s", path)
		}
	}
	if calls != 2 {
		t.Fatal("invalid paths reached network")
	}
	c.HTTP.Transport = transport(func(r *http.Request) (*http.Response, error) {
		return response(200, `{"_links":{"self":{"href":"https://eu.api.blizzard.com/data/wow/playable-class/index?namespace=static-1.15.9_70003-classic1x-eu"}}}`), nil
	})
	if _, err = c.Fetch(context.Background(), "/data/wow/playable-class/index", "static-forever-eu"); !errors.Is(err, ErrNamespace) {
		t.Fatal("Classic data accepted as Forever")
	}
}
func TestMissingSecretsNoNetworkAndNoRedirect(t *testing.T) {
	c := Client{HTTP: &http.Client{Transport: transport(func(*http.Request) (*http.Response, error) { t.Fatal("network without credential"); return nil, nil })}}
	if _, err := c.Fetch(context.Background(), Seeds[0], "static-forever-eu"); !errors.Is(err, ErrCredentials) {
		t.Fatal(err)
	}
	calls := 0
	c = Client{ClientID: "id", ClientSecret: "secret", HTTP: &http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return response(200, `{"access_token":"token","expires_in":3600}`), nil
		}
		out := response(302, "")
		out.Header.Set("Location", "https://evil.example/")
		return out, nil
	})}}
	if _, err := c.Fetch(context.Background(), Seeds[0], "static-forever-eu"); err == nil || calls != 2 {
		t.Fatal("redirect followed")
	}
}
func TestNormalizeDoesNotInventAndQuarantinesTestNames(t *testing.T) {
	d := Document{Path: "/data/wow/quest/1", Namespace: "static-1.60.1_70399-forever-eu", Locale: "en_GB", Payload: map[string]any{"id": float64(1), "title": "A source-backed quest", "description": "Narrative text not imported", "requirements": map[string]any{}, "secret": "not selected"}}
	b := Normalize(d, "eu", time.Now())
	if len(b.Entities) != 1 || len(b.Edges) != 0 || strings.Contains(b.Entities[0].Description, "Narrative") || strings.Contains(b.Entities[0].Description, "secret") {
		t.Fatalf("%+v", b)
	}
	if b.Source.ExpiresAt == nil || b.Source.ExpiresAt.After(time.Now().Add(29*24*time.Hour)) {
		t.Fatal("TTL absent")
	}
	if b.Source.Provenance.EffectiveVersion != d.Namespace {
		t.Fatal("missing build")
	}
	d.Path = "/data/wow/profession/1"
	d.Payload = map[string]any{"name": "Test Profession [DNT]"}
	if len(Normalize(d, "eu", time.Now()).Entities) != 0 {
		t.Fatal("internal row promoted")
	}
	var payload map[string]any
	_ = json.Unmarshal([]byte(`{"page":1,"pageCount":2,"results":[],"evil":{"href":"https://evil.example/data/wow/item/1?namespace=static-1.60.1_70399-forever-eu"}}`), &payload)
	d.Path = "/data/wow/search/item?_page=1&_pageSize=100&orderby=id"
	d.Payload = payload
	c := Client{Region: "eu"}
	links := c.Links(d)
	if len(links) != 1 || !strings.Contains(links[0], "_page=2") {
		t.Fatalf("pagination %+v", links)
	}
}

func TestCappedSearchSplitsBoundsAndIdentityIncludesBuild(t *testing.T) {
	d := Document{Path: "/data/wow/search/item?_page=1&_pageSize=100&id=%5B0%2C9999%5D&orderby=id", Namespace: "static-1.60.1_70399-forever-eu", Locale: "en_GB", Payload: map[string]any{"resultCountCapped": true, "page": float64(1), "pageCount": float64(10), "results": []any{}}}
	c := Client{Region: "eu"}
	links := c.Links(d)
	if len(links) != 2 {
		t.Fatalf("partition links %+v", links)
	}
	for _, link := range links {
		if !AllowedPath(link) || strings.Contains(link, "_page=2") {
			t.Fatalf("invalid capped continuation %s", link)
		}
	}
	d.Path = "/data/wow/item/25"
	d.Payload = map[string]any{"name": "A stable item ID"}
	a := Normalize(d, "eu", time.Now())
	d.Namespace = "static-1.60.1_70790-forever-eu"
	b := Normalize(d, "eu", time.Now())
	if a.Entities[0].ID == b.Entities[0].ID || a.Source.ID == b.Source.ID {
		t.Fatal("builds conflated under shared IDs")
	}
}
