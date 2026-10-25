-- The ten system leave types for every tenant.
--
-- Migration 000046 seeded them only as templates under the zero UUID and said "the application
-- layer copies these to each tenant on first access". No code ever did, and ListByTenant reads
-- tenant_id = $1 only: every real tenant had no leave type, so no one could file leave or a sick
-- note ("krank" is looked up by key), and there is no route to create one.
--
-- lean: copies for the tenants that exist now; a tenant created later needs the same copy at
-- provisioning (BACKLOG-NEXT seed-leave-types-on-tenant-provisioning), with one server per
-- customer the default tenant is the only one today.
SET LOCAL row_security = off;

INSERT INTO hr_leave_types (tenant_id, key, name, color, deducts_from_balance, requires_approval,
                            requires_au_document, is_system, sort_order)
SELECT t.id, s.key, s.name, s.color, s.deducts_from_balance, s.requires_approval,
       s.requires_au_document, s.is_system, s.sort_order
FROM tenants t
CROSS JOIN hr_leave_types s
WHERE s.tenant_id = '00000000-0000-0000-0000-000000000000'
  AND t.id <> '00000000-0000-0000-0000-000000000000'
ON CONFLICT (tenant_id, key) DO NOTHING;
