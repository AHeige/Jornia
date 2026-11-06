package ingestion

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"wisp/internal/storage"
)

type Worker struct {
	Store          *storage.Store
	Client         *Client
	MaxRequests    int
	ReprojectLimit int
}
type Report struct {
	Status      string `json:"status"`
	Requests    int    `json:"requests"`
	Documents   int    `json:"documents"`
	Entities    int    `json:"normalized_records"`
	Pending     int    `json:"pending"`
	Namespace   string `json:"namespace,omitempty"`
	Reprojected int    `json:"reprojected_documents"`
}

func (w *Worker) Run(ctx context.Context) (report Report, runErr error) {
	report.Status = "running"
	if w.MaxRequests <= 0 {
		w.MaxRequests = 200
	}
	if w.MaxRequests > 2000 {
		return report, errors.New("batch request limit exceeds 2000")
	}
	if w.ReprojectLimit <= 0 {
		w.ReprojectLimit = 200
	}
	if w.ReprojectLimit > 2000 {
		return report, errors.New("cache reprojection limit exceeds 2000")
	}
	if err := w.Client.init(); err != nil {
		return report, err
	}
	lock, err := w.Store.Pool.Acquire(ctx)
	if err != nil {
		return report, err
	}
	defer lock.Release()
	var locked bool
	if err = lock.QueryRow(ctx, `SELECT pg_try_advisory_lock(hashtextextended(current_database()||':'||current_schema()||':wisp-ingestion',0))`).Scan(&locked); err != nil {
		return report, err
	}
	if !locked {
		report.Status = "already_running"
		return report, nil
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = lock.Exec(cleanup, `SELECT pg_advisory_unlock(hashtextextended(current_database()||':'||current_schema()||':wisp-ingestion',0))`)
	}()
	var runID int64
	if err = w.Store.Pool.QueryRow(ctx, `INSERT INTO ingestion_runs(provider,status) VALUES('blizzard','running') RETURNING id`).Scan(&runID); err != nil {
		return report, err
	}
	defer func() {
		code := ""
		if runErr != nil {
			report.Status = "failed"
			code = "ingestion_failed"
		}
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = w.Store.Pool.Exec(cleanup, `UPDATE ingestion_runs SET finished_at=now(),status=$1,requests=$2,documents=$3,error_code=nullif($4,'') WHERE id=$5`, report.Status, report.Requests, report.Documents, code, runID)
	}()
	// TTL is enforced independently of a working credential or upstream availability.
	if err = w.Purge(ctx); err != nil {
		return report, err
	}
	if w.Client.ClientID == "" || w.Client.ClientSecret == "" {
		report.Status = "awaiting_credentials"
		return report, nil
	}
	probe, err := w.Client.Fetch(ctx, "/data/wow/playable-class/index", "static-forever-"+w.Client.Region)
	report.Requests++
	if err != nil {
		return report, err
	}
	report.Namespace = probe.Namespace
	if _, err = w.Store.Pool.Exec(ctx, `INSERT INTO api_ingest_state(key,value) VALUES($1,$2) ON CONFLICT(key) DO UPDATE SET value=excluded.value,updated_at=now()`, "namespace:"+w.Client.Region+":"+w.Client.Locale, probe.Namespace); err != nil {
		return report, err
	}
	// Reproject cached records after a schema change without resetting their original expiry.
	for rebuilt := 0; rebuilt < w.ReprojectLimit; rebuilt++ {
		var d Document
		var raw []byte
		err = w.Store.Pool.QueryRow(ctx, `SELECT path,namespace,locale,payload,fetched_at FROM api_ingest_documents WHERE projection_version<2 AND expires_at>now() AND namespace=$1 AND locale=$2 ORDER BY fetched_at,path LIMIT 1`, probe.Namespace, w.Client.Locale).Scan(&d.Path, &d.Namespace, &d.Locale, &raw, &d.FetchedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			break
		}
		if err != nil {
			return report, err
		}
		if err = json.Unmarshal(raw, &d.Payload); err != nil {
			return report, err
		}
		if err = w.Save(ctx, d); err != nil {
			return report, err
		}
		report.Reprojected++
	}

	if err = w.Save(ctx, probe); err != nil {
		return report, err
	}
	report.Documents++
	report.Entities += len(Normalize(probe, w.Client.Region, time.Now()).Entities)
	// Retire the exploratory unpartitioned search jobs; all current search roots carry explicit ID bounds.
	if _, err = w.Store.Pool.Exec(ctx, `DELETE FROM api_ingest_jobs WHERE namespace=$1 AND locale=$2 AND path LIKE '/data/wow/search/%' AND path !~ '([?&])id='`, probe.Namespace, w.Client.Locale); err != nil {
		return report, err
	}
	for _, path := range Seeds {
		if err = w.enqueue(ctx, path, probe.Namespace); err != nil {
			return report, err
		}
	}
	for report.Requests < w.MaxRequests {
		var path, namespace, locale string
		err = w.Store.Pool.QueryRow(ctx, `SELECT path,namespace,locale FROM api_ingest_jobs WHERE namespace=$1 AND locale=$2 AND next_attempt<=now() ORDER BY CASE WHEN path LIKE '%/index' OR path LIKE '%/search/%' THEN 0 ELSE 1 END,next_attempt,path LIMIT 1`, probe.Namespace, w.Client.Locale).Scan(&path, &namespace, &locale)
		if errors.Is(err, pgx.ErrNoRows) {
			break
		}
		if err != nil {
			return report, err
		}
		d, fetchErr := w.Client.Fetch(ctx, path, namespace)
		report.Requests++
		if fetchErr != nil {
			status := "retry"
			delay := 15 * time.Minute
			if d.Status == 404 {
				status = "not_available"
				delay = 24 * time.Hour
			}
			if d.Status == 429 {
				status = "rate_limited"
				delay = time.Hour
			}
			_, err = w.Store.Pool.Exec(ctx, `UPDATE api_ingest_jobs SET status=$1,http_status=$2,checked_at=now(),next_attempt=now()+$3 * interval '1 second' WHERE path=$4 AND namespace=$5 AND locale=$6`, status, d.Status, int64(delay/time.Second), path, namespace, locale)
			if err != nil {
				return report, err
			}
			if d.Status == 401 || d.Status == 403 || d.Status == 429 || d.Status == 0 {
				report.Status = status
				break
			}
			continue
		}
		if err = w.Save(ctx, d); err != nil {
			return report, err
		}
		report.Documents++
		report.Entities += len(Normalize(d, w.Client.Region, time.Now()).Entities)
	}
	if err = w.Store.Pool.QueryRow(ctx, `SELECT count(*) FROM api_ingest_jobs WHERE namespace=$1 AND locale=$2 AND (status<>'current' OR next_attempt<=now())`, probe.Namespace, w.Client.Locale).Scan(&report.Pending); err != nil {
		return report, err
	}
	if report.Status == "running" {
		report.Status = "batch_complete"
	}
	return report, nil
}
func (w *Worker) enqueue(ctx context.Context, path, namespace string) error {
	if !AllowedPath(path) {
		return nil
	}
	_, err := w.Store.Pool.Exec(ctx, `INSERT INTO api_ingest_jobs(path,namespace,locale) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, path, namespace, w.Client.Locale)
	return err
}
func (w *Worker) Save(ctx context.Context, d Document) error {
	if !namespacePattern.MatchString(d.Namespace) || !strings.HasSuffix(d.Namespace, "-"+w.Client.Region) || !AllowedPath(d.Path) {
		return ErrNamespace
	}
	now := d.FetchedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	bundle := Normalize(d, w.Client.Region, now)
	raw, err := json.Marshal(d.Payload)
	if err != nil {
		return err
	}
	checksum := fmt.Sprintf("%x", sha256.Sum256(raw))
	tx, err := w.Store.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `INSERT INTO api_ingest_documents(path,namespace,locale,checksum,payload,fetched_at,expires_at,projection_version) VALUES($1,$2,$3,$4,$5,$6,$7,2) ON CONFLICT(path,namespace,locale) DO UPDATE SET checksum=excluded.checksum,payload=excluded.payload,fetched_at=excluded.fetched_at,expires_at=excluded.expires_at,projection_version=2`, d.Path, d.Namespace, d.Locale, checksum, raw, now, now.Add(28*24*time.Hour))
	if err != nil {
		return err
	}
	src := bundle.Source
	incoming := []string{}
	for _, e := range bundle.Entities {
		incoming = append(incoming, e.ID)
	}
	// A changed index must not leave old rows claiming the new response as their evidence.
	if _, err = tx.Exec(ctx, `DELETE FROM edges WHERE (subject_id,game_version) IN(SELECT id,game_version FROM entities WHERE source_id=$1 AND NOT(id=ANY($2))) OR (object_id,game_version) IN(SELECT id,game_version FROM entities WHERE source_id=$1 AND NOT(id=ANY($2)))`, src.ID, incoming); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM entities WHERE source_id=$1 AND NOT(id=ANY($2))`, src.ID, incoming); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO sources(id,title,uri,revision,license,validation_status,provenance,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(id) DO UPDATE SET revision=excluded.revision,provenance=excluded.provenance,expires_at=excluded.expires_at`, src.ID, src.Title, src.URI, src.Revision, src.License, src.ValidationStatus, src.Provenance, src.ExpiresAt)
	if err != nil {
		return err
	}
	rootID, _ := identity(d.Path, d.Namespace, d.Locale)
	for _, e := range bundle.Entities {
		conflict := "ON CONFLICT(id,game_version) DO NOTHING"
		if e.ID == rootID {
			conflict = "ON CONFLICT(id,game_version) DO UPDATE SET name=excluded.name,description=excluded.description,source_id=excluded.source_id"
		}
		_, err = tx.Exec(ctx, `INSERT INTO entities(id,game_version,kind,name,description,spoiler,source_id) VALUES($1,'forever',$2,$3,$4,false,$5) `+conflict, e.ID, e.Kind, e.Name, e.Description, src.ID)
		if err != nil {
			return err
		}
	}
	// Replace only this response's references; other sources and curated facts are untouched.
	if _, err = tx.Exec(ctx, `DELETE FROM edges WHERE source_id=$1`, src.ID); err != nil {
		return err
	}
	for _, e := range bundle.Edges {
		_, err = tx.Exec(ctx, `INSERT INTO edges(id,game_version,subject_id,predicate,object_id,source_id) VALUES($1,'forever',$2,$3,$4,$5) ON CONFLICT(id,game_version) DO UPDATE SET source_id=excluded.source_id`, e.ID, e.SubjectID, e.Predicate, e.ObjectID, src.ID)
		if err != nil {
			return err
		}
	}
	_, err = tx.Exec(ctx, `INSERT INTO api_ingest_jobs(path,namespace,locale,status,checked_at,http_status,next_attempt) VALUES($1,$2,$3,'current',$4,200,$4::timestamptz+interval '24 hours') ON CONFLICT(path,namespace,locale) DO UPDATE SET status='current',checked_at=excluded.checked_at,http_status=200,next_attempt=excluded.next_attempt`, d.Path, d.Namespace, d.Locale, now)
	if err != nil {
		return err
	}
	for _, path := range w.Client.Links(d) {
		_, err = tx.Exec(ctx, `INSERT INTO api_ingest_jobs(path,namespace,locale) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, path, d.Namespace, d.Locale)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
func (w *Worker) Purge(ctx context.Context) error {
	tx, err := w.Store.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	// Edges that reference expired entities must be removed before those entities.
	for _, sql := range []string{`DELETE FROM edges WHERE source_id IN(SELECT id FROM sources WHERE expires_at<=now()) OR (subject_id,game_version) IN(SELECT e.id,e.game_version FROM entities e JOIN sources s ON s.id=e.source_id WHERE s.expires_at<=now()) OR (object_id,game_version) IN(SELECT e.id,e.game_version FROM entities e JOIN sources s ON s.id=e.source_id WHERE s.expires_at<=now())`, `DELETE FROM entities WHERE source_id IN(SELECT id FROM sources WHERE expires_at<=now())`, `DELETE FROM sources WHERE expires_at<=now()`, `DELETE FROM api_ingest_documents WHERE expires_at<=now()`, `DELETE FROM api_ingest_jobs WHERE coalesce(checked_at,next_attempt)<now()-interval '28 days'`} {
		if _, err = tx.Exec(ctx, sql); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
