-- 000307 Kommissionierung: Picklisten-Kopf und -Positionen
-- Form uebernommen von inventur_sessions/inventur_counts (000186): Kopf,
-- Positionen, Statusfortschritt, buchender Abschluss.

BEGIN;
SET LOCAL row_security = off;

CREATE TABLE IF NOT EXISTS picking_lists (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID         NOT NULL,
    reference   TEXT         NOT NULL,
    status      TEXT         NOT NULL DEFAULT 'open' CHECK (status IN ('open','picking','completed')),
    assigned_to UUID         NULL,
    created_by  UUID         NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_picking_lists_tenant
    ON picking_lists (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_picking_lists_status
    ON picking_lists (tenant_id, status)
    WHERE status IN ('open','picking');

CREATE TABLE IF NOT EXISTS picking_list_items (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID        NOT NULL,
    picking_list_id    UUID        NOT NULL REFERENCES picking_lists(id) ON DELETE CASCADE,
    item_id            UUID        NOT NULL REFERENCES inventory_items(id) ON DELETE CASCADE,
    quantity_requested BIGINT      NOT NULL CHECK (quantity_requested > 0),
    quantity_picked    BIGINT      NOT NULL DEFAULT 0 CHECK (quantity_picked >= 0),
    location           TEXT        NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (picking_list_id, item_id)
);

CREATE INDEX IF NOT EXISTS idx_picking_list_items_list
    ON picking_list_items (picking_list_id);
CREATE INDEX IF NOT EXISTS idx_picking_list_items_tenant
    ON picking_list_items (tenant_id);

-- RLS aktivieren
CALL enable_tenant_rls('picking_lists');
CALL enable_tenant_rls('picking_list_items');

COMMIT;
