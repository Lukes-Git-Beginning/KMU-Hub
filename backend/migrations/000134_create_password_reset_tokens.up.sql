-- Migration 000134: password_reset_tokens
-- Single-use, short-lived tokens for the forgot-password flow.
-- tenant_id NOT NULL follows Option-B/RLS convention.
-- token_hash is SHA-256 hex of the opaque plaintext token (same
-- pattern as refresh_tokens and invitations).

CREATE TABLE password_reset_tokens (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID        NOT NULL,
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  VARCHAR(64) NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Fast lookup by hash (primary query path).
CREATE UNIQUE INDEX idx_password_reset_tokens_hash
    ON password_reset_tokens (token_hash);

-- Useful for "invalidate all pending resets for a user" queries.
CREATE INDEX idx_password_reset_tokens_user_id
    ON password_reset_tokens (user_id);

-- Partial index supports expiry-sweeps and single-use enforcement
-- without scanning rows that are already consumed.
CREATE INDEX idx_password_reset_tokens_expires_at_pending
    ON password_reset_tokens (expires_at)
    WHERE used_at IS NULL;
