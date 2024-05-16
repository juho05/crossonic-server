-- +migrate Up
CREATE TABLE song_ratings (
  song_id text NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
  user_name text NOT NULL REFERENCES users(name) ON DELETE CASCADE,
  rating int NOT NULL,
  PRIMARY KEY (song_id, user_name)
);

CREATE TABLE album_ratings (
  album_id text NOT NULL REFERENCES albums(id) ON DELETE CASCADE,
  user_name text NOT NULL REFERENCES users(name) ON DELETE CASCADE,
  rating int NOT NULL,
  PRIMARY KEY (album_id, user_name)
);

CREATE TABLE artist_ratings (
  artist_id text NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
  user_name text NOT NULL REFERENCES users(name) ON DELETE CASCADE,
  rating int NOT NULL,
  PRIMARY KEY (artist_id, user_name)
);

CREATE TABLE song_stars (
  song_id text NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
  user_name text NOT NULL REFERENCES users(name) ON DELETE CASCADE,
  created timestamptz NOT NULL,
  PRIMARY KEY (song_id, user_name)
);

CREATE TABLE album_stars (
  album_id text NOT NULL REFERENCES albums(id) ON DELETE CASCADE,
  user_name text NOT NULL REFERENCES users(name) ON DELETE CASCADE,
  created timestamptz NOT NULL,
  PRIMARY KEY (album_id, user_name)
);

CREATE TABLE artist_stars (
  artist_id text NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
  user_name text NOT NULL REFERENCES users(name) ON DELETE CASCADE,
  created timestamptz NOT NULL,
  PRIMARY KEY (artist_id, user_name)
);

-- +migrate Down
DROP TABLE song_ratings;
DROP TABLE album_ratings;
DROP TABLE artist_ratings;
DROP TABLE song_stars;
DROP TABLE album_stars;
DROP TABLE artist_stars;