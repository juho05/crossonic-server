-- name: GetScrobbleDurationSumMS :one
SELECT COALESCE(SUM(duration_ms), 0) FROM scrobbles WHERE user_name = $1 AND duration_ms IS NOT NULL AND now_playing = false AND time >= sqlc.arg('start') AND time < sqlc.arg('end');

-- name: GetScrobbleDistinctSongCount :one
SELECT COALESCE(COUNT(DISTINCT song_id), 0) FROM scrobbles WHERE user_name = $1 AND now_playing = false AND time >= sqlc.arg('start') AND time < sqlc.arg('end');

-- name: GetScrobbleDistinctAlbumCount :one
SELECT COALESCE(COUNT(DISTINCT album_id), 0) FROM scrobbles WHERE user_name = $1 AND now_playing = false AND time >= sqlc.arg('start') AND time < sqlc.arg('end');

-- name: GetScrobbleDistinctArtistCount :one
SELECT COALESCE(COUNT(DISTINCT song_artist.artist_id), 0) FROM scrobbles
INNER JOIN song_artist ON scrobbles.song_id = song_artist.song_id
WHERE scrobbles.user_name = $1 AND scrobbles.now_playing = false AND scrobbles.time >= sqlc.arg('start') AND scrobbles.time < sqlc.arg('end');