-- Migration 000236: helpdesk_ticket_fields
-- Adds description, category and a per-tenant sequential ticket_number to tickets,
-- plus a counter table for the numbering. Backfills existing tickets and seeds the
-- counters so new tickets continue the per-tenant sequence.

BEGIN;

ALTER TABLE tickets
    ADD COLUMN description   TEXT,
    ADD COLUMN category      TEXT,
    ADD COLUMN ticket_number INTEGER NOT NULL DEFAULT 0;

-- Per-tenant counter: next_number is the value to assign to the NEXT ticket.
CREATE TABLE helpdesk_ticket_counters (
    tenant_id   UUID    PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    next_number INTEGER NOT NULL DEFAULT 1
);

-- RLS: standard tenant isolation (matches pattern in 000232).
ALTER TABLE helpdesk_ticket_counters ENABLE ROW LEVEL SECURITY;
ALTER TABLE helpdesk_ticket_counters FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON helpdesk_ticket_counters
    USING      (tenant_id = current_tenant_id() OR is_system_context())
    WITH CHECK (tenant_id = current_tenant_id() OR is_system_context());

-- Backfill: number existing tickets per tenant by creation order.
UPDATE tickets t SET ticket_number = s.rn
FROM (
    SELECT id, ROW_NUMBER() OVER (PARTITION BY tenant_id ORDER BY created_at, id) AS rn
    FROM tickets
) s
WHERE t.id = s.id;

-- Seed counters to max(ticket_number)+1 per tenant so the sequence continues.
INSERT INTO helpdesk_ticket_counters (tenant_id, next_number)
SELECT tenant_id, MAX(ticket_number) + 1
FROM tickets
GROUP BY tenant_id
ON CONFLICT (tenant_id) DO UPDATE SET next_number = EXCLUDED.next_number;

COMMIT;
