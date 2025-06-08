-- +migrate Up
ALTER TABLE albums DROP CONSTRAINT albums_music_brainz_id_key;
ALTER TABLE albums ADD CONSTRAINT albums_release_mbid_key UNIQUE (release_mbid);

-- +migrate Down
ALTER TABLE albums ADD CONSTRAINT albums_music_brainz_id_key UNIQUE (music_brainz_id);
ALTER TABLE albums DROP CONSTRAINT albums_release_mbid_key;
