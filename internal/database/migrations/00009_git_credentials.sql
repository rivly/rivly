-- +goose Up
CREATE TABLE git_credentials (
    id         INTEGER PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE,
    username   TEXT NOT NULL,
    token_enc  BLOB NOT NULL,
    created_by TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at INTEGER NOT NULL DEFAULT (unixepoch())
);

-- +goose Down
DROP TABLE git_credentials;
