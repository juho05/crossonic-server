-- +migrate Up
ALTER TABLE albums
ADD COLUMN info_updated timestamptz,
ADD COLUMN description text,
ADD COLUMN lastfm_mbid text,
ADD COLUMN lastfm_url text;

ALTER TABLE artists
ADD COLUMN info_updated timestamptz,
ADD COLUMN biography text,
ADD COLUMN lastfm_mbid text,
ADD COLUMN lastfm_url text;

-- +migrate Down
ALTER TABLE albums
DROP COLUMN info_updated,
DROP COLUMN description,
DROP COLUMN lastfm_mbid,
DROP COLUMN lastfm_url;

ALTER TABLE artists
DROP COLUMN info_updated,
DROP COLUMN biography,
DROP COLUMN lastfm_mbid,
DROP COLUMN lastfm_url;