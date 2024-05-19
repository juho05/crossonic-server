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
-- name: FindAlbumsAlphabeticalByName :many
SELECT albums.*, COALESCE(tracks.count, 0) AS track_count, COALESCE(tracks.duration_ms, 0) AS duration_ms, album_stars.created as starred, album_ratings.rating AS user_rating, COALESCE(avgr.rating, 0) AS avg_rating FROM albums
LEFT JOIN (
  SELECT album_id, COUNT(*) AS count, SUM(duration_ms) AS duration_ms FROM songs GROUP BY album_id
) tracks ON tracks.album_id = albums.id
LEFT JOIN album_stars ON album_stars.album_id = albums.id AND album_stars.user_name = $1
LEFT JOIN (
  SELECT album_id, AVG(album_ratings.rating) AS rating FROM album_ratings GROUP BY album_id
) avgr ON avgr.album_id = albums.id
LEFT JOIN album_ratings ON album_ratings.album_id = albums.id AND album_ratings.user_name = $1
WHERE (cast(sqlc.narg(from_year) as int) IS NULL OR (albums.year IS NOT NULL AND albums.year >= sqlc.arg(from_year)))
AND (cast(sqlc.narg(to_year) as int) IS NULL OR (albums.year IS NOT NULL AND albums.year <= sqlc.arg(to_year)))
AND (sqlc.arg('genres_lower')::text[] IS NULL OR EXISTS (
  SELECT album_genre.album_id, genres.name FROM album_genre
  JOIN genres ON album_genre.genre_name = genres.name
  WHERE album_genre.album_id = albums.id AND lower(album_genre.genre_name) = any(sqlc.arg('genres_lower')::text[])
))
ORDER BY lower(albums.name)
OFFSET $2 LIMIT $3;
-- name: FindAlbumsNewest :many
SELECT albums.*, COALESCE(tracks.count, 0) AS track_count, COALESCE(tracks.duration_ms, 0) AS duration_ms, album_stars.created as starred, album_ratings.rating AS user_rating, COALESCE(avgr.rating, 0) AS avg_rating FROM albums
LEFT JOIN (
  SELECT album_id, COUNT(*) AS count, SUM(duration_ms) AS duration_ms FROM songs GROUP BY album_id
) tracks ON tracks.album_id = albums.id
LEFT JOIN album_stars ON album_stars.album_id = albums.id AND album_stars.user_name = $1
LEFT JOIN (
  SELECT album_id, AVG(album_ratings.rating) AS rating FROM album_ratings GROUP BY album_id
) avgr ON avgr.album_id = albums.id
LEFT JOIN album_ratings ON album_ratings.album_id = albums.id AND album_ratings.user_name = $1
WHERE (cast(sqlc.narg(from_year) as int) IS NULL OR (albums.year IS NOT NULL AND albums.year >= sqlc.arg(from_year)))
AND (cast(sqlc.narg(to_year) as int) IS NULL OR (albums.year IS NOT NULL AND albums.year <= sqlc.arg(to_year)))
AND (sqlc.arg('genres_lower')::text[] IS NULL OR EXISTS (
  SELECT album_genre.album_id, genres.name FROM album_genre
  JOIN genres ON album_genre.genre_name = genres.name
  WHERE album_genre.album_id = albums.id AND lower(album_genre.genre_name) = any(sqlc.arg('genres_lower')::text[])
))
ORDER BY albums.created DESC, lower(albums.name)
OFFSET $2 LIMIT $3;
-- name: FindAlbumsHighestRated :many
SELECT albums.*, COALESCE(tracks.count, 0) AS track_count, COALESCE(tracks.duration_ms, 0) AS duration_ms, album_stars.created as starred, album_ratings.rating AS user_rating, COALESCE(avgr.rating, 0) AS avg_rating FROM albums
LEFT JOIN (
  SELECT album_id, COUNT(*) AS count, SUM(duration_ms) AS duration_ms FROM songs GROUP BY album_id
) tracks ON tracks.album_id = albums.id
LEFT JOIN album_stars ON album_stars.album_id = albums.id AND album_stars.user_name = $1
LEFT JOIN (
  SELECT album_id, AVG(album_ratings.rating) AS rating FROM album_ratings GROUP BY album_id
) avgr ON avgr.album_id = albums.id
LEFT JOIN album_ratings ON album_ratings.album_id = albums.id AND album_ratings.user_name = $1
WHERE (cast(sqlc.narg(from_year) as int) IS NULL OR (albums.year IS NOT NULL AND albums.year >= sqlc.arg(from_year)))
AND (cast(sqlc.narg(to_year) as int) IS NULL OR (albums.year IS NOT NULL AND albums.year <= sqlc.arg(to_year)))
AND (sqlc.arg('genres_lower')::text[] IS NULL OR EXISTS (
  SELECT album_genre.album_id, genres.name FROM album_genre
  JOIN genres ON album_genre.genre_name = genres.name
  WHERE album_genre.album_id = albums.id AND lower(album_genre.genre_name) = any(sqlc.arg('genres_lower')::text[])
))
ORDER BY COALESCE(album_ratings.rating, 0) DESC, lower(albums.name)
OFFSET $2 LIMIT $3;
-- name: FindAlbumsStarred :many
SELECT albums.*, COALESCE(tracks.count, 0) AS track_count, COALESCE(tracks.duration_ms, 0) AS duration_ms, album_stars.created as starred, album_ratings.rating AS user_rating, COALESCE(avgr.rating, 0) AS avg_rating FROM albums
LEFT JOIN (
  SELECT album_id, COUNT(*) AS count, SUM(duration_ms) AS duration_ms FROM songs GROUP BY album_id
) tracks ON tracks.album_id = albums.id
LEFT JOIN album_stars ON album_stars.album_id = albums.id AND album_stars.user_name = $1
LEFT JOIN (
  SELECT album_id, AVG(album_ratings.rating) AS rating FROM album_ratings GROUP BY album_id
) avgr ON avgr.album_id = albums.id
LEFT JOIN album_ratings ON album_ratings.album_id = albums.id AND album_ratings.user_name = $1
WHERE album_stars.created IS NOT NULL
AND (cast(sqlc.narg(from_year) as int) IS NULL OR (albums.year IS NOT NULL AND albums.year >= sqlc.arg(from_year)))
AND (cast(sqlc.narg(to_year) as int) IS NULL OR (albums.year IS NOT NULL AND albums.year <= sqlc.arg(to_year)))
AND (sqlc.arg('genres_lower')::text[] IS NULL OR EXISTS (
  SELECT album_genre.album_id, genres.name FROM album_genre
  JOIN genres ON album_genre.genre_name = genres.name
  WHERE album_genre.album_id = albums.id AND lower(album_genre.genre_name) = any(sqlc.arg('genres_lower')::text[])
))
ORDER BY album_stars.created DESC, lower(albums.name)
OFFSET $2 LIMIT $3;
-- name: FindAlbumsRandom :many
SELECT albums.*, COALESCE(tracks.count, 0) AS track_count, COALESCE(tracks.duration_ms, 0) AS duration_ms, album_stars.created as starred, album_ratings.rating AS user_rating, COALESCE(avgr.rating, 0) AS avg_rating FROM albums
LEFT JOIN (
  SELECT album_id, COUNT(*) AS count, SUM(duration_ms) AS duration_ms FROM songs GROUP BY album_id
) tracks ON tracks.album_id = albums.id
LEFT JOIN album_stars ON album_stars.album_id = albums.id AND album_stars.user_name = $1
LEFT JOIN (
  SELECT album_id, AVG(album_ratings.rating) AS rating FROM album_ratings GROUP BY album_id
) avgr ON avgr.album_id = albums.id
LEFT JOIN album_ratings ON album_ratings.album_id = albums.id AND album_ratings.user_name = $1
WHERE (cast(sqlc.narg(from_year) as int) IS NULL OR (albums.year IS NOT NULL AND albums.year >= sqlc.arg(from_year)))
AND (cast(sqlc.narg(to_year) as int) IS NULL OR (albums.year IS NOT NULL AND albums.year <= sqlc.arg(to_year)))
AND (sqlc.arg('genres_lower')::text[] IS NULL OR EXISTS (
  SELECT album_genre.album_id, genres.name FROM album_genre
  JOIN genres ON album_genre.genre_name = genres.name
  WHERE album_genre.album_id = albums.id AND lower(album_genre.genre_name) = any(sqlc.arg('genres_lower')::text[])
))
ORDER BY random()
LIMIT $2;
-- name: FindAlbumsByYear :many
SELECT albums.*, COALESCE(tracks.count, 0) AS track_count, COALESCE(tracks.duration_ms, 0) AS duration_ms, album_stars.created as starred, album_ratings.rating AS user_rating, COALESCE(avgr.rating, 0) AS avg_rating FROM albums
LEFT JOIN (
  SELECT album_id, COUNT(*) AS count, SUM(duration_ms) AS duration_ms FROM songs GROUP BY album_id
) tracks ON tracks.album_id = albums.id
LEFT JOIN album_stars ON album_stars.album_id = albums.id AND album_stars.user_name = $1
LEFT JOIN (
  SELECT album_id, AVG(album_ratings.rating) AS rating FROM album_ratings GROUP BY album_id
) avgr ON avgr.album_id = albums.id
LEFT JOIN album_ratings ON album_ratings.album_id = albums.id AND album_ratings.user_name = $1
WHERE albums.year IS NOT NULL
AND albums.year >= sqlc.arg(from_year)
AND albums.year <= sqlc.arg(to_year)
AND (sqlc.arg('genres_lower')::text[] IS NULL OR EXISTS (
  SELECT album_genre.album_id, genres.name FROM album_genre
  JOIN genres ON album_genre.genre_name = genres.name
  WHERE album_genre.album_id = albums.id AND lower(album_genre.genre_name) = any(sqlc.arg('genres_lower')::text[])
))
ORDER BY albums.year, lower(albums.name)
OFFSET $2 LIMIT $3;
-- name: FindAlbumsByGenre :many
SELECT albums.*, COALESCE(tracks.count, 0) AS track_count, COALESCE(tracks.duration_ms, 0) AS duration_ms, album_stars.created as starred, album_ratings.rating AS user_rating, COALESCE(avgr.rating, 0) AS avg_rating FROM albums
LEFT JOIN (
  SELECT album_id, COUNT(*) AS count, SUM(duration_ms) AS duration_ms FROM songs GROUP BY album_id
) tracks ON tracks.album_id = albums.id
LEFT JOIN album_stars ON album_stars.album_id = albums.id AND album_stars.user_name = $1
LEFT JOIN (
  SELECT album_id, AVG(album_ratings.rating) AS rating FROM album_ratings GROUP BY album_id
) avgr ON avgr.album_id = albums.id
LEFT JOIN album_ratings ON album_ratings.album_id = albums.id AND album_ratings.user_name = $1
WHERE (cast(sqlc.narg(from_year) as int) IS NULL OR albums.year >= sqlc.arg(from_year))
AND (cast(sqlc.narg(to_year) as int) IS NULL OR albums.year <= sqlc.arg(to_year))
AND EXISTS (
  SELECT album_genre.album_id, genres.name FROM album_genre
  JOIN genres ON album_genre.genre_name = genres.name
  WHERE album_genre.album_id = albums.id AND lower(album_genre.genre_name) = any(sqlc.arg('genres_lower')::text[])
)
ORDER BY lower(albums.name)
OFFSET $2 LIMIT $3;
-- name: FindAlbum :one
SELECT albums.*, COALESCE(tracks.count, 0) AS track_count, COALESCE(tracks.duration_ms, 0) AS duration_ms, album_stars.created as starred, album_ratings.rating AS user_rating, COALESCE(avgr.rating, 0) AS avg_rating FROM albums
LEFT JOIN (
  SELECT album_id, COUNT(*) AS count, SUM(duration_ms) AS duration_ms FROM songs GROUP BY album_id
) tracks ON tracks.album_id = albums.id
LEFT JOIN album_stars ON album_stars.album_id = albums.id AND album_stars.user_name = $1
LEFT JOIN (
  SELECT album_id, AVG(album_ratings.rating) AS rating FROM album_ratings GROUP BY album_id
) avgr ON avgr.album_id = albums.id
LEFT JOIN album_ratings ON album_ratings.album_id = albums.id AND album_ratings.user_name = $1
WHERE albums.id = $2;
-- name: FindAlbumsByArtist :many
SELECT albums.*, COALESCE(tracks.count, 0) AS track_count, COALESCE(tracks.duration_ms, 0) AS duration_ms, album_stars.created as starred, album_ratings.rating AS user_rating, COALESCE(avgr.rating, 0) AS avg_rating FROM albums
LEFT JOIN (
  SELECT album_id, COUNT(*) AS count, SUM(duration_ms) AS duration_ms FROM songs GROUP BY album_id
) tracks ON tracks.album_id = albums.id
LEFT JOIN album_stars ON album_stars.album_id = albums.id AND album_stars.user_name = $1
LEFT JOIN (
  SELECT album_id, AVG(album_ratings.rating) AS rating FROM album_ratings GROUP BY album_id
) avgr ON avgr.album_id = albums.id
LEFT JOIN album_ratings ON album_ratings.album_id = albums.id AND album_ratings.user_name = $1
WHERE EXISTS (
  SELECT album_artist.album_id, album_artist.artist_id FROM album_artist
  WHERE album_artist.album_id = albums.id AND album_artist.artist_id = $2
);
-- name: SearchAlbums :many
SELECT albums.*, COALESCE(tracks.count, 0) AS track_count, COALESCE(tracks.duration_ms, 0) AS duration_ms, album_stars.created as starred, album_ratings.rating AS user_rating, COALESCE(avgr.rating, 0) AS avg_rating FROM albums
LEFT JOIN (
  SELECT album_id, COUNT(*) AS count, SUM(duration_ms) AS duration_ms FROM songs GROUP BY album_id
) tracks ON tracks.album_id = albums.id
LEFT JOIN album_stars ON album_stars.album_id = albums.id AND album_stars.user_name = $1
LEFT JOIN (
  SELECT album_id, AVG(album_ratings.rating) AS rating FROM album_ratings GROUP BY album_id
) avgr ON avgr.album_id = albums.id
LEFT JOIN album_ratings ON album_ratings.album_id = albums.id AND album_ratings.user_name = $1
WHERE position(lower(sqlc.arg(search_str)) in lower(albums.name)) > 0
ORDER BY position(lower(sqlc.arg(search_str)) in lower(albums.name)), lower(albums.name)
OFFSET $2 LIMIT $3;