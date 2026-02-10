-- Remove role-permission grants for calendar and resources
DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions WHERE resource IN ('calendars', 'resources')
);

-- Remove calendar and resource permissions
DELETE FROM permissions WHERE resource IN ('calendars', 'resources');

-- Remove calendar notification event types
DELETE FROM event_types WHERE module_id = 'calendar';

DROP TABLE IF EXISTS public_holidays;
