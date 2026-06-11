DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions
    WHERE name IN (
        'work_labels:read', 'work_labels:write', 'work_labels:delete',
        'work_custom_fields:read', 'work_custom_fields:write', 'work_custom_fields:delete'
    )
);

DELETE FROM permissions
WHERE name IN (
    'work_labels:read', 'work_labels:write', 'work_labels:delete',
    'work_custom_fields:read', 'work_custom_fields:write', 'work_custom_fields:delete'
);
