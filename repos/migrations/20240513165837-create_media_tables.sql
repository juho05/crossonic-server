-- +migrate Up

CREATE TABLE genres (
  name text NOT NULL PRIMARY KEY
);

CREATE TABLE artists (
  id text NOT NULL PRIMARY KEY,
  name text NOT NULL UNIQUE,
  created timestamptz NOT NULL,
  updated timestamptz NOT NULL,
  music_brainz_id text
);

CREATE TABLE albums (
  id text NOT NULL PRIMARY KEY,
  name text NOT NULL,
  created timestamptz NOT NULL,
  updated timestamptz NOT NULL,
  year int,
  record_labels text,
  music_brainz_id text UNIQUE,
  release_types text,
  is_compilation boolean,
  replay_gain real,
  replay_gain_peak real
);

CREATE TABLE songs (
  id text NOT NULL PRIMARY KEY,
  path text NOT NULL UNIQUE,
  album_id text REFERENCES albums(id),
  title text NOT NULL,
  track int,
  year int,
  size bigint NOT NULL,
  content_type text NOT NULL,
  duration_ms int NOT NULL,
  bit_rate int NOT NULL,
  sampling_rate int NOT NULL,
  channel_count int NOT NULL,
  disc_number int,
  created timestamptz NOT NULL,
  updated timestamptz NOT NULL,
  bpm int,
  music_brainz_id text UNIQUE,
  replay_gain real,
  replay_gain_peak real,
  lyrics text
);

CREATE TABLE song_artist (
  song_id text NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
  artist_id text NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
  PRIMARY KEY (song_id, artist_id)
);

CREATE TABLE album_artist (
  album_id text NOT NULL REFERENCES albums(id) ON DELETE CASCADE,
  artist_id text NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
  PRIMARY KEY (album_id, artist_id)
);

CREATE TABLE song_genre (
  song_id text NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
  genre_name text NOT NULL REFERENCES genres(name) ON DELETE CASCADE,
  PRIMARY KEY (song_id, genre_name)
);

CREATE TABLE album_genre (
  album_id text NOT NULL REFERENCES albums(id) ON DELETE CASCADE,
  genre_name text NOT NULL REFERENCES genres(name) ON DELETE CASCADE,
  PRIMARY KEY (album_id, genre_name)
);

-- +migrate Down
DROP TABLE album_genre;
DROP TABLE song_genre;
DROP TABLE album_artist;
DROP TABLE song_artist;
DROP TABLE songs;
DROP TABLE albums;
DROP TABLE artists;
DROP TABLE genres;