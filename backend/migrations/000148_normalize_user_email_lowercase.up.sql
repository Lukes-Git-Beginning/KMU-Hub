-- Migration 000148: normalize user + invitation email to lowercase
-- Case-sensitive email allowed duplicate accounts (User@x vs user@x) and
-- silently broke password reset on case mismatch. Backfill existing rows to
-- lowercase, then move the unique indexes onto lower(email) so future
-- case-duplicates are impossible. Mirrors the contacts pattern (000007:47).
--
-- PRE-FLIGHT (run against prod BEFORE deploy; both MUST return 0 rows, else the
-- unique index creation fails mid-migration and leaves drift — deploy.sh rolls
-- back code only, not DB):
--   SELECT lower(email), count(*) FROM users
--     GROUP BY lower(email) HAVING count(*) > 1;
--   SELECT lower(email), count(*) FROM invitations WHERE accepted_at IS NULL
--     GROUP BY lower(email) HAVING count(*) > 1;

BEGIN;

-- users: backfill mixed-case rows, then swap unique index to lower(email).
UPDATE users SET email = lower(email) WHERE email <> lower(email);
DROP INDEX idx_users_email;
CREATE UNIQUE INDEX idx_users_email ON users (lower(email));

-- invitations: backfill, then swap the pending-unique index to lower(email).
UPDATE invitations SET email = lower(email) WHERE email <> lower(email);
DROP INDEX idx_invitations_email_pending;
CREATE UNIQUE INDEX idx_invitations_email_pending
    ON invitations (lower(email)) WHERE accepted_at IS NULL;

COMMIT;
