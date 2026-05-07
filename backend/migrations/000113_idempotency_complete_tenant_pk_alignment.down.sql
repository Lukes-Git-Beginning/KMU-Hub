-- Rollback: drop partial index added in 000113
DROP INDEX IF EXISTS idx_idempotency_keys_tenant_inflight;
