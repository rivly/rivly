-- +goose Up
ALTER TABLE registries ADD COLUMN name TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE registries DROP COLUMN name;
