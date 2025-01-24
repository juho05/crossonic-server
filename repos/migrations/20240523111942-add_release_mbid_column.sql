-- +migrate Up
ALTER TABLE albums ADD COLUMN release_mbid text;

-- +migrate Down
ALTER TABLE albums DROP COLUMN release_mbid;
