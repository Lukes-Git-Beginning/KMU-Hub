-- Migration 000148 (down): restore the case-sensitive email unique indexes.
-- NOTE: the lowercase backfill of existing rows is NOT reversed — original
-- mixed-case values are lost. Only the index expressions are restored.

BEGIN;

DROP INDEX idx_users_email;
CREATE UNIQUE INDEX idx_users_email ON users (email);

DROP INDEX idx_invitations_email_pending;
CREATE UNIQUE INDEX idx_invitations_email_pending
    ON invitations (email) WHERE accepted_at IS NULL;

COMMIT;
