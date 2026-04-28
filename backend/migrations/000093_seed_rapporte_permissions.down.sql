-- Sprint 2 W2.A Rapporte — rollback ACL seed

DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions
    WHERE resource IN ('rapporte:report', 'rapporte:line', 'rapporte:attachment')
);

DELETE FROM permissions
WHERE resource IN ('rapporte:report', 'rapporte:line', 'rapporte:attachment');
