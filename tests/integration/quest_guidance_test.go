package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/AHeige/Jornia/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"testing"
	"time"
)

func TestVerifiedQuestFollowupsAreExactBuildFreshAndProviderIndependent(t *testing.T) {
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
	schema := fmt.Sprintf("jornia_guidance_%d", time.Now().UnixNano())
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
	store := &storage.Store{Pool: pool, ExcludeForeverDB: true}
	if err = store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	for i, c := range []struct {
		id, status, build string
		expired           bool
	}{{"verified", "verified", "70790", false}, {"reviewed", "source_checked", "70790", false}, {"wrong-build", "verified", "70862", false}, {"expired", "verified", "70790", true}, {"foreverdb:test", "verified", "70790", false}} {
		expiry := time.Now().Add(time.Hour)
		if c.expired {
			expiry = time.Now().Add(-time.Hour)
		}
		prov, _ := json.Marshal(map[string]string{"effective_version": c.build})
		_, err = pool.Exec(ctx, `INSERT INTO sources(id,title,uri,revision,license,validation_status,provenance,expires_at) VALUES($1,'Synthetic verified source','https://example.test','test','test',$2,$3,$4)`, c.id, c.status, prov, expiry)
		if err != nil {
			t.Fatal(err)
		}
		_, err = pool.Exec(ctx, `INSERT INTO quest_followups VALUES('forever','70790',376,$1,$2,$3)`, 100+i, "Synthetic next quest", c.id)
		if err != nil {
			t.Fatal(err)
		}
	}
	g, err := store.QuestGuidance(ctx, "forever", "70790", []int{376})
	if err != nil || len(g.Edges) != 1 || len(g.Sources) != 1 || g.Sources[0].ID != "verified" {
		t.Fatal(g, err)
	}
	for _, args := range []struct {
		version, build string
		ids            []int
	}{{"forever", "70862", []int{376}}, {"classic", "70790", []int{376}}, {"forever", "70790", []int{999}}} {
		g, err = store.QuestGuidance(ctx, args.version, args.build, args.ids)
		if err != nil || len(g.Edges) != 0 {
			t.Fatal("cross-scope guidance", g, err)
		}
	}
}
