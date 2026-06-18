CREATE TABLE IF NOT EXISTS vehicle_documents (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    vehicle_id  UUID NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
    doc_type    TEXT NOT NULL CHECK (doc_type IN ('registration','insurance','tuev','other')),
    name        TEXT NOT NULL,
    object_key  TEXT NOT NULL,
    upload_date DATE NOT NULL DEFAULT CURRENT_DATE,
    expiry_date DATE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_vehicle_documents_vehicle_id ON vehicle_documents(vehicle_id);
CREATE INDEX IF NOT EXISTS idx_vehicle_documents_tenant_id ON vehicle_documents(tenant_id);
CALL enable_tenant_rls('vehicle_documents');
