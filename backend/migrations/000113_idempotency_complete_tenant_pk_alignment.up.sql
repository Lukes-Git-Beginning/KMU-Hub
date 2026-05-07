-- Idempotency Complete / Tenant PK alignment tracking migration.
--
-- Context:
--   000105 created idempotency_keys with PK (key) and indexes:
--     - idx_idempotency_keys_tenant_user  (tenant_id, user_id, created_at DESC)
--     - idx_idempotency_keys_expires      (expires_at)
--   000108 promoted the PK to (tenant_id, key) — the composite key prevents
--     cross-tenant cache-replay (P0 #17).
--
--   The PK (tenant_id, key) already covers the primary lookup in Reserve().
--   However, the Complete() path additionally filters on completed_at IS NULL
--   and updates completed_at + response fields.  A partial index on
--   (tenant_id, key) WHERE completed_at IS NULL gives the planner a tight
--   scan for in-flight keys without touching completed rows.
--
--   App-code fix for Complete() ON CONFLICT targeting the composite PK is
--   handled in Stream 2C (no DDL required here).

CREATE INDEX IF NOT EXISTS idx_idempotency_keys_tenant_inflight
    ON idempotency_keys(tenant_id, key)
    WHERE completed_at IS NULL;
