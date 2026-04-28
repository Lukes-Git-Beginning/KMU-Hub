-- Rollback 000100: Remove rapporte:report:approve permission

DELETE FROM role_permissions
WHERE permission_id = (SELECT id FROM permissions WHERE name = 'rapporte:report:approve');

DELETE FROM permissions WHERE name = 'rapporte:report:approve';
