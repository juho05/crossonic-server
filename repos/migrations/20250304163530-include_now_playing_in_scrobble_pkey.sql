-- +migrate Up
ALTER TABLE scrobbles DROP CONSTRAINT scrobbles_pkey;
ALTER TABLE scrobbles ADD CONSTRAINT scrobbles_pkey PRIMARY KEY (user_name, song_id, time, now_playing);

-- +migrate Down
DELETE FROM scrobbles WHERE now_playing = true;
ALTER TABLE scrobbles DROP CONSTRAINT scrobbles_pkey;
ALTER TABLE scrobbles ADD CONSTRAINT scrobbles_pkey PRIMARY KEY (user_name, song_id, time);
