package foreverdb

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"wisp/internal/storage"
)

type Worker struct {
	Store       *storage.Store
	Client      *Client
	MaxRequests int
}
type Report struct {
	Status    string `json:"status"`
	Build     string `json:"build"`
	Requests  int    `json:"requests"`
	Documents int    `json:"documents"`
	Records   int    `json:"projected_records"`
	Pending   int    `json:"pending"`
}

var Seeds = []string{"/api/v1/quests/376", "/api/v1/quests/3901", "/api/v1/classes/mage", "/api/v1/quests?zone=Tirisfal&limit=100", "/api/v1/quests?zone=Ashenvale&limit=100", "/api/v1/classes", "/api/v1/professions", "/api/v1/zones?limit=100", "/api/v1/dungeons?limit=100", "/api/v1/battlegrounds", "/api/v1/vendors?limit=100", "/api/v1/itemsets?limit=100", "/api/v1/quests?limit=100", "/api/v1/items?limit=100", "/api/v1/spells?limit=100", "/api/v1/changes-seen?limit=100", "/api/v1/builds?limit=100", "/api/v1/buffs-nerfs?limit=100", "/api/v1/merchants-favor", "/api/v1/poisons", "/api/v1/disenchant", "/api/v1/gathering/herbalism", "/api/v1/gathering/mining", "/api/v1/gathering/skinning", "/api/v1/gathering/fishing", "/api/v1/articles?limit=100", "/api/v1/posts?limit=100"}

