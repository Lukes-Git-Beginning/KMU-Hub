DO $$
DECLARE v_admin_role_id UUID;
BEGIN
    SELECT id INTO v_admin_role_id FROM roles WHERE name = 'admin' LIMIT 1;
    IF v_admin_role_id IS NOT NULL THEN
        INSERT INTO permissions (id, name, description) VALUES
            (gen_random_uuid(), 'fuhrpark:fuel:read',     'Tankprotokoll lesen'),
            (gen_random_uuid(), 'fuhrpark:fuel:write',    'Tankprotokoll schreiben'),
            (gen_random_uuid(), 'fuhrpark:trip:read',     'Fahrtenbuch lesen'),
            (gen_random_uuid(), 'fuhrpark:trip:write',    'Fahrtenbuch schreiben'),
            (gen_random_uuid(), 'fuhrpark:document:read', 'Fahrzeugdokumente lesen'),
            (gen_random_uuid(), 'fuhrpark:document:write','Fahrzeugdokumente schreiben'),
            (gen_random_uuid(), 'fuhrpark:gps:read',      'GPS-Tracking lesen'),
            (gen_random_uuid(), 'fuhrpark:gps:write',     'GPS-Positionen schreiben')
        ON CONFLICT (name) DO NOTHING;
        INSERT INTO role_permissions (role_id, permission_id)
        SELECT v_admin_role_id, p.id FROM permissions p
        WHERE p.name IN (
            'fuhrpark:fuel:read','fuhrpark:fuel:write',
            'fuhrpark:trip:read','fuhrpark:trip:write',
            'fuhrpark:document:read','fuhrpark:document:write',
            'fuhrpark:gps:read','fuhrpark:gps:write'
        )
        ON CONFLICT DO NOTHING;
    END IF;
END $$;
