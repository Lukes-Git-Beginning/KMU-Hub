-- member works the tickets assigned to them (decision 2026-10-25,
-- feat-helpdesk-assigned-tickets-edit-for-members): helpdesk:ticket:edit at scope own. The
-- service narrows edit, close, reopen and CSAT to tickets whose assignee is the caller; assigning,
-- merging and internal notes stay with agents beyond own scope. The key exists since 000256;
-- admin, it_admin and manager hold it at scope all.
INSERT INTO role_permissions (role_id, permission_id, scope)
SELECT r.id, p.id, 'own'
FROM roles r, permissions p
WHERE r.name = 'member' AND r.tenant_id IS NULL
  AND p.name IN ('helpdesk:ticket:edit')
ON CONFLICT (role_id, permission_id) DO NOTHING;
