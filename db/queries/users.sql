-- name: CreateUser :exec
INSERT INTO users (
  name, encrypted_password
) VALUES (
  $1, $2
);

-- name: FindUsers :many
SELECT * FROM users;

-- name: FindUser :one
SELECT * FROM users WHERE name = $1;

-- name: DeleteUser :one
DELETE FROM users WHERE name = $1 RETURNING name;