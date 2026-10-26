-- member deletes the tasks they created (decision 2026-10-25,
-- feat-work-own-task-delete-for-members): work:task:delete at scope own. The
-- gateway passes the caller as owner_id and the service deletes only a task
-- whose created_by is the caller (404 otherwise). Deleting projects stays with
-- manager (projects:delete / work:project:delete). The key exists since 000256;
-- admin holds it at scope all, manager at scope team.
INSERT INTO role_permissions (role_id, permission_id, scope)
SELECT r.id, p.id, 'own'
FROM roles r, permissions p
WHERE r.name = 'member' AND r.tenant_id IS NULL
  AND p.name IN ('work:task:delete')
ON CONFLICT (role_id, permission_id) DO NOTHING;
