-- Revert Sprint 1 S1.2 Berichte — ACL seed

DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions WHERE resource = 'berichte:reports'
);

DELETE FROM permissions WHERE resource = 'berichte:reports';
