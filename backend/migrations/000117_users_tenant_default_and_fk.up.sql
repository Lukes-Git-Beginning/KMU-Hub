-- Sprint 4 Welle 0.5 — fix Zero-UUID tenant_id on users + add fk_users_tenant.
-- 2026-05-10
--
-- Pre-state: backend/internal/auth/service.go Register/AcceptInvitation never
-- set User.TenantID, leaving it as uuid.Nil ('00000000-0000-0000-0000-000000000000').
-- That tenant does not exist in the tenants table — only the bootstrap sentinel
-- '00000000-0000-0000-0000-000000000001' (created by migration 000114) is real.
-- This migration repairs already-orphaned rows and prevents future drift via FK.
--
-- The FK is created NOT VALID and validated separately so the lock is briefer
-- (NOT VALID adds catalog metadata only; VALIDATE CONSTRAINT scans rows under
-- a SHARE UPDATE EXCLUSIVE lock that does not block reads/writes on rows that
-- already pass).

BEGIN;

-- 1) Backfill orphaned Zero-UUID users onto the bootstrap default tenant.
UPDATE users
   SET tenant_id = '00000000-0000-0000-0000-000000000001'
 WHERE tenant_id = '00000000-0000-0000-0000-000000000000';

-- 2) Asserts: post-backfill no user may carry a non-existent tenant.
DO $$
BEGIN
    IF (SELECT COUNT(*) FROM users WHERE tenant_id = '00000000-0000-0000-0000-000000000000') > 0 THEN
        RAISE EXCEPTION 'users with zero-uuid tenant_id remain after backfill';
    END IF;

    IF (SELECT COUNT(*) FROM users u
         WHERE NOT EXISTS (SELECT 1 FROM tenants t WHERE t.id = u.tenant_id)) > 0 THEN
        RAISE EXCEPTION 'users referencing non-existent tenant remain — manual triage required';
    END IF;
END$$;

-- 3) Add FK constraint NOT VALID, then validate online.
ALTER TABLE users
    ADD CONSTRAINT fk_users_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) NOT VALID;

ALTER TABLE users VALIDATE CONSTRAINT fk_users_tenant;

-- 4) Safety net: NOT NULL was already set by migration 000104, but enforce again
-- in case a downstream environment skipped it.
ALTER TABLE users ALTER COLUMN tenant_id SET NOT NULL;

COMMIT;
