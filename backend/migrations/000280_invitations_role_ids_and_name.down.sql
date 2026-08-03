-- Reverse of 000280. Dropping role_ids loses the ability to name a custom role
-- in an invitation; the legacy `role` column still holds the preset name for
-- every row this migration backfilled from, so the pre-000280 accept path
-- (resolve by name) keeps working for those.

BEGIN;

ALTER TABLE invitations DROP COLUMN IF EXISTS role_ids;
ALTER TABLE invitations DROP COLUMN IF EXISTS first_name;
ALTER TABLE invitations DROP COLUMN IF EXISTS last_name;

COMMENT ON COLUMN invitations.role IS NULL;

COMMIT;
