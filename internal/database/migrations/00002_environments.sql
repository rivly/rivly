-- +goose Up
CREATE TABLE environments (
    id         INTEGER PRIMARY KEY,
    name       TEXT NOT NULL,
    kind       TEXT NOT NULL DEFAULT 'local',
    url        TEXT NOT NULL,
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at INTEGER NOT NULL DEFAULT (unixepoch())
);

-- +goose Down
DROP TABLE environments;
