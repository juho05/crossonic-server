-- name: FindGenre :one
SELECT * FROM genres WHERE name = $1;
-- name: FindAllGenres :many
SELECT * FROM genres;
-- name: CreateGenre :exec
INSERT INTO genres (name) VALUES ($1) ON CONFLICT DO NOTHING;
-- name: DeleteGenre :exec
DELETE FROM genres WHERE name = $1;
-- name: DeleteAllGenres :exec
DELETE FROM genres;