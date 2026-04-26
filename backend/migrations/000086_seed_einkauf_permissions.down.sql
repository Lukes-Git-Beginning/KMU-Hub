-- Sprint 2 W1.A Einkauf — rollback ACL seed

DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions
    WHERE resource IN ('einkauf:supplier', 'einkauf:po', 'einkauf:line')
);

DELETE FROM permissions
WHERE resource IN ('einkauf:supplier', 'einkauf:po', 'einkauf:line');
