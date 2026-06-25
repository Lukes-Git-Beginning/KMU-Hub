-- Revoke the manager + member grants added in the up migration.
-- The permissions themselves are NOT removed (they predate this migration and
-- remain granted to admin); only the manager/member role_permissions rows go.

-- Manager: remove grants for the five Welle-2 module groups.
DELETE FROM role_permissions
WHERE role_id = (SELECT id FROM roles WHERE name = 'manager')
  AND permission_id IN (
    SELECT id FROM permissions WHERE resource IN (
      'berichte:reports',
      'helpdesk',
      'wiki:articles', 'wiki:categories',
      'formulare:schemas', 'formulare:submissions', 'formulare:webhooks',
      'vertraege:contract', 'vertraege:party', 'vertraege:reminder'
    )
  );

-- Member: remove the read-only + self-service subset.
DELETE FROM role_permissions
WHERE role_id = (SELECT id FROM roles WHERE name = 'member')
  AND permission_id IN (
    SELECT id FROM permissions WHERE name IN (
      'berichte:reports:read',
      'helpdesk:read',
      'wiki:articles:read', 'wiki:categories:read',
      'formulare:schemas:read', 'formulare:submissions:read', 'formulare:webhooks:read',
      'formulare:submissions:write',
      'vertraege:contract:read', 'vertraege:party:read', 'vertraege:reminder:read'
    )
  );
