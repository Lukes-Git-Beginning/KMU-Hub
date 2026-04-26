-- Rollback Sprint 2 S1.5 Vertraege — ACL seed

DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions
    WHERE resource IN ('vertraege:contract', 'vertraege:party', 'vertraege:reminder')
);

DELETE FROM permissions
WHERE resource IN ('vertraege:contract', 'vertraege:party', 'vertraege:reminder');
