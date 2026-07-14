-- name: ListGitCredentials :many
SELECT * FROM git_credentials ORDER BY name;

-- name: GetGitCredential :one
SELECT * FROM git_credentials WHERE id = ? LIMIT 1;

-- name: CreateGitCredential :one
INSERT INTO git_credentials (name, username, token_enc, created_by)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: UpdateGitCredential :one
UPDATE git_credentials
SET name = ?, username = ?, token_enc = ?, updated_at = unixepoch()
WHERE id = ?
RETURNING *;

-- name: DeleteGitCredential :exec
DELETE FROM git_credentials WHERE id = ?;
