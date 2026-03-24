-- +migrate Up
CREATE TABLE music_folder_artists (
    music_folder_id int NOT NULL REFERENCES music_folders(id) ON DELETE CASCADE ON UPDATE CASCADE,
    artist_id text NOT NULL REFERENCES artists(id) ON DELETE CASCADE ON UPDATE CASCADE,
    PRIMARY KEY (music_folder_id,artist_id)
);
INSERT INTO music_folder_artists (music_folder_id,artist_id) SELECT 1, artists.id FROM artists;

ALTER TABLE music_folder_users ADD PRIMARY KEY (music_folder_id,user_name);

-- +migrate Down
ALTER TABLE music_folder_users DROP CONSTRAINT music_folder_users_pkey;
DROP TABLE music_folder_artists;
