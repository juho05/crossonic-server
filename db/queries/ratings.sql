-- name: SetSongRating :exec
INSERT INTO song_ratings (song_id,user_name,rating) VALUES ($1, $2, $3) ON CONFLICT(song_id,user_name) DO UPDATE SET rating = $3;
-- name: SetAlbumRating :exec
INSERT INTO album_ratings (album_id,user_name,rating) VALUES ($1, $2, $3) ON CONFLICT(album_id,user_name) DO UPDATE SET rating = $3;
-- name: SetArtistRating :exec
INSERT INTO artist_ratings (artist_id,user_name,rating) VALUES ($1, $2, $3) ON CONFLICT(artist_id,user_name) DO UPDATE SET rating = $3;
-- name: RemoveSongRating :exec
DELETE FROM song_ratings WHERE user_name = $1 AND song_id = $2;
-- name: RemoveAlbumRating :exec
DELETE FROM album_ratings WHERE user_name = $1 AND album_id = $2;
-- name: RemoveArtistRating :exec
DELETE FROM artist_ratings WHERE user_name = $1 AND artist_id = $2;
-- name: StarSong :exec
INSERT INTO song_stars (song_id, user_name, created) VALUES ($1, $2, NOW()) ON CONFLICT(song_id,user_name) DO NOTHING;
-- name: StarAlbum :exec
INSERT INTO album_stars (album_id, user_name, created) VALUES ($1, $2, NOW()) ON CONFLICT(album_id,user_name) DO NOTHING;
-- name: StarArtist :exec
INSERT INTO artist_stars (artist_id, user_name, created) VALUES ($1, $2, NOW()) ON CONFLICT(artist_id,user_name) DO NOTHING;
-- name: UnstarSong :exec
DELETE FROM song_stars WHERE user_name = $1 AND song_id = $2;
-- name: UnstarAlbum :exec
DELETE FROM album_stars WHERE user_name = $1 AND album_id = $2;
-- name: UnstarArtist :exec
DELETE FROM artist_stars WHERE user_name = $1 AND artist_id = $2;
-- name: SetLBFeedbackUpdated :copyfrom
INSERT INTO lb_feedback_updated (song_id,user_name,mbid) VALUES ($1,$2,$3);
-- name: RemoveLBFeedbackUpdated :exec
DELETE FROM lb_feedback_updated WHERE user_name = $1 AND song_id = any(sqlc.arg('song_ids')::text[]);
-- name: FindLBFeedbackUpdatedSongIDsInMBIDListNotStarred :many
SELECT lb_feedback_updated.song_id FROM lb_feedback_updated LEFT JOIN song_stars ON song_stars.user_name = $1 AND song_stars.song_id = lb_feedback_updated.song_id WHERE lb_feedback_updated.user_name = $1 AND song_stars.song_id IS NULL AND lb_feedback_updated.mbid = any(sqlc.arg('song_mbids')::text[]);
-- name: DeleteLBFeedbackUpdatedStarsNotInMBIDList :execresult
DELETE FROM song_stars WHERE song_stars.user_name = $1 AND song_stars.song_id IN (
  SELECT lb_feedback_updated.song_id FROM lb_feedback_updated WHERE lb_feedback_updated.user_name = $1 AND NOT (lb_feedback_updated.mbid = any(sqlc.arg('song_mbids')::text[]))
);
-- name: StarSongs :copyfrom
INSERT INTO song_stars (song_id, user_name, created) VALUES ($1, $2, $3);
-- name: FindNotLBUpdatedSongs :many
SELECT songs.*, albums.name as album_name, song_stars.created as starred FROM songs
LEFT JOIN albums ON albums.id = songs.album_id
LEFT JOIN song_stars ON song_stars.song_id = songs.id AND song_stars.user_name = $1
LEFT JOIN lb_feedback_updated ON lb_feedback_updated.user_name = $1 AND lb_feedback_updated.song_id = songs.id
WHERE lb_feedback_updated.song_id IS NULL;