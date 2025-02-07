-- +migrate Up
ALTER TABLE artists DROP CONSTRAINT artists_name_key;

-- +migrate Down
ALTER TABLE artists ADD CONSTRAINT artists_name_key UNIQUE (name);