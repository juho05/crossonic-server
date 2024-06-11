-- name: InsertSystemValueIfNotExists :one
INSERT INTO system (key, value) VALUES ($1, $2) ON CONFLICT (key) DO UPDATE SET key = $1 RETURNING *;
-- name: ReplaceSystemValue :one
INSERT INTO system (key, value) VALUES ($1, $2) ON CONFLICT (key) DO UPDATE SET key = $1, value = $2 RETURNING *;
-- name: GetSystemValue :one
SELECT * FROM system WHERE key = $1;