-- Sprint 2 S1.5 Vertraege — schema

-- ============================================================================
-- contracts
-- ============================================================================
CREATE TABLE contracts (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID        NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    contract_number    TEXT        NOT NULL,
    title              TEXT        NOT NULL,
    contract_type      TEXT        NOT NULL CHECK (contract_type IN ('rental','service','employment','nda','other')),
    status             TEXT        NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','active','expired','terminated')),
    starts_on          DATE        NOT NULL,
    ends_on            DATE        NULL,    -- NULL = open-ended contract
    document_url       TEXT        NULL,    -- MinIO path; populated by Sprint 3 document integration
    notes              TEXT        NOT NULL DEFAULT '',
    created_by         UUID        NULL REFERENCES users(id) ON DELETE SET NULL,
    signature_provider VARCHAR(50) NULL,    -- Phase D: Skribble integration placeholder
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (tenant_id, contract_number)
);

CREATE INDEX idx_contracts_tenant_status ON contracts (tenant_id, status);
CREATE INDEX idx_contracts_ends_on       ON contracts (ends_on) WHERE status = 'active';

-- ============================================================================
-- contract_parties
-- ============================================================================
CREATE TABLE contract_parties (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID        NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    contract_id      UUID        NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
    party_type       TEXT        NOT NULL CHECK (party_type IN ('contact','company','external')),
    contact_id       UUID        NULL REFERENCES contacts(id) ON DELETE SET NULL,
    company_id       UUID        NULL REFERENCES companies(id) ON DELETE SET NULL,
    external_name    TEXT        NULL,   -- used when party_type = 'external'
    role_in_contract TEXT        NOT NULL,
    signed_on        DATE        NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_contract_parties_contract ON contract_parties (contract_id);
CREATE INDEX idx_contract_parties_contact  ON contract_parties (contact_id) WHERE contact_id IS NOT NULL;

-- ============================================================================
-- contract_reminders
-- ============================================================================
CREATE TABLE contract_reminders (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID        NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    contract_id   UUID        NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
    remind_at     TIMESTAMPTZ NOT NULL,
    reminder_type TEXT        NOT NULL CHECK (reminder_type IN ('renewal','expiry','payment','custom')),
    subject       TEXT        NOT NULL,
    message       TEXT        NOT NULL DEFAULT '',
    status        TEXT        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','sent','cancelled')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_at       TIMESTAMPTZ NULL
);

-- Partial index for the reminder worker query performance
CREATE INDEX idx_contract_reminders_pending ON contract_reminders (remind_at) WHERE status = 'pending';
