-- +migrate Up
CREATE TABLE system (
  key text NOT NULL PRIMARY KEY,
  value text NOT NULL
);

-- +migrate Down
DROP TABLE system;
