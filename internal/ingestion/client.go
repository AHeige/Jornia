// Package ingestion reads only authorized public game-data API endpoints.
package ingestion

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var ErrCredentials = errors.New("Blizzard developer credentials are missing")
var ErrNamespace = errors.New("API response did not prove the requested Forever namespace")
var pathPattern = regexp.MustCompile(`^/data/wow/(search/(item|spell|creature)|playable-class|playable-race|playable-specialization|profession|recipe|item-class|item-set|item|spell|creature|creature-family|creature-type|quest|reputation-faction|reputation-tiers|journal-instance|journal-encounter|journal-expansion|mount|achievement|achievement-category|title|power-type)(/[a-z0-9-]+)*$`)
var namespacePattern = regexp.MustCompile(`^static(?:-[0-9]+\.[0-9]+\.[0-9]+_[0-9]+)?-forever-(eu|us|kr|tw)$`)

type Client struct {
	Region, Locale, ClientID, ClientSecret string
	HTTP                                   *http.Client
	Interval                               time.Duration
	token                                  string
	tokenUntil, last                       time.Time
}
type Document struct {
	Path, Namespace, Locale string
	FetchedAt               time.Time
	Payload                 map[string]any
	Status                  int
}

func (c *Client) init() error {
	if c.Region == "" {
		c.Region = "eu"
	}
	if c.Locale == "" {
		c.Locale = "en_GB"
	}
	if !regexp.MustCompile(`^(eu|us|kr|tw)$`).MatchString(c.Region) {
		return errors.New("unsupported region")
	}
	if !regexp.MustCompile(`^[a-z]{2}_[A-Z]{2}$`).MatchString(c.Locale) {
		return errors.New("invalid locale")
	}
	if c.HTTP == nil {
		c.HTTP = &http.Client{Timeout: 20 * time.Second}
	}
	if c.HTTP.CheckRedirect == nil {
		c.HTTP.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	}
	return nil
}
func (c *Client) Token(ctx context.Context) error {
	if err := c.init(); err != nil {
		return err
	}
	if c.ClientID == "" || c.ClientSecret == "" {
		return ErrCredentials
	}
	if c.token != "" && time.Now().Before(c.tokenUntil) {
		return nil
	}
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://oauth.battle.net/token", strings.NewReader("grant_type=client_credentials"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(c.ClientID+":"+c.ClientSecret)))
	req.Header.Set("User-Agent", "Wisp-Knowledge/0.1")
	response, err := c.HTTP.Do(req)
	if err != nil {
		return errors.New("OAuth transport failed")
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return fmt.Errorf("OAuth HTTP %d", response.StatusCode)
	}
	var token struct {
		Access  string `json:"access_token"`
		Expires int    `json:"expires_in"`
	}
	if err = json.NewDecoder(io.LimitReader(response.Body, 32768)).Decode(&token); err != nil || token.Access == "" || token.Expires <= 60 {
		return errors.New("invalid OAuth response")
	}
	c.token = token.Access
	c.tokenUntil = time.Now().Add(time.Duration(token.Expires-60) * time.Second)
	return nil
}
func AllowedPath(path string) bool {
	u, err := url.Parse(path)
	if err != nil || u.IsAbs() || u.Host != "" || u.Fragment != "" || !pathPattern.MatchString(u.Path) {
		return false
	}
	for key, values := range u.Query() {
		if len(values) != 1 {
			return false
		}
		switch key {
		case "_page", "_pageSize":
			if !regexp.MustCompile(`^[0-9]{1,6}$`).MatchString(values[0]) {
				return false
			}
		case "id":
			if !regexp.MustCompile(`^\[[0-9]{1,10},[0-9]{1,10}\]$`).MatchString(values[0]) {
				return false
			}
			bounds := strings.Split(strings.Trim(values[0], "[]"), ",")
			lo, _ := strconv.ParseInt(bounds[0], 10, 64)
			hi, _ := strconv.ParseInt(bounds[1], 10, 64)
			if lo < 0 || hi < lo || hi > 2147483647 {
				return false
			}
		case "orderby":
			if values[0] != "id" {
				return false
			}
		default:
			return false
		}
	}
	return true
}
func (c *Client) Fetch(ctx context.Context, path, namespace string) (Document, error) {
	d := Document{Path: path, Namespace: namespace, Locale: c.Locale}
	if err := c.init(); err != nil {
		return d, err
	}
	if !AllowedPath(path) || !namespacePattern.MatchString(namespace) || !strings.HasSuffix(namespace, "-"+c.Region) {
		return d, errors.New("API path or namespace rejected")
	}
	if err := c.Token(ctx); err != nil {
		return d, err
	}
	if wait := c.Interval - time.Since(c.last); wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return d, ctx.Err()
		case <-timer.C:
		}
	}
	c.last = time.Now()
	endpoint, _ := url.Parse("https://" + c.Region + ".api.blizzard.com" + path)
	q := endpoint.Query()
	q.Set("namespace", namespace)
	q.Set("locale", c.Locale)
	endpoint.RawQuery = q.Encode()
	req, _ := http.NewRequestWithContext(ctx, "GET", endpoint.String(), nil)
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("User-Agent", "Wisp-Knowledge/0.1")
	response, err := c.HTTP.Do(req)
	if err != nil {
		return d, errors.New("game-data transport failed")
	}
	defer response.Body.Close()
	d.Status = response.StatusCode
	if response.StatusCode != 200 {
		return d, fmt.Errorf("game-data HTTP %d", response.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, (4<<20)+1))
	if err != nil || len(raw) > 4<<20 {
		return d, errors.New("API response exceeds limit")
	}
	if json.Unmarshal(raw, &d.Payload) != nil {
		return d, errors.New("invalid API JSON")
	}
	// A response must identify itself; retail or Classic fallback must never enter Forever.
	links, _ := d.Payload["_links"].(map[string]any)
	self, _ := links["self"].(map[string]any)
	href, _ := self["href"].(string)
	u, err := url.Parse(href)
	actual := ""
	if err == nil && u.Scheme == "https" && u.Host == c.Region+".api.blizzard.com" {
		actual = u.Query().Get("namespace")
	}
	if actual == "" && strings.HasPrefix(path, "/data/wow/search/") {
		if results, ok := d.Payload["results"].([]any); ok {
			valid := true
			for _, entry := range results {
				obj, ok := entry.(map[string]any)
				if !ok {
					valid = false
					break
				}
				key, _ := obj["key"].(map[string]any)
				href, _ := key["href"].(string)
				u, err := url.Parse(href)
				if err != nil || u.Scheme != "https" || u.Host != c.Region+".api.blizzard.com" || u.User != nil {
					valid = false
					break
				}
				ns := u.Query().Get("namespace")
				if !namespacePattern.MatchString(ns) || !strings.HasSuffix(ns, "-"+c.Region) || (actual != "" && actual != ns) {
					valid = false
					break
				}
				actual = ns
			}
			if !valid {
				return d, ErrNamespace
			}
			// An empty pinned result contributes no game facts.
			if len(results) == 0 && namespace != "static-forever-"+c.Region {
				actual = namespace
			}
		}
	}
	if !namespacePattern.MatchString(actual) || !strings.HasSuffix(actual, "-"+c.Region) {
		return d, ErrNamespace
	}
	if namespace != "static-forever-"+c.Region && actual != namespace {
		return d, ErrNamespace
	}
	d.Namespace = actual
	d.Locale = c.Locale
	d.FetchedAt = time.Now().UTC()
	return d, nil
}

