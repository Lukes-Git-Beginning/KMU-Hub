-- Migration 000238: inbox_canned_responses
-- Tenant-scoped reply templates ("Textbausteine") for the unified inbox. Kept
-- separate from helpdesk.canned_responses so the notification service stays
-- independent of the helpdesk service (graceful degradation).

BEGIN;

CREATE TABLE inbox_canned_responses (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name       TEXT        NOT NULL,
    body       TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_inbox_canned_tenant ON inbox_canned_responses (tenant_id);

-- RLS: standard tenant isolation (matches pattern in 000232).
ALTER TABLE inbox_canned_responses ENABLE ROW LEVEL SECURITY;
ALTER TABLE inbox_canned_responses FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON inbox_canned_responses
    USING      (tenant_id = current_tenant_id() OR is_system_context())
    WITH CHECK (tenant_id = current_tenant_id() OR is_system_context());

COMMIT;
