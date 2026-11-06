ALTER TABLE sources ADD COLUMN expires_at timestamptz;
CREATE INDEX sources_expiry ON sources(expires_at) WHERE expires_at IS NOT NULL;
CREATE TABLE api_ingest_jobs (
  path text NOT NULL,
  namespace text NOT NULL,
  locale text NOT NULL,
  status text NOT NULL DEFAULT 'pending',
  next_attempt timestamptz NOT NULL DEFAULT now(),
  checked_at timestamptz,
  http_status integer,
  PRIMARY KEY(path,namespace,locale)
);
CREATE TABLE api_ingest_documents (
  path text NOT NULL,
  namespace text NOT NULL,
  locale text NOT NULL,
  checksum text NOT NULL,
  payload jsonb NOT NULL,
  fetched_at timestamptz NOT NULL DEFAULT now(),
  expires_at timestamptz NOT NULL,
  PRIMARY KEY(path,namespace,locale)
);
CREATE INDEX api_jobs_due ON api_ingest_jobs(next_attempt,status);
CREATE INDEX api_documents_expiry ON api_ingest_documents(expires_at);
CREATE TABLE ingestion_runs (
 id bigserial PRIMARY KEY,
 started_at timestamptz NOT NULL DEFAULT now(),
 finished_at timestamptz,
 provider text NOT NULL,
 status text NOT NULL,
 requests integer NOT NULL DEFAULT 0,
 documents integer NOT NULL DEFAULT 0,
 error_code text
);
