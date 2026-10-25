-- kommunikation:team_inbox:read for member and manager (decision 2026-10-25: team inbox for members,
-- bounded by membership).
--
-- Until now every inbox read route required the coarse inbox:read, which only admin holds (000129).
-- The gateway accepts this key additively on the read routes a team inbox member needs (messages,
-- unread count, single message, thread, team inbox list, member list). Each of them is bounded to the
-- caller in the inbox service: own messages, single messages by owner/assignee/team membership, team
-- inboxes and team message lists by membership. Scope 'own' says exactly that; admin keeps 'all'.
--
-- Takes effect after the next login: permissions are baked into the JWT.
INSERT INTO permissions (name, resource, action)
VALUES ('kommunikation:team_inbox:read', 'kommunikation:team_inbox', 'read')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id, scope)
SELECT r.id, p.id, 'own'
FROM roles r, permissions p
WHERE r.name IN ('member', 'manager') AND r.tenant_id IS NULL
  AND p.name = 'kommunikation:team_inbox:read'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id, scope)
SELECT r.id, p.id, 'all'
FROM roles r, permissions p
WHERE r.name = 'admin' AND r.tenant_id IS NULL
  AND p.name = 'kommunikation:team_inbox:read'
ON CONFLICT (role_id, permission_id) DO NOTHING;
