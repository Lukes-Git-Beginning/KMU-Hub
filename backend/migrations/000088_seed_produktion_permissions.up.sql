-- Sprint 2 W1.A Produktion — ACL seed

INSERT INTO permissions (name, resource, action) VALUES
    ('produktion:order:read',   'produktion:order',   'read'),
    ('produktion:order:write',  'produktion:order',   'write'),
    ('produktion:booking:read', 'produktion:booking', 'read'),
    ('produktion:booking:write','produktion:booking', 'write'),
    ('produktion:plan:read',    'produktion:plan',    'read'),
    ('produktion:plan:write',   'produktion:plan',    'write')
ON CONFLICT (name) DO NOTHING;

-- Grant all produktion permissions to admin role.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin'
  AND p.resource IN ('produktion:order', 'produktion:booking', 'produktion:plan')
ON CONFLICT (role_id, permission_id) DO NOTHING;
