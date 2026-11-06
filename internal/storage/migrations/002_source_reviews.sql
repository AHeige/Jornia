ALTER TABLE sources DROP CONSTRAINT sources_validation_status_check;
ALTER TABLE sources ADD CONSTRAINT sources_validation_status_check
  CHECK (validation_status IN ('fixture','unverified','source_checked','verified'));
ALTER TABLE sources ADD COLUMN provenance jsonb NOT NULL DEFAULT '{}';
CREATE TABLE knowledge_datasets (
  game_version text NOT NULL,
  revision text NOT NULL,
  checksum text NOT NULL,
  imported_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (game_version,revision)
);
