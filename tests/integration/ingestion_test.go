package integration

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"wisp/internal/ingestion"
	"wisp/internal/storage"
	"wisp/internal/wisp"
)

func TestOfficialImportRetrievalAndExpiry(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("requires test PostgreSQL")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("wisp_ingest_%d", time.Now().UnixNano())
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE")
	cfg, _ := pgxpool.ParseConfig(dsn)
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	cfg.MaxConns = 4
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	store := &storage.Store{Pool: pool}
	if err = store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err = store.Seed(ctx); err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `INSERT INTO api_ingest_state(key,value) VALUES('namespace:eu:en_GB','static-1.60.1_70399-forever-eu')`)
	if err != nil {
		t.Fatal(err)
	}
	worker := ingestion.Worker{Store: store, Client: &ingestion.Client{Region: "eu", Locale: "en_GB"}}
	// A growing API catalog must not shadow a same-named curated relationship.
	duplicate := ingestion.Document{Path: "/data/wow/quest/999991", Namespace: "static-1.60.1_70399-forever-eu", Locale: "en_GB", Payload: map[string]any{"title": "Raene's Cleansing"}}
	if err = worker.Save(ctx, duplicate); err != nil {
		t.Fatal(err)
	}
	duplicateService := wisp.Service{Store: store}
	duplicateAnswer, err := duplicateService.Ask(ctx, wisp.Request{Question: "Who gives Raene's Cleansing?", GameVersion: "forever", Mode: "auto"})
	if err != nil || duplicateAnswer.Status != "answered" || !strings.Contains(duplicateAnswer.Text, "Raene Wolfrunner") {
		t.Fatal("catalog hid curated quest giver", duplicateAnswer, err)
	}
	parenthetical, err := duplicateService.Ask(ctx, wisp.Request{Question: "Who do I turn in The Ashenvale Hunt (The Barrens) to?", GameVersion: "forever", Mode: "auto"})
	if err != nil || parenthetical.Status != "answered" {
		t.Fatal("punctuation hid curated quest", parenthetical, err)
	}
	// More records than the old whole-scope reader's first-100 window.
	for i := 1; i <= 240; i++ {
		d := ingestion.Document{Path: fmt.Sprintf("/data/wow/creature/%d", i), Namespace: "static-1.60.1_70399-forever-eu", Locale: "en_GB", Payload: map[string]any{"name": fmt.Sprintf("Archive Creature %d", i), "id": float64(i)}}
		if err = worker.Save(ctx, d); err != nil {
			t.Fatal(err)
		}
	}
	service := wisp.Service{Store: store}
	answer, err := service.Ask(ctx, wisp.Request{Question: "Vad är Archive Creature 240?", GameVersion: "forever", Mode: "auto"})
	if err != nil || answer.Status != "answered" || len(answer.Sources) != 1 || answer.Sources[0].ExpiresAt == nil {
		t.Fatalf("retrieval beyond 100: %+v %v", answer, err)
	}
	// A shared numeric ID in a different build is a distinct fact, never a silent merge.
	newer := ingestion.Document{Path: "/data/wow/creature/240", Namespace: "static-1.60.1_70790-forever-eu", Locale: "en_GB", Payload: map[string]any{"name": "Archive Creature 240", "level": float64(50)}}
	if err = worker.Save(ctx, newer); err != nil {
		t.Fatal(err)
	}
	answer, err = service.Ask(ctx, wisp.Request{Question: "Vad är Archive Creature 240?", GameVersion: "forever", Mode: "auto"})
	if err != nil || strings.Contains(answer.Text, "level=50") {
		t.Fatal("unpublished build entered serving graph")
	}
	_, _ = pool.Exec(ctx, `UPDATE api_ingest_state SET value=$1 WHERE key='namespace:eu:en_GB'`, newer.Namespace)
	answer, err = service.Ask(ctx, wisp.Request{Question: "Vad är Archive Creature 240?", GameVersion: "forever", Mode: "auto"})
	if err != nil || !strings.Contains(answer.Text, "level=50") || len(answer.Sources) != 1 {
		t.Fatal("build-scoped serving failed")
	}
	_, _ = pool.Exec(ctx, `UPDATE api_ingest_state SET value='static-1.60.1_70399-forever-eu' WHERE key='namespace:eu:en_GB'`)
	doc := ingestion.Document{Path: "/data/wow/quest/1001", Namespace: "static-1.60.1_70399-forever-eu", Locale: "en_GB", Payload: map[string]any{"title": "A linked quest", "rewards": map[string]any{"items": []any{map[string]any{"name": "Proof Sword", "key": map[string]any{"href": "https://eu.api.blizzard.com/data/wow/item/1002?namespace=static-1.60.1_70399-forever-eu"}}}}}}
	if err = worker.Save(ctx, doc); err != nil {
		t.Fatal(err)
	}
	item := ingestion.Document{Path: "/data/wow/item/1002", Namespace: doc.Namespace, Locale: doc.Locale, Payload: map[string]any{"name": "Proof Sword", "item_level": float64(42)}}
	if err = worker.Save(ctx, item); err != nil {
		t.Fatal(err)
	}
	if err = worker.Save(ctx, doc); err != nil {
		t.Fatal(err)
	}
	g, err := store.Retrieve(ctx, "forever", false, "Berätta om Proof Sword")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range g.Entities {
		if e.Kind == "item" {
			found = strings.Contains(e.Description, "item_level=42")
		}
	}
	if !found {
		t.Fatal("index reference overwrote resource detail")
	}
	_, err = pool.Exec(ctx, `UPDATE sources SET expires_at=now()-interval '1 minute' WHERE id LIKE 'blizzard-api:%'`)
	if err != nil {
		t.Fatal(err)
	}
	answer, err = service.Ask(ctx, wisp.Request{Question: "Vad är Archive Creature 240?", GameVersion: "forever", Mode: "auto"})
	if err != nil || answer.Status != "unknown" {
		t.Fatal("expired API fact served")
	}
	_, _ = pool.Exec(ctx, `UPDATE api_ingest_documents SET expires_at=now()-interval '1 minute'`)
	report, err := worker.Run(ctx)
	if err != nil || report.Status != "awaiting_credentials" || report.Requests != 0 {
		t.Fatalf("no-credential purge %+v %v", report, err)
	}
	var count int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM entities WHERE id LIKE 'blizzard-api:%'`).Scan(&count)
	if count != 0 {
		t.Fatal("expired records retained")
	}
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM entities WHERE game_version='forever'`).Scan(&count)
	if count != 49 {
		t.Fatal("curated graph changed by API purge")
	}
}
