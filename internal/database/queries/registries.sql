-- name: ListRegistries :many
SELECT * FROM registries ORDER BY server;

-- name: GetRegistry :one
SELECT * FROM registries WHERE id = ? LIMIT 1;

-- name: GetRegistryByServer :one
SELECT * FROM registries WHERE server = ? LIMIT 1;

-- name: CreateRegistry :one
INSERT INTO registries (name, server, username, password_enc, created_by)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateRegistry :one
UPDATE registries
SET name = ?, server = ?, username = ?, password_enc = ?, updated_at = unixepoch()
WHERE id = ?
RETURNING *;

-- name: DeleteRegistry :exec
DELETE FROM registries WHERE id = ?;
