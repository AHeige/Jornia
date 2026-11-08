package integration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/AHeige/Jornia/internal/admin"
	"github.com/AHeige/Jornia/internal/httpapi"
	"github.com/AHeige/Jornia/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestInviteAdminAuthCSRFHashOnlyAndRevocation(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("requires PostgreSQL")
	}
	ctx := context.Background()
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	schema := fmt.Sprintf("wisp_admin_%d", time.Now().UnixNano())
	if _, err = db.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer db.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE")
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
	password := strings.Repeat("p", 64)
	origin := "http://127.0.0.1:18983"
	app, err := admin.New(pool, origin, password)
	if err != nil {
		t.Fatal(err)
	}
	call := func(method, path, body, csrf string, auth bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Host = "127.0.0.1:18983"
		if auth {
			r.SetBasicAuth("admin", password)
		}
		if method == "POST" {
			r.Header.Set("Origin", origin)
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("X-Wisp-CSRF", csrf)
		}
		w := httptest.NewRecorder()
		app.ServeHTTP(w, r)
		return w
	}
	if w := call("GET", "/admin/invites", "", "", false); w.Code != 401 {
		t.Fatal(w.Code)
	}
	if w := call("POST", "/admin/invites", `{"label":"Friend","days":30,"daily_limit":50}`, "", true); w.Code != 403 {
		t.Fatal("missing CSRF accepted", w.Code)
	}
	w := call("GET", "/admin/session", "", "", true)
	var session map[string]string
	json.Unmarshal(w.Body.Bytes(), &session)
	w = call("POST", "/admin/invites", `{"label":"Friend","days":30,"daily_limit":50}`, session["csrf"], true)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var made map[string]any
	json.Unmarshal(w.Body.Bytes(), &made)
	token := made["token"].(string)
	if len(token) != 64 {
		t.Fatal("bad token")
	}
	hash := sha256.Sum256([]byte(token))
	if hex.EncodeToString(hash[:]) != made["id"] {
		t.Fatal("hash mismatch")
	}
	list := call("GET", "/admin/invites", "", "", true)
	if list.Code != 200 || strings.Contains(list.Body.String(), token) {
		t.Fatal("plaintext code leaked", list.Code)
	}
	public := httpapi.Public(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }), pool, t.TempDir())
	health := func() int {
		r := httptest.NewRequest("GET", "/healthz", nil)
		r.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		public.ServeHTTP(w, r)
		return w.Code
	}
	if health() != 200 {
		t.Fatal("invite not usable")
	}
	w = call("POST", "/admin/invites/"+made["id"].(string)+"/revoke", `{}`, session["csrf"], true)
	if w.Code != 200 || health() != 401 {
		t.Fatal("revocation ineffective", w.Code)
	}
	var count int
	pool.QueryRow(ctx, `SELECT count(*) FROM invite_admin_audit`).Scan(&count)
	if count != 2 {
		t.Fatal("audit missing", count)
	}
}
