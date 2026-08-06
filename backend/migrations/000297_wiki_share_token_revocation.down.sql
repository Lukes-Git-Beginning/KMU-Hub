-- Rolling back drops the revocation record: tokens cut while this was in
-- place come back alive on the way down. Revoke them again -- or delete the
-- rows outright -- before applying this.

BEGIN;

ALTER TABLE wiki_share_tokens
    DROP COLUMN IF EXISTS revoked_at,
    DROP COLUMN IF EXISTS created_by;

COMMIT;
