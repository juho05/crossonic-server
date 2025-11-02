-- +migrate Up
ALTER TABLE song_ratings
DROP CONSTRAINT song_ratings_user_name_fkey,
ADD CONSTRAINT song_ratings_user_name_fkey FOREIGN KEY(user_name) REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE album_ratings
DROP CONSTRAINT album_ratings_user_name_fkey,
ADD CONSTRAINT album_ratings_user_name_fkey FOREIGN KEY(user_name) REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE artist_ratings
DROP CONSTRAINT artist_ratings_user_name_fkey,
ADD CONSTRAINT artist_ratings_user_name_fkey FOREIGN KEY(user_name) REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE song_stars
DROP CONSTRAINT song_stars_user_name_fkey,
ADD CONSTRAINT song_stars_user_name_fkey FOREIGN KEY(user_name) REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE album_stars
DROP CONSTRAINT album_stars_user_name_fkey,
ADD CONSTRAINT album_stars_user_name_fkey FOREIGN KEY(user_name) REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE artist_stars
DROP CONSTRAINT artist_stars_user_name_fkey,
ADD CONSTRAINT artist_stars_user_name_fkey FOREIGN KEY(user_name) REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE scrobbles
DROP CONSTRAINT scrobbles_user_name_fkey,
ADD CONSTRAINT scrobbles_user_name_fkey FOREIGN KEY(user_name) REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE playlists
DROP CONSTRAINT playlists_owner_fkey,
ADD CONSTRAINT playlists_owner_fkey FOREIGN KEY(owner) REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE internet_radio_stations
DROP CONSTRAINT internet_radio_stations_user_name_fkey,
ADD CONSTRAINT internet_radio_stations_user_name_fkey FOREIGN KEY(user_name) REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE lb_feedback_status
DROP CONSTRAINT lb_feedback_status_user_name_fkey,
ADD CONSTRAINT lb_feedback_status_user_name_fkey FOREIGN KEY(user_name) REFERENCES users(name) ON UPDATE CASCADE ON DELETE CASCADE;

-- +migrate Down
ALTER TABLE song_ratings
DROP CONSTRAINT song_ratings_user_name_fkey,
ADD CONSTRAINT song_ratings_user_name_fkey FOREIGN KEY(user_name) REFERENCES users(name) ON DELETE CASCADE;

ALTER TABLE album_ratings
DROP CONSTRAINT album_ratings_user_name_fkey,
ADD CONSTRAINT album_ratings_user_name_fkey FOREIGN KEY(user_name) REFERENCES users(name) ON DELETE CASCADE;

ALTER TABLE artist_ratings
DROP CONSTRAINT artist_ratings_user_name_fkey,
ADD CONSTRAINT artist_ratings_user_name_fkey FOREIGN KEY(user_name) REFERENCES users(name) ON DELETE CASCADE;

ALTER TABLE song_stars
DROP CONSTRAINT song_stars_user_name_fkey,
ADD CONSTRAINT song_stars_user_name_fkey FOREIGN KEY(user_name) REFERENCES users(name) ON DELETE CASCADE;

ALTER TABLE album_stars
DROP CONSTRAINT album_stars_user_name_fkey,
ADD CONSTRAINT album_stars_user_name_fkey FOREIGN KEY(user_name) REFERENCES users(name) ON DELETE CASCADE;

ALTER TABLE artist_stars
DROP CONSTRAINT artist_stars_user_name_fkey,
ADD CONSTRAINT artist_stars_user_name_fkey FOREIGN KEY(user_name) REFERENCES users(name) ON DELETE CASCADE;

ALTER TABLE scrobbles
DROP CONSTRAINT scrobbles_user_name_fkey,
ADD CONSTRAINT scrobbles_user_name_fkey FOREIGN KEY(user_name) REFERENCES users(name) ON DELETE CASCADE;

ALTER TABLE playlists
DROP CONSTRAINT playlists_owner_fkey,
ADD CONSTRAINT playlists_owner_fkey FOREIGN KEY(owner) REFERENCES users(name) ON DELETE CASCADE;

ALTER TABLE internet_radio_stations
DROP CONSTRAINT internet_radio_stations_user_name_fkey,
ADD CONSTRAINT internet_radio_stations_user_name_fkey FOREIGN KEY(user_name) REFERENCES users(name) ON DELETE CASCADE;

ALTER TABLE lb_feedback_status
DROP CONSTRAINT lb_feedback_status_user_name_fkey,
ADD CONSTRAINT lb_feedback_status_user_name_fkey FOREIGN KEY(user_name) REFERENCES users(name) ON DELETE CASCADE;
