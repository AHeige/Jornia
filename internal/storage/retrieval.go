package storage

import (
	"context"
	"strings"
	"unicode"
)

// Retrieve finds anchors in SQL before loading a bounded two-hop neighborhood.
// Corpus size is independent of the model's context window.
func (s *Store) Retrieve(ctx context.Context, version string, spoilers bool, question string) (Graph, error) {
	g := Graph{GameVersion: version, Entities: []Entity{}, Edges: []Edge{}, Sources: []Source{}}
	normalized := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return unicode.ToLower(r)
		}
		return ' '
	}, question)
	normalized = " " + strings.Join(strings.Fields(normalized), " ") + " "
	queryWords := []string{}
	for _, term := range strings.Fields(normalized) {
		if !searchStopwords[term] {
			queryWords = append(queryWords, term)
		}
	}
	words := strings.Join(queryWords, " | ")
	stopNames := []string{}
	for word := range searchStopwords {
		stopNames = append(stopNames, word)
	}
	if words == "" {
		return g, nil
	}
	rows, err := s.Pool.Query(ctx, `SELECT e.id FROM entities e JOIN sources s ON s.id=e.source_id WHERE e.game_version=$1 AND (NOT $6 OR s.id NOT LIKE 'foreverdb:%') AND lower(e.name)<>ALL($5) AND to_tsvector('simple',e.name||' '||e.id) @@ to_tsquery('simple',$4) AND ($2 OR NOT e.spoiler) AND s.validation_status IN ('fixture','source_checked','verified') AND (s.expires_at IS NULL OR s.expires_at>now()) AND (s.id NOT LIKE 'blizzard-api:%' OR s.provenance->>'effective_version' IN(SELECT value FROM api_ingest_state WHERE key LIKE 'namespace:%')) AND (s.id NOT LIKE 'foreverdb:%' OR s.provenance->>'effective_version' IN(SELECT value FROM api_ingest_state WHERE key='foreverdb:build')) AND (position(' '||btrim(regexp_replace(lower(e.name),'[^[:alnum:]]+',' ','g'))||' ' in $3)>0 OR position(' '||btrim(regexp_replace(lower(e.id),'[^[:alnum:]]+',' ','g'))||' ' in $3)>0) ORDER BY length(e.name) DESC,e.id LIMIT 24`, version, spoilers, normalized, words, stopNames, s.ExcludeForeverDB)
	if err != nil {
		return g, err
	}
	ids := []string{}
	seen := map[string]bool{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return g, err
		}
		ids = append(ids, id)
		seen[id] = true
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return g, err
	}
	if len(ids) == 0 {
		// An AND query across meaningful words is deliberately conservative:
		// partial-name/factual matches are candidates, never inferred facts.
		terms := []string{}
		for _, term := range strings.Fields(normalized) {
			if len([]rune(term)) < 3 || searchStopwords[term] {
				continue
			}
			terms = append(terms, term)
		}
		if len(terms) > 12 {
			terms = terms[:12]
		}
		if len(terms) > 0 {
			rows, err = s.Pool.Query(ctx, `SELECT e.id FROM entities e JOIN sources s ON s.id=e.source_id WHERE e.game_version=$1 AND (NOT $4 OR s.id NOT LIKE 'foreverdb:%') AND to_tsvector('simple',e.name||' '||e.description) @@ to_tsquery('simple',$3) AND ($2 OR NOT e.spoiler) AND s.validation_status IN ('fixture','source_checked','verified') AND (s.expires_at IS NULL OR s.expires_at>now()) AND (s.id NOT LIKE 'blizzard-api:%' OR s.provenance->>'effective_version' IN(SELECT value FROM api_ingest_state WHERE key LIKE 'namespace:%')) AND (s.id NOT LIKE 'foreverdb:%' OR s.provenance->>'effective_version' IN(SELECT value FROM api_ingest_state WHERE key='foreverdb:build')) ORDER BY e.id LIMIT 8`, version, spoilers, strings.Join(terms, " & "), s.ExcludeForeverDB)
			if err != nil {
				return g, err
			}
			for rows.Next() {
				var id string
				if err = rows.Scan(&id); err != nil {
					rows.Close()
					return g, err
				}
				ids = append(ids, id)
				seen[id] = true
			}
			err = rows.Err()
			rows.Close()
			if err != nil {
				return g, err
			}
		}
	}
	if len(ids) == 0 {
		return g, nil
	}
	g.Anchors = append([]string(nil), ids...)
	edgeIDs := map[string]bool{}
	sourceIDs := map[string]bool{}
	for hop := 0; hop < 2; hop++ {
		rows, err = s.Pool.Query(ctx, `WITH candidates AS MATERIALIZED (SELECT id,game_version,subject_id,predicate,object_id,source_id FROM edges WHERE game_version=$1 AND (subject_id=ANY($2) OR object_id=ANY($2)) ORDER BY id LIMIT 400) SELECT e.id,e.subject_id,e.predicate,e.object_id,e.source_id FROM candidates e JOIN sources s ON s.id=e.source_id JOIN entities a ON a.id=e.subject_id AND a.game_version=e.game_version JOIN sources sa ON sa.id=a.source_id JOIN entities b ON b.id=e.object_id AND b.game_version=e.game_version JOIN sources sb ON sb.id=b.source_id WHERE e.game_version=$1 AND (NOT $4 OR (s.id NOT LIKE 'foreverdb:%' AND sa.id NOT LIKE 'foreverdb:%' AND sb.id NOT LIKE 'foreverdb:%')) AND (e.subject_id=ANY($2) OR e.object_id=ANY($2)) AND ($3 OR (NOT a.spoiler AND NOT b.spoiler)) AND s.validation_status IN ('fixture','source_checked','verified') AND sa.validation_status IN ('fixture','source_checked','verified') AND sb.validation_status IN ('fixture','source_checked','verified') AND (s.expires_at IS NULL OR s.expires_at>now()) AND (s.id NOT LIKE 'blizzard-api:%' OR s.provenance->>'effective_version' IN(SELECT value FROM api_ingest_state WHERE key LIKE 'namespace:%')) AND (s.id NOT LIKE 'foreverdb:%' OR s.provenance->>'effective_version' IN(SELECT value FROM api_ingest_state WHERE key='foreverdb:build')) AND (sa.expires_at IS NULL OR sa.expires_at>now()) AND (sa.id NOT LIKE 'blizzard-api:%' OR sa.provenance->>'effective_version' IN(SELECT value FROM api_ingest_state WHERE key LIKE 'namespace:%')) AND (sa.id NOT LIKE 'foreverdb:%' OR sa.provenance->>'effective_version' IN(SELECT value FROM api_ingest_state WHERE key='foreverdb:build')) AND (sb.expires_at IS NULL OR sb.expires_at>now()) AND (sb.id NOT LIKE 'blizzard-api:%' OR sb.provenance->>'effective_version' IN(SELECT value FROM api_ingest_state WHERE key LIKE 'namespace:%')) AND (sb.id NOT LIKE 'foreverdb:%' OR sb.provenance->>'effective_version' IN(SELECT value FROM api_ingest_state WHERE key='foreverdb:build')) ORDER BY e.id LIMIT 200`, version, ids, spoilers, s.ExcludeForeverDB)
		if err != nil {
			return g, err
		}
		for rows.Next() {
			var e Edge
			if err = rows.Scan(&e.ID, &e.SubjectID, &e.Predicate, &e.ObjectID, &e.SourceID); err != nil {
				rows.Close()
				return g, err
			}
			if !edgeIDs[e.ID] {
				g.Edges = append(g.Edges, e)
				edgeIDs[e.ID] = true
				sourceIDs[e.SourceID] = true
			}
			for _, id := range []string{e.SubjectID, e.ObjectID} {
				if !seen[id] {
					seen[id] = true
					ids = append(ids, id)
				}
			}
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return g, err
		}
	}
	rows, err = s.Pool.Query(ctx, `SELECT id,game_version,kind,name,description,spoiler,source_id FROM entities WHERE game_version=$1 AND id=ANY($2) ORDER BY id`, version, ids)
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
		sourceIDs[e.SourceID] = true
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return g, err
	}
	sources := []string{}
	for id := range sourceIDs {
		sources = append(sources, id)
	}
	rows, err = s.Pool.Query(ctx, `SELECT id,title,uri,revision,license,validation_status,provenance,expires_at FROM sources WHERE id=ANY($1) ORDER BY id`, sources)
	if err != nil {
		return g, err
	}
	for rows.Next() {
		var src Source
		if err = rows.Scan(&src.ID, &src.Title, &src.URI, &src.Revision, &src.License, &src.ValidationStatus, &src.Provenance, &src.ExpiresAt); err != nil {
			rows.Close()
			return g, err
		}
		g.Sources = append(g.Sources, src)
	}
	err = rows.Err()
	rows.Close()
	return g, err
}

// This affects query syntax only; no translation or English-only data filter.
var searchStopwords = map[string]bool{"who": true, "gives": true, "is": true, "what": true, "where": true, "which": true, "how": true, "does": true, "the": true, "for": true, "can": true, "tell": true, "about": true, "please": true, "quest": true, "vad": true, "var": true, "hur": true, "vilken": true, "vilka": true, "och": true, "jag": true, "om": true, "är": true, "med": true}
