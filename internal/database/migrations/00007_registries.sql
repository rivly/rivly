-- +goose Up
CREATE TABLE registries (
    id           INTEGER PRIMARY KEY,
    server       TEXT NOT NULL UNIQUE,
    username     TEXT NOT NULL,
    password_enc BLOB NOT NULL,
    created_by   TEXT NOT NULL DEFAULT '',
    created_at   INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at   INTEGER NOT NULL DEFAULT (unixepoch())
);

-- +goose Down
DROP TABLE registries;
