-- Remove the readonly, it_admin and hr_admin grants of search:read added by the up migration. The
-- key itself and the admin, manager and member grants predate it and stay.
DELETE FROM role_permissions
WHERE permission_id IN (SELECT id FROM permissions WHERE name = 'search:read')
  AND role_id IN (SELECT id FROM roles WHERE tenant_id IS NULL AND name IN ('readonly', 'it_admin', 'hr_admin'));
