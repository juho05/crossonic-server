-- +migrate Up
CREATE TABLE lb_feedback_updated (
  song_id text NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
  user_name text NOT NULL REFERENCES users(name) ON DELETE CASCADE,
  mbid text NOT NULL,
  PRIMARY KEY (song_id,user_name)
);

-- +migrate Down
DROP TABLE lb_feedback_updated;