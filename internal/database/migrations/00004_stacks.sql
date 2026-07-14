-- +goose Up
CREATE TABLE stacks (
    id         INTEGER PRIMARY KEY,
    env_id     INTEGER NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    content    TEXT NOT NULL,
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at INTEGER NOT NULL DEFAULT (unixepoch()),
    UNIQUE (env_id, name)
);

-- +goose Down
DROP TABLE stacks;
