-- +migrate Up
ALTER TABLE songs
ADD COLUMN search_text text;

UPDATE songs SET search_text = ' ' || lower(title) || ' ';

ALTER TABLE songs
ALTER COLUMN search_text SET NOT NULL;

ALTER TABLE albums
ADD COLUMN search_text text;

UPDATE albums SET search_text = ' ' || lower(name) || ' ';

ALTER TABLE albums
ALTER COLUMN search_text SET NOT NULL;

ALTER TABLE artists
ADD COLUMN search_text text;

UPDATE artists SET search_text = ' ' || lower(name) || ' ';

ALTER TABLE artists
ALTER COLUMN search_text SET NOT NULL;

INSERT INTO system (key, value) VALUES ('needs-full-scan', '1') ON CONFLICT (key) DO NOTHING;

-- +migrate Down
ALTER TABLE songs
DROP COLUMN search_text;
ALTER TABLE albums
DROP COLUMN search_text;
ALTER TABLE artists
DROP COLUMN search_text;
