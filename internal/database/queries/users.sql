-- name: CountUsers :one
SELECT count(*) FROM users;

-- name: CreateUser :one
INSERT INTO users (email, display_name, role)
VALUES (?, ?, ?)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = ? LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = ? LIMIT 1;

-- name: CreatePasswordCredential :one
INSERT INTO credentials (user_id, type, secret)
VALUES (?, 'password', ?)
RETURNING *;

-- name: GetPasswordCredential :one
SELECT * FROM credentials
WHERE user_id = ? AND type = 'password'
LIMIT 1;

-- name: UpdateUserProfile :one
UPDATE users
SET display_name = ?, updated_at = unixepoch()
WHERE id = ?
RETURNING *;

-- name: UpdatePasswordCredential :exec
UPDATE credentials
SET secret = ?
WHERE user_id = ? AND type = 'password';
