-- calendars:delete, messages:delete, files:delete for member and manager at scope 'own'.
--
-- Migration 129 gave the three delete keys to admin only, so every member and manager answered 403
-- on deleting their own event or calendar, chat message or chat file, although the UI offers the
-- action on exactly those items. The scope is enforced by the services, not the gateway:
-- calendar.Service.Delete and event.Service.DeleteEvent require owner/creator, message.Service.Delete
-- and chat file.Service.Delete require the author/uploader or a channel moderator, and the event
-- category delete filters on user_id. 'own' records that intent in the grant.
--
-- Takes effect after the next login: permissions are baked into the JWT.
INSERT INTO role_permissions (role_id, permission_id, scope)
SELECT r.id, p.id, 'own'
FROM roles r, permissions p
WHERE r.tenant_id IS NULL
  AND r.name IN ('manager', 'member')
  AND p.name IN ('calendars:delete', 'messages:delete', 'files:delete')
ON CONFLICT (role_id, permission_id) DO NOTHING;
