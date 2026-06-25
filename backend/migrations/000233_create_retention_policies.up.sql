-- Migration: 000233_create_retention_policies
-- DSGVO Art. 5(1)(e) Speicherbegrenzung — tenant-scoped retention policies.
-- Follows the same RLS pattern as ip_access_rules (Migration 000124).

CREATE TABLE retention_policies (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID         NOT NULL,
    resource_type VARCHAR(64)  NOT NULL,
    retention_days INT         NOT NULL CHECK (retention_days > 0),
    action        VARCHAR(16)  NOT NULL CHECK (action IN ('delete', 'anonymize')),
    enabled       BOOLEAN      NOT NULL DEFAULT true,
    description   TEXT,
    created_by    UUID         REFERENCES users(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_retention_policies_tenant_resource UNIQUE (tenant_id, resource_type)
);

CREATE INDEX idx_retention_policies_tenant_id ON retention_policies (tenant_id);

-- Enable RLS using the project-standard stored procedure (defined in Migration 000118).
CALL enable_tenant_rls('retention_policies');
