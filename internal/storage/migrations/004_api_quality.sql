CREATE INDEX entities_search_words ON entities USING gin(to_tsvector('simple', name||' '||id));
-- The upstream catalog also contains internal/test rows. Keep raw evidence but do not serve them as game knowledge.
DELETE FROM edges WHERE (subject_id,game_version) IN(SELECT id,game_version FROM entities WHERE id LIKE 'blizzard-api:%' AND name ~* '(\[(DNT|DND|UNUSED|PH)\]|^zz|^unused |^deprecated |^internal )') OR (object_id,game_version) IN(SELECT id,game_version FROM entities WHERE id LIKE 'blizzard-api:%' AND name ~* '(\[(DNT|DND|UNUSED|PH)\]|^zz|^unused |^deprecated |^internal )');
DELETE FROM entities WHERE id LIKE 'blizzard-api:%' AND name ~* '(\[(DNT|DND|UNUSED|PH)\]|^zz|^unused |^deprecated |^internal )';
