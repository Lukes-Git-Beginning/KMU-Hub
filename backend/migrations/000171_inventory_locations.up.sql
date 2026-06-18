-- 000171 Lagerorte (inventory_locations) + location_id FK auf inventory_items

BEGIN;
SET LOCAL row_security = off;

CREATE TABLE IF NOT EXISTS inventory_locations (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID         NOT NULL,
    name        TEXT         NOT NULL,
    address     TEXT         NOT NULL DEFAULT '',
    type        TEXT         NOT NULL DEFAULT 'warehouse' CHECK (type IN ('warehouse','store','vehicle')),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ  NULL,
    UNIQUE (tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_inventory_locations_tenant
    ON inventory_locations (tenant_id)
    WHERE deleted_at IS NULL;

ALTER TABLE inventory_items
    ADD COLUMN IF NOT EXISTS location_id UUID NULL REFERENCES inventory_locations(id) ON DELETE SET NULL;

-- RLS aktivieren
CALL enable_tenant_rls('inventory_locations');

COMMIT;
