-- Leads as a contact lifecycle stage, not a separate table.
--
-- A lead is the same person as the contact they become; a second table would
-- create two truths for one person and break duplicate detection. Existing
-- rows are contacts that already made it past qualification, so they default
-- to 'customer' -- no NULL hole, and today's behaviour is unchanged.
--
-- RLS: contacts already carries tenant_id NOT NULL and a tenant policy
-- (migration 000070 / the RLS wave); adding columns inherits both.

ALTER TABLE contacts
    ADD COLUMN lifecycle_stage VARCHAR(20) NOT NULL DEFAULT 'customer',
    ADD COLUMN lead_source VARCHAR(20),
    ADD COLUMN lead_score SMALLINT,
    ADD COLUMN lead_temperature VARCHAR(10),
    ADD COLUMN lead_status VARCHAR(20),
    -- Free-text company for leads that have no companies row yet (CSV/dialer
    -- intake names an employer long before anyone creates the company record).
    ADD COLUMN lead_company VARCHAR(255);

ALTER TABLE contacts
    ADD CONSTRAINT chk_contacts_lifecycle_stage
        CHECK (lifecycle_stage IN ('lead', 'qualified', 'customer')),
    ADD CONSTRAINT chk_contacts_lead_source
        CHECK (lead_source IS NULL OR lead_source IN ('manual', 'csv', 'dialer')),
    ADD CONSTRAINT chk_contacts_lead_score
        CHECK (lead_score IS NULL OR (lead_score >= 0 AND lead_score <= 100)),
    ADD CONSTRAINT chk_contacts_lead_temperature
        CHECK (lead_temperature IS NULL OR lead_temperature IN ('hot', 'warm', 'cold')),
    ADD CONSTRAINT chk_contacts_lead_status
        CHECK (lead_status IS NULL OR lead_status IN ('new', 'contacted', 'qualified', 'disqualified'));

-- Partial: leads are the minority of contacts, and the lead inbox only ever
-- asks for the non-customer rows of one tenant.
CREATE INDEX idx_contacts_lifecycle_stage
    ON contacts (tenant_id, lifecycle_stage, created_at DESC)
    WHERE lifecycle_stage <> 'customer';

COMMENT ON COLUMN contacts.lifecycle_stage IS 'lead -> qualified -> customer; a lead is a contact, never a separate row';
COMMENT ON COLUMN contacts.lead_temperature IS 'manual override only; NULL means derive from lead_score';
