-- +migrate Up
CREATE TABLE playlists (
  id text NOT NULL PRIMARY KEY,
  name text NOT NULL,
  created timestamptz NOT NULL,
  updated timestamptz NOT NULL,
  owner text NOT NULL REFERENCES users(name) ON DELETE CASCADE,
  public boolean NOT NULL DEFAULT false,
  comment text
);

CREATE TABLE playlist_song (
  playlist_id text NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,
  song_id text NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
  track int NOT NULL,
  PRIMARY KEY (playlist_id,track)
);

-- +migrate Down
DROP TABLE playlist_song;
DROP TABLE playlists;
