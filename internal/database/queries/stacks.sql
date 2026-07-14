-- name: ListStacks :many
SELECT * FROM stacks WHERE env_id = ? ORDER BY name;

-- name: GetStack :one
SELECT * FROM stacks WHERE env_id = ? AND name = ? LIMIT 1;

-- name: UpsertStack :one
INSERT INTO stacks (env_id, name, content)
VALUES (?, ?, ?)
ON CONFLICT (env_id, name) DO UPDATE SET
    content = excluded.content,
    updated_at = unixepoch()
RETURNING *;

-- name: DeleteStack :exec
DELETE FROM stacks WHERE env_id = ? AND name = ?;
