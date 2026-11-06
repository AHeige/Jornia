package storage

// Categories describe answerable domains, not a claim that an index is complete.
// Items/gear and professions/recipes overlap and must never be summed as a total.
type KnowledgeCategory struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Kinds           []string `json:"entity_kinds"`
	Records         int      `json:"records"`
	DetailDocuments int      `json:"active_api_detail_documents"`
	Status          string   `json:"status"`
	FieldCoverage   string   `json:"field_coverage"`
	Required        []string `json:"required_for_useful_answers"`
	NextSources     []string `json:"next_sources"`
	Derived         bool     `json:"derived"`
}

func CategoryCatalog(counts, details map[string]int) []KnowledgeCategory {
	type definition struct {
		id, name                            string
		kinds, resources, required, sources []string
		derived                             bool
	}
	definitions := []definition{
		{"quests", "Quests", []string{"quest", "quest_chain"}, []string{"quest"}, []string{"objectives", "requirements", "rewards", "starts_and_ends", "typed_chain_relationships", "locations"}, []string{"blizzard", "foreverdb:quests"}, false},
		{"items", "Items", []string{"item"}, []string{"item"}, []string{"stats", "requirements", "effects", "availability", "acquisition"}, []string{"blizzard", "foreverdb:items"}, false},
		{"gear", "Gear and sets", []string{"item", "item_set"}, []string{"item", "item-set"}, []string{"equip_slots", "stats", "set_bonuses", "class_restrictions", "enchants", "acquisition"}, []string{"blizzard", "foreverdb:items", "foreverdb:itemsets"}, false},
		{"zones", "Zones and places", []string{"continent", "zone", "location", "settlement"}, nil, []string{"map_ids", "subzones", "coordinates", "connections", "activities", "level_ranges"}, []string{"blizzard", "foreverdb:zones", "curated"}, false},
		{"npcs", "NPCs, vendors and trainers", []string{"npc"}, []string{"creature"}, []string{"positions", "roles", "services", "quest_links", "observation_provenance"}, []string{"blizzard", "foreverdb:quests", "foreverdb:vendors", "foreverdb:professions"}, false},
		{"classes", "Classes and races", []string{"class", "race"}, []string{"playable-class", "playable-race"}, []string{"abilities", "learned_levels", "resources", "class_race_restrictions"}, []string{"blizzard", "foreverdb:classes"}, false},
		{"specs", "Specializations and roles", []string{"specialization"}, []string{"playable-specialization"}, []string{"forever_applicability", "roles", "abilities", "talent_links"}, []string{"blizzard", "foreverdb:classes"}, false},
		{"abilities", "Spells and abilities", []string{"ability"}, []string{"spell"}, []string{"ranks", "costs", "cast_times", "cooldowns", "effects", "learned_levels"}, []string{"blizzard", "foreverdb:spells", "foreverdb:classes"}, false},
		{"talents", "Talent trees", []string{"talent", "talent_tree"}, nil, []string{"tiers", "columns", "ranks", "prerequisites", "spell_effects", "build_version"}, []string{"foreverdb:classes"}, false},
		{"professions", "Primary and secondary professions", []string{"profession"}, []string{"profession"}, []string{"primary_secondary_type", "ranks", "skill_caps", "trainers", "recipes"}, []string{"blizzard", "foreverdb:professions"}, false},
		{"recipes", "Recipes and crafting", []string{"recipe"}, []string{"recipe"}, []string{"reagents", "produced_items", "skill_thresholds", "learning_sources", "costs"}, []string{"blizzard", "foreverdb:professions"}, false},
		{"gathering", "Gathering and materials", []string{"gathering_node"}, nil, []string{"materials", "zones", "skill_requirements", "positions", "observed_vs_classic"}, []string{"foreverdb:gathering"}, false},
		{"dungeons", "Dungeons", []string{"dungeon"}, []string{"journal-instance"}, []string{"entrances", "level_ranges", "access_requirements", "bosses", "mechanics", "loot"}, []string{"blizzard", "foreverdb:dungeons"}, false},
		{"raids", "Raids", []string{"raid"}, nil, []string{"entrances", "access_requirements", "bosses", "mechanics", "loot", "roles"}, []string{"foreverdb:dungeons"}, false},
		{"bosses", "Bosses and encounters", []string{"encounter", "boss"}, []string{"journal-encounter"}, []string{"abilities", "mechanics", "instance_links", "loot", "strategy_sources"}, []string{"blizzard", "foreverdb:dungeons", "curated"}, false},
		{"travel", "Travel and transport", []string{"flight_path", "transport"}, nil, []string{"routes", "connections", "faction_restrictions", "travel_times"}, []string{"curated", "player_opt_in_observations"}, false},
		{"reputation", "Factions and reputation", []string{"faction"}, []string{"reputation-faction"}, []string{"ranks", "rewards", "earning_methods", "vendor_links"}, []string{"blizzard", "foreverdb:vendors", "foreverdb:merchants-favor"}, false},
		{"pvp", "PvP", []string{"battleground", "arena"}, nil, []string{"rules", "level_brackets", "objectives", "rewards", "source_assurance"}, []string{"foreverdb:battlegrounds"}, false},
		{"events", "World events", []string{"world_event"}, nil, []string{"dates", "activities", "locations", "requirements", "rewards"}, []string{"official_posts", "curated"}, false},
		{"collectibles", "Collections", []string{"mount", "achievement", "title"}, []string{"mount", "achievement", "title"}, []string{"availability", "requirements", "acquisition", "build_version"}, []string{"blizzard", "curated"}, false},
		{"lore", "Lore and storylines", []string{"lore", "storyline"}, nil, []string{"characters", "events", "chronology", "zone_links", "spoiler_labels", "reuse_rights"}, []string{"licensed_editorial", "original_curated_summaries"}, false},
		{"builds", "Character builds", []string{"character_build"}, nil, []string{"talent_dependencies", "playstyle", "abilities", "assumptions", "dated_evidence"}, []string{"derived_from_verified_talents_and_abilities"}, true},
		{"gear_combinations", "Gear combinations", []string{"gear_combination"}, nil, []string{"eligible_items", "stat_rules", "set_bonuses", "constraints", "objective", "explanation"}, []string{"derived_from_item_and_character_facts"}, true},
		{"quest_routes", "Quest routes", []string{"quest_route"}, nil, []string{"quest_dependencies", "coordinates", "travel_graph", "player_constraints", "explanation"}, []string{"derived_from_quest_and_travel_facts"}, true},
		{"experiences", "Places and experiences", []string{"experience_guide"}, nil, []string{"atmosphere", "scenery", "activities", "duration", "difficulty", "subjective_labels", "spoiler_labels"}, []string{"original_editorial", "local_player_preferences_not_central_journals"}, false},
	}
	out := make([]KnowledgeCategory, 0, len(definitions))
	for _, d := range definitions {
		c := KnowledgeCategory{ID: d.id, Name: d.name, Kinds: d.kinds, Status: "missing", FieldCoverage: "not_measured", Required: d.required, NextSources: d.sources, Derived: d.derived}
		for _, k := range d.kinds {
			c.Records += counts[k]
		}
		for _, k := range d.resources {
			c.DetailDocuments += details[k]
		}
		if c.Records > 0 {
			c.Status = "partial"
		}
		if c.Derived {
			c.Status = "not_implemented"
		}
		out = append(out, c)
	}
	return out
}
