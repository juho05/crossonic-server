-- +migrate Up
ALTER TABLE scrobbles
DROP CONSTRAINT scrobbles_song_id_fkey;
CREATE INDEX scrobbles_song_id_idx ON scrobbles(song_id);

-- +migrate Down
DROP INDEX scrobbles_song_id_idx;

ALTER TABLE scrobbles
ADD CONSTRAINT scrobbles_song_id_fkey
FOREIGN KEY (song_id) REFERENCES songs(id) ON DELETE CASCADE;