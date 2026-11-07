-- Separate provider accounting; reservations are committed before every HTTP attempt.
CREATE TABLE provider_daily_usage (
 provider text NOT NULL,
 utc_day date NOT NULL,
 requests integer NOT NULL CHECK (requests >= 0),
 PRIMARY KEY(provider,utc_day)
);
ALTER TABLE api_ingest_jobs ADD COLUMN priority integer NOT NULL DEFAULT 10;
