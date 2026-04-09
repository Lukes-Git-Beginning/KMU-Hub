-- Dialer module permissions
INSERT INTO permissions (name, resource, action) VALUES
    ('dialer:campaigns:read',  'dialer:campaigns', 'read'),
    ('dialer:campaigns:write', 'dialer:campaigns', 'write'),
    ('dialer:calls:read',      'dialer:calls',     'read'),
    ('dialer:calls:write',     'dialer:calls',     'write'),
    ('dialer:outcomes:manage', 'dialer:outcomes',   'manage'),
    ('dialer:agent:manage',    'dialer:agent',      'manage')
ON CONFLICT (name) DO NOTHING;

-- Admin: all dialer permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin'
  AND p.resource LIKE 'dialer:%'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Manager: campaign read/write + call read/write
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'manager'
  AND p.name IN (
    'dialer:campaigns:read', 'dialer:campaigns:write',
    'dialer:calls:read',     'dialer:calls:write'
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Member: campaign read + call write + agent self-manage
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'member'
  AND p.name IN (
    'dialer:campaigns:read',
    'dialer:calls:write',
    'dialer:agent:manage'
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;
