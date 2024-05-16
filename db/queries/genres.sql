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
-- name: FindGenresWithCount :many
SELECT genres.name, COALESCE(al.count, 0) AS album_count, COALESCE(so.count, 0) AS song_count FROM genres
LEFT JOIN (
  SELECT genre_name, COUNT(*) AS count FROM album_genre GROUP BY genre_name
) al ON al.genre_name = genres.name
LEFT JOIN (
  SELECT genre_name, COUNT(*) AS count FROM song_genre GROUP BY genre_name
) so ON so.genre_name = genres.name
ORDER BY lower(genres.name);
-- name: FindGenresByAlbums :many
SELECT album_genre.album_id, genres.name FROM album_genre
JOIN genres ON album_genre.genre_name = genres.name
WHERE album_genre.album_id = any(sqlc.arg('album_ids')::text[]);