-- +migrate Up
ALTER TABLE users ADD COLUMN listenbrainz_scrobble bool DEFAULT true;
ALTER TABLE users ADD COLUMN listenbrainz_sync_feedback bool DEFAULT false;

-- retain current default behavior of feedback sync being on by default for existing connections
UPDATE users SET listenbrainz_sync_feedback = true WHERE listenbrainz_username IS NOT NULL;

-- +migrate Down
ALTER TABLE users DROP COLUMN listenbrainz_scrobble;
ALTER TABLE users DROP COLUMN listenbrainz_sync_feedback;
