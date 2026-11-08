package storage

import (
	"strings"
	"testing"
)

func TestTrainingOptionsRespectLevelAndExcludeHistoricalAbilities(t *testing.T) {
	raw := []byte(`{"data":{"abilities":[{"name":"Fireball","learned":"trainer","source":"Classic fallback","ranks":[{"rank":"1","level":1,"spellId":133},{"rank":"2","level":6,"spellId":143}]},{"name":"Frostbolt","learned":"trainer","ranks":[{"rank":"1","level":4,"spellId":116},{"rank":"2","level":8,"spellId":205}]},{"name":"Chilled","learned":"trainer","ranks":[{"rank":"","level":1,"spellId":12484}]},{"name":"Historical spell","learned":"sod","ranks":[{"level":1,"spellId":999}]},{"name":"Hidden spell","learned":"trainer","announced":false,"ranks":[{"level":1,"spellId":998}]}]}}`)
	rows, err := trainingOptions(raw, 3)
	if err != nil || len(rows) != 2 {
		t.Fatal(err, rows)
	}
	if rows[0].SpellID != 133 || rows[1].SpellID != 116 || rows[1].Level != 4 || !strings.Contains(string(rows[0].Origin["source"]), "Classic") {
		t.Fatal(rows)
	}
	rows, err = trainingOptions(raw, 6)
	if err != nil || len(rows) != 3 {
		t.Fatal(err, rows)
	}
	for _, r := range rows {
		if r.Timing == "next_listed_training_level" && r.Level != 8 {
			t.Fatal(r)
		}
	}
}
