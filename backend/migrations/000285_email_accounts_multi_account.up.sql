-- Allow multiple email accounts per user (account switcher / unified inbox).
-- Existing rows are exactly one per user (enforced by the constraint being
-- dropped below), so DEFAULT true on the new column makes every current
-- account the user's default without a separate backfill step.
ALTER TABLE email_accounts DROP CONSTRAINT email_accounts_user_id_key;
ALTER TABLE email_accounts ADD COLUMN is_default BOOLEAN NOT NULL DEFAULT true;

-- Exactly one default account per user, enforced by the database, not just
-- application logic.
CREATE UNIQUE INDEX idx_email_accounts_user_default ON email_accounts (user_id) WHERE is_default;
