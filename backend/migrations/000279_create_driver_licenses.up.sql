-- Fuehrerscheinkontrolle (backlog unit g-fuhrpark-license-check, Lauf 4).
-- Halter-Pflicht in DACH: wer Fahrzeuge stellt, muss die Fahrerlaubnis seiner
-- Fahrer regelmaessig pruefen und das dokumentieren. Eine Zeile pro
-- durchgefuehrter Kontrolle (Historie), nicht ein Status-Feld pro Fahrer --
-- die letzte Zeile je driver_id ist der aktuelle Stand.

CREATE TABLE IF NOT EXISTS driver_licenses (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID NOT NULL,
    driver_id            UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    license_classes      TEXT[] NOT NULL DEFAULT '{}',
    expiry_date          DATE,
    checked_at           DATE NOT NULL DEFAULT CURRENT_DATE,
    next_check_due_date  DATE NOT NULL,
    notes                TEXT,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_driver_licenses_driver_id ON driver_licenses(driver_id);
CREATE INDEX IF NOT EXISTS idx_driver_licenses_tenant_id ON driver_licenses(tenant_id);

CALL enable_tenant_rls('driver_licenses');

-- Permissions: same admin-only precedent as fuhrpark:document (000196) --
-- vehicle/trip/fuel compliance records are broader roles, but attaching a
-- legal check to a specific employee is kept to admin until a real need for
-- manager access shows up.
DO $$
DECLARE v_admin_role_id UUID;
BEGIN
    SELECT id INTO v_admin_role_id FROM roles WHERE name = 'admin' AND tenant_id IS NULL LIMIT 1;
    IF v_admin_role_id IS NOT NULL THEN
        INSERT INTO permissions (name, resource, action) VALUES
            ('fuhrpark:license:read',  'fuhrpark:license', 'read'),
            ('fuhrpark:license:write', 'fuhrpark:license', 'write')
        ON CONFLICT (name) DO NOTHING;
        INSERT INTO role_permissions (role_id, permission_id)
        SELECT v_admin_role_id, p.id FROM permissions p
        WHERE p.name IN ('fuhrpark:license:read', 'fuhrpark:license:write')
        ON CONFLICT DO NOTHING;
    END IF;
END $$;
