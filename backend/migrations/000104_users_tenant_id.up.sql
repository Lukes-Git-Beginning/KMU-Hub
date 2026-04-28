-- Add tenant_id to users table for JWT claim extraction and cross-tenant isolation.
-- Single-tenant installs: all existing users get the default sentinel tenant.
-- Multi-tenant (Option-B): tenant_id is populated at registration/invitation time.
-- Postgres 11+ stores constant DEFAULT in catalog; no row rewrite, brief lock only.
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001';

CREATE INDEX IF NOT EXISTS idx_users_tenant_id ON users (tenant_id);
