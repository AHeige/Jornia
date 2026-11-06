CREATE TABLE sources (
  id text PRIMARY KEY,
  title text NOT NULL,
  uri text NOT NULL,
  revision text NOT NULL,
  license text NOT NULL,
  validation_status text NOT NULL CHECK (validation_status IN ('fixture','unverified','verified')),
  recorded_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE entities (
  id text NOT NULL,
  game_version text NOT NULL,
  kind text NOT NULL,
  name text NOT NULL,
  description text NOT NULL,
  spoiler boolean NOT NULL DEFAULT false,
  source_id text NOT NULL REFERENCES sources(id),
  PRIMARY KEY (id, game_version)
);
CREATE INDEX entities_name ON entities(game_version, lower(name));
CREATE TABLE edges (
  id text NOT NULL,
  game_version text NOT NULL,
  subject_id text NOT NULL,
  predicate text NOT NULL,
  object_id text NOT NULL,
  source_id text NOT NULL REFERENCES sources(id),
  PRIMARY KEY (id, game_version),
  FOREIGN KEY (subject_id, game_version) REFERENCES entities(id, game_version),
  FOREIGN KEY (object_id, game_version) REFERENCES entities(id, game_version)
);
CREATE INDEX edges_object ON edges(game_version, object_id);
