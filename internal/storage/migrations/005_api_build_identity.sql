CREATE TABLE api_ingest_state(key text PRIMARY KEY,value text NOT NULL,updated_at timestamptz NOT NULL DEFAULT now());
ALTER TABLE api_ingest_documents ADD COLUMN projection_version integer NOT NULL DEFAULT 0;
-- Rebuild only this prototype's API projection with build-scoped identities; raw, expiring evidence is preserved.
DELETE FROM edges WHERE source_id LIKE 'blizzard-api:%' OR subject_id LIKE 'blizzard-api:%' OR object_id LIKE 'blizzard-api:%';
DELETE FROM entities WHERE id LIKE 'blizzard-api:%';
DELETE FROM sources WHERE id LIKE 'blizzard-api:%';
