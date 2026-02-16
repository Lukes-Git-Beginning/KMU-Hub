-- Migration 000042: Add visibility and owner_id to contacts table
-- Supports shared (visible to all) and personal (visible to owner only) contacts.

ALTER TABLE contacts ADD COLUMN visibility VARCHAR(20) NOT NULL DEFAULT 'shared';
ALTER TABLE contacts ADD CONSTRAINT chk_contacts_visibility CHECK (visibility IN ('shared', 'personal'));
ALTER TABLE contacts ADD COLUMN owner_id UUID REFERENCES users(id);

CREATE INDEX idx_contacts_visibility ON contacts(visibility);
CREATE INDEX idx_contacts_owner ON contacts(owner_id);
