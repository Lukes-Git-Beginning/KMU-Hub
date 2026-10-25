-- Removes the tenant copies of 000345 that no leave request uses yet; a used one stays, the
-- request would otherwise lose its type (FK).
SET LOCAL row_security = off;

DELETE FROM hr_leave_types lt
USING hr_leave_types s
WHERE s.tenant_id = '00000000-0000-0000-0000-000000000000'
  AND lt.tenant_id <> '00000000-0000-0000-0000-000000000000'
  AND lt.key = s.key
  AND lt.is_system
  AND NOT EXISTS (SELECT 1 FROM hr_leave_requests r WHERE r.leave_type_id = lt.id);
