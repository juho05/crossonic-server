-- +migrate Up
ALTER TABLE albums
    ADD COLUMN disc_titles text;

INSERT INTO system (key, value) VALUES ('needs-full-scan', '1') ON CONFLICT (key) DO NOTHING;

-- +migrate Down
ALTER TABLE albums
    DROP COLUMN disc_titles;
