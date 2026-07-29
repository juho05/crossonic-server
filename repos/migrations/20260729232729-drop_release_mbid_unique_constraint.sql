-- +migrate Up
ALTER TABLE albums DROP CONSTRAINT albums_release_mbid_key;

-- +migrate Down
ALTER TABLE albums ADD CONSTRAINT albums_release_mbid_key UNIQUE (release_mbid);
