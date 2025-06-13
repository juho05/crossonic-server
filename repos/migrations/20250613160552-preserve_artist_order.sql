-- +migrate Up
ALTER TABLE song_artist
ADD COLUMN index int NOT NULL DEFAULT 0;

ALTER TABLE album_artist
ADD COLUMN index int NOT NULL DEFAULT 0;

-- +migrate Down
ALTER TABLE song_artist
DROP COLUMN index;
ALTER TABLE album_artist
DROP COLUMN index;
