DROP INDEX IF EXISTS idx_email_accounts_user_default;
ALTER TABLE email_accounts DROP COLUMN is_default;
ALTER TABLE email_accounts ADD CONSTRAINT email_accounts_user_id_key UNIQUE (user_id);
