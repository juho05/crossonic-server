-- name: FindAlbumsByNameWithArtistMatchCount :many
SELECT albums.id, albums.music_brainz_id, COUNT(artists.id) AS artist_matches FROM albums
LEFT JOIN album_artist ON albums.id = album_artist.album_id
LEFT JOIN artists ON album_artist.artist_id = artists.id AND artists.name = any(sqlc.arg('artist_names')::text[])
WHERE albums.name = $1
GROUP BY albums.id, albums.music_brainz_id;
-- name: CreateAlbum :one
INSERT INTO albums
(id, name, created, updated, year, record_labels, music_brainz_id, release_types, is_compilation, replay_gain, replay_gain_peak)
VALUES($1, $2, NOW(), NOW(), $3, $4, $5, $6, $7, $8, $9)
RETURNING *;
-- name: UpdateAlbum :exec
UPDATE albums
SET name = $2, year = $3, record_labels = $4, release_types = $5, is_compilation = $6, replay_gain = $7, replay_gain_peak = $8, updated = NOW()
WHERE id = $1;
-- name: DeleteAlbumsLastUpdatedBefore :exec
DELETE FROM albums WHERE updated < $1;
-- name: DeleteAlbumArtists :exec
DELETE FROM album_artist WHERE album_id = $1;
-- name: CreateAlbumArtists :copyfrom
INSERT INTO album_artist (album_id,artist_id) VALUES ($1, $2);
-- name: DeleteAlbumGenres :exec
DELETE FROM album_genre WHERE album_id = $1;
-- name: CreateAlbumGenres :copyfrom
INSERT INTO album_genre (album_id,genre_name) VALUES ($1, $2);