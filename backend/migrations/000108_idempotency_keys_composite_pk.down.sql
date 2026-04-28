BEGIN;

-- Revert composite PK back to single-column PK on (key).
-- Note: this rollback intentionally does NOT add a FK tenant_id -> tenants(id)
-- because the sentinel UUID (00000000-0000-0000-0000-000000000001) used during
-- the Option-B Phase 1 backfill (000106) would violate such a constraint for
-- legacy rows that predate real Tenant records. The FK will be added in a
-- dedicated cleanup migration after Pilot-1 when legacy rows are purged.
ALTER TABLE idempotency_keys DROP CONSTRAINT IF EXISTS idempotency_keys_pkey;
ALTER TABLE idempotency_keys ADD PRIMARY KEY (key);

COMMIT;
