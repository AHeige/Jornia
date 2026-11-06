CREATE TABLE IF NOT EXISTS tester_access (
 token_hash text PRIMARY KEY,
 label text NOT NULL,
 expires_at timestamptz NOT NULL,
 revoked boolean NOT NULL DEFAULT false,
 daily_limit integer NOT NULL DEFAULT 50 CHECK (daily_limit BETWEEN 1 AND 500)
);
CREATE TABLE IF NOT EXISTS tester_requests (
 token_hash text NOT NULL REFERENCES tester_access(token_hash),
 request_id text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(token_hash,request_id)
);
CREATE INDEX IF NOT EXISTS tester_requests_day ON tester_requests(token_hash,created_at);
