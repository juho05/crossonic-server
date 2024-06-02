-- +migrate Up
ALTER TABLE songs DROP CONSTRAINT songs_music_brainz_id_key;

-- +migrate Down
ALTER TABLE songs ADD CONSTRAINT songs_music_brainz_id_key UNIQUE (music_brainz_id);