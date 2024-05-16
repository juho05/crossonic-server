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