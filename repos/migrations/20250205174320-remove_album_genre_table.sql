-- +migrate Up
DROP TABLE album_genre;

-- +migrate Down
CREATE TABLE album_genre (
  album_id text NOT NULL REFERENCES albums(id) ON DELETE CASCADE,
  genre_name text NOT NULL REFERENCES genres(name) ON DELETE CASCADE,
  PRIMARY KEY (album_id, genre_name)
);