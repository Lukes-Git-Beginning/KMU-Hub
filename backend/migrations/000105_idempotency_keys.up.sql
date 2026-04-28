CREATE TABLE IF NOT EXISTS idempotency_keys (
    key            TEXT PRIMARY KEY,
    tenant_id      UUID NOT NULL,
    user_id        UUID NOT NULL,
    method         TEXT NOT NULL,
    path           TEXT NOT NULL,
    request_hash   TEXT NOT NULL,
    response_status INT NULL,
    response_body  JSONB NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at   TIMESTAMPTZ NULL,
    expires_at     TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '24 hours')
);
CREATE INDEX IF NOT EXISTS idx_idempotency_keys_tenant_user
    ON idempotency_keys(tenant_id, user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_idempotency_keys_expires
    ON idempotency_keys(expires_at);
