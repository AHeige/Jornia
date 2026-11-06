// Package armory fetches only explicitly requested public Forever profiles.
// Profile data is never inserted into the shared world knowledge database.
package armory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Item struct {
	ID      int    `json:"id"`
	Slot    string `json:"slot"`
	Name    string `json:"name"`
	Quality string `json:"quality"`
}
type Snapshot struct {
	Name      string   `json:"name"`
	Surname   string   `json:"surname"`
	Region    string   `json:"region"`
	Level     int      `json:"level"`
	Race      string   `json:"race"`
	Class     string   `json:"class"`
	Gender    string   `json:"gender"`
	Image     string   `json:"image"`
	Avatar    string   `json:"avatar"`
	Items     []Item   `json:"items"`
	FetchedAt int64    `json:"fetched_at"`
	LastLogin int64    `json:"last_login"`
	ExpiresAt int64    `json:"expires_at"`
	Source    string   `json:"source"`
	Warnings  []string `json:"warnings"`
}
type Lookup struct {
	Name    string `json:"name"`
	Surname string `json:"surname"`
	Region  string `json:"region"`
}
type Client struct {
	slots                  chan struct{}
	once                   sync.Once
	ClientID, ClientSecret string
	HTTP                   *http.Client
	mu                     sync.Mutex
	token                  string
	until                  time.Time
}

var namePattern = regexp.MustCompile(`^[\p{L}\p{N}][\p{L}\p{N}'-]{0,47}$`)

