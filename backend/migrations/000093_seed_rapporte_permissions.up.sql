-- Sprint 2 W2.A Rapporte — ACL seed

INSERT INTO permissions (name, resource, action) VALUES
    ('rapporte:report:read',      'rapporte:report',     'read'),
    ('rapporte:report:write',     'rapporte:report',     'write'),
    ('rapporte:line:read',        'rapporte:line',       'read'),
    ('rapporte:line:write',       'rapporte:line',       'write'),
    ('rapporte:attachment:read',  'rapporte:attachment', 'read'),
    ('rapporte:attachment:write', 'rapporte:attachment', 'write')
ON CONFLICT (name) DO NOTHING;

-- Grant all rapporte permissions to admin role
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin'
  AND p.resource IN ('rapporte:report', 'rapporte:line', 'rapporte:attachment')
ON CONFLICT (role_id, permission_id) DO NOTHING;
