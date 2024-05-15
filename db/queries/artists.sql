-- name: CreateArtist :one
INSERT INTO artists
(id, name, created, updated, music_brainz_id)
VALUES ($1, $2, NOW(), NOW(), $3)
RETURNING *;
-- name: UpdateArtist :exec
UPDATE artists SET name = $2, music_brainz_id = $3, updated = NOW() WHERE id = $1;
-- name: DeleteArtistsLastUpdatedBefore :exec
DELETE FROM artists WHERE updated < $1;
-- name: FindArtistsByName :many
SELECT * FROM artists WHERE name = any(sqlc.arg('artist_names')::text[]);