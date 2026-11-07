package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
	"wisp/internal/foreverdb"
	"wisp/internal/storage"
)

func TestForeverDBIdempotenceBuildIsolationSearchCoverageAndFailures(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("requires PostgreSQL")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("wisp_fdb_%d", time.Now().UnixNano())
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE")
	cfg, _ := pgxpool.ParseConfig(dsn)
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	store := &storage.Store{Pool: pool}
	for i := 0; i < 2; i++ {
		if err = store.Migrate(ctx); err != nil {
			t.Fatal(err)
		}
	}
	worker := foreverdb.Worker{Store: store}
	var doc foreverdb.Document
	json.Unmarshal([]byte(`{"data":{"id":1,"name":"Elixir of Minor Strength","description":"Made with silverleaf"},"meta":{"build":"1.60.1.70790","client":"dev-test"}}`), &doc)
	for i := 0; i < 2; i++ {
		if _, err = worker.Save(ctx, "/api/v1/items/1", doc); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	pool.QueryRow(ctx, "SELECT count(*) FROM entities").Scan(&count)
	if count != 1 {
		t.Fatal("duplicate entity", count)
	}
	pool.Exec(ctx, `INSERT INTO api_ingest_state(key,value) VALUES('foreverdb:build','1.60.1.70790')`)
	g, err := store.Retrieve(ctx, "forever", false, "elixir minor strength")
	if err != nil || len(g.Anchors) != 1 {
		t.Fatal("factual fallback", err, len(g.Anchors))
	}
	coverage, err := store.Coverage(ctx, "forever")
	if err != nil || coverage["complete"] != false {
		t.Fatal("coverage", err)
	}
	pool.Exec(ctx, `UPDATE api_ingest_state SET value='1.60.1.70791' WHERE key='foreverdb:build'`)
	g, err = store.Retrieve(ctx, "forever", false, "Elixir of Minor Strength")
	if err != nil || len(g.Entities) != 0 {
		t.Fatal("old build leaked", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/quests/376":
			w.WriteHeader(404)
			return
		case "/api/v1/quests/3901":
			w.WriteHeader(502)
			return
		}
		fmt.Fprint(w, `{"data":[],"meta":{"build":"1.60.1.70790","client":"dev-test"}}`)
	}))
	defer server.Close()
	worker.Client = &foreverdb.Client{Key: "synthetic", DeveloperID: "dev-test", BaseURL: server.URL, Interval: time.Nanosecond}
	worker.MaxRequests = 4
	r, err := worker.Run(ctx)
	if err != nil || r.Status != "batch_complete" || r.Requests != 4 {
		t.Fatal("isolated 404/502 stopped whole import", r, err)
	}
	var reserved int
	pool.QueryRow(ctx, `SELECT requests FROM provider_daily_usage WHERE provider='foreverdb'`).Scan(&reserved)
	if reserved != 4 {
		t.Fatal("quota reservation mismatch", reserved)
	}
}
