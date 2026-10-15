-- +goose Up
-- Tokens issued before this moment are rejected. It is bumped on every password
-- change, which signs the user out everywhere. The epoch default means "never
-- changed", so existing sessions keep working when this migration is applied.
ALTER TABLE users
    ADD COLUMN password_changed_at TIMESTAMPTZ NOT NULL DEFAULT '1970-01-01 00:00:00+00';

-- +goose Down
ALTER TABLE users DROP COLUMN password_changed_at;
