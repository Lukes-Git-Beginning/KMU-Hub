-- Add tenant_id to users table for JWT claim extraction and cross-tenant isolation.
-- Single-tenant installs: all existing users get the default sentinel tenant.
-- Multi-tenant (Option-B): tenant_id is populated at registration/invitation time.
ALTER TABLE users
    ADD COLUMN tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001';

CREATE INDEX idx_users_tenant_id ON users (tenant_id);
