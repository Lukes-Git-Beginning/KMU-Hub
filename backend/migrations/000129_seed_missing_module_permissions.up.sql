-- Seed permissions referenced by gateway RequirePermission() guards that were
-- never inserted by any migration. Without these rows the guarded routes
-- returned 403 "insufficient permissions" for EVERY user, including admins
-- (documents, email, finance, formulare, helpdesk, hr, inbox, wiki, ...).
--
-- Grants are admin-only for now: that strictly widens access compared to the
-- status quo (nobody had these permissions) without making a product decision
-- about manager/member visibility into sensitive modules (HR, finance).
-- Per-module manager/member mappings are a deliberate follow-up.

INSERT INTO permissions (name, resource, action) VALUES
    ('automations:read',            'automations',            'read'),
    ('automations:write',           'automations',            'write'),
    ('calendars:delete',            'calendars',              'delete'),
    ('documents:read',              'documents',              'read'),
    ('documents:write',             'documents',              'write'),
    ('documents:delete',            'documents',              'delete'),
    ('email:read',                  'email',                  'read'),
    ('email:write',                 'email',                  'write'),
    ('email:delete',                'email',                  'delete'),
    ('finance:read',                'finance',                'read'),
    ('finance:write',               'finance',                'write'),
    ('finance:delete',              'finance',                'delete'),
    ('finance:admin',               'finance',                'admin'),
    ('formulare:schemas:read',      'formulare:schemas',      'read'),
    ('formulare:schemas:write',     'formulare:schemas',      'write'),
    ('formulare:submissions:read',  'formulare:submissions',  'read'),
    ('formulare:submissions:write', 'formulare:submissions',  'write'),
    ('formulare:webhooks:read',     'formulare:webhooks',     'read'),
    ('formulare:webhooks:write',    'formulare:webhooks',     'write'),
    ('helpdesk:read',               'helpdesk',               'read'),
    ('helpdesk:write',              'helpdesk',               'write'),
    ('hr:read',                     'hr',                     'read'),
    ('hr:write',                    'hr',                     'write'),
    ('hr:admin',                    'hr',                     'admin'),
    ('inbox:read',                  'inbox',                  'read'),
    ('inbox:write',                 'inbox',                  'write'),
    ('recording:write',             'recording',              'write'),
    ('recordings:admin',            'recordings',             'admin'),
    ('resources:delete',            'resources',              'delete'),
    ('search:read',                 'search',                 'read'),
    ('settings:write',              'settings',               'write'),
    ('wiki:articles:read',          'wiki:articles',          'read'),
    ('wiki:articles:write',         'wiki:articles',          'write'),
    ('wiki:categories:read',        'wiki:categories',        'read'),
    ('wiki:categories:write',       'wiki:categories',        'write')
ON CONFLICT (name) DO NOTHING;

-- Admin: all permissions seeded above
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin'
  AND p.name IN (
    'automations:read', 'automations:write',
    'calendars:delete',
    'documents:read', 'documents:write', 'documents:delete',
    'email:read', 'email:write', 'email:delete',
    'finance:read', 'finance:write', 'finance:delete', 'finance:admin',
    'formulare:schemas:read', 'formulare:schemas:write',
    'formulare:submissions:read', 'formulare:submissions:write',
    'formulare:webhooks:read', 'formulare:webhooks:write',
    'helpdesk:read', 'helpdesk:write',
    'hr:read', 'hr:write', 'hr:admin',
    'inbox:read', 'inbox:write',
    'recording:write', 'recordings:admin',
    'resources:delete',
    'search:read',
    'settings:write',
    'wiki:articles:read', 'wiki:articles:write',
    'wiki:categories:read', 'wiki:categories:write'
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;
