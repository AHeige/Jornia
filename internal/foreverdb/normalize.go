package foreverdb

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
	"wisp/internal/storage"
)

type Bundle struct {
	Source   storage.Source
	Entities []storage.Entity
	Edges    []storage.Edge
}

var kinds = map[string]string{"items": "item", "spells": "ability", "quests": "quest", "vendors": "npc", "dungeons": "dungeon", "zones": "zone", "classes": "class", "itemsets": "item_set", "professions": "profession", "gathering": "gathering_guide", "battlegrounds": "battleground", "changes-seen": "observed_change", "builds": "game_build", "buffs-nerfs": "class_change", "merchants-favor": "currency_guide", "poisons": "profession_guide", "disenchant": "profession_guide", "articles": "article", "posts": "official_post"}

func object(v any) map[string]any { m, _ := v.(map[string]any); return m }
func str(v any) string            { s, _ := v.(string); return s }
func array(v any) []any           { a, _ := v.([]any); return a }
func identity(v any) string {
	switch x := v.(type) {
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case string:
		return x
	}
	return ""
}
func hash(v string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(v)))[:24] }
func clean(v any, depth int) any {
	if depth > 7 {
		return "nested detail omitted"
	}
	switch x := v.(type) {
	case map[string]any:
		r := map[string]any{}
		for k, value := range x {
			switch k {
			case "icon", "image", "portrait", "media":
				continue
			}
			r[k] = clean(value, depth+1)
		}
		return r
	case []any:
		r := []any{}
		for i, value := range x {
			if i >= 100 {
				r = append(r, "additional rows omitted from projection")
				break
			}
			r = append(r, clean(value, depth+1))
		}
		return r
	case string:
		if len(x) > 2000 {
			return "long field omitted from projection"
		}
		return x
	default:
		return v
	}
}
func Normalize(path string, d Document, now time.Time) Bundle {
	u, _ := url.Parse(path)
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	category := parts[2]
	detail := len(parts) > 3
	rank := "list"
	if detail {
		rank = "detail"
	}
	sourceID := "foreverdb:" + d.Meta.Build + ":" + rank + ":" + hash(path)
	raw, _ := json.Marshal(d)
	digest := fmt.Sprintf("%x", sha256.Sum256(raw))
	expiry := now.Add(28 * 24 * time.Hour)
	b := Bundle{Source: storage.Source{ID: sourceID, Title: "Data from ForeverDB.net", URI: "https://foreverdb.net", Revision: d.Meta.Build + ":" + digest, License: "ForeverDB developer API terms: visible attribution required; no resale; original rights retained", ValidationStatus: "source_checked", ExpiresAt: &expiry, Provenance: &storage.Provenance{CheckedAt: now.UTC().Format(time.RFC3339), EffectiveVersion: d.Meta.Build, VerificationMethod: "Authorized ForeverDB API response checked; client records, Classic fallbacks and player observations retain their own labels. Not independently verified by Wisp.", Attribution: "Data from ForeverDB.net", HistoryURI: "https://foreverdb.net" + path, LicenseURI: "https://foreverdb.net/developers", Redistribution: "Application use under approved developer access; no resale or open-data relicensing", Changes: "Structured factual projection; media excluded; observation counts and original source labels preserved", ReviewSHA256: digest}}}
	add := func(id, kind, name string, row map[string]any) string {
		if name == "" || len(name) > 512 {
			return ""
		}
		c := object(clean(row, 0))
		facts := map[string]any{}
		size := 0
		for _, key := range []string{"id", "name", "slug", "level", "requiredLevel", "side", "zone", "category", "kind", "knownInForever", "source", "levelSource", "requiredLevelSource", "seenInForever", "placeSeenInForever", "players", "learned", "status", "announced", "notAnnouncedReason", "betaEvent", "speculation", "speculationReasons", "dataFromBuild", "newInForever", "type", "location", "mapId", "uiMapId", "x", "y", "chain", "description", "stats", "damage", "speed", "quality", "inventoryType", "costs", "requirements", "effects", "ranks", "tier", "column", "maxRanks", "prerequisites", "skill", "reagents", "produces", "makes", "spellId", "effect", "tools", "needsNearby", "learnedFrom", "trainerCost", "trainerCostSource", "reputationFaction", "levelRange", "recipeCount", "changedFromClassic", "maxSkill", "liveClient", "cost", "rank", "line", "prerequisite", "relation", "relationSource", "title", "summary", "published", "date", "url", "tags"} {
			v, ok := c[key]
			if !ok || key == "description" && kind == "quest" {
				continue
			}
			encoded, _ := json.Marshal(v)
			if size+len(encoded) > 5500 {
				continue
			}
			facts[key] = v
			size += len(encoded)
		}
		encoded, _ := json.Marshal(facts)
		description := name + " — ForeverDB build " + d.Meta.Build + ". API record, not proof of live availability. Preserve each field's Classic/Forever/client/observation source. Alternatives are not all required; observations do not prove drop rates. Fields may be partial: " + string(encoded)
		b.Entities = append(b.Entities, storage.Entity{ID: id, GameVersion: "forever", Kind: kind, Name: name, Description: description, SourceID: sourceID})
		return id
	}
	edge := func(a, p, z string) {
		if a != "" && z != "" {
			b.Edges = append(b.Edges, storage.Edge{ID: "foreverdb:" + hash(a+p+z), SubjectID: a, Predicate: p, ObjectID: z, SourceID: sourceID})
		}
	}
	var rows []any
	if a, ok := d.Data.([]any); ok {
		rows = a
	} else {
		rows = []any{d.Data}
	}
	for _, value := range rows {
		r := object(value)
		if r == nil {
			continue
		}
		name := str(r["name"])
		if name == "" {
			name = str(r["title"])
		}
		id := identity(r["id"])
		if id == "" {
			id = str(r["slug"])
		}
		if id == "" {
			canonical, _ := json.Marshal(r)
			id = hash(string(canonical))
		}
		if name == "" {
			name = category + " " + id
		}
		kind := kinds[category]
		if kind == "" {
			continue
		}
		if category == "dungeons" && str(r["kind"]) == "raid" {
			kind = "raid"
		}
		root := add("foreverdb:"+d.Meta.Build+":"+category+":"+id, kind, name, r)
		if z := object(r["zone"]); str(z["n"]) != "" {
			zid := add(root+":zone", "zone", str(z["n"]), z)
			edge(root, "in_zone", zid)
		}
		if category == "quests" {
			for _, role := range []string{"starts", "ends"} {
				n := object(r[role])
				nname := str(n["name"])
				if nname == "" && object(n["location"]) != nil {
					nname = name + " " + role + " map marker"
				}
				nk := str(n["type"])
				if nk == "" {
					nk = "quest_map_marker"
				}
				nid := add(root+":"+role, nk, nname, n)
				edge(root, map[string]string{"starts": "source_reported_given_by", "ends": "source_reported_turned_in_to"}[role], nid)
			}
			chain := object(r["chain"])
			for _, direction := range []string{"previous", "next"} {
				for _, link := range array(chain[direction]) {
					n := object(link)
					child := add(root+":chain:"+direction+":"+identity(n["questId"]), "quest_reference", str(n["name"]), n)
					edge(root, "source_reported_"+direction+"_quest", child)
				}
			}
		}
		if category == "classes" {
			for _, v := range array(r["abilities"]) {
				a := object(v)
				aid := add(root+":ability:"+hash(str(a["name"])), "ability", str(a["name"]), a)
				edge(root, "has_client_ability_record", aid)
			}
			for _, v := range array(r["talentTrees"]) {
				tree := object(v)
				tid := add(root+":tree:"+hash(str(tree["name"])), "talent_tree", name+" "+str(tree["name"]), tree)
				edge(root, "has_talent_tree", tid)
				for _, v := range array(tree["talents"]) {
					talent := object(v)
					iid := add(tid+":talent:"+identity(talent["id"]), "talent", str(talent["name"]), talent)
					edge(tid, "contains_talent", iid)
				}
			}
		}
		if category == "professions" {
			for _, v := range array(r["recipes"]) {
				recipe := object(v)
				rid := add(root+":recipe:"+identity(recipe["spellId"]), "recipe", str(recipe["name"]), recipe)
				edge(root, "has_recipe_record", rid)
			}
			for _, v := range array(r["trainers"]) {
				trainer := object(v)
				nid := add(root+":trainer:"+identity(trainer["npcId"]), "npc", str(trainer["name"]), trainer)
				edge(root, "source_reported_trainer", nid)
			}
		}
	}
	return b
}