var Seeds = []string{"/data/wow/playable-class/index", "/data/wow/playable-race/index", "/data/wow/playable-specialization/index", "/data/wow/profession/index", "/data/wow/quest/index", "/data/wow/quest/area/index", "/data/wow/quest/category/index", "/data/wow/quest/type/index", "/data/wow/item-class/index", "/data/wow/item-set/index", "/data/wow/reputation-faction/index", "/data/wow/reputation-tiers/index", "/data/wow/journal-instance/index", "/data/wow/journal-encounter/index", "/data/wow/journal-expansion/index", "/data/wow/mount/index", "/data/wow/achievement/index", "/data/wow/achievement-category/index", "/data/wow/creature-family/index", "/data/wow/creature-type/index", "/data/wow/title/index", "/data/wow/power-type/index", "/data/wow/search/item?_page=1&_pageSize=100&orderby=id&id=%5B0%2C2147483647%5D", "/data/wow/search/spell?_page=1&_pageSize=100&orderby=id&id=%5B0%2C2147483647%5D", "/data/wow/search/creature?_page=1&_pageSize=100&orderby=id&id=%5B0%2C2147483647%5D"}

func (c *Client) Links(d Document) []string {
	seen := map[string]bool{}
	var visit func(any)
	visit = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			if href, ok := x["href"].(string); ok {
				u, err := url.Parse(href)
				if err == nil && u.Scheme == "https" && u.Host == c.Region+".api.blizzard.com" && u.User == nil && u.Fragment == "" {
					q := u.Query()
					ns := q.Get("namespace")
					if ns == d.Namespace {
						q.Del("namespace")
						q.Del("locale")
						path := u.Path
						if len(q) > 0 {
							path += "?" + q.Encode()
						}
						if AllowedPath(path) {
							seen[path] = true
						}
					}
				}
			}
			for _, v := range x {
				visit(v)
			}
		case []any:
			for _, v := range x {
				visit(v)
			}
		}
	}
	visit(d.Payload)
	// Blizzard search can cap the result set. Split an ID interval instead of claiming capped pages cover the corpus.
	if capped, _ := d.Payload["resultCountCapped"].(bool); capped && strings.HasPrefix(d.Path, "/data/wow/search/") {
		u, _ := url.Parse(d.Path)
		q := u.Query()
		bounds := strings.Split(strings.Trim(q.Get("id"), "[]"), ",")
		lo, hi := int64(0), int64(2147483647)
		if len(bounds) == 2 {
			lo, _ = strconv.ParseInt(bounds[0], 10, 64)
			hi, _ = strconv.ParseInt(bounds[1], 10, 64)
		}
		if hi > lo {
			mid := lo + (hi-lo)/2
			for _, pair := range [][2]int64{{lo, mid}, {mid + 1, hi}} {
				next := *u
				query := next.Query()
				query.Set("id", fmt.Sprintf("[%d,%d]", pair[0], pair[1]))
				query.Set("_page", "1")
				next.RawQuery = query.Encode()
				seen[next.String()] = true
			}
			result := []string{}
			for path := range seen {
				result = append(result, path)
			}
			return result
		}
	}
	// Search pagination is explicit, bounded and uses the same pinned namespace.
	page, _ := d.Payload["page"].(float64)
	count, _ := d.Payload["pageCount"].(float64)
	if count > 10000 {
		count = 10000
	}
	if page > 0 && page < count && strings.HasPrefix(d.Path, "/data/wow/search/") {
		u, _ := url.Parse(d.Path)
		q := u.Query()
		q.Set("_page", fmt.Sprintf("%.0f", page+1))
		u.RawQuery = q.Encode()
		seen[u.String()] = true
	}
	result := []string{}
	for path := range seen {
		result = append(result, path)
	}
	return result
}
