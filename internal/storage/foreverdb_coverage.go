package storage

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
)

// Catalog identities, fetched detail documents and graph projections are different
// measures. In particular, quest-scoped NPC observations are not unique NPCs.
func (s *Store) foreverDBCoverage(ctx context.Context) (map[string]any, error) {
	var build string
	err := s.Pool.QueryRow(ctx, `SELECT value FROM api_ingest_state WHERE key='foreverdb:build'`).Scan(&build)
	if err == pgx.ErrNoRows {
		return map[string]any{"configured": false, "complete": false}, nil
	}
	if err != nil {
		return nil, err
	}
	namespace := "foreverdb-" + build
	var meta []byte
	err = s.Pool.QueryRow(ctx, `SELECT payload->'data' FROM api_ingest_documents WHERE namespace=$1 AND path='/api/v1/meta' AND expires_at>now()`, namespace).Scan(&meta)
	if err != nil && err != pgx.ErrNoRows {
		return nil, err
	}
	var totals any
	if len(meta) > 0 {
		_ = json.Unmarshal(meta, &totals)
	}
	rows, err := s.Pool.Query(ctx, `WITH docs AS (SELECT split_part(split_part(path,'?',1),'/',4) AS category,path,payload FROM api_ingest_documents WHERE namespace=$1 AND expires_at>now() AND path<>'/api/v1/meta'), listed AS (SELECT category,count(DISTINCT coalesce(r->>'id',r->>'slug',r->>'name',r->>'title')) AS records FROM docs CROSS JOIN LATERAL jsonb_array_elements(CASE WHEN jsonb_typeof(payload->'data')='array' THEN payload->'data' ELSE '[]'::jsonb END) r WHERE split_part(path,'?',1) ~ '^/api/v1/[^/]+$' GROUP BY category), counts AS (SELECT category,count(*) AS documents,count(*) FILTER(WHERE split_part(path,'?',1) ~ '^/api/v1/[^/]+/[^/]+$') AS details FROM docs GROUP BY category) SELECT c.category,c.documents,c.details,coalesce(l.records,0) FROM counts c LEFT JOIN listed l USING(category) ORDER BY c.category`, namespace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	categories := []map[string]any{}
	for rows.Next() {
		var category string
		var documents, details, listed int
		if err = rows.Scan(&category, &documents, &details, &listed); err != nil {
			return nil, err
		}
		categories = append(categories, map[string]any{"category": category, "documents": documents, "detail_documents": details, "unique_listed_identities": listed, "field_completeness": "not_measured"})
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	jobs := map[string]int{}
	rows, err = s.Pool.Query(ctx, `SELECT status,count(*) FROM api_ingest_jobs WHERE namespace=$1 GROUP BY status`, namespace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var count int
		if err = rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		jobs[status] = count
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return map[string]any{"configured": true, "build": build, "complete": false, "catalog_metadata": totals, "categories": categories, "jobs": jobs, "refresh": "hourly bounded batches; catalog/class/profession refresh 24 hours; quest details 7 days; other details 21 days; expiry 28 days; backlog and rate limits can delay refresh", "attribution": "Data from ForeverDB.net", "url": "https://foreverdb.net", "warning": "Catalog coverage is not verified field completeness. Scoped observations overlap. Classic and client leftovers retain their origin; availability is not inferred."}, nil
}
