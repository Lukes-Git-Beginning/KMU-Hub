-- ============================================================================
-- Link helpdesk tickets to CRM contacts/companies (backend-gaps.md §helpdesk
-- "contact_id/org_id ins Ticket-Modell"). Both nullable: not every ticket has
-- a CRM counterpart. Tenant ownership is verified at the service layer
-- (ContactExists/CompanyExists), the FK only guards referential integrity.
-- ============================================================================

ALTER TABLE tickets
    ADD COLUMN contact_id UUID NULL REFERENCES contacts(id) ON DELETE SET NULL,
    ADD COLUMN org_id     UUID NULL REFERENCES companies(id) ON DELETE SET NULL;

CREATE INDEX idx_tickets_contact_id ON tickets (tenant_id, contact_id) WHERE contact_id IS NOT NULL;
CREATE INDEX idx_tickets_org_id ON tickets (tenant_id, org_id) WHERE org_id IS NOT NULL;
