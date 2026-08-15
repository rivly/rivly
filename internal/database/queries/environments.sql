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

-- name: UpdateEnvironment :one
UPDATE environments
SET
    name = @name,
    kind = @kind,
    url = @url,
    snapshot = CASE WHEN url = @url THEN snapshot END,
    snapshot_at = CASE WHEN url = @url THEN snapshot_at END,
    updated_at = unixepoch()
WHERE id = @id
RETURNING *;

-- name: DeleteEnvironment :execrows
DELETE FROM environments WHERE id = ?;
