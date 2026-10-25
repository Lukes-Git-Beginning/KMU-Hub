DELETE FROM email_attachments WHERE message_id IS NULL;
DROP INDEX IF EXISTS idx_email_attachments_account;
ALTER TABLE email_attachments DROP CONSTRAINT IF EXISTS email_attachments_owner_check;
ALTER TABLE email_attachments ALTER COLUMN message_id SET NOT NULL;
ALTER TABLE email_attachments DROP COLUMN IF EXISTS account_id;
