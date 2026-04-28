BEGIN;

-- P0 #17: idempotency_keys PRIMARY KEY promoted to (tenant_id, key) to prevent
-- cross-tenant cache replay once HardMode is enabled. Without this, a key
-- produced by Tenant A could be replayed as Tenant B's response.
ALTER TABLE idempotency_keys DROP CONSTRAINT IF EXISTS idempotency_keys_pkey;
ALTER TABLE idempotency_keys ADD PRIMARY KEY (tenant_id, key);

-- Also update the ON CONFLICT target in any existing stored procedures
-- (none at time of authoring — Gateway uses application-level conflict detection).

COMMIT;
