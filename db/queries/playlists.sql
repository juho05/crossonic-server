-- name: CreatePlaylist :exec
INSERT INTO playlists (id,name,created,updated,owner,public,comment)
VALUES ($1,$2,NOW(),NOW(),$3,$4,$5);
-- name: UpdatePlaylist :exec
UPDATE playlists SET name = $3, updated = NOW(), public = $4, comment = $5
WHERE id = $1 AND owner = $2;
-- name: UpdatePlaylistName :execresult
UPDATE playlists SET name = $3, updated = NOW()
WHERE id = $1 AND owner = $2;
-- name: CheckPlaylistExists :one
SELECT EXISTS (SELECT id FROM playlists WHERE id = $1 AND owner = $2);
-- name: FindPlaylist :one
SELECT playlists.*, COALESCE(tracks.count, 0) AS track_count, COALESCE(tracks.duration_ms, 0) FROM playlists
LEFT JOIN (
  SELECT playlist_song.playlist_id, COUNT(*) AS count, SUM(songs.duration_ms) AS duration_ms FROM playlist_song
  JOIN songs ON songs.id = playlist_song.song_id
  GROUP BY playlist_song.playlist_id
) tracks ON tracks.playlist_id = playlists.id
WHERE playlists.id = $1 AND (playlists.owner = sqlc.arg('user') OR playlists.public = true);
-- name: FindPlaylists :many
SELECT playlists.*, COALESCE(tracks.count, 0) AS track_count, COALESCE(tracks.duration_ms, 0) FROM playlists
LEFT JOIN (
  SELECT playlist_song.playlist_id, COUNT(*) AS count, SUM(songs.duration_ms) AS duration_ms FROM playlist_song
  JOIN songs ON songs.id = playlist_song.song_id
  GROUP BY playlist_song.playlist_id
) tracks ON tracks.playlist_id = playlists.id
WHERE playlists.owner = sqlc.arg('user') OR playlists.public = true
ORDER BY playlists.updated DESC, playlists.name ASC, playlists.created DESC;
-- name: GetPlaylistOwner :one
SELECT owner FROM playlists WHERE id = $1;
-- name: GetPlaylistTracks :many
SELECT songs.*, albums.name as album_name, albums.replay_gain as album_replay_gain, albums.replay_gain_peak as album_replay_gain_peak, song_stars.created as starred, song_ratings.rating AS user_rating, COALESCE(avgr.rating, 0) AS avg_rating
FROM playlist_song
JOIN songs ON songs.id = playlist_song.song_id
LEFT JOIN albums ON albums.id = songs.album_id
LEFT JOIN song_stars ON song_stars.song_id = songs.id AND song_stars.user_name = $2
LEFT JOIN (
  SELECT song_id, AVG(song_ratings.rating) AS rating FROM song_ratings GROUP BY song_id
) avgr ON avgr.song_id = songs.id
LEFT JOIN song_ratings ON song_ratings.song_id = songs.id AND song_ratings.user_name = $2
WHERE playlist_song.playlist_id = $1
ORDER BY playlist_song.track;
-- name: GetPlaylistTrackNumbers :many
SELECT track FROM playlist_song WHERE playlist_id = $1 ORDER BY track;
-- name: UpdatePlaylistTrackNumbers :exec
UPDATE playlist_song SET track = track + sqlc.arg('add') WHERE playlist_song.playlist_id = $1 AND track >= sqlc.arg('min_track') AND track <= sqlc.arg('max_track');
-- name: ClearPlaylist :exec
DELETE FROM playlist_song WHERE playlist_id = $1;
-- name: AddPlaylistTracks :copyfrom
INSERT INTO playlist_song (playlist_id,song_id,track) VALUES ($1,$2,$3);
-- name: RemovePlaylistTracks :exec
DELETE FROM playlist_song WHERE playlist_id = $1 AND track = any(sqlc.arg('tracks')::int[]);
-- name: DeletePlaylist :execresult
DELETE FROM playlists WHERE id = $1 AND owner = $2;