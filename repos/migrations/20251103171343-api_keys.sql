-- +migrate Up
CREATE TABLE api_keys (
    user_name TEXT NOT NULL REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE,
    name TEXT NOT NULL,
    value_hash BYTEA NOT NULL,
    created timestamptz NOT NULL,
    PRIMARY KEY (user_name, name)
);
CREATE INDEX api_keys_value_hash_idx ON api_keys(value_hash);

ALTER TABLE users ADD COLUMN hashed_password TEXT;
ALTER TABLE users ALTER COLUMN encrypted_password DROP NOT NULL;

-- +migrate Down
DROP TABLE api_keys;
ALTER TABLE users DROP COLUMN hashed_password;
ALTER TABLE users ALTER COLUMN encrypted_password SET NOT NULL;
