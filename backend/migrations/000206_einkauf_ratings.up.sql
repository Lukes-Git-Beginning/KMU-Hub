-- 000206 Einkauf Supplier Ratings

BEGIN;
SET LOCAL row_security = off;

CREATE TABLE IF NOT EXISTS supplier_ratings (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID        NOT NULL,
    supplier_id UUID        NOT NULL REFERENCES suppliers(id) ON DELETE CASCADE,
    category    TEXT        NOT NULL CHECK (category IN ('quality', 'delivery', 'price')),
    rating      INT         NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment     TEXT        NOT NULL DEFAULT '',
    rated_by    UUID,
    rated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_supplier_ratings_tenant ON supplier_ratings (tenant_id, supplier_id);

CALL enable_tenant_rls('supplier_ratings');

COMMIT;
