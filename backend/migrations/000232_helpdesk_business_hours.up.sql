-- Migration 000232: helpdesk_business_hours
-- One config row per tenant: weekly schedule + holidays + timezone.
-- JSONB keeps the schema flexible without per-day columns.

BEGIN;

CREATE TABLE helpdesk_business_hours (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    -- JSONB: { "Montag": {"active":true,"start":"08:00","end":"17:30"}, ... }
    schedule   JSONB       NOT NULL DEFAULT '{}'::jsonb,
    -- JSONB: [{"id":"h-1","date":"2026-01-01","name":"Neujahr"}, ...]
    holidays   JSONB       NOT NULL DEFAULT '[]'::jsonb,
    timezone   TEXT        NOT NULL DEFAULT 'Europe/Berlin',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id)
);

CREATE INDEX idx_helpdesk_business_hours_tenant
    ON helpdesk_business_hours (tenant_id);

-- RLS: standard tenant isolation (matches pattern in 000146)
ALTER TABLE helpdesk_business_hours ENABLE ROW LEVEL SECURITY;
ALTER TABLE helpdesk_business_hours FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON helpdesk_business_hours
    USING  (tenant_id = current_tenant_id() OR is_system_context())
    WITH CHECK (tenant_id = current_tenant_id() OR is_system_context());

COMMIT;
