-- name: CreateUser :exec
INSERT INTO users (
  name, encrypted_password
) VALUES (
  $1, $2
);

-- name: UpdateUserListenBrainzConnection :one
UPDATE users SET encrypted_listenbrainz_token = $2, listenbrainz_username = $3 WHERE name = $1 RETURNING *;

-- name: FindUsers :many
SELECT * FROM users;

-- name: FindUser :one
SELECT * FROM users WHERE name = $1;

-- name: DeleteUser :one
DELETE FROM users WHERE name = $1 RETURNING name;