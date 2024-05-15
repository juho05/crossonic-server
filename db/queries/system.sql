-- name: InsertSystemValueIfNotExists :one
INSERT INTO system (key, value) VALUES ($1, $2) ON CONFLICT (key) DO UPDATE SET key = $1 RETURNING *;