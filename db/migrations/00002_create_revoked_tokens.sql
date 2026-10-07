-- +goose Up
-- Logout support: JWTs are stateless, so we remember revoked token IDs (jti)
-- until they would have expired anyway.
CREATE TABLE revoked_tokens (
    jti        TEXT        PRIMARY KEY,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_revoked_tokens_expires_at ON revoked_tokens (expires_at);

-- +goose Down
DROP TABLE revoked_tokens;