func (q Lookup) Valid() bool {
	return namePattern.MatchString(q.Name) && namePattern.MatchString(q.Surname) && (q.Region == "eu" || q.Region == "us" || q.Region == "kr" || q.Region == "tw")
}
func mediaURL(raw, region string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host != "render.worldofwarcraft.com" || u.User != nil || u.RawQuery != "" || !strings.HasPrefix(u.Path, "/forever-"+region+"/character/") {
		return ""
	}
	return u.String()
}
func (c *Client) tokenFor(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.HTTP == nil {
		c.HTTP = &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	if c.ClientID == "" || c.ClientSecret == "" {
		return "", errors.New("Armory is not configured on the server")
	}
	if c.token != "" && time.Now().Before(c.until) {
		return c.token, nil
	}
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://oauth.battle.net/token", strings.NewReader("grant_type=client_credentials"))
	req.SetBasicAuth(c.ClientID, c.ClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := c.HTTP.Do(req)
	if err != nil {
		return "", errors.New("Blizzard authentication is unavailable")
	}
	defer response.Body.Close()
	var token struct {
		Access  string `json:"access_token"`
		Expires int    `json:"expires_in"`
	}
	if response.StatusCode != 200 || json.NewDecoder(io.LimitReader(response.Body, 32768)).Decode(&token) != nil || token.Access == "" || token.Expires <= 60 {
		return "", errors.New("Blizzard authentication failed")
	}
	c.token = token.Access
	c.until = time.Now().Add(time.Duration(token.Expires-60) * time.Second)
	return c.token, nil
}
func (c *Client) fetch(ctx context.Context, token, region, path string, out any) error {
	endpoint := "https://" + region + ".api.blizzard.com" + path + "?namespace=profile-forever-" + region + "&locale=en_GB"
	req, _ := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return errors.New("Blizzard profile connection failed")
	}
	defer res.Body.Close()
	if res.StatusCode == 404 {
		return errors.New("Character not found in Forever. Check the full name and region; recently created characters may not be published yet")
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("Blizzard profile HTTP %d", res.StatusCode)
	}
	var raw map[string]json.RawMessage
	bytes, err := io.ReadAll(io.LimitReader(res.Body, (2<<20)+1))
	if err != nil || len(bytes) > 2<<20 {
		return errors.New("Profile response is too large")
	}
	if json.Unmarshal(bytes, &raw) != nil {
		return errors.New("Invalid profile response")
	}
	var links struct {
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
	}
	_ = json.Unmarshal(raw["_links"], &links)
	expectedPath, _ := url.PathUnescape(path)
	u, err := url.Parse(links.Self.Href)
	if err != nil || u.Scheme != "https" || u.Host != region+".api.blizzard.com" || u.Path != expectedPath || u.Query().Get("namespace") != "profile-forever-"+region {
		return errors.New("Profile did not prove the requested Forever source")
	}
	return json.Unmarshal(bytes, out)
}
func (c *Client) Fetch(ctx context.Context, q Lookup) (Snapshot, error) {
	s := Snapshot{Items: []Item{}, Warnings: []string{}}
	if !q.Valid() {
		return s, errors.New("Enter the character's first name, surname and region")
	}
	token, err := c.tokenFor(ctx)
	if err != nil {
		return s, err
	}
	slug := strings.ToLower(q.Name + "-" + q.Surname)
	path := "/profile/wow/character/" + url.PathEscape(slug)
	var p struct {
		ID           int    `json:"id"`
		Name         string `json:"name"`
		Surname      string `json:"surname"`
		Level        int    `json:"level"`
		Gender, Race struct {
			Name string `json:"name"`
		}
		Class struct {
			Name string `json:"name"`
		} `json:"character_class"`
		LastLogin int64 `json:"last_login_timestamp"`
	}
	if err = c.fetch(ctx, token, q.Region, path, &p); err != nil {
		return s, err
	}
	if !strings.EqualFold(p.Name+"-"+p.Surname, slug) || p.ID <= 0 || p.Level < 1 || p.Level > 100 {
		return s, errors.New("Profile identity did not match the requested character")
	}
	now := time.Now().Unix()
	s = Snapshot{Name: p.Name, Surname: p.Surname, Region: q.Region, Level: p.Level, Race: p.Race.Name, Class: p.Class.Name, Gender: p.Gender.Name, LastLogin: p.LastLogin / 1000, FetchedAt: now, ExpiresAt: now + 28*86400, Source: "https://" + q.Region + ".api.blizzard.com" + path + "?namespace=profile-forever-" + q.Region, Items: []Item{}, Warnings: []string{}}
	var gear struct {
		Character struct {
			ID int `json:"id"`
		} `json:"character"`
		Items []struct {
			Item struct {
				ID int `json:"id"`
			} `json:"item"`
			Slot struct {
				Name string `json:"name"`
			} `json:"slot"`
			Quality struct {
				Type string `json:"type"`
			} `json:"quality"`
			Name string `json:"name"`
		} `json:"equipped_items"`
	}
	if err = c.fetch(ctx, token, q.Region, path+"/equipment", &gear); err != nil {
		s.Warnings = append(s.Warnings, "Equipment is unavailable.")
	} else if gear.Character.ID != p.ID {
		return Snapshot{}, errors.New("Equipment belongs to a different character")
	} else {
		for _, item := range gear.Items {
			if len(s.Items) >= 25 {
				break
			}
			if item.Item.ID > 0 {
				s.Items = append(s.Items, Item{ID: item.Item.ID, Slot: item.Slot.Name, Name: item.Name, Quality: item.Quality.Type})
			}
		}
	}
	var media struct {
		Character struct {
			ID int `json:"id"`
		} `json:"character"`
		Assets []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"assets"`
	}
	if err = c.fetch(ctx, token, q.Region, path+"/character-media", &media); err != nil {
		s.Warnings = append(s.Warnings, "Character portrait is unavailable.")
	} else if media.Character.ID != p.ID {
		return Snapshot{}, errors.New("Portrait belongs to a different character")
	} else {
		for _, a := range media.Assets {
			if a.Key == "main-raw" {
				s.Image = mediaURL(a.Value, q.Region)
			}
			if a.Key == "avatar" {
				s.Avatar = mediaURL(a.Value, q.Region)
			}
		}
	}
	return s, nil
}
func (c *Client) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.once.Do(func() { c.slots = make(chan struct{}, 2) })
	select {
	case c.slots <- struct{}{}:
		defer func() { <-c.slots }()
	default:
		http.Error(w, "Character service is busy; try again later.", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	var q Lookup
	d := json.NewDecoder(io.LimitReader(r.Body, 4096))
	d.DisallowUnknownFields()
	if d.Decode(&q) != nil {
		w.WriteHeader(400)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Invalid character lookup"})
		return
	}
	snapshot, err := c.Fetch(r.Context(), q)
	if err != nil {
		w.WriteHeader(502)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(snapshot)
}
