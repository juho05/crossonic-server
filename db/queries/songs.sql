-- name: FindSongCount :one
SELECT COUNT(*) as song_count FROM songs;
-- name: FindSong :one
SELECT songs.*, albums.name as album_name FROM songs LEFT JOIN albums ON songs.album_id = albums.id WHERE songs.id = $1;
-- name: FindSongByMusicBrainzID :one
SELECT songs.*, albums.name as album_name FROM songs LEFT JOIN albums ON songs.album_id = albums.id WHERE songs.music_brainz_id = $1;
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