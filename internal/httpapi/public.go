package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"path/filepath"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Exact allowlist: static visuals must not expose arbitrary files in the release directory.
var publicAssets = map[string]string{
	"/media/character-preview.jpg":      "media/character-preview.jpg",
	"/media/expedition.jpg":             "media/expedition.jpg",
	"/media/wisp.png":                   "media/wisp.png",
	"/media/ingame-chat-preview.jpg":    "media/ingame-chat-preview.jpg",
	"/media/ingame-journey-preview.jpg": "media/ingame-journey-preview.jpg",
	"/download.css":                     "download.css",
	"/media/forest.jpg":                 "media/forest.jpg",
	"/media/addon-preview.jpg":          "media/addon-preview.jpg",
	"/media/journey-preview.jpg":        "media/journey-preview.jpg",
}

var requestID = regexp.MustCompile(`^[a-zA-Z0-9_-]{16,80}$`)
var releaseName = regexp.MustCompile(`^wisp-[0-9]+\.[0-9]+\.[0-9]+-(windows-amd64|darwin-arm64)\.zip(\.sha256)?$`)

// Public exposes only the tester API and explicitly named distribution files.
// It never serves database files, journals, environment files or private endpoints.
func Public(api http.Handler, pool *pgxpool.Pool, releases string, profiles ...http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		if file, ok := publicAssets[r.URL.Path]; ok && r.Method == "GET" {
			http.ServeFile(w, r, filepath.Join(releases, file))
			return
		}
		if r.Method == "GET" && r.URL.Path == "/style.css" {
			http.ServeFile(w, r, filepath.Join(releases, "style.css"))
			return
		}
		if r.Method == "GET" && r.URL.Path == "/" {
			http.ServeFile(w, r, filepath.Join(releases, "index.html"))
			return
		}
		if r.Method == "GET" && len(r.URL.Path) > 10 && r.URL.Path[:10] == "/releases/" {
			name := r.URL.Path[10:]
			if !releaseName.MatchString(name) {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
			http.ServeFile(w, r, filepath.Join(releases, name))
			return
		}
		if !(r.Method == "GET" && r.URL.Path == "/healthz" || r.Method == "POST" && (r.URL.Path == "/api/wisp/ask" || r.URL.Path == "/api/wisp/character" && len(profiles) == 1)) {
			http.NotFound(w, r)
			return
		}
		token := r.Header.Get("Authorization")
		if len(token) != 7+64 || token[:7] != "Bearer " {
			write(w, 401, map[string]string{"error": "test access required"})
			return
		}
		hash := sha256.Sum256([]byte(token[7:]))
		digest := hex.EncodeToString(hash[:])
		ctx := r.Context()
		tx, err := pool.Begin(ctx)
		if err != nil {
			write(w, 503, map[string]string{"error": "access unavailable"})
			return
		}
		defer tx.Rollback(ctx)
		var limit int
		err = tx.QueryRow(ctx, `SELECT daily_limit FROM tester_access WHERE token_hash=$1 AND NOT revoked AND expires_at>now() FOR UPDATE`, digest).Scan(&limit)
		if err != nil {
			write(w, 401, map[string]string{"error": "test access expired or invalid"})
			return
		}
		if r.Method == "POST" {
			id := r.Header.Get("X-Wisp-Request-ID")
			if !requestID.MatchString(id) {
				write(w, 400, map[string]string{"error": "valid request identity required"})
				return
			}
			var exists bool
			if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM tester_requests WHERE token_hash=$1 AND request_id=$2)`, digest, id).Scan(&exists); err != nil {
				write(w, 503, map[string]string{"error": "access unavailable"})
				return
			}
			if exists {
				write(w, 409, map[string]string{"error": "request already accepted; automatic replay is disabled"})
				return
			}
			var count int
			err = tx.QueryRow(ctx, `SELECT count(*) FROM tester_requests WHERE token_hash=$1 AND created_at>=date_trunc('day',now() AT TIME ZONE 'UTC') AT TIME ZONE 'UTC'`, digest).Scan(&count)
			if err != nil {
				write(w, 503, map[string]string{"error": "access unavailable"})
				return
			}
			if count >= limit {
				w.Header().Set("Retry-After", "3600")
				write(w, 429, map[string]string{"error": "daily test limit reached"})
				return
			}
			if _, err = tx.Exec(ctx, `INSERT INTO tester_requests(token_hash,request_id) VALUES($1,$2)`, digest, id); err != nil {
				write(w, 503, map[string]string{"error": "access unavailable"})
				return
			}
		}
		if err = tx.Commit(ctx); err != nil {
			write(w, 503, map[string]string{"error": "access unavailable"})
			return
		}
		// Deadlines, output budgets and global inference limits remain enforced by the API.
		ctx, cancel := context.WithTimeout(ctx, 38*time.Second)
		defer cancel()
		if r.URL.Path == "/api/wisp/character" {
			profiles[0].ServeHTTP(w, r.WithContext(ctx))
			return
		}
		api.ServeHTTP(w, r.WithContext(ctx))
	})
}
