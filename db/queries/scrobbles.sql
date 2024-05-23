-- name: CreateScrobbles :copyfrom
INSERT INTO scrobbles
(user_name,song_id,album_id,time,song_duration_ms,duration_ms,submitted_to_listenbrainz,now_playing)
VALUES
($1,$2,$3,$4,$5,$6,$7,$8);
-- name: DeleteNowPlaying :exec
DELETE FROM scrobbles WHERE now_playing = true AND (user_name = $1 OR EXTRACT(EPOCH FROM (NOW() - time))*1000 > song_duration_ms*3);
-- name: GetNowPlaying :one
SELECT * FROM scrobbles WHERE user_name = $1 AND now_playing = true AND EXTRACT(EPOCH FROM (NOW() - time))*1000 < song_duration_ms*3;
-- name: GetNowPlayingSongs :many
SELECT scrobbles.user_name, scrobbles.time, songs.*, albums.name as album_name, albums.replay_gain as album_replay_gain, albums.replay_gain_peak as album_replay_gain_peak, song_stars.created as starred, song_ratings.rating AS user_rating, COALESCE(avgr.rating, 0) AS avg_rating FROM songs
JOIN scrobbles ON scrobbles.song_id = songs.id
LEFT JOIN albums ON albums.id = songs.album_id
LEFT JOIN song_stars ON song_stars.song_id = songs.id AND song_stars.user_name = $1
LEFT JOIN (
  SELECT song_id, AVG(song_ratings.rating) AS rating FROM song_ratings GROUP BY song_id
) avgr ON avgr.song_id = songs.id
LEFT JOIN song_ratings ON song_ratings.song_id = songs.id AND song_ratings.user_name = $1
WHERE scrobbles.now_playing = true AND EXTRACT(EPOCH FROM (NOW() - time))*1000 < scrobbles.song_duration_ms*3
ORDER BY scrobbles.time DESC;
-- name: FindUnsubmittedLBScrobbles :many
SELECT * FROM scrobbles JOIN users ON scrobbles.user_name = users.name WHERE users.listenbrainz_username IS NOT NULL AND now_playing = false AND submitted_to_listenbrainz = false AND (duration_ms >= 4*60*1000 OR duration_ms >= song_duration_ms*0.5);
-- name: SetLBSubmittedByUsers :exec
UPDATE scrobbles SET submitted_to_listenbrainz = true WHERE user_name = any(sqlc.arg('user_names')::text[]) AND now_playing = false AND (duration_ms >= 4*60*1000 OR duration_ms >= song_duration_ms*0.5);