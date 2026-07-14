-- +goose Up
ALTER TABLE stacks ADD COLUMN created_by TEXT NOT NULL DEFAULT '';
ALTER TABLE stacks ADD COLUMN updated_by TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE stacks DROP COLUMN created_by;
ALTER TABLE stacks DROP COLUMN updated_by;
