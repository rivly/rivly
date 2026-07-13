-- name: CountEnvironments :one
SELECT count(*) FROM environments;

-- name: ListEnvironments :many
SELECT * FROM environments ORDER BY id;

-- name: GetEnvironment :one
SELECT * FROM environments WHERE id = ? LIMIT 1;

-- name: CreateEnvironment :one
INSERT INTO environments (name, kind, url)
VALUES (?, ?, ?)
RETURNING *;

-- name: UpdateEnvironmentSnapshot :exec
UPDATE environments
SET snapshot = ?, snapshot_at = unixepoch()
WHERE id = ?;
