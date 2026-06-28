-- Migration 000240 — attachments for inventory items.
-- Mirrors the fuhrpark vehicle_documents pattern (migration 000194): the file is
-- uploaded browser-direct via the presigned-upload service (scope=inventar) and
-- only its object_key is registered here. tenant_id + RLS keep it tenant-scoped.
CREATE TABLE IF NOT EXISTS inventory_item_attachments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    item_id     UUID NOT NULL REFERENCES inventory_items(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    object_key  TEXT NOT NULL,
    file_type   TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_inventory_item_attachments_item_id ON inventory_item_attachments(item_id);
CREATE INDEX IF NOT EXISTS idx_inventory_item_attachments_tenant_id ON inventory_item_attachments(tenant_id);
CALL enable_tenant_rls('inventory_item_attachments');
