package storage

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql fixtures/*.json datasets/*.json
var files embed.FS

const FixtureVersion = "fixture-ashenvale-v1"

type Source struct {
	ExpiresAt        *time.Time  `json:"expires_at,omitempty"`
	ID               string      `json:"id"`
	Title            string      `json:"title"`
	URI              string      `json:"uri"`
	Revision         string      `json:"revision"`
	License          string      `json:"license"`
	ValidationStatus string      `json:"validation_status"`
	Provenance       *Provenance `json:"provenance,omitempty"`
}

// Provenance distinguishes a source review from verification against a game client.
type Provenance struct {
	CheckedAt          string   `json:"checked_at"`
	EffectiveVersion   string   `json:"effective_version"`
	VerificationMethod string   `json:"verification_method"`
	Attribution        string   `json:"attribution"`
	HistoryURI         string   `json:"history_uri"`
	LicenseURI         string   `json:"license_uri"`
	LicensePolicyURI   string   `json:"license_policy_uri"`
	Changes            string   `json:"changes"`
	ReviewSHA256       string   `json:"review_sha256"`
	ReviewedClaims     []string `json:"reviewed_claims"`
	Redistribution     string   `json:"redistribution"`
}
type Entity struct {
	ID          string `json:"id"`
	GameVersion string `json:"game_version"`
	Kind        string `json:"kind"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Spoiler     bool   `json:"spoiler"`
	SourceID    string `json:"source_id"`
}
type Edge struct {
	ID        string `json:"id"`
	SubjectID string `json:"subject_id"`
	Predicate string `json:"predicate"`
	ObjectID  string `json:"object_id"`
	SourceID  string `json:"source_id"`
}
type Graph struct {
	GameVersion string   `json:"game_version"`
	Entities    []Entity `json:"entities"`
	Edges       []Edge   `json:"edges"`
	Sources     []Source `json:"sources"`
}
type Store struct{ Pool *pgxpool.Pool }

// Migrate applies immutable, checksummed migrations under a transaction lock.
func (s *Store) Migrate(ctx context.Context) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(872331001)`); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (name text PRIMARY KEY, checksum text NOT NULL, applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		return err
	}
	entries, err := files.ReadDir("migrations")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		// macOS external volumes can create AppleDouble sidecars next to SQL files.
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		b, err := files.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return err
		}
		sum := sha256.Sum256(b)
		checksum := hex.EncodeToString(sum[:])
		var old string
		rows, err := tx.Query(ctx, `SELECT checksum FROM schema_migrations WHERE name=$1`, entry.Name())
		if err != nil {
			return err
		}
		if rows.Next() {
			err = rows.Scan(&old)
		}
		rows.Close()
		if err != nil {
			return err
		}
		if rows.Err() != nil {
			return rows.Err()
		}
		if old != "" {
			if old != checksum {
				return fmt.Errorf("migration %s checksum changed", entry.Name())
			}
			continue
		}
		if _, err = tx.Exec(ctx, string(b)); err != nil {
			return fmt.Errorf("migration %s: %w", entry.Name(), err)
		}
		if _, err = tx.Exec(ctx, `INSERT INTO schema_migrations(name,checksum) VALUES($1,$2)`, entry.Name(), checksum); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) Seed(ctx context.Context) error {
	if err := s.seedFile(ctx, "fixtures/ashenvale.json"); err != nil {
		return err
	}
	entries, err := files.ReadDir("datasets")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		if err = s.seedFile(ctx, "datasets/"+entry.Name()); err != nil {
			return fmt.Errorf("seed %s: %w", entry.Name(), err)
		}
	}
	return nil
}

func (s *Store) seedFile(ctx context.Context, path string) error {
	b, err := files.ReadFile(path)
	if err != nil {
		return err
	}
	var data struct {
		Version  string   `json:"version"`
		Revision string   `json:"dataset_revision"`
		Source   Source   `json:"source"`
		Sources  []Source `json:"sources"`
		Entities []Entity `json:"entities"`
		Edges    []Edge   `json:"edges"`
	}
	if err = json.Unmarshal(b, &data); err != nil {
		return err
	}
	if data.Source.ID != "" {
		data.Sources = []Source{data.Source}
	}
	if data.Revision == "" {
		data.Revision = data.Version
	}
	checksum := fmt.Sprintf("%x", sha256.Sum256(b))
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(872331002)`); err != nil {
		return err
	}
	var old string
	rows, err := tx.Query(ctx, `SELECT checksum FROM knowledge_datasets WHERE game_version=$1 AND revision=$2`, data.Version, data.Revision)
	if err != nil {
		return err
	}
	if rows.Next() {
		err = rows.Scan(&old)
	}
	rows.Close()
	if err != nil {
		return err
	}
	if rows.Err() != nil {
		return rows.Err()
	}
	if old != "" {
		if old != checksum {
			return fmt.Errorf("immutable dataset %s checksum changed", data.Revision)
		}
		return tx.Commit(ctx)
	}
	for _, src := range data.Sources {
		if src.ID == "" || src.URI == "" || src.Revision == "" || src.License == "" {
			return fmt.Errorf("source provenance is incomplete")
		}
		if data.Version == "forever" && (src.ValidationStatus != "source_checked" && src.ValidationStatus != "verified") {
			return fmt.Errorf("Forever source must be reviewed")
		}
		if data.Version == "forever" && (src.Provenance == nil || src.Provenance.CheckedAt == "" || src.Provenance.VerificationMethod == "") {
			return fmt.Errorf("Forever source review is incomplete")
		}
		var provenance any = src.Provenance
		if src.Provenance == nil {
			provenance = map[string]any{}
		}
		_, err = tx.Exec(ctx, `INSERT INTO sources(id,title,uri,revision,license,validation_status,provenance) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(id) DO NOTHING`, src.ID, src.Title, src.URI, src.Revision, src.License, src.ValidationStatus, provenance)
		if err != nil {
			return err
		}
	}
	defaultSource := data.Source.ID
	for _, e := range data.Entities {
		if e.SourceID == "" {
			e.SourceID = defaultSource
		}
		_, err = tx.Exec(ctx, `INSERT INTO entities(id,game_version,kind,name,description,spoiler,source_id) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(id,game_version) DO NOTHING`, e.ID, data.Version, e.Kind, e.Name, e.Description, e.Spoiler, e.SourceID)
		if err != nil {
			return err
		}
	}
	for _, e := range data.Edges {
		if e.SourceID == "" {
			e.SourceID = defaultSource
		}
		_, err = tx.Exec(ctx, `INSERT INTO edges(id,game_version,subject_id,predicate,object_id,source_id) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(id,game_version) DO NOTHING`, e.ID, data.Version, e.SubjectID, e.Predicate, e.ObjectID, e.SourceID)
		if err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, `INSERT INTO knowledge_datasets(game_version,revision,checksum) VALUES($1,$2,$3)`, data.Version, data.Revision, checksum); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Graph loads a bounded public knowledge scope. Private player data is not stored here.
func (s *Store) Graph(ctx context.Context, version string, spoilers bool) (Graph, error) {
	g := Graph{GameVersion: version, Entities: []Entity{}, Edges: []Edge{}, Sources: []Source{}}
	rows, err := s.Pool.Query(ctx, `SELECT e.id,e.game_version,e.kind,e.name,e.description,e.spoiler,e.source_id FROM entities e JOIN sources s ON s.id=e.source_id WHERE e.game_version=$1 AND ($2 OR NOT e.spoiler) AND s.validation_status IN ('fixture','source_checked','verified') AND (s.expires_at IS NULL OR s.expires_at>now()) AND (s.id NOT LIKE 'blizzard-api:%' OR s.provenance->>'effective_version' IN(SELECT value FROM api_ingest_state WHERE key LIKE 'namespace:%')) ORDER BY e.id LIMIT 100`, version, spoilers)
	if err != nil {
		return g, err
	}
	for rows.Next() {
		var e Entity
		if err = rows.Scan(&e.ID, &e.GameVersion, &e.Kind, &e.Name, &e.Description, &e.Spoiler, &e.SourceID); err != nil {
			rows.Close()
			return g, err
		}
		g.Entities = append(g.Entities, e)
	}
	rows.Close()
	if rows.Err() != nil {
		return g, rows.Err()
	}
	ids := []string{}
	srcs := map[string]bool{}
	for _, e := range g.Entities {
		ids = append(ids, e.ID)
		srcs[e.SourceID] = true
	}
	rows, err = s.Pool.Query(ctx, `SELECT e.id,e.subject_id,e.predicate,e.object_id,e.source_id FROM edges e JOIN sources s ON s.id=e.source_id WHERE e.game_version=$1 AND e.subject_id=ANY($2) AND e.object_id=ANY($2) AND s.validation_status IN ('fixture','source_checked','verified') AND (s.expires_at IS NULL OR s.expires_at>now()) AND (s.id NOT LIKE 'blizzard-api:%' OR s.provenance->>'effective_version' IN(SELECT value FROM api_ingest_state WHERE key LIKE 'namespace:%')) ORDER BY e.id LIMIT 200`, version, ids)
	if err != nil {
		return g, err
	}
	for rows.Next() {
		var e Edge
		if err = rows.Scan(&e.ID, &e.SubjectID, &e.Predicate, &e.ObjectID, &e.SourceID); err != nil {
			rows.Close()
			return g, err
		}
		g.Edges = append(g.Edges, e)
		srcs[e.SourceID] = true
	}
	rows.Close()
	if rows.Err() != nil {
		return g, rows.Err()
	}
	sourceIDs := []string{}
	for id := range srcs {
		sourceIDs = append(sourceIDs, id)
	}
	sort.Strings(sourceIDs)
	rows, err = s.Pool.Query(ctx, `SELECT id,title,uri,revision,license,validation_status,provenance FROM sources WHERE id=ANY($1) ORDER BY id`, sourceIDs)
	if err != nil {
		return g, err
	}
	for rows.Next() {
		var src Source
		if err = rows.Scan(&src.ID, &src.Title, &src.URI, &src.Revision, &src.License, &src.ValidationStatus, &src.Provenance); err != nil {
			rows.Close()
			return g, err
		}
		g.Sources = append(g.Sources, src)
	}
	rows.Close()
	return g, rows.Err()
}

// Coverage reports installed scope, never a percentage of the complete game universe.
func (s *Store) Coverage(ctx context.Context, version string) (map[string]any, error) {
	counts := map[string]int{}
	rows, err := s.Pool.Query(ctx, `SELECT e.kind,count(*) FROM entities e JOIN sources s ON s.id=e.source_id WHERE e.game_version=$1 AND s.validation_status IN ('source_checked','verified') AND (s.expires_at IS NULL OR s.expires_at>now()) AND (s.id NOT LIKE 'blizzard-api:%' OR s.provenance->>'effective_version' IN(SELECT value FROM api_ingest_state WHERE key LIKE 'namespace:%')) GROUP BY e.kind`, version)
	if err != nil {
		return nil, err
	}
	total := 0
	for rows.Next() {
		var kind string
		var count int
		if err = rows.Scan(&kind, &count); err != nil {
			rows.Close()
			return nil, err
		}
		counts[kind] = count
		total += count
	}
	rows.Close()
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	jobs := map[string]int{}
	rows, err = s.Pool.Query(ctx, `SELECT status,count(*) FROM api_ingest_jobs GROUP BY status`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var status string
		var count int
		if err = rows.Scan(&status, &count); err != nil {
			rows.Close()
			return nil, err
		}
		jobs[status] = count
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	namespaces := []string{}
	rows, err = s.Pool.Query(ctx, `SELECT value FROM api_ingest_state WHERE key LIKE 'namespace:%' ORDER BY key`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var value string
		if err = rows.Scan(&value); err != nil {
			rows.Close()
			return nil, err
		}
		namespaces = append(namespaces, value)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	var documents int
	err = s.Pool.QueryRow(ctx, `SELECT count(*) FROM api_ingest_documents WHERE expires_at>now()`).Scan(&documents)
	if err != nil {
		return nil, err
	}
	details := map[string]int{}
	rows, err = s.Pool.Query(ctx, `SELECT split_part(path,'/',4),count(*) FROM api_ingest_documents WHERE expires_at>now() AND path ~ '^/data/wow/[a-z-]+/[0-9]+$' GROUP BY 1`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var kind string
		var count int
		if err = rows.Scan(&kind, &count); err != nil {
			rows.Close()
			return nil, err
		}
		details[kind] = count
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	locales := []string{}
	rows, err = s.Pool.Query(ctx, `SELECT DISTINCT locale FROM api_ingest_documents WHERE expires_at>now() ORDER BY locale`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var locale string
		if err = rows.Scan(&locale); err != nil {
			rows.Close()
			return nil, err
		}
		locales = append(locales, locale)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	var runStatus string
	var checkedAt *time.Time
	err = s.Pool.QueryRow(ctx, `SELECT status,finished_at FROM ingestion_runs ORDER BY id DESC LIMIT 1`).Scan(&runStatus, &checkedAt)
	if err != nil && err != pgx.ErrNoRows {
		return nil, err
	}
	return map[string]any{"game_version": version, "entities": total, "by_kind": counts, "complete": false, "scope": "Curated Ashenvale plus build-scoped official Forever API metadata; not complete game knowledge or proof of live availability.", "unknown_total_game_entities": true, "ingestion": map[string]any{"provider": "blizzard", "namespaces": namespaces, "locales": locales, "detailed_documents_by_resource": details, "record_counts_are_not_field_completeness": true, "documents": documents, "jobs": jobs, "latest_run_status": runStatus, "latest_run_finished_at": checkedAt, "refresh_hours": 24, "retention_days": 28, "search_id_bounds": []int{0, 2147483647}}, "unverified_domains": []string{"complete quests and objectives", "complete NPC positions", "drop chances", "quest eligibility", "live-client availability", "all game data outside supported API endpoints"}}, nil
}
