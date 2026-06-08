-- ADR-0007 — Finance line items: JSONB → relational tables.
-- Sprint 4, 2026-06-08.
--
-- Adds three relational line-item tables (invoice/quote/credit-note) plus
-- dedicated locked_at/locked_by columns on finance_invoices (replacing the
-- snapshot_data JSONB lock hack, service_gobd.go).
--
-- This migration is ADDITIVE / non-breaking: the existing line_items JSONB
-- columns stay in place. The backfill runs separately in 000133. The JSONB
-- columns are dropped only in a later sprint after the relational cutover is
-- confirmed (clean cutover — no dual-write window, no prod finance data yet).
--
-- tax_rate CHECK is DACH-safe (>= 0 AND <= 100), NOT DE-only (0/7/19), so
-- AT (20/10/13%) and CH (8.1/2.6/3.8%) invoice lines are accepted without a
-- future migration.
--
-- RLS: the new tables carry a denormalized tenant_id (NOT NULL) and are
-- activated via the standard enable_tenant_rls() procedure from 000118,
-- matching the finance tables enabled in 000122.

BEGIN;

SET LOCAL row_security = off;

-- ============================================================================
-- finance_invoice_lines
-- ============================================================================
CREATE TABLE finance_invoice_lines (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id   UUID NOT NULL REFERENCES finance_invoices(id) ON DELETE CASCADE,
    tenant_id    UUID NOT NULL,  -- denormalized for tenant-isolation queries / RLS without JOIN
    position     INT  NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    quantity     NUMERIC(15,4) NOT NULL,
    unit_price   NUMERIC(15,4) NOT NULL,
    tax_rate     NUMERIC(5,2)  NOT NULL DEFAULT 0,
    line_total   NUMERIC(15,4) NOT NULL,
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_invoice_lines_quantity   CHECK (quantity > 0),
    CONSTRAINT chk_invoice_lines_unit_price CHECK (unit_price >= 0),
    CONSTRAINT chk_invoice_lines_tax_rate   CHECK (tax_rate >= 0 AND tax_rate <= 100),
    CONSTRAINT chk_invoice_lines_position   CHECK (position >= 1)
);
CREATE INDEX idx_invoice_lines_invoice ON finance_invoice_lines(invoice_id);
CREATE INDEX idx_invoice_lines_tenant  ON finance_invoice_lines(tenant_id);

-- ============================================================================
-- finance_quote_lines
-- ============================================================================
CREATE TABLE finance_quote_lines (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    quote_id     UUID NOT NULL REFERENCES finance_quotes(id) ON DELETE CASCADE,
    tenant_id    UUID NOT NULL,
    position     INT  NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    quantity     NUMERIC(15,4) NOT NULL,
    unit_price   NUMERIC(15,4) NOT NULL,
    tax_rate     NUMERIC(5,2)  NOT NULL DEFAULT 0,
    line_total   NUMERIC(15,4) NOT NULL,
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_quote_lines_quantity   CHECK (quantity > 0),
    CONSTRAINT chk_quote_lines_unit_price CHECK (unit_price >= 0),
    CONSTRAINT chk_quote_lines_tax_rate   CHECK (tax_rate >= 0 AND tax_rate <= 100),
    CONSTRAINT chk_quote_lines_position   CHECK (position >= 1)
);
CREATE INDEX idx_quote_lines_quote  ON finance_quote_lines(quote_id);
CREATE INDEX idx_quote_lines_tenant ON finance_quote_lines(tenant_id);

-- ============================================================================
-- finance_credit_note_lines
-- ============================================================================
CREATE TABLE finance_credit_note_lines (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    credit_note_id UUID NOT NULL REFERENCES finance_credit_notes(id) ON DELETE CASCADE,
    tenant_id      UUID NOT NULL,
    position       INT  NOT NULL,
    description    TEXT NOT NULL DEFAULT '',
    quantity       NUMERIC(15,4) NOT NULL,
    unit_price     NUMERIC(15,4) NOT NULL,
    tax_rate       NUMERIC(5,2)  NOT NULL DEFAULT 0,
    line_total     NUMERIC(15,4) NOT NULL,
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_credit_note_lines_quantity   CHECK (quantity > 0),
    CONSTRAINT chk_credit_note_lines_unit_price CHECK (unit_price >= 0),
    CONSTRAINT chk_credit_note_lines_tax_rate   CHECK (tax_rate >= 0 AND tax_rate <= 100),
    CONSTRAINT chk_credit_note_lines_position   CHECK (position >= 1)
);
CREATE INDEX idx_credit_note_lines_credit_note ON finance_credit_note_lines(credit_note_id);
CREATE INDEX idx_credit_note_lines_tenant      ON finance_credit_note_lines(tenant_id);

-- ============================================================================
-- GoBD lock columns on finance_invoices (replaces snapshot_data lock hack).
-- ============================================================================
ALTER TABLE finance_invoices
    ADD COLUMN locked_at TIMESTAMPTZ,
    ADD COLUMN locked_by UUID;

-- ============================================================================
-- Row-Level-Security — standard tenant_isolation policy (procedure from 000118).
-- ============================================================================
CALL enable_tenant_rls('finance_invoice_lines');
CALL enable_tenant_rls('finance_quote_lines');
CALL enable_tenant_rls('finance_credit_note_lines');

COMMIT;
