-- Remove the member and manager grants of search:read added by the up migration. The key itself
-- and the admin grant predate it (migration 129) and stay.
DELETE FROM role_permissions
WHERE permission_id IN (SELECT id FROM permissions WHERE name = 'search:read')
  AND role_id IN (SELECT id FROM roles WHERE tenant_id IS NULL AND name IN ('manager', 'member'));
