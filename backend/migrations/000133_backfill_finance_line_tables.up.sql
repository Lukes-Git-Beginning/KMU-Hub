-- ADR-0007 — Backfill the relational line-item tables from the existing
-- line_items JSONB blobs, and backfill the new locked_at/locked_by columns
-- from the snapshot_data JSONB lock marker.
-- Sprint 4, 2026-06-08.
--
-- Idempotent:
--   * Line backfill skips any parent that already has relational lines
--     (NOT EXISTS guard) — re-running yields identical row counts and never
--     duplicates or deletes app-created rows.
--   * Lock backfill only fills rows where locked_at IS NULL.
--
-- Position is taken from the JSON array ordinality (1-based, always satisfies
-- the position >= 1 CHECK). The old client-side string id is intentionally
-- discarded — it was never a real PK (ADR-0007 core weakness); fresh
-- gen_random_uuid() PKs are assigned via column DEFAULT.
--
-- Runs as the migration role (superuser, BYPASSRLS), so the FORCE-RLS
-- tenant_isolation policy on the new tables does not block these inserts.

BEGIN;

SET LOCAL row_security = off;

-- ============================================================================
-- finance_invoice_lines
-- ============================================================================
INSERT INTO finance_invoice_lines (
    invoice_id, tenant_id, position, description,
    quantity, unit_price, tax_rate, line_total, created_at, updated_at
)
SELECT
    fi.id,
    fi.tenant_id,
    t.ord::int,
    COALESCE(t.elem->>'description', ''),
    (t.elem->>'quantity')::numeric,
    (t.elem->>'unit_price')::numeric,
    COALESCE(NULLIF(t.elem->>'tax_rate', '')::numeric, 0),
    COALESCE(
        NULLIF(t.elem->>'line_total', '')::numeric,
        (t.elem->>'quantity')::numeric * (t.elem->>'unit_price')::numeric
    ),
    fi.created_at,
    fi.updated_at
FROM finance_invoices fi,
     jsonb_array_elements(fi.line_items) WITH ORDINALITY AS t(elem, ord)
WHERE fi.line_items IS NOT NULL
  AND jsonb_typeof(fi.line_items) = 'array'
  AND fi.line_items <> '[]'::jsonb
  AND NOT EXISTS (SELECT 1 FROM finance_invoice_lines l WHERE l.invoice_id = fi.id);

-- ============================================================================
-- finance_quote_lines
-- ============================================================================
INSERT INTO finance_quote_lines (
    quote_id, tenant_id, position, description,
    quantity, unit_price, tax_rate, line_total, created_at, updated_at
)
SELECT
    fq.id,
    fq.tenant_id,
    t.ord::int,
    COALESCE(t.elem->>'description', ''),
    (t.elem->>'quantity')::numeric,
    (t.elem->>'unit_price')::numeric,
    COALESCE(NULLIF(t.elem->>'tax_rate', '')::numeric, 0),
    COALESCE(
        NULLIF(t.elem->>'line_total', '')::numeric,
        (t.elem->>'quantity')::numeric * (t.elem->>'unit_price')::numeric
    ),
    fq.created_at,
    fq.updated_at
FROM finance_quotes fq,
     jsonb_array_elements(fq.line_items) WITH ORDINALITY AS t(elem, ord)
WHERE fq.line_items IS NOT NULL
  AND jsonb_typeof(fq.line_items) = 'array'
  AND fq.line_items <> '[]'::jsonb
  AND NOT EXISTS (SELECT 1 FROM finance_quote_lines l WHERE l.quote_id = fq.id);

-- ============================================================================
-- finance_credit_note_lines
-- ============================================================================
INSERT INTO finance_credit_note_lines (
    credit_note_id, tenant_id, position, description,
    quantity, unit_price, tax_rate, line_total, created_at, updated_at
)
SELECT
    fcn.id,
    fcn.tenant_id,
    t.ord::int,
    COALESCE(t.elem->>'description', ''),
    (t.elem->>'quantity')::numeric,
    (t.elem->>'unit_price')::numeric,
    COALESCE(NULLIF(t.elem->>'tax_rate', '')::numeric, 0),
    COALESCE(
        NULLIF(t.elem->>'line_total', '')::numeric,
        (t.elem->>'quantity')::numeric * (t.elem->>'unit_price')::numeric
    ),
    fcn.created_at,
    fcn.updated_at
FROM finance_credit_notes fcn,
     jsonb_array_elements(fcn.line_items) WITH ORDINALITY AS t(elem, ord)
WHERE fcn.line_items IS NOT NULL
  AND jsonb_typeof(fcn.line_items) = 'array'
  AND fcn.line_items <> '[]'::jsonb
  AND NOT EXISTS (SELECT 1 FROM finance_credit_note_lines l WHERE l.credit_note_id = fcn.id);

-- ============================================================================
-- Lock backfill: move locked_at/locked_by out of snapshot_data JSONB into
-- the dedicated columns (idempotent — only fills NULL locked_at).
-- ============================================================================
UPDATE finance_invoices
SET locked_at = (snapshot_data->>'locked_at')::timestamptz,
    locked_by = NULLIF(snapshot_data->>'locked_by', '')::uuid
WHERE snapshot_data IS NOT NULL
  AND snapshot_data ? 'locked_at'
  AND locked_at IS NULL;

COMMIT;
