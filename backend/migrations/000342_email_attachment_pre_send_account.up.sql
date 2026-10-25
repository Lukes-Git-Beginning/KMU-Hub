-- A pre-send upload has no message yet: UploadAttachment stored message_id = uuid.Nil, which
-- violates the FK on email_messages(id) and failed every upload. The attachment now carries
-- its mailbox (account_id) and may wait without a message; the ownership check follows
-- attachment -> account -> caller until a message exists.
ALTER TABLE email_attachments ADD COLUMN account_id UUID REFERENCES email_accounts(id) ON DELETE CASCADE;
ALTER TABLE email_attachments ALTER COLUMN message_id DROP NOT NULL;
ALTER TABLE email_attachments ADD CONSTRAINT email_attachments_owner_check
    CHECK (message_id IS NOT NULL OR account_id IS NOT NULL);
CREATE INDEX idx_email_attachments_account ON email_attachments(account_id) WHERE account_id IS NOT NULL;
