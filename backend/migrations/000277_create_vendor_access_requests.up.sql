-- ============================================================================
-- vendor_access_requests: GDAP-light v3 (RBAC R-5 B) — Zentria requests a
-- time-boxed support window into a tenant's data; the tenant's own admin
-- approves/declines/counter-proposes/revokes. This is an ordinary tenant-
-- scoped, authenticated resource, NOT an external unauthenticated endpoint —
-- the backlog premise that assumed the latter was wrong (verified against
-- desktop/src/renderer/src/api/vendor-access.ts, which calls
-- authenticatedRequest like every other admin API).
--
-- Who creates a row is intentionally out of scope here: the frontend contract
-- (vendor-access.ts) has no create call, only list/approve/decline/
-- counter-propose/revoke. Origination is presumably a future Zentria-operator
-- tool outside this tenant-facing API — the repository still exposes
-- CreateRequest for that future caller and for tests, just no HTTP route.
--
-- agents is JSONB (an array of {name}) because Zentria staff are not tenant
-- users and have no row to join against. scope is a Postgres array of area
-- IDs from the frontend's static VENDOR_ACCESS_AREAS catalogue (crm,
-- finance, ..., hr_data, salary) — sensitivity of an area is a frontend
-- constant, mirrored server-side in the service layer, not stored per row.
-- approved_by/revoked_by reference tenant users (the reviewer), resolved to
-- a display name at read time via a join — same as p1b-roles-list's
-- memberCount join, not stored as a name.
-- ============================================================================

CREATE TABLE vendor_access_requests (
    id                     UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id              UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    reason                 TEXT        NOT NULL,
    description            TEXT        NOT NULL DEFAULT '',
    ticket_ref             TEXT        NULL,
    agents                 JSONB       NOT NULL DEFAULT '[]',
    scope                  TEXT[]      NOT NULL DEFAULT '{}',
    requested_start        DATE        NOT NULL,
    duration_days          INTEGER     NOT NULL CHECK (duration_days > 0),
    expires_at             DATE        NOT NULL,
    status                 TEXT        NOT NULL DEFAULT 'pending'
        CONSTRAINT vendor_access_requests_status_check CHECK (status IN
            ('pending', 'counter_proposed', 'active', 'declined', 'expired', 'revoked', 'completed')),
    counter_proposed_start DATE        NULL,
    approved_at            TIMESTAMPTZ NULL,
    approved_by            UUID        NULL REFERENCES users(id) ON DELETE SET NULL,
    sensitive_ack          BOOLEAN     NULL,
    revoked_at             TIMESTAMPTZ NULL,
    revoked_by             UUID        NULL REFERENCES users(id) ON DELETE SET NULL,
    completed_at           TIMESTAMPTZ NULL,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_vendor_access_requests_tenant_status
    ON vendor_access_requests (tenant_id, status, created_at DESC);

CALL enable_tenant_rls('vendor_access_requests');
