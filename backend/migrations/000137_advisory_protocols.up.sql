-- Migration 000137 — Advisory Protocols (Beratungsprotokolle)
--
-- Adds the advisory_protocols table for legal financial advisory documentation
-- per MiFID II / §64 WpHG / §16-18 FinVermV / §61 VVG (IDD).
-- ESG preferences per EU-DelegVO 2021/1253.
-- 10-year retention per FinVermV §18a + civil-law statutes of limitation.
--
-- Immutability decision: service-side enforcement (not DB trigger).
-- Rationale: existing repo pattern (consent_records, finance) enforces business
-- rules in Go service layer for testability + gracefulness. The `status` column
-- (draft | finalized) is the single source of truth; the service rejects any
-- UPDATE/DELETE on a finalized record with ErrProtocolFinalized.
--
-- Products: stored as JSONB array. The FE AdvisoryProduct list is variable-length
-- (0–N products per session) and has no FK relationship — JSONB is idiomatic here.
-- All other fields are typed columns for query performance + stricter constraints.

-- ============================================================================
-- Contacts extension: referred_by + client_segment
-- ============================================================================

ALTER TABLE contacts
    ADD COLUMN IF NOT EXISTS referred_by_contact_id UUID REFERENCES contacts(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS client_segment CHAR(1) CHECK (client_segment IN ('A', 'B', 'C'));

CREATE INDEX IF NOT EXISTS idx_contacts_referred_by ON contacts(referred_by_contact_id) WHERE referred_by_contact_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_contacts_client_segment ON contacts(client_segment) WHERE client_segment IS NOT NULL;

-- ============================================================================
-- advisory_protocols table
-- ============================================================================

CREATE TYPE advisory_status AS ENUM ('draft', 'finalized');

CREATE TABLE advisory_protocols (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID NOT NULL,
    contact_id              UUID NOT NULL REFERENCES contacts(id) ON DELETE RESTRICT,
    created_by              UUID NOT NULL,
    status                  advisory_status NOT NULL DEFAULT 'draft',
    handed_over_at          TIMESTAMPTZ,                       -- set when finalized/handed-over

    -- § 1 Gesprächskopf
    date                    DATE,
    time_from               TIME,
    time_to                 TIME,
    location                VARCHAR(20),                       -- office|phone|video|onsite
    advisor                 TEXT NOT NULL DEFAULT '',
    occasion                VARCHAR(20),                       -- initial|followup|event
    occasion_note           TEXT NOT NULL DEFAULT '',
    customer_category       VARCHAR(20),                       -- private|professional

    -- § 2 Kunde & Profil
    birth_date              DATE,
    marital_status          TEXT NOT NULL DEFAULT '',
    tax_status              TEXT NOT NULL DEFAULT '',

    -- § 3 Kenntnisse & Erfahrungen (§16 FinVermV)
    known_asset_classes     TEXT[] NOT NULL DEFAULT '{}',      -- stocks|funds|etf|bonds|derivatives|realestate|crypto
    past_transactions       TEXT NOT NULL DEFAULT '',
    financial_education     TEXT NOT NULL DEFAULT '',
    professional_experience TEXT NOT NULL DEFAULT '',
    self_assessment         SMALLINT CHECK (self_assessment BETWEEN 1 AND 5),

    -- § 4 Finanzielle Situation
    monthly_net_income      NUMERIC(14,2),
    recurring_liabilities   NUMERIC(14,2),
    liquid_assets           NUMERIC(14,2),
    current_investments     NUMERIC(14,2),
    real_estate             TEXT NOT NULL DEFAULT '',
    existing_insurance      TEXT NOT NULL DEFAULT '',
    max_loss_capacity_abs   NUMERIC(14,2),
    max_loss_capacity_pct   NUMERIC(5,2),

    -- § 5 Anlageziele & Risikoprofil (PRIIP SRI 1-7)
    investment_purpose      TEXT[] NOT NULL DEFAULT '{}',      -- retirement|liquidity|growth|saving|speculation
    horizon                 VARCHAR(10) NOT NULL DEFAULT '',   -- lt1|1to3|3to5|5to10|gt10
    risk_tolerance          TEXT NOT NULL DEFAULT '',
    risk_capacity           TEXT NOT NULL DEFAULT '',
    risk_class              SMALLINT NOT NULL DEFAULT 3 CHECK (risk_class BETWEEN 1 AND 7),
    esg_preference          BOOLEAN NOT NULL DEFAULT FALSE,
    esg_details             TEXT NOT NULL DEFAULT '',
    one_time_amount         NUMERIC(14,2),
    monthly_savings         NUMERIC(14,2),

    -- § 6 Besprochene Produkte (variable-length list — JSONB)
    -- Each element: {id, name, isin?, category?, riskClass(1-7), chances?, risks, costsOneTime?, costsRunning?, recommended}
    products                JSONB NOT NULL DEFAULT '[]'::jsonb,

    -- § 7 Empfehlung & Geeignetheitsbegründung
    recommendation_summary  TEXT NOT NULL DEFAULT '',
    suitability_reasoning   TEXT NOT NULL DEFAULT '',
    goal_reference          TEXT NOT NULL DEFAULT '',
    alternatives            TEXT NOT NULL DEFAULT '',
    not_recommended         TEXT NOT NULL DEFAULT '',

    -- § 8 Abschluss & Compliance
    main_concerns           TEXT NOT NULL DEFAULT '',
    warnings_given          TEXT[] NOT NULL DEFAULT '{}',      -- risk|costs|conflicts|pastPerformance|liquidity
    document_delivered      BOOLEAN NOT NULL DEFAULT FALSE,
    document_delivered_date DATE,
    delivery_form           VARCHAR(10) NOT NULL DEFAULT 'email', -- paper|email|portal
    advisor_signature       TEXT NOT NULL DEFAULT '',
    customer_confirmation   BOOLEAN NOT NULL DEFAULT FALSE,
    document_waiver         BOOLEAN NOT NULL DEFAULT FALSE,
    followup_date           DATE,
    internal_notes          TEXT NOT NULL DEFAULT '',

    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_advisory_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

-- Performance indexes
CREATE INDEX idx_advisory_protocols_contact   ON advisory_protocols(contact_id, tenant_id);
CREATE INDEX idx_advisory_protocols_tenant    ON advisory_protocols(tenant_id);
CREATE INDEX idx_advisory_protocols_status    ON advisory_protocols(status) WHERE status = 'draft';
CREATE INDEX idx_advisory_protocols_date      ON advisory_protocols(date DESC) WHERE date IS NOT NULL;
CREATE INDEX idx_advisory_protocols_created   ON advisory_protocols(created_at DESC);

-- ============================================================================
-- RBAC — permission seeds
-- ============================================================================
-- advisory-protocols:read   — list and view protocols (finalized are read-only for all)
-- advisory-protocols:write  — create drafts and update them before finalization
-- advisory-protocols:delete — delete draft-only protocols; separate from write
--
-- Grant: admin + manager get full access; member gets read (can view their own
-- protocols) — delete is admin/manager only (legal compliance).

INSERT INTO permissions (name, resource, action) VALUES
    ('advisory-protocols:read',   'advisory-protocols', 'read'),
    ('advisory-protocols:write',  'advisory-protocols', 'write'),
    ('advisory-protocols:delete', 'advisory-protocols', 'delete')
ON CONFLICT (name) DO NOTHING;

-- admin gets all three
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'admin'
  AND p.name IN ('advisory-protocols:read', 'advisory-protocols:write', 'advisory-protocols:delete')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- manager gets read + write (not delete)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'manager'
  AND p.name IN ('advisory-protocols:read', 'advisory-protocols:write')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- member gets read only
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'member'
  AND p.name = 'advisory-protocols:read'
ON CONFLICT (role_id, permission_id) DO NOTHING;
