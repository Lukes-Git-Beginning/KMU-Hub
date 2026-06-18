-- Migration 000164 DOWN

DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions
    WHERE resource IN ('rapporte:measurement', 'rapporte:template')
);

DELETE FROM permissions
WHERE resource IN ('rapporte:measurement', 'rapporte:template');
