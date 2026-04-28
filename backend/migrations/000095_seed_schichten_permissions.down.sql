-- Sprint 2 W2.A Schichten — ACL rollback

DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions
    WHERE resource IN ('schichten:shift', 'schichten:assignment', 'schichten:template')
);

DELETE FROM permissions
WHERE resource IN ('schichten:shift', 'schichten:assignment', 'schichten:template');
