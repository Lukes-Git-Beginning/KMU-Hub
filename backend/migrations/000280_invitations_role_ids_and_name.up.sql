-- Migration 000280: make an invitation carry role IDS and a name.
--
-- Two gaps, both surfaced by the account admin surface (A-1 Benutzerverwaltung,
-- desktop useAdminUsers.ts) whose invite form sends `{email, firstName?,
-- lastName?, roles[]}`:
--
--   1. invitations.role is a single VARCHAR holding a legacy PRESET NAME
--      ("admin"/"manager"/"member"). The frontend sends an ARRAY OF ROLE IDS,
--      because that is what the roster returns (AdminUser.roles). Names cannot
--      express a custom role at all: roles is unique per
--      (COALESCE(tenant_id, zero), name), so two tenants may each own a custom
--      role called "Buchhaltung" and a name says nothing about which.
--   2. invitations never stored a name, so an invited row in the roster showed
--      an empty first/last name even when the admin had typed one.
--
-- role_ids becomes the AUTHORITATIVE field for what an accepted invitation
-- grants. role stays and stays NOT NULL, but only as the legacy display name
-- (GET /api/v1/invitations still returns it, and the ProvisionTenant path still
-- writes "admin" there) — nothing decides anything on it any more.
--
-- This also closes a real defect in the accept path, which resolved the role
-- with `WHERE r.name = $1` under sysctx (RLS off): the moment any tenant owned
-- a custom role named "admin", accepting an invitation assigned EVERY matching
-- row, foreign tenants' roles included. Resolving by id cannot do that.

BEGIN;

ALTER TABLE invitations ADD COLUMN IF NOT EXISTS role_ids UUID[] NOT NULL DEFAULT '{}';
ALTER TABLE invitations ADD COLUMN IF NOT EXISTS first_name VARCHAR(100) NOT NULL DEFAULT '';
ALTER TABLE invitations ADD COLUMN IF NOT EXISTS last_name VARCHAR(100) NOT NULL DEFAULT '';

-- Backfill from the name, restricted to the system presets (tenant_id IS NULL)
-- because that is the only thing role could ever legitimately hold. A row whose
-- name resolves to nothing keeps the empty array: it was already unacceptable
-- (the accept path raised role_not_found on zero matches), so this preserves
-- the behaviour instead of inventing a role for it.
UPDATE invitations i
SET role_ids = ARRAY(
        SELECT r.id FROM roles r
        WHERE r.tenant_id IS NULL AND r.name = i.role
    )
WHERE cardinality(i.role_ids) = 0;

COMMENT ON COLUMN invitations.role_ids IS
    'Roles the accepted account receives. Authoritative; role is the legacy display name only.';
COMMENT ON COLUMN invitations.role IS
    'Legacy preset name, display only since migration 000280. Assignment resolves role_ids.';

COMMIT;
