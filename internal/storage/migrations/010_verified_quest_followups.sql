-- Only explicitly reviewed quest-chain claims belong here. Player observations
-- alone never establish prerequisite or follow-up relationships.
CREATE TABLE quest_followups (
 game_version text NOT NULL,
 client_build text NOT NULL CHECK (client_build ~ '^[0-9]{1,10}$'),
 quest_id integer NOT NULL CHECK (quest_id > 0),
 next_quest_id integer NOT NULL CHECK (next_quest_id > 0 AND next_quest_id <> quest_id),
 next_quest_title text NOT NULL CHECK (length(next_quest_title) BETWEEN 1 AND 200),
 source_id text NOT NULL REFERENCES sources(id),
 PRIMARY KEY (game_version,client_build,quest_id,next_quest_id,source_id)
);
