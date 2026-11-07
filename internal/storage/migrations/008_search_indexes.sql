-- Existing GIN name/ID search remains the primary anchor path.
-- Cover both directions of graph expansion and provider scheduling/build lookup.
CREATE INDEX edges_subject ON edges(game_version,subject_id);
CREATE INDEX source_provider_build ON sources((provenance->>'effective_version'),id);
CREATE INDEX api_jobs_provider_priority_due ON api_ingest_jobs(namespace,locale,priority,next_attempt,path);
-- Full factual search is available for bounded lexical fallback without scanning descriptions.
CREATE INDEX entities_fact_search_words ON entities USING gin(to_tsvector('simple',name||' '||description));
