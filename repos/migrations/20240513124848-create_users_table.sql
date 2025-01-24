-- +migrate Up
CREATE TABLE users (
  name text NOT NULL PRIMARY KEY,
  encrypted_password bytea NOT NULL
);

-- +migrate Down
DROP TABLE users;
