-- +goose Up
ALTER TABLE participants ADD COLUMN IF NOT EXISTS secret_hash bytea;

-- +goose Down
SELECT 1;
