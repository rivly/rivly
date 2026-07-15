-- name: ListStacks :many
SELECT * FROM stacks WHERE env_id = ? ORDER BY name;

-- name: GetStack :one
SELECT * FROM stacks WHERE env_id = ? AND name = ? LIMIT 1;

-- name: UpsertStack :one
INSERT INTO stacks (
    env_id, name, content, env, created_by, updated_by,
    source, git_url, git_ref, git_path, git_credential_id, git_commit
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (env_id, name) DO UPDATE SET
    content = excluded.content,
    env = excluded.env,
    updated_by = excluded.updated_by,
    source = excluded.source,
    git_url = excluded.git_url,
    git_ref = excluded.git_ref,
    git_path = excluded.git_path,
    git_credential_id = excluded.git_credential_id,
    git_commit = excluded.git_commit,
    updated_at = unixepoch()
RETURNING *;

-- name: DeleteStack :exec
DELETE FROM stacks WHERE env_id = ? AND name = ?;
