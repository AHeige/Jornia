package ingestion

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"wisp/internal/storage"
)

var internalName = regexp.MustCompile(`(?i)(\[(DNT|DND|UNUSED|PH)\]|^zz|^unused |^deprecated |^internal )`)
var entityPath = regexp.MustCompile(`^/data/wow/([a-z-]+)/([0-9]+)$`)
var kinds = map[string]string{"item": "item", "spell": "ability", "creature": "npc", "quest": "quest", "profession": "profession", "recipe": "recipe", "playable-class": "class", "playable-race": "race", "playable-specialization": "specialization", "reputation-faction": "faction", "journal-instance": "dungeon", "journal-encounter": "encounter", "mount": "mount", "achievement": "achievement", "title": "title", "item-class": "item_class", "item-set": "item_set", "creature-family": "creature_family", "creature-type": "creature_type", "power-type": "power_type"}

type Bundle struct {
	Source   storage.Source
	Entities []storage.Entity
	Edges    []storage.Edge
}

func localized(v any, locale string) string {
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	if m, ok := v.(map[string]any); ok {
		if s, ok := m[locale].(string); ok {
			return strings.TrimSpace(s)
		}
	}
	return ""
}
func identity(path, namespace, locale string) (string, string) {
	u, _ := url.Parse(path)
	m := entityPath.FindStringSubmatch(u.Path)
	if m == nil || kinds[m[1]] == "" {
		return "", ""
	}
	return "blizzard-api:" + namespace + ":" + locale + ":" + m[1] + ":" + m[2], kinds[m[1]]
}
func Normalize(d Document, region string, now time.Time) Bundle {
	payload, _ := json.Marshal(d.Payload)
	sum := fmt.Sprintf("%x", sha256.Sum256(payload))
	expiry := now.Add(28 * 24 * time.Hour)
	sourceID := "blizzard-api:" + d.Namespace + ":" + d.Locale + ":" + fmt.Sprintf("%x", sha256.Sum256([]byte(d.Path)))[:24]
	sourceURL, _ := url.Parse("https://" + region + ".api.blizzard.com" + d.Path)
	sourceQuery := sourceURL.Query()
	sourceQuery.Set("namespace", d.Namespace)
	sourceQuery.Set("locale", d.Locale)
	sourceURL.RawQuery = sourceQuery.Encode()
	b := Bundle{Entities: []storage.Entity{}, Edges: []storage.Edge{}, Source: storage.Source{ID: sourceID, Title: "Blizzard Entertainment · WoW Forever Game Data", URI: sourceURL.String(), Revision: d.Namespace + ":" + sum, License: "Blizzard Developer API Terms of Use; no open-data relicensing", ValidationStatus: "source_checked", ExpiresAt: &expiry, Provenance: &storage.Provenance{CheckedAt: now.UTC().Format(time.RFC3339), EffectiveVersion: d.Namespace, VerificationMethod: "Authorized game-data API; response self/resource-link namespaces checked for Forever. This is API evidence, not in-game verification.", Attribution: "Game data provided by Blizzard Entertainment", HistoryURI: sourceURL.String(), LicenseURI: "https://www.blizzard.com/en-us/legal/a2989b50-5f16-43b1-abec-2ae17cc09dd6/blizzard-developer-api-terms-of-use", Redistribution: "Application use under API terms; refresh/delete within 28 days. No personal/profile endpoints or assets.", Changes: "Selected identifiers, names, numeric metadata and explicitly linked resource references; no copied quest narrative or media.", ReviewSHA256: sum}}}
	add := func(path string, obj map[string]any) string {
		id, kind := identity(path, d.Namespace, d.Locale)
		name := localized(obj["name"], d.Locale)
		if name == "" {
			name = localized(obj["title"], d.Locale)
		}
		if id == "" || name == "" || len(name) > 512 || internalName.MatchString(name) {
			return ""
		}
		description := name + " is listed as " + kind + " in Blizzard's Forever game-data API (" + d.Namespace + ")."
		for _, key := range []string{"level", "required_level", "item_level", "quality", "max_count", "purchase_price", "sell_price"} {
			if n, ok := obj[key].(float64); ok {
				description += " " + key + "=" + strconv.FormatFloat(n, 'f', -1, 64) + "."
			}
		}
		b.Entities = append(b.Entities, storage.Entity{ID: id, GameVersion: "forever", Kind: kind, Name: name, Description: description, SourceID: sourceID})
		return id
	}
	rootID := add(d.Path, d.Payload)
	// Search records contain data plus a resource key; indexes contain linked name/id entries.
	var visit func(any, string)
	visit = func(v any, parentKey string) {
		switch obj := v.(type) {
		case []any:
			for _, entry := range obj {
				visit(entry, parentKey)
			}
		case map[string]any:
			var href string
			if key, ok := obj["key"].(map[string]any); ok {
				href, _ = key["href"].(string)
			}
			if href == "" {
				if key, ok := obj["_links"].(map[string]any); ok {
					if self, ok := key["self"].(map[string]any); ok {
						href, _ = self["href"].(string)
					}
				}
			}
			if href != "" {
				u, err := url.Parse(href)
				if err == nil && u.Scheme == "https" && u.Host == region+".api.blizzard.com" && u.Query().Get("namespace") == d.Namespace && AllowedPath(u.Path) {
					data := obj
					if nested, ok := obj["data"].(map[string]any); ok {
						data = nested
					}
					target := add(u.Path, data)
					if rootID != "" && target != "" && target != rootID {
						edgeID := fmt.Sprintf("%x", sha256.Sum256([]byte(rootID+":"+parentKey+":"+target)))
						b.Edges = append(b.Edges, storage.Edge{ID: "api-reference:" + edgeID, SubjectID: rootID, Predicate: "api_reference_" + parentKey, ObjectID: target, SourceID: sourceID})
					}
				}
			}
			for key, value := range obj {
				if key != "_links" && key != "key" {
					visit(value, key)
				}
			}
		}
	}
	visit(d.Payload, "resource")
	// Keep only one canonical row per resource in a response.
	seen := map[string]bool{}
	entities := []storage.Entity{}
	for _, e := range b.Entities {
		if !seen[e.ID] {
			seen[e.ID] = true
			entities = append(entities, e)
		}
	}
	b.Entities = entities
	return b
}
