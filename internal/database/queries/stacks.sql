-- name: ListStacks :many
SELECT * FROM stacks WHERE env_id = ? ORDER BY name;

-- name: GetStack :one
SELECT * FROM stacks WHERE env_id = ? AND name = ? LIMIT 1;

-- name: ListAutoUpdateStacks :many
SELECT * FROM stacks WHERE source = 'git' AND git_auto_update = 1;

-- name: UpsertStack :one
INSERT INTO stacks (
    env_id, name, content, env, created_by, updated_by,
    source, git_url, git_ref, git_path, git_credential_id, git_commit,
    git_auto_update, git_poll_interval, git_remote_hash
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
    git_auto_update = excluded.git_auto_update,
    git_poll_interval = excluded.git_poll_interval,
    git_remote_hash = excluded.git_remote_hash,
    git_last_error = '',
    updated_at = unixepoch()
RETURNING *;

-- name: MarkStackChecked :exec
UPDATE stacks
SET git_last_checked_at = unixepoch(), git_last_error = ?, git_remote_hash = ?
WHERE id = ?;

-- name: ApplyStackGitUpdate :exec
UPDATE stacks
SET content = ?, git_commit = ?, git_remote_hash = ?, updated_by = ?,
    updated_at = unixepoch(), git_last_checked_at = unixepoch(), git_last_error = ''
WHERE id = ?;

-- name: DeleteStack :exec
DELETE FROM stacks WHERE env_id = ? AND name = ?;
