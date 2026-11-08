package wisp

import "regexp"

// These aliases select a retrieval strategy, never restrict supported languages.
// Unrecognized phrasing retains ordinary retrieval; no game facts are inferred.
var classIntent = regexp.MustCompile(`(?i)\b(spells?|abilit(y|ies)|rotation|talents?|talanger|besvärjelser|förmågor|sorts|hechizos|zauber)\b|法术|技能|天赋|呪文|スキル|талант|заклинан`)
var classNames = map[int]string{1: "Warrior", 2: "Paladin", 3: "Hunter", 4: "Rogue", 5: "Priest", 7: "Shaman", 8: "Mage", 9: "Warlock", 11: "Druid"}

func classKnowledgeRequest(r Request) bool {
	return r.Snapshot != nil && classNames[r.Snapshot.ClassID] != "" && classIntent.MatchString(r.Question)
}

var classSpellIntent = regexp.MustCompile(`(?i)\b(spells?|abilit(y|ies)|rotation|besvärjelser|förmågor|sorts|hechizos|zauber)\b|法术|技能|呪文|スキル|заклинан`)
