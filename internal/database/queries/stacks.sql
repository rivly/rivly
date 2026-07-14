-- name: ListStacks :many
SELECT * FROM stacks WHERE env_id = ? ORDER BY name;

-- name: GetStack :one
SELECT * FROM stacks WHERE env_id = ? AND name = ? LIMIT 1;

-- name: UpsertStack :one
INSERT INTO stacks (env_id, name, content, env, created_by, updated_by)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT (env_id, name) DO UPDATE SET
    content = excluded.content,
    env = excluded.env,
    updated_by = excluded.updated_by,
    updated_at = unixepoch()
RETURNING *;

-- name: DeleteStack :exec
DELETE FROM stacks WHERE env_id = ? AND name = ?;
