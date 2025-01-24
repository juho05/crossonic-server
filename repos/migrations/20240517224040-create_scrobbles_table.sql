-- +migrate Up
CREATE TABLE scrobbles (
  user_name text NOT NULL REFERENCES users(name) ON DELETE CASCADE,
  song_id text NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
  album_id text REFERENCES albums(id) ON DELETE SET NULL,
  time timestamptz NOT NULL,
  song_duration_ms int NOT NULL,
  duration_ms int,
  submitted_to_listenbrainz boolean NOT NULL DEFAULT false,
  now_playing boolean NOT NULL DEFAULT false,
  PRIMARY KEY(user_name, song_id, time)
);

-- +migrate Down
DROP TABLE scrobbles;