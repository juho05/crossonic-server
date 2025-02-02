-- +migrate Up
CREATE TABLE internet_radio_stations (
  id text NOT NULL PRIMARY KEY,
  name text NOT NULL,
  stream_url text NOT NULL,
  created timestamptz NOT NULL,
  updated timestamptz NOT NULL,
  user_name text NOT NULL REFERENCES users(name) ON DELETE CASCADE,
  homepage_url text
);

-- +migrate Down
DROP TABLE internet_radio_stations;
