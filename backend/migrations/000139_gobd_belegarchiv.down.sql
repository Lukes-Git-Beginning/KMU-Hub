-- Migration 000139 — GoBD Belegarchiv (down)

BEGIN;

-- Remove RBAC seeds
DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions
    WHERE name IN ('gobd-archive:read', 'gobd-archive:write', 'gobd-archive:admin')
);

DELETE FROM permissions
WHERE name IN ('gobd-archive:read', 'gobd-archive:write', 'gobd-archive:admin');

-- Drop tables (CASCADE handles indexes + constraints)
DROP TABLE IF EXISTS gobd_document_events CASCADE;
DROP TABLE IF EXISTS gobd_documents CASCADE;

-- Drop ENUMs
DROP TYPE IF EXISTS gobd_event_type;
DROP TYPE IF EXISTS gobd_document_type;

COMMIT;
