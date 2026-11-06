package ingestion

import (
	"strings"
	"testing"
	"time"
)

func TestExplicitFieldsStayBoundedAndDoNotInventMechanics(t *testing.T) {
	d := Document{Path: "/data/wow/item/117", Namespace: "static-1.60.1_70399-forever-eu", Locale: "en_GB", Payload: map[string]any{
		"name": "Tough Jerky", "required_level": float64(1), "quality": map[string]any{"name": "Common", "type": "COMMON"},
		"preview_item": map[string]any{"spells": []any{map[string]any{"description": "Restores 58 health over 18 sec.", "key": map[string]any{"href": "https://ignored.example/asset"}}}},
		"media":        map[string]any{"id": float64(117)},
	}}
	b := Normalize(d, "eu", time.Now())
	if len(b.Entities) != 1 {
		t.Fatal(b)
	}
	text := b.Entities[0].Description
	for _, want := range []string{"required_level=1", "Common", "Restores 58 health", "partial projection"} {
		if !strings.Contains(text, want) {
			t.Fatal(text)
		}
	}
	for _, bad := range []string{"ignored.example", "drop_chance", "available_in_game", "media"} {
		if strings.Contains(text, bad) {
			t.Fatal(text)
		}
	}
	d.Payload["description"] = strings.Repeat("x", 10000)
	if len(Normalize(d, "eu", time.Now()).Entities[0].Description) > 2100 {
		t.Fatal("unbounded projection")
	}
}
