-- ============================================================================
-- Third slice of backlog unit g-rls-tenant-scoped-admin-writes, following
-- 000273 (two_factor_policy) and 000274 (presence_config, dashboard_defaults):
-- storage_quotas held one global row and both counters wrote it without a
-- WHERE clause (IncrementUsedBytes/DecrementUsedBytes,
-- chat/file/postgres_repository.go:186ff), so every tenant's upload consumed
-- the same shared max_bytes budget as everyone else's.
--
-- Unlike the two previous tables, the old used_bytes cannot be replicated onto
-- every tenant the way max_bytes is: it is a SUM across all tenants' files,
-- and copying it verbatim would tell every tenant it was already using the
-- whole installation's storage. chat_files has carried tenant_id and RLS
-- since 000115/000122, so the honest backfill recomputes used_bytes per
-- tenant from SUM(file_size) over its own non-deleted files instead of
-- reading the old counter at all.
--
-- max_bytes is replicated 1:1 from the old global row, the same lossless
-- reading 000273/000274 used: it was the quota every tenant operated under,
-- so it remains the quota every tenant operates under until an admin changes
-- it.
--
-- No provisioning seed for new tenants: IncrementUsedBytes becomes an upsert
-- (a tenant's first upload creates its row, picking up the column default for
-- max_bytes), so a tenant with no row has used_bytes 0 by construction --
-- exactly the code default GetStorageQuota's caller now falls back to.
-- ============================================================================

ALTER TABLE storage_quotas
    ADD COLUMN tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE;

-- The table never had a unique key beyond id, so in principle several global
-- rows could exist; take the most recently edited one as the quota policy to
-- replicate, same defensive read as 000274 used for presence_config.
INSERT INTO storage_quotas (id, tenant_id, max_bytes, used_bytes, updated_at)
SELECT gen_random_uuid(), t.id, q.max_bytes,
       COALESCE((
           SELECT SUM(cf.file_size) FROM chat_files cf
            WHERE cf.tenant_id = t.id AND cf.is_deleted = FALSE
       ), 0),
       q.updated_at
FROM (
    SELECT * FROM storage_quotas
     WHERE tenant_id IS NULL
     ORDER BY updated_at DESC, id
     LIMIT 1
) q
CROSS JOIN tenants t;

DELETE FROM storage_quotas WHERE tenant_id IS NULL;

ALTER TABLE storage_quotas ALTER COLUMN tenant_id SET NOT NULL;

-- One quota row per tenant. Also the conflict target IncrementUsedBytes
-- upserts on.
CREATE UNIQUE INDEX idx_storage_quotas_tenant ON storage_quotas (tenant_id);

CALL enable_tenant_rls('storage_quotas');
