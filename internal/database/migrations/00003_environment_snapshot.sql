-- +goose Up
ALTER TABLE environments ADD COLUMN snapshot TEXT;
ALTER TABLE environments ADD COLUMN snapshot_at INTEGER;

-- +goose Down
ALTER TABLE environments DROP COLUMN snapshot_at;
ALTER TABLE environments DROP COLUMN snapshot;
