-- Backfill for fix-einkauf-po-total (backlog unit, Lauf 4).
--
-- e91cdf2a (2026-07-26) fixed RecomputePOTotal to run on every line mutation
-- going forward, but that only touches purchase_orders whose lines are
-- edited again after the fix landed. Any PO whose lines were added earlier
-- and never touched since is still stuck at total_amount = 0. This recomputes
-- every existing purchase order once, using the exact same net-sum formula
-- (SQL SUM, not float64) as RecomputePOTotal in postgres_repository.go.
--
-- Idempotent: re-running yields the same totals, since it always reads the
-- current po_lines rather than accumulating onto the existing total_amount.

BEGIN;

UPDATE purchase_orders po
SET total_amount = COALESCE((
        SELECT SUM(l.quantity * l.unit_price)
        FROM po_lines l
        WHERE l.po_id = po.id AND l.tenant_id = po.tenant_id
    ), 0),
    updated_at = NOW()
WHERE po.total_amount <> COALESCE((
        SELECT SUM(l.quantity * l.unit_price)
        FROM po_lines l
        WHERE l.po_id = po.id AND l.tenant_id = po.tenant_id
    ), 0);

COMMIT;
