-- Rollback of migration 000260: email rules.

BEGIN;

ALTER TABLE email_messages
    DROP COLUMN IF EXISTS label_ids;

DROP TABLE IF EXISTS email_rules;

COMMIT;
