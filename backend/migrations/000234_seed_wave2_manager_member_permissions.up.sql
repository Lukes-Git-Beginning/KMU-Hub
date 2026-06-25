-- Grant manager + member roles their permissions for the five Welle-2 modules
-- (berichte, helpdesk, wiki, formulare, vertraege).
--
-- These modules were seeded admin-only (migrations 000080/000090/000129), so
-- until now every manager and member hit 403 on all of these routes. The
-- permissions already exist; only the role_permissions rows are missing.
--
-- Product decision (2026-06-26):
--   manager -> full operational access (read+write) on all five modules.
--   member  -> read-only across all five, plus self-service form submission
--              (formulare:submissions:write). helpdesk stays read-only for member
--              because its permission is flat -- helpdesk:write would also expose
--              SLA/queue/canned-response config.

-- Manager: every permission whose resource belongs to one of the five modules.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'manager'
  AND p.resource IN (
    'berichte:reports',
    'helpdesk',
    'wiki:articles', 'wiki:categories',
    'formulare:schemas', 'formulare:submissions', 'formulare:webhooks',
    'vertraege:contract', 'vertraege:party', 'vertraege:reminder'
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Member: read-only across all five modules, plus self-service form submission.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'member'
  AND p.name IN (
    'berichte:reports:read',
    'helpdesk:read',
    'wiki:articles:read', 'wiki:categories:read',
    'formulare:schemas:read', 'formulare:submissions:read', 'formulare:webhooks:read',
    'formulare:submissions:write',
    'vertraege:contract:read', 'vertraege:party:read', 'vertraege:reminder:read'
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;
