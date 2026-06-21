-- +goose Up
ALTER TABLE users
ALTER COLUMN refresh_token DROP NOT NULL;

-- +goose Down
ALTER TABLE users
ALTER COLUMN refresh_token SET NOT NULL;
