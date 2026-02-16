-- Migration 000041: Create email tables
-- Creates the complete email data layer: accounts, folders, messages,
-- contact links, attachments, and signatures.

-- Email accounts (one per user, vault-encrypted IMAP/SMTP credentials)
CREATE TABLE email_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    email_address VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    imap_host VARCHAR(255) NOT NULL,
    imap_port INTEGER NOT NULL DEFAULT 993,
    smtp_host VARCHAR(255) NOT NULL,
    smtp_port INTEGER NOT NULL DEFAULT 587,
    username VARCHAR(255) NOT NULL,
    password_encrypted TEXT NOT NULL,
    use_ssl BOOLEAN NOT NULL DEFAULT true,
    last_sync_at TIMESTAMPTZ,
    sync_enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(user_id)
);
CREATE INDEX idx_email_accounts_user ON email_accounts(user_id);

-- Email folders (mapped from IMAP)
CREATE TABLE email_folders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES email_accounts(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    imap_name VARCHAR(255) NOT NULL,
    folder_type VARCHAR(50) NOT NULL DEFAULT 'custom',
    uid_validity BIGINT NOT NULL DEFAULT 0,
    highest_uid BIGINT NOT NULL DEFAULT 0,
    message_count INTEGER NOT NULL DEFAULT 0,
    unread_count INTEGER NOT NULL DEFAULT 0,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(account_id, imap_name)
);
CREATE INDEX idx_email_folders_account ON email_folders(account_id);

-- Email messages (cached from IMAP)
CREATE TABLE email_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES email_accounts(id) ON DELETE CASCADE,
    folder_id UUID NOT NULL REFERENCES email_folders(id) ON DELETE CASCADE,
    uid BIGINT NOT NULL,
    message_id VARCHAR(998),
    in_reply_to VARCHAR(998),
    "references" TEXT[],
    thread_id UUID,
    from_name VARCHAR(255),
    from_email VARCHAR(255) NOT NULL,
    to_addresses JSONB NOT NULL DEFAULT '[]',
    cc_addresses JSONB NOT NULL DEFAULT '[]',
    bcc_addresses JSONB NOT NULL DEFAULT '[]',
    subject VARCHAR(998),
    preview TEXT,
    body_text TEXT,
    body_html TEXT,
    is_read BOOLEAN NOT NULL DEFAULT false,
    is_starred BOOLEAN NOT NULL DEFAULT false,
    is_draft BOOLEAN NOT NULL DEFAULT false,
    has_attachments BOOLEAN NOT NULL DEFAULT false,
    date TIMESTAMPTZ NOT NULL,
    size_bytes BIGINT NOT NULL DEFAULT 0,
    raw_headers TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(folder_id, uid)
);
CREATE INDEX idx_email_messages_account ON email_messages(account_id);
CREATE INDEX idx_email_messages_folder ON email_messages(folder_id);
CREATE INDEX idx_email_messages_thread ON email_messages(thread_id);
CREATE INDEX idx_email_messages_date ON email_messages(date DESC);
CREATE INDEX idx_email_messages_message_id ON email_messages(message_id);
CREATE INDEX idx_email_messages_from ON email_messages(from_email);

-- Full-text search on email messages
ALTER TABLE email_messages ADD COLUMN search_vector tsvector;
CREATE INDEX idx_email_messages_search ON email_messages USING GIN(search_vector);

CREATE OR REPLACE FUNCTION email_messages_search_vector_update() RETURNS trigger AS $$
BEGIN
    NEW.search_vector := to_tsvector('german',
        coalesce(NEW.subject, '') || ' ' ||
        coalesce(NEW.body_text, '') || ' ' ||
        coalesce(NEW.from_name, '') || ' ' ||
        coalesce(NEW.from_email, '')
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_email_messages_search_vector
    BEFORE INSERT OR UPDATE ON email_messages
    FOR EACH ROW EXECUTE FUNCTION email_messages_search_vector_update();

-- Email-CRM contact links (junction table)
CREATE TABLE email_contact_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES email_messages(id) ON DELETE CASCADE,
    contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    link_type VARCHAR(20) NOT NULL DEFAULT 'auto',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(message_id, contact_id)
);
CREATE INDEX idx_email_contact_links_message ON email_contact_links(message_id);
CREATE INDEX idx_email_contact_links_contact ON email_contact_links(contact_id);

-- Email attachments (metadata; files stored in MinIO)
CREATE TABLE email_attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES email_messages(id) ON DELETE CASCADE,
    filename VARCHAR(255) NOT NULL,
    content_type VARCHAR(255) NOT NULL,
    size_bytes BIGINT NOT NULL DEFAULT 0,
    minio_key VARCHAR(512) NOT NULL,
    content_id VARCHAR(255),
    is_inline BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_email_attachments_message ON email_attachments(message_id);

-- Email signatures (per-user HTML signatures)
CREATE TABLE email_signatures (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    name VARCHAR(255) NOT NULL DEFAULT 'Standard',
    html_content TEXT NOT NULL DEFAULT '',
    is_default BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_email_signatures_user ON email_signatures(user_id);
