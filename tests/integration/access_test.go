package integration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
	"wisp/internal/httpapi"
	"wisp/internal/storage"
)

func TestTesterAccessIsolationAndLimits(t *testing.T) {
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
	schema := fmt.Sprintf("wisp_access_%d", time.Now().UnixNano())
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
	store := storage.Store{Pool: pool}
	if err = store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	token := strings.Repeat("a", 64)
	hash := sha256.Sum256([]byte(token))
	digest := hex.EncodeToString(hash[:])
	_, err = pool.Exec(ctx, `INSERT INTO tester_access(token_hash,label,expires_at,daily_limit) VALUES($1,'test',now()+interval '1 hour',2)`, digest)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	profileCalls := 0
	handler := httpapi.Public(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(200) }), pool, t.TempDir(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { profileCalls++; w.WriteHeader(200) }))
	check := func(path, token, id string, want int) {
		t.Helper()
		r := httptest.NewRequest("POST", path, strings.NewReader(`{}`))
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		r.Header.Set("X-Wisp-Request-ID", id)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("%s got %d want %d: %s", path, w.Code, want, w.Body.String())
		}
	}
	check("/api/wisp/ask", "", "request-0000000001", 401)
	check("/api/knowledge", token, "request-0000000001", 404)
	check("/api/wisp/ask", token, "request-0000000001", 200)
	check("/api/wisp/ask", token, "request-0000000001", 409)
	check("/api/wisp/character", "", "request-0000000002", 401)
	check("/api/wisp/character", token, "request-0000000002", 200)
	check("/api/wisp/character", token, "request-0000000002", 409)
	check("/api/wisp/ask", token, "request-0000000003", 429)
	if calls != 1 || profileCalls != 1 {
		t.Fatal("unexpected backend calls", calls)
	}
	_, err = pool.Exec(ctx, `UPDATE tester_access SET revoked=true`)
	if err != nil {
		t.Fatal(err)
	}
	check("/api/wisp/ask", token, "request-0000000004", 401)
}
