-- +migrate Up
ALTER TABLE songs
ADD COLUMN cover_id text;

-- +migrate Down
ALTER TABLE songs
DROP COLUMN cover_id;
