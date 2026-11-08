package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/AHeige/Jornia/internal/foreverdb"
	"github.com/AHeige/Jornia/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"testing"
	"time"
)

func TestQuestIdentityRetrievalKeepsBuildExpiryAndProviderBoundaries(t *testing.T) {
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
	schema := fmt.Sprintf("jornia_quest_identity_%d", time.Now().UnixNano())
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
	s := &storage.Store{Pool: pool}
	if err = s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	w := foreverdb.Worker{Store: s}
	var d foreverdb.Document
	if err = json.Unmarshal([]byte(`{"data":{"id":376,"name":"English quest title","ends":{"type":"npc","name":"Test Recipient","location":{"zone":"Test Zone","x":10,"y":20},"source":"Classic, unconfirmed in Forever"}},"meta":{"build":"1.60.1.70862","client":"test"}}`), &d); err != nil {
		t.Fatal(err)
	}
	if _, err = w.Save(ctx, "/api/v1/quests/376", d); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO api_ingest_state(key,value) VALUES('foreverdb:build','1.60.1.70862')`); err != nil {
		t.Fatal(err)
	}
	for _, question := range []string{"Vad gör jag nu?", "任务接下来怎么做？", ""} {
		g, e := s.RetrieveQuestContext(ctx, "forever", false, question, []int{376, 376})
		if e != nil || len(g.Anchors) != 1 || len(g.Edges) != 1 || len(g.Entities) != 2 {
			t.Fatalf("ID lookup: %v %+v", e, g)
		}
		if g.Sources[0].ValidationStatus != "source_checked" {
			t.Fatal("promoted source")
		}
	}

	if err = json.Unmarshal([]byte(`{"data":{"id":8,"name":"Mage","abilities":[{"name":"Fireball","learned":"trainer","ranks":[{"rank":"Rank 1","level":1,"spellId":133}]},{"name":"Frostbolt","learned":"trainer","ranks":[{"rank":"Rank 1","level":4,"spellId":116}]}]},"meta":{"build":"1.60.1.70862","client":"test"}}`), &d); err != nil {
		t.Fatal(err)
	}
	if _, err = w.Save(ctx, "/api/v1/classes/mage", d); err != nil {
		t.Fatal(err)
	}
	classGraph, e := s.ClassOptions(ctx, "forever", "Mage", 3)
	if e != nil || len(classGraph.Entities) != 2 || len(classGraph.Sources) != 1 {
		t.Fatal("numeric class identity", e, classGraph)
	}
	hiddenClass, e := (&storage.Store{Pool: pool, ExcludeForeverDB: true}).ClassOptions(ctx, "forever", "Mage", 3)
	if e != nil || len(hiddenClass.Entities) != 0 {
		t.Fatal("paused class leaked", e)
	}
	assertHidden := func(store *storage.Store, version string) {
		t.Helper()
		g, e := store.RetrieveQuestContext(ctx, version, false, "Unrelated localized question", []int{376})
		if e != nil || len(g.Entities) != 0 {
			t.Fatalf("identity boundary: %v %+v", e, g)
		}
	}
	assertHidden(&storage.Store{Pool: pool, ExcludeForeverDB: true}, "forever")
	assertHidden(s, "retail")
	if _, err = pool.Exec(ctx, `UPDATE sources SET expires_at=now()-interval '1 second'`); err != nil {
		t.Fatal(err)
	}
	assertHidden(s, "forever")
	hiddenClass, e = s.ClassOptions(ctx, "forever", "Mage", 3)
	if e != nil || len(hiddenClass.Entities) != 0 {
		t.Fatal("expired or old class leaked", e)
	}
	if _, err = pool.Exec(ctx, `UPDATE sources SET expires_at=now()+interval '1 day'; UPDATE api_ingest_state SET value='1.60.1.70863' WHERE key='foreverdb:build'`); err != nil {
		t.Fatal(err)
	}
	assertHidden(s, "forever")
	hiddenClass, e = s.ClassOptions(ctx, "forever", "Mage", 3)
	if e != nil || len(hiddenClass.Entities) != 0 {
		t.Fatal("expired or old class leaked", e)
	}
}
