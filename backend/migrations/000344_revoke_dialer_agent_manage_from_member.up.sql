-- dialer:agent:manage back off the member preset.
--
-- Migration 000068 gave it to member as "agent self-manage". The key is the supervisor right: it
-- names another agent on PUT/GET /dialer/agent-status, opens the campaign roster and the agent
-- dashboard for any agent. An agent's own status runs on dialer:calls:write since e7afd4d7 and the
-- handlers pin it to the caller (agentStatusTarget), so the grant only ever widened reach.
-- manager and admin keep it.
--
-- Takes effect after the next login: permissions are baked into the JWT.
DELETE FROM role_permissions rp
USING roles r, permissions p
WHERE rp.role_id = r.id
  AND rp.permission_id = p.id
  AND r.tenant_id IS NULL
  AND r.name = 'member'
  AND p.name = 'dialer:agent:manage';
