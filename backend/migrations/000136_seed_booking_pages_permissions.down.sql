-- Rollback 000136 — remove booking-pages RBAC seed
DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions
    WHERE name IN ('booking-pages:read', 'booking-pages:write', 'booking-pages:delete')
);

DELETE FROM permissions
WHERE name IN ('booking-pages:read', 'booking-pages:write', 'booking-pages:delete');
