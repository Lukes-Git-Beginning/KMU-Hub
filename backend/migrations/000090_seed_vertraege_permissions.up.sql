-- Sprint 2 S1.5 Vertraege — ACL seed

INSERT INTO permissions (name, resource, action) VALUES
    ('vertraege:contract:read',  'vertraege:contract', 'read'),
    ('vertraege:contract:write', 'vertraege:contract', 'write'),
    ('vertraege:party:read',     'vertraege:party',    'read'),
    ('vertraege:party:write',    'vertraege:party',    'write'),
    ('vertraege:reminder:read',  'vertraege:reminder', 'read'),
    ('vertraege:reminder:write', 'vertraege:reminder', 'write')
ON CONFLICT (name) DO NOTHING;

-- Grant all vertraege permissions to admin role
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'admin'
  AND p.resource IN ('vertraege:contract', 'vertraege:party', 'vertraege:reminder')
ON CONFLICT (role_id, permission_id) DO NOTHING;
