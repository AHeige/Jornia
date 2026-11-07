-- Admin metadata only: never plaintext invite codes, questions or journals.
CREATE TABLE invite_admin_audit (
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 action text NOT NULL CHECK(action IN ('created','revoked')),
 token_hash text NOT NULL,
 label text NOT NULL,
 occurred_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX invite_admin_audit_time ON invite_admin_audit(occurred_at);
