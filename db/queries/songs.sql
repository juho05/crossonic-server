-- name: FindSongCount :one
SELECT COUNT(*) as song_count FROM songs;
-- name: FindSong :one
SELECT songs.*, albums.name as album_name, albums.music_brainz_id as album_music_brainz_id, albums.release_mbid as album_release_mbid FROM songs LEFT JOIN albums ON songs.album_id = albums.id WHERE songs.id = $1;
-- name: FindSongs :many
SELECT songs.*, albums.name as album_name, albums.music_brainz_id as album_music_brainz_id, albums.release_mbid as album_release_mbid FROM songs LEFT JOIN albums ON songs.album_id = albums.id WHERE songs.id = any(sqlc.arg('song_ids')::text[]);
-- name: FindSongWithoutAlbum :one
SELECT songs.* FROM songs WHERE songs.id = $1;
-- name: FindSongsByMusicBrainzID :many
SELECT songs.*, albums.name as album_name, albums.music_brainz_id as album_music_brainz_id, albums.release_mbid as album_release_mbid FROM songs LEFT JOIN albums ON songs.album_id = albums.id WHERE songs.music_brainz_id = $1;
-- name: FindSongByPath :one
SELECT songs.*, albums.name as album_name FROM songs LEFT JOIN albums ON songs.album_id = albums.id WHERE songs.path = $1;
-- name: CreateSong :one
INSERT INTO songs
(id, path, album_id, title, track, year, size, content_type, duration_ms, bit_rate, sampling_rate, channel_count, disc_number, created, updated, bpm, music_brainz_id, replay_gain, replay_gain_peak, lyrics, cover_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW(), NOW(), $14, $15, $16, $17, $18, $19) RETURNING *;
-- name: UpdateSong :exec
UPDATE songs SET path=$2,album_id=$3,title=$4,track=$5,year=$6,size=$7,content_type=$8,duration_ms=$9,bit_rate=$10,sampling_rate=$11,channel_count=$12,disc_number=$13,updated=NOW(),bpm=$14,music_brainz_id=$15,replay_gain=$16,replay_gain_peak=$17,lyrics=$18,cover_id=$19
WHERE id = $1;
-- name: DeleteSongsLastUpdatedBefore :exec
DELETE FROM songs WHERE updated < $1;
-- name: DeleteSongArtists :exec
DELETE FROM song_artist WHERE song_id = $1;
-- name: CreateSongArtists :copyfrom
INSERT INTO song_artist (song_id,artist_id) VALUES ($1, $2);
-- name: DeleteSongGenres :exec
DELETE FROM song_genre WHERE song_id = $1;
-- name: CreateSongGenres :copyfrom
INSERT INTO song_genre (song_id,genre_name) VALUES ($1, $2);
-- name: FindRandomSongs :many
SELECT songs.*, albums.name as album_name, albums.replay_gain as album_replay_gain, albums.replay_gain_peak as album_replay_gain_peak, song_stars.created as starred, song_ratings.rating AS user_rating, COALESCE(avgr.rating, 0) AS avg_rating FROM songs
LEFT JOIN albums ON albums.id = songs.album_id
LEFT JOIN song_stars ON song_stars.song_id = songs.id AND song_stars.user_name = $1
LEFT JOIN (
  SELECT song_id, AVG(song_ratings.rating) AS rating FROM song_ratings GROUP BY song_id
) avgr ON avgr.song_id = songs.id
LEFT JOIN song_ratings ON song_ratings.song_id = songs.id AND song_ratings.user_name = $1
WHERE (cast(sqlc.narg(from_year) as int) IS NULL OR (songs.year IS NOT NULL AND songs.year >= sqlc.arg(from_year)))
AND (cast(sqlc.narg(to_year) as int) IS NULL OR (songs.year IS NOT NULL AND songs.year <= sqlc.arg(to_year)))
AND (sqlc.arg('genres_lower')::text[] IS NULL OR EXISTS (
  SELECT song_genre.song_id, genres.name FROM song_genre
  JOIN genres ON song_genre.genre_name = genres.name
  WHERE song_genre.song_id = songs.id AND lower(song_genre.genre_name) = any(sqlc.arg('genres_lower')::text[])
))
ORDER BY random()
LIMIT $2;
-- name: FindSongsByAlbum :many
SELECT songs.*, albums.name as album_name, albums.replay_gain as album_replay_gain, albums.replay_gain_peak as album_replay_gain_peak, song_stars.created as starred, song_ratings.rating AS user_rating, COALESCE(avgr.rating, 0) AS avg_rating FROM songs
LEFT JOIN albums ON albums.id = songs.album_id
LEFT JOIN song_stars ON song_stars.song_id = songs.id AND song_stars.user_name = $1
LEFT JOIN (
  SELECT song_id, AVG(song_ratings.rating) AS rating FROM song_ratings GROUP BY song_id
) avgr ON avgr.song_id = songs.id
LEFT JOIN song_ratings ON song_ratings.song_id = songs.id AND song_ratings.user_name = $1
WHERE albums.id = $2
ORDER BY songs.disc_number, songs.track;
-- name: GetStreamInfo :one
SELECT songs.path, songs.bit_rate, songs.content_type, songs.duration_ms, songs.channel_count FROM songs WHERE songs.id = $1;
-- name: SearchSongs :many
SELECT songs.*, albums.name as album_name, albums.replay_gain as album_replay_gain, albums.replay_gain_peak as album_replay_gain_peak, song_stars.created as starred, song_ratings.rating AS user_rating, COALESCE(avgr.rating, 0) AS avg_rating FROM songs
LEFT JOIN albums ON albums.id = songs.album_id
LEFT JOIN song_stars ON song_stars.song_id = songs.id AND song_stars.user_name = $1
LEFT JOIN (
  SELECT song_id, AVG(song_ratings.rating) AS rating FROM song_ratings GROUP BY song_id
) avgr ON avgr.song_id = songs.id
LEFT JOIN song_ratings ON song_ratings.song_id = songs.id AND song_ratings.user_name = $1
WHERE position(lower(sqlc.arg(search_str)) in lower(songs.title)) > 0
ORDER BY position(lower(sqlc.arg(search_str)) in lower(songs.title)), lower(songs.title)
OFFSET $2 LIMIT $3;
-- name: GetMedianReplayGain :one
SELECT COALESCE(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY songs.replay_gain), 0) FROM songs;