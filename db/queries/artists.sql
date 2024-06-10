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
-- name: FindAlbumArtists :many
SELECT artists.*, COALESCE(aa.count, 0) AS album_count, artist_stars.created as starred, artist_ratings.rating AS user_rating, COALESCE(avgr.rating, 0) AS avg_rating FROM artists
LEFT JOIN (
  SELECT artist_id, COUNT(*) AS count FROM album_artist GROUP BY artist_id
) aa ON aa.artist_id = artists.id
LEFT JOIN artist_stars ON artist_stars.artist_id = artists.id AND artist_stars.user_name = $1
LEFT JOIN (
  SELECT artist_id, AVG(artist_ratings.rating) AS rating FROM artist_ratings GROUP BY artist_id
) avgr ON avgr.artist_id = artists.id
LEFT JOIN artist_ratings ON artist_ratings.artist_id = artists.id AND artist_ratings.user_name = $1
WHERE COALESCE(aa.count, 0) > 0
ORDER BY lower(artists.name);
-- name: FindArtistRefsByAlbums :many
SELECT album_artist.album_id, artists.id, artists.name FROM album_artist
JOIN artists ON album_artist.artist_id = artists.id
WHERE album_artist.album_id = any(sqlc.arg('album_ids')::text[]);
-- name: FindArtistRefsBySongs :many
SELECT song_artist.song_id, artists.id, artists.name, artists.music_brainz_id FROM song_artist
JOIN artists ON song_artist.artist_id = artists.id
WHERE song_artist.song_id = any(sqlc.arg('song_ids')::text[]);
-- name: FindAlbumArtistRefsBySongs :many
SELECT songs.id as song_id, artists.id, artists.name FROM songs
JOIN albums ON songs.album_id = albums.id
JOIN album_artist ON album_artist.album_id = albums.id
JOIN artists ON album_artist.artist_id = artists.id
WHERE songs.id = any(sqlc.arg('song_ids')::text[]);
-- name: FindArtist :one
SELECT artists.*, artist_stars.created as starred, artist_ratings.rating AS user_rating, COALESCE(avgr.rating, 0) AS avg_rating FROM artists
LEFT JOIN artist_stars ON artist_stars.artist_id = artists.id AND artist_stars.user_name = $1
LEFT JOIN (
  SELECT artist_id, AVG(artist_ratings.rating) AS rating FROM artist_ratings GROUP BY artist_id
) avgr ON avgr.artist_id = artists.id
LEFT JOIN artist_ratings ON artist_ratings.artist_id = artists.id AND artist_ratings.user_name = $1
WHERE artists.id = $2;
-- name: FindArtistSimple :one
SELECT * FROM artists WHERE id = $1;
-- name: SearchAlbumArtists :many
SELECT artists.*, COALESCE(aa.count, 0) AS album_count, artist_stars.created as starred, artist_ratings.rating AS user_rating, COALESCE(avgr.rating, 0) AS avg_rating FROM artists
LEFT JOIN (
  SELECT artist_id, COUNT(*) AS count FROM album_artist GROUP BY artist_id
) aa ON aa.artist_id = artists.id
LEFT JOIN artist_stars ON artist_stars.artist_id = artists.id AND artist_stars.user_name = $1
LEFT JOIN (
  SELECT artist_id, AVG(artist_ratings.rating) AS rating FROM artist_ratings GROUP BY artist_id
) avgr ON avgr.artist_id = artists.id
LEFT JOIN artist_ratings ON artist_ratings.artist_id = artists.id AND artist_ratings.user_name = $1
WHERE COALESCE(aa.count, 0) > 0 AND position(lower(sqlc.arg(search_str)) in lower(artists.name)) > 0
ORDER BY position(lower(sqlc.arg(search_str)) in lower(artists.name)), lower(artists.name)
OFFSET $2 LIMIT $3;