-- +goose Up
ALTER TABLE stacks ADD COLUMN git_auto_update INTEGER NOT NULL DEFAULT 0;
ALTER TABLE stacks ADD COLUMN git_poll_interval INTEGER NOT NULL DEFAULT 300;
ALTER TABLE stacks ADD COLUMN git_remote_hash TEXT NOT NULL DEFAULT '';
ALTER TABLE stacks ADD COLUMN git_last_checked_at INTEGER NOT NULL DEFAULT 0;
ALTER TABLE stacks ADD COLUMN git_last_error TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE stacks DROP COLUMN git_last_error;
ALTER TABLE stacks DROP COLUMN git_last_checked_at;
ALTER TABLE stacks DROP COLUMN git_remote_hash;
ALTER TABLE stacks DROP COLUMN git_poll_interval;
ALTER TABLE stacks DROP COLUMN git_auto_update;
