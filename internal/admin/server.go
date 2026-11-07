// Package admin owns Wisp's small, SSH-only invite management surface.
package admin

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

//go:embed web/index.html web/app.js web/style.css
var assets embed.FS

type Server struct {
	Pool     *pgxpool.Pool
	Origin   string
	password [32]byte
	csrf     string
	mu       sync.Mutex
	attempts int
	window   time.Time
}

func New(pool *pgxpool.Pool, origin, password string) (*Server, error) {
	u, e := url.Parse(origin)
	if e != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.Port() == "" || u.Path != "" || u.User != nil || len(password) < 32 {
		return nil, errors.New("admin requires a loopback origin and a strong separate password")
	}
	return &Server{Pool: pool, Origin: origin, password: sha256.Sum256([]byte(password)), csrf: randomToken()}, nil
}
func randomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("secure random unavailable")
	}
	return hex.EncodeToString(b)
}
func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self'; img-src 'self'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'")
	u, _ := url.Parse(s.Origin)
	if r.Host != u.Host {
		http.Error(w, "Unknown host", 403)
		return
	}
	user, password, ok := r.BasicAuth()
	digest := sha256.Sum256([]byte(password))
	if !ok || user != "admin" || subtle.ConstantTimeCompare(digest[:], s.password[:]) != 1 {
		s.mu.Lock()
		if time.Since(s.window) > time.Minute {
			s.window = time.Now()
			s.attempts = 0
		}
		s.attempts++
		attempts := s.attempts
		s.mu.Unlock()
		if attempts > 5 {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "Try again later", 429)
			return
		}
		w.Header().Set("WWW-Authenticate", `Basic realm="Wisp private admin", charset="UTF-8"`)
		http.Error(w, "Admin sign-in required", 401)
		return
	}
	if r.Method != "GET" && (r.Method != "POST" || r.Header.Get("Origin") != s.Origin || r.Header.Get("X-Wisp-CSRF") != s.csrf || !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json")) {
		http.Error(w, "Request rejected", 403)
		return
	}
	switch {
	case r.Method == "GET" && r.URL.Path == "/admin/session":
		write(w, 200, map[string]string{"csrf": s.csrf})
		return
	case r.Method == "GET" && r.URL.Path == "/admin/invites":
		s.list(w, r)
		return
	case r.Method == "POST" && r.URL.Path == "/admin/invites":
		s.create(w, r)
		return
	case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/admin/invites/") && strings.HasSuffix(r.URL.Path, "/revoke"):
		s.revoke(w, r)
		return
	case r.Method == "GET":
		file := map[string]string{"/": "index.html", "/app.js": "app.js", "/style.css": "style.css"}[r.URL.Path]
		if file != "" {
			body, _ := assets.ReadFile("web/" + file)
			types := map[string]string{"index.html": "text/html; charset=utf-8", "app.js": "text/javascript; charset=utf-8", "style.css": "text/css; charset=utf-8"}
			w.Header().Set("Content-Type", types[file])
			w.Write(body)
			return
		}
	}
	http.NotFound(w, r)
}
func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	rows, err := s.Pool.Query(r.Context(), `SELECT a.token_hash,a.label,a.expires_at,a.revoked,a.daily_limit,count(q.request_id) FILTER(WHERE q.created_at>=(now() AT TIME ZONE 'UTC')::date AT TIME ZONE 'UTC'),max(q.created_at) FROM tester_access a LEFT JOIN tester_requests q ON q.token_hash=a.token_hash GROUP BY a.token_hash ORDER BY a.expires_at DESC LIMIT 200`)
	if err != nil {
		write(w, 503, map[string]string{"error": "Invites could not be loaded."})
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, label string
		var expiry time.Time
		var revoked bool
		var limit, usage int
		var last *time.Time
		if rows.Scan(&id, &label, &expiry, &revoked, &limit, &usage, &last) != nil {
			write(w, 503, map[string]string{"error": "Invites could not be loaded."})
			return
		}
		out = append(out, map[string]any{"id": id, "label": label, "expires_at": expiry, "revoked": revoked, "daily_limit": limit, "used_today": usage, "last_used_at": last})
	}
	if rows.Err() != nil {
		write(w, 503, map[string]string{"error": "Invites could not be loaded."})
		return
	}
	write(w, 200, out)
}
func (s *Server) create(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Label      string `json:"label"`
		Days       int    `json:"days"`
		DailyLimit int    `json:"daily_limit"`
	}
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048))
	d.DisallowUnknownFields()
	if d.Decode(&input) != nil {
		write(w, 400, map[string]string{"error": "Check the invite details."})
		return
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		write(w, 400, map[string]string{"error": "One request expected."})
		return
	}
	input.Label = strings.TrimSpace(input.Label)
	if input.Label == "" || len(input.Label) > 80 || !utf8.ValidString(input.Label) || input.Days < 1 || input.Days > 90 || input.DailyLimit < 1 || input.DailyLimit > 500 {
		write(w, 400, map[string]string{"error": "Use a label, 1–90 days and 1–500 requests per day."})
		return
	}
	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		write(w, 503, map[string]string{"error": "Invite was not created."})
		return
	}
	defer tx.Rollback(r.Context())
	if _, err = tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(872331109)`); err != nil {
		write(w, 503, map[string]string{"error": "Invite was not created."})
		return
	}
	var count int
	err = tx.QueryRow(r.Context(), `SELECT count(*) FROM tester_access WHERE NOT revoked AND expires_at>now()`).Scan(&count)
	if err != nil || count >= 100 {
		write(w, 409, map[string]string{"error": "Active invite limit reached, or storage unavailable."})
		return
	}
	token := randomToken()
	digest := sha256.Sum256([]byte(token))
	id := hex.EncodeToString(digest[:])
	expiry := time.Now().UTC().Add(time.Duration(input.Days) * 24 * time.Hour)
	if _, err = tx.Exec(r.Context(), `INSERT INTO tester_access(token_hash,label,expires_at,daily_limit) VALUES($1,$2,$3,$4)`, id, input.Label, expiry, input.DailyLimit); err == nil {
		_, err = tx.Exec(r.Context(), `INSERT INTO invite_admin_audit(action,token_hash,label) VALUES('created',$1,$2)`, id, input.Label)
		if err == nil {
			err = tx.Commit(r.Context())
		}
	}
	if err != nil {
		write(w, 503, map[string]string{"error": "Invite was not created. Refresh the list before retrying."})
		return
	}
	write(w, 201, map[string]any{"id": id, "token": token, "label": input.Label, "expires_at": expiry, "daily_limit": input.DailyLimit})
}

var hashID = regexp.MustCompile(`^[0-9a-f]{64}$`)

func (s *Server) revoke(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/admin/invites/"), "/revoke")
	if !hashID.MatchString(id) {
		write(w, 400, map[string]string{"error": "Unknown invite."})
		return
	}
	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		write(w, 503, map[string]string{"error": "Revocation not confirmed."})
		return
	}
	defer tx.Rollback(r.Context())
	var label string
	err = tx.QueryRow(r.Context(), `UPDATE tester_access SET revoked=true WHERE token_hash=$1 RETURNING label`, id).Scan(&label)
	if err != nil {
		write(w, 404, map[string]string{"error": "Invite not found or revocation unconfirmed."})
		return
	}
	if _, err = tx.Exec(r.Context(), `INSERT INTO invite_admin_audit(action,token_hash,label) VALUES('revoked',$1,$2)`, id, label); err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		write(w, 503, map[string]string{"error": "Revocation not confirmed. Refresh before retrying."})
		return
	}
	write(w, 200, map[string]bool{"revoked": true})
}
