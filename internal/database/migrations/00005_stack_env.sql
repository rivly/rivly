-- +goose Up
ALTER TABLE stacks ADD COLUMN env TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE stacks DROP COLUMN env;
