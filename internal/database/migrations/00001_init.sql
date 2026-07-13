-- +goose Up
CREATE TABLE users (
    id                INTEGER PRIMARY KEY,
    email             TEXT NOT NULL UNIQUE,
    email_verified_at INTEGER,
    display_name      TEXT NOT NULL DEFAULT '',
    role              TEXT NOT NULL DEFAULT 'admin',
    created_at        INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at        INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE credentials (
    id                  INTEGER PRIMARY KEY,
    user_id             INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type                TEXT NOT NULL,
    secret              TEXT,
    provider            TEXT,
    provider_account_id TEXT,
    created_at          INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE INDEX idx_credentials_user_id ON credentials(user_id);

CREATE TABLE tokens (
    id         INTEGER PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    purpose    TEXT NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at INTEGER NOT NULL,
    used_at    INTEGER
);

CREATE INDEX idx_tokens_user_id ON tokens(user_id);

CREATE TABLE sessions (
    token  TEXT PRIMARY KEY,
    data   BLOB NOT NULL,
    expiry REAL NOT NULL
);

CREATE INDEX sessions_expiry_idx ON sessions(expiry);

-- +goose Down
DROP TABLE sessions;
DROP TABLE tokens;
DROP TABLE credentials;
DROP TABLE users;
