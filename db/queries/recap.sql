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

-- name: GetScrobbleTopSongsByDuration :many
SELECT songs.*, SUM(scrobbles.duration_ms) as total_duration_ms, albums.name as album_name, albums.replay_gain as album_replay_gain, albums.replay_gain_peak as album_replay_gain_peak, song_stars.created as starred, song_ratings.rating AS user_rating, COALESCE(avgr.rating, 0) AS avg_rating FROM scrobbles
INNER JOIN songs ON scrobbles.song_id = songs.id
LEFT JOIN albums ON albums.id = songs.album_id
LEFT JOIN song_stars ON song_stars.song_id = songs.id AND song_stars.user_name = $1
LEFT JOIN (
  SELECT song_id, AVG(song_ratings.rating) AS rating FROM song_ratings GROUP BY song_id
) avgr ON avgr.song_id = songs.id
LEFT JOIN song_ratings ON song_ratings.song_id = songs.id AND song_ratings.user_name = $1
WHERE scrobbles.user_name = $1 AND scrobbles.now_playing = false AND scrobbles.time >= sqlc.arg('start') AND scrobbles.time < sqlc.arg('end')
GROUP BY songs.id, albums.id, song_stars.created, song_ratings.rating, avgr.rating
ORDER BY SUM(scrobbles.duration_ms) DESC LIMIT $2 OFFSET $3;