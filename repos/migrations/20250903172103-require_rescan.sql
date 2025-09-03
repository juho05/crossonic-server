-- +migrate Up
INSERT INTO system (key, value) VALUES ('needs-full-scan', '1') ON CONFLICT (key) DO NOTHING;

-- +migrate Down
