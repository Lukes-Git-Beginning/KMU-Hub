-- Revoke the manager + member grants added in the up migration.
-- The permissions themselves are NOT removed (they predate this migration and
-- remain granted to admin); only the manager/member role_permissions rows go.

-- Manager: remove grants for the five module groups.
DELETE FROM role_permissions
WHERE role_id = (SELECT id FROM roles WHERE name = 'manager')
  AND permission_id IN (
    SELECT id FROM permissions WHERE resource IN (
      'booking-pages',
      'schichten:shift', 'schichten:template', 'schichten:assignment', 'schichten:swap',
      'hr:time_category', 'hr:time_template', 'hr:time_project',
      'inventar:item', 'inventar:movement', 'inventar:warning', 'inventar:location', 'inventar:inventur',
      'einkauf:po', 'einkauf:line', 'einkauf:supplier', 'einkauf:catalog', 'einkauf:contract', 'einkauf:rating'
    )
  );

-- Member: remove the self-service subset.
DELETE FROM role_permissions
WHERE role_id = (SELECT id FROM roles WHERE name = 'member')
  AND permission_id IN (
    SELECT id FROM permissions WHERE name IN (
      'schichten:swap:read', 'schichten:swap:create',
      'booking-pages:read',
      'inventar:item:read', 'inventar:movement:read', 'inventar:warning:read',
      'inventar:location:read', 'inventar:inventur:read'
    )
  );
