-- +migrate Up
ALTER TABLE users ADD COLUMN encrypted_listenbrainz_token bytea;
ALTER TABLE users ADD COLUMN listenbrainz_username text;

-- +migrate Down
ALTER TABLE users DROP COLUMN encrypted_listenbrainz_token;
ALTER TABLE users DROP COLUMN listenbrainz_username;