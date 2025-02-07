-- +migrate Up
ALTER TABLE songs
DROP COLUMN cover_id;

-- +migrate Down
ALTER TABLE songs
ADD COLUMN cover_id text;
