-- +migrate Up
ALTER TABLE playlist_song
DROP CONSTRAINT playlist_song_pkey,
ADD CONSTRAINT playlist_song_pkey PRIMARY KEY (playlist_id,track) DEFERRABLE INITIALLY IMMEDIATE;

-- +migrate Down
ALTER TABLE
playlist_song DROP CONSTRAINT playlist_song_pkey,
ADD CONSTRAINT playlist_song_pkey PRIMARY KEY (playlist_id,track);
