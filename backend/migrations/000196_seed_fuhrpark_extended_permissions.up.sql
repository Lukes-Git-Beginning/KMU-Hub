DO $$
DECLARE v_admin_role_id UUID;
BEGIN
    SELECT id INTO v_admin_role_id FROM roles WHERE name = 'admin' LIMIT 1;
    IF v_admin_role_id IS NOT NULL THEN
        INSERT INTO permissions (name, resource, action) VALUES
            ('fuhrpark:fuel:read',      'fuhrpark:fuel',     'read'),
            ('fuhrpark:fuel:write',     'fuhrpark:fuel',     'write'),
            ('fuhrpark:trip:read',      'fuhrpark:trip',     'read'),
            ('fuhrpark:trip:write',     'fuhrpark:trip',     'write'),
            ('fuhrpark:document:read',  'fuhrpark:document', 'read'),
            ('fuhrpark:document:write', 'fuhrpark:document', 'write'),
            ('fuhrpark:gps:read',       'fuhrpark:gps',      'read'),
            ('fuhrpark:gps:write',      'fuhrpark:gps',      'write')
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
