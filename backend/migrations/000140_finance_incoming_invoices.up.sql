-- Migration 000140 — Incoming Invoices (E-Rechnung Eingang)
--
-- Adds the finance_incoming_invoices table for processing inbound e-invoices.
-- Supports ZUGFeRD/Factur-X CII (Cross-Industry Invoice) and XRechnung UBL
-- formats as required by the German E-Rechnungsverordnung / EU Directive 2014/55/EU.
--
-- Design decisions:
-- - line_items and tax_breakdown stored as JSONB (same pattern as outgoing invoices)
-- - original_xml preserved verbatim for audit / re-processing
-- - UNIQUE(tenant_id, supplier_name, invoice_number) for duplicate-import protection
-- - status lifecycle: received → reviewed → booked | rejected (service-enforced)
-- - Permission: reuses existing finance resource (read/write/admin) — no new seeds needed
-- - RLS via standard enable_tenant_rls() procedure from Migration 000118

BEGIN;

SET LOCAL row_security = off;

CREATE TABLE finance_incoming_invoices (
    id                  UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID          NOT NULL,
    supplier_name       TEXT          NOT NULL,
    supplier_address    TEXT          NOT NULL DEFAULT '',
    supplier_vat_id     TEXT          NOT NULL DEFAULT '',
    invoice_number      TEXT          NOT NULL,
    invoice_date        DATE          NOT NULL,
    due_date            DATE,
    currency            CHAR(3)       NOT NULL DEFAULT 'EUR',
    line_items          JSONB         NOT NULL DEFAULT '[]'::jsonb,
    tax_breakdown       JSONB         NOT NULL DEFAULT '[]'::jsonb,
    subtotal            NUMERIC(15,2) NOT NULL DEFAULT 0,
    total_tax           NUMERIC(15,2) NOT NULL DEFAULT 0,
    gross_total         NUMERIC(15,2) NOT NULL DEFAULT 0,
    source_format       TEXT          NOT NULL CHECK (source_format IN ('zugferd_cii', 'xrechnung_ubl', 'xml_only')),
    original_xml        TEXT          NOT NULL,
    original_filename   TEXT          NOT NULL DEFAULT '',
    status              TEXT          NOT NULL DEFAULT 'received' CHECK (status IN ('received', 'reviewed', 'booked', 'rejected')),
    created_by          UUID          NOT NULL,
    created_at          TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_incoming_invoices_tenant
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,

    -- Dedup guard: same supplier + same supplier-assigned number per tenant
    CONSTRAINT uq_incoming_invoice_dedup
        UNIQUE (tenant_id, supplier_name, invoice_number)
);

-- Indexes
CREATE INDEX idx_incoming_invoices_tenant
    ON finance_incoming_invoices(tenant_id);

CREATE INDEX idx_incoming_invoices_tenant_date
    ON finance_incoming_invoices(tenant_id, invoice_date DESC);

CREATE INDEX idx_incoming_invoices_tenant_status
    ON finance_incoming_invoices(tenant_id, status);

-- Row-Level-Security via standard procedure (Migration 000118)
CALL enable_tenant_rls('finance_incoming_invoices');

COMMIT;