func (w *Worker) enqueue(ctx context.Context, path, namespace string, priority int) error {
	if !AllowedPath(path) {
		return errors.New("unapproved ForeverDB path")
	}
	_, err := w.Store.Pool.Exec(ctx, `INSERT INTO api_ingest_jobs(path,namespace,locale,priority) VALUES($1,$2,'en',$3) ON CONFLICT(path,namespace,locale) DO UPDATE SET priority=least(api_ingest_jobs.priority,excluded.priority)`, path, namespace, priority)
	return err
}
func (w *Worker) reserve(ctx context.Context) error {
	var n int
	err := w.Store.Pool.QueryRow(ctx, `INSERT INTO provider_daily_usage(provider,utc_day,requests) VALUES('foreverdb',(now() AT TIME ZONE 'UTC')::date,1) ON CONFLICT(provider,utc_day) DO UPDATE SET requests=provider_daily_usage.requests+1 WHERE provider_daily_usage.requests<8000 RETURNING requests`).Scan(&n)
	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New("provider daily safety budget reached")
	}
	return err
}
func (w *Worker) Run(ctx context.Context) (report Report, runErr error) {
	report.Status = "running"
	if w.MaxRequests <= 0 {
		w.MaxRequests = 200
	}
	if w.MaxRequests > 600 {
		return report, errors.New("ForeverDB batch limit exceeds 600")
	}
	lock, err := w.Store.Pool.Acquire(ctx)
	if err != nil {
		return report, err
	}
	defer lock.Release()
	var locked bool
	if err = lock.QueryRow(ctx, `SELECT pg_try_advisory_lock(hashtextextended(current_database()||':'||current_schema()||':wisp-foreverdb',0))`).Scan(&locked); err != nil {
		return report, err
	}
	if !locked {
		report.Status = "already_running"
		return report, nil
	}
	defer func() {
		c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		lock.Exec(c, `SELECT pg_advisory_unlock(hashtextextended(current_database()||':'||current_schema()||':wisp-foreverdb',0))`)
	}()
	var notBefore int64
	_ = w.Store.Pool.QueryRow(ctx, `SELECT value::bigint FROM api_ingest_state WHERE key='foreverdb:not-before'`).Scan(&notBefore)
	if notBefore > time.Now().Unix() {
		report.Status = "provider_cooldown"
		return report, nil
	}
	var runID int64
	if err = w.Store.Pool.QueryRow(ctx, `INSERT INTO ingestion_runs(provider,status) VALUES('foreverdb','running') RETURNING id`).Scan(&runID); err != nil {
		return report, err
	}
	defer func() {
		code := ""
		if runErr != nil {
			report.Status = "failed"
			code = "foreverdb_import_failed"
		}
		c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		w.Store.Pool.Exec(c, `UPDATE ingestion_runs SET finished_at=now(),status=$1,requests=$2,documents=$3,error_code=nullif($4,'') WHERE id=$5`, report.Status, report.Requests, report.Documents, code, runID)
	}()
	fetch := func(path string) (Document, error) {
		if e := w.reserve(ctx); e != nil {
			report.Status = "daily_safety_budget"
			return Document{}, e
		}
		report.Requests++
		return w.Client.Fetch(ctx, path)
	}
	meta, err := fetch("/api/v1/meta")
	if err != nil {
		w.cooldown(ctx, err)
		return report, err
	}
	report.Build = meta.Meta.Build
	namespace := "foreverdb-" + report.Build
	// Only expose this provider's current build. Other sources keep their own version boundaries.
	if _, err = w.Store.Pool.Exec(ctx, `INSERT INTO api_ingest_state(key,value) VALUES('foreverdb:build',$1) ON CONFLICT(key) DO UPDATE SET value=excluded.value,updated_at=now()`, report.Build); err != nil {
		return report, err
	}
	// Retain provider catalog totals separately from projected entities.
	if _, err = w.Save(ctx, "/api/v1/meta", meta); err != nil {
		return report, err
	}
	report.Documents++
	// Replan from cached responses too: a newer importer can discover independent
	// pages without refetching an already current catalog or bypassing backoff.
	cached, err := w.Store.Pool.Query(ctx, `SELECT path,payload FROM api_ingest_documents WHERE namespace=$1 AND expires_at>now()`, namespace)
	if err != nil {
		return report, err
	}
	planned := map[string]int{}
	for cached.Next() {
		var path string
		var payload []byte
		if err = cached.Scan(&path, &payload); err != nil {
			cached.Close()
			return report, err
		}
		var d Document
		if json.Unmarshal(payload, &d) == nil {
			for path, p := range Links(path, d) {
				planned[path] = p
			}
		}
	}
	err = cached.Err()
	cached.Close()
	if err != nil {
		return report, err
	}
	for path, p := range planned {
		if err = w.enqueue(ctx, path, namespace, p); err != nil {
			return report, err
		}
	}
	for i, path := range Seeds {
		priority := 5
		if i < 5 {
			priority = 0
		}
		if err = w.enqueue(ctx, path, namespace, priority); err != nil {
			return report, err
		}
	}
	transientErrors := 0
	for report.Requests < w.MaxRequests {
		var path string
		err = w.Store.Pool.QueryRow(ctx, `SELECT path FROM api_ingest_jobs WHERE namespace=$1 AND locale='en' AND next_attempt<=now() ORDER BY priority,next_attempt,path LIMIT 1`, namespace).Scan(&path)
		if errors.Is(err, pgx.ErrNoRows) {
			break
		}
		if err != nil {
			return report, err
		}
		doc, e := fetch(path)
		if e != nil {
			if report.Status == "daily_safety_budget" {
				break
			}
			var he HTTPError
			status := 0
			if errors.As(e, &he) {
				status = he.Status
			}
			delay := time.Hour
			if status == 404 {
				delay = 24 * time.Hour
			}
			_, err = w.Store.Pool.Exec(ctx, `UPDATE api_ingest_jobs SET status='retry',checked_at=now(),http_status=$1,next_attempt=now()+$2 * interval '1 second' WHERE path=$3 AND namespace=$4 AND locale='en'`, status, int64(delay/time.Second), path, namespace)
			if err != nil {
				return report, err
			}
			if status == 404 {
				continue
			}
			if status >= 500 {
				transientErrors++
				if transientErrors < 3 {
					continue
				}
			}
			w.cooldown(ctx, e)
			report.Status = "provider_paused"
			break
		}
		if doc.Meta.Build != report.Build {
			return report, errors.New("ForeverDB build changed during batch; no mixed-build projection")
		}
		transientErrors = 0
		count, e := w.Save(ctx, path, doc)
		if e != nil {
			return report, e
		}
		report.Documents++
		report.Records += count
		for next, priority := range Links(path, doc) {
			if err = w.enqueue(ctx, next, namespace, priority); err != nil {
				return report, err
			}
		}
	}
	if err = w.Store.Pool.QueryRow(ctx, `SELECT count(*) FROM api_ingest_jobs WHERE namespace=$1 AND (status<>'current' OR next_attempt<=now())`, namespace).Scan(&report.Pending); err != nil {
		return report, err
	}
	if report.Status == "running" {
		report.Status = "batch_complete"
	}
	return report, nil
}
func (w *Worker) cooldown(ctx context.Context, e error) {
	delay := time.Hour
	var he HTTPError
	if errors.As(e, &he) && he.Status == 429 {
		if sec, err := strconv.Atoi(he.RetryAfter); err == nil && sec > 0 {
			delay = time.Duration(min(sec, 86400)) * time.Second
		} else if until, err := http.ParseTime(he.RetryAfter); err == nil && time.Until(until) > 0 {
			delay = time.Until(until)
		}
	}
	w.Store.Pool.Exec(ctx, `INSERT INTO api_ingest_state(key,value) VALUES('foreverdb:not-before',$1) ON CONFLICT(key) DO UPDATE SET value=excluded.value,updated_at=now()`, strconv.FormatInt(time.Now().Add(delay).Unix(), 10))
}
func Links(path string, d Document) map[string]int {
	out := map[string]int{}
	if d.Meta.Next != "" {
		out[d.Meta.Next] = 8
	}
	u, _ := url.Parse(path)
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	category := parts[2]
	// A failed page must not prevent importing other documented pages in the same catalog.
	if d.Meta.Total > 100 && d.Meta.Total <= 200000 && (u.Query().Get("offset") == "" || u.Query().Get("offset") == "0") {
		for offset := 100; offset < d.Meta.Total; offset += 100 {
			q := u.Query()
			q.Set("limit", "100")
			q.Set("offset", strconv.Itoa(offset))
			page := u.Path + "?" + q.Encode()
			if AllowedPath(page) {
				out[page] = 8
			}
		}
	}
	if len(parts) != 3 {
		return out
	}
	for _, v := range array(d.Data) {
		r := object(v)
		var id string
		priority := 30
		switch category {
		case "classes":
			id = strings.ToLower(str(r["name"]))
			priority = 1
		case "professions":
			id = str(r["slug"])
			priority = 1
		case "quests":
			id = identity(r["id"])
			priority = 12
		case "dungeons", "itemsets", "vendors":
			id = identity(r["id"])
			priority = 18
		case "items", "spells":
			id = identity(r["id"])
		default:
			continue
		}
		if id != "" {
			p := "/api/v1/" + category + "/" + url.PathEscape(id)
			if category == "professions" {
				p += "?limit=100"
			}
			if AllowedPath(p) {
				out[p] = priority
			}
		}
	}
	return out
}
func (w *Worker) Save(ctx context.Context, path string, d Document) (int, error) {
	now := time.Now().UTC()
	b := Normalize(path, d, now)
	raw, _ := json.Marshal(d)
	sum := fmt.Sprintf("%x", sha256.Sum256(raw))
	tx, err := w.Store.Pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `INSERT INTO api_ingest_documents(path,namespace,locale,checksum,payload,fetched_at,expires_at,projection_version) VALUES($1,$2,'en',$3,$4,$5,$6,2) ON CONFLICT(path,namespace,locale) DO UPDATE SET checksum=excluded.checksum,payload=excluded.payload,fetched_at=excluded.fetched_at,expires_at=excluded.expires_at,projection_version=2`, path, "foreverdb-"+d.Meta.Build, sum, raw, now, b.Source.ExpiresAt)
	if err != nil {
		return 0, err
	}
	s := b.Source
	_, err = tx.Exec(ctx, `INSERT INTO sources(id,title,uri,revision,license,validation_status,provenance,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(id) DO UPDATE SET revision=excluded.revision,provenance=excluded.provenance,expires_at=excluded.expires_at`, s.ID, s.Title, s.URI, s.Revision, s.License, s.ValidationStatus, s.Provenance, s.ExpiresAt)
	if err != nil {
		return 0, err
	}
	ids := []string{}
	for _, e := range b.Entities {
		ids = append(ids, e.ID)
	}
	if _, err = tx.Exec(ctx, `DELETE FROM edges WHERE source_id=$1 OR (subject_id,game_version) IN(SELECT id,game_version FROM entities WHERE source_id=$1 AND NOT(id=ANY($2))) OR (object_id,game_version) IN(SELECT id,game_version FROM entities WHERE source_id=$1 AND NOT(id=ANY($2)))`, s.ID, ids); err != nil {
		return 0, err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM entities WHERE source_id=$1 AND NOT(id=ANY($2))`, s.ID, ids); err != nil {
		return 0, err
	}
	for _, e := range b.Entities {
		_, err = tx.Exec(ctx, `INSERT INTO entities(id,game_version,kind,name,description,spoiler,source_id) VALUES($1,'forever',$2,$3,$4,false,$5) ON CONFLICT(id,game_version) DO UPDATE SET name=excluded.name,description=excluded.description,source_id=excluded.source_id WHERE entities.source_id=excluded.source_id OR position(':detail:' in excluded.source_id)>0 OR position(':list:' in entities.source_id)>0`, e.ID, e.Kind, e.Name, e.Description, e.SourceID)
		if err != nil {
			return 0, err
		}
	}
	for _, e := range b.Edges {
		_, err = tx.Exec(ctx, `INSERT INTO edges(id,game_version,subject_id,predicate,object_id,source_id) VALUES($1,'forever',$2,$3,$4,$5) ON CONFLICT(id,game_version) DO UPDATE SET source_id=excluded.source_id`, e.ID, e.SubjectID, e.Predicate, e.ObjectID, e.SourceID)
		if err != nil {
			return 0, err
		}
	}
	delay := refreshDelay(path)
	_, err = tx.Exec(ctx, `UPDATE api_ingest_jobs SET status='current',checked_at=now(),http_status=200,next_attempt=now()+$1 * interval '1 second' WHERE path=$2 AND namespace=$3 AND locale='en'`, int64(delay/time.Second), path, "foreverdb-"+d.Meta.Build)
	if err != nil {
		return 0, err
	}
	return len(b.Entities), tx.Commit(ctx)
}

// A 60k-detail catalog cannot be revisited daily within an 8k/day budget.
// Prefer daily catalog discovery and critical class/profession data, with dated
// quest details weekly and the large item/spell tail every three weeks.
func refreshDelay(path string) time.Duration {
	p := strings.Split(strings.Split(path, "?")[0], "/")
	if len(p) == 4 {
		return 24 * time.Hour
	}
	if len(p) > 4 {
		switch p[3] {
		case "classes", "professions", "gathering":
			return 24 * time.Hour
		case "quests":
			return 7 * 24 * time.Hour
		}
	}
	return 21 * 24 * time.Hour
}
