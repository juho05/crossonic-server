-- +migrate Up
CREATE TABLE music_folders (
    id int NOT NULL PRIMARY KEY,
    name text NOT NULL,
    path text NOT NULL
);

CREATE TABLE music_folder_users (
    music_folder_id int NOT NULL REFERENCES music_folders(id) ON DELETE CASCADE ON UPDATE CASCADE,
    user_name text NOT NULL REFERENCES users(name) ON DELETE CASCADE ON UPDATE CASCADE
);

INSERT INTO music_folders (id, name, path) VALUES (1, 'Default', '');
INSERT INTO music_folder_users (music_folder_id, user_name) SELECT 1, users.name FROM users;

ALTER TABLE songs ADD COLUMN music_folder_id int REFERENCES music_folders(id) ON DELETE SET NULL ON UPDATE CASCADE;
ALTER TABLE albums ADD COLUMN music_folder_id int REFERENCES music_folders(id) ON DELETE SET NULL ON UPDATE CASCADE;

-- +migrate Down
ALTER TABLE albums DROP COLUMN music_folder_id;
ALTER TABLE songs DROP COLUMN music_folder_id;
DROP TABLE music_folder_users;
DROP TABLE music_folders;