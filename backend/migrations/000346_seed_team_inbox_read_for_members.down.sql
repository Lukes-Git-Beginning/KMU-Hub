-- Remove kommunikation:team_inbox:read and every grant of it added since the up migration.
DELETE FROM role_permissions
WHERE permission_id IN (SELECT id FROM permissions WHERE name = 'kommunikation:team_inbox:read');

DELETE FROM permissions WHERE name = 'kommunikation:team_inbox:read';
