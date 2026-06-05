-- Remove exactly the permissions seeded by the up migration (explicit name
-- list — resources like 'calendars' had pre-existing read/write rows that
-- must survive a rollback).
DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions WHERE name IN (
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
);

DELETE FROM permissions WHERE name IN (
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
);
