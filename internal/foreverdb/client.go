// Package foreverdb owns our read-only, bounded import of the approved developer API.
package foreverdb

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type Document struct {
	Data any `json:"data"`
	Meta struct {
		Build  string `json:"build"`
		Client string `json:"client"`
		Next   string `json:"next"`
		Total  int    `json:"total"`
	} `json:"meta"`
}
type Client struct {
	Key, DeveloperID, BaseURL string
	HTTP                      *http.Client
	Interval                  time.Duration
	last                      time.Time
}
type HTTPError struct {
	Status     int
	RetryAfter string
}

func (e HTTPError) Error() string { return "ForeverDB request rejected" }

var pathPattern = regexp.MustCompile(`^/api/v1/(meta|items|spells|quests|vendors|dungeons|zones|classes|itemsets|professions|gathering|battlegrounds|changes-seen|builds|buffs-nerfs|merchants-favor|poisons|disenchant|articles|posts)(/[a-zA-Z0-9-]+)?$`)
var buildPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$`)

func AllowedPath(path string) bool {
	u, e := url.Parse(path)
	if e != nil || u.IsAbs() || u.Host != "" || u.Fragment != "" || !pathPattern.MatchString(u.Path) {
		return false
	}
	for k, v := range u.Query() {
		if len(v) != 1 || len(v[0]) > 100 {
			return false
		}
		switch k {
		case "limit", "offset", "q", "zone", "side", "new", "changed", "verified", "kind", "class", "build", "direction", "type", "change":
		default:
			return false
		}
	}
	return len(path) < 500
}
func (c *Client) Fetch(ctx context.Context, path string) (d Document, err error) {
	if !AllowedPath(path) || c.Key == "" || c.DeveloperID == "" {
		return d, errors.New("invalid ForeverDB request configuration")
	}
	base := c.BaseURL
	if base == "" {
		base = "https://foreverdb.net"
	}
	u, e := url.Parse(base)
	if e != nil || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" || !(u.Scheme == "https" && u.Host == "foreverdb.net" || u.Scheme == "http" && strings.HasPrefix(u.Host, "127.0.0.1:")) {
		return d, errors.New("invalid ForeverDB origin")
	}
	interval := c.Interval
	if interval <= 0 {
		interval = 1100 * time.Millisecond
	}
	if wait := time.Until(c.last.Add(interval)); wait > 0 {
		select {
		case <-ctx.Done():
			return d, ctx.Err()
		case <-time.After(wait):
		}
	}
	c.last = time.Now()
	req, e := http.NewRequestWithContext(ctx, "GET", base+path, nil)
	if e != nil {
		return d, e
	}
	req.Header.Set("X-API-Key", c.Key)
	req.Header.Set("User-Agent", "Wisp/0.7.1 (server data import)")
	req.Header.Set("Accept", "application/json")
	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	r, e := client.Do(req)
	if e != nil {
		return d, errors.New("ForeverDB connection failed")
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		return d, HTTPError{Status: r.StatusCode, RetryAfter: r.Header.Get("Retry-After")}
	}
	raw, e := io.ReadAll(io.LimitReader(r.Body, 4<<20+1))
	if e != nil || len(raw) > 4<<20 {
		return d, errors.New("ForeverDB response size rejected")
	}
	if json.Unmarshal(raw, &d) != nil || !buildPattern.MatchString(d.Meta.Build) || d.Meta.Client != c.DeveloperID || d.Data == nil {
		return d, errors.New("ForeverDB response identity/schema rejected")
	}
	if d.Meta.Next != "" && !AllowedPath(d.Meta.Next) {
		return d, errors.New("ForeverDB next-page origin rejected")
	}
	return d, nil
}
