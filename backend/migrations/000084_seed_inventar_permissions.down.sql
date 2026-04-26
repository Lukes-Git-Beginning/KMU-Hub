-- Sprint 2 W1.A Inventar — revert ACL seed

DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions
    WHERE resource IN ('inventar:item', 'inventar:movement', 'inventar:warning')
);

DELETE FROM permissions
WHERE resource IN ('inventar:item', 'inventar:movement', 'inventar:warning');
