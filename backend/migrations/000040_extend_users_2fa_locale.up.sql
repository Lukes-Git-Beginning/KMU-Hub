-- Extend users table with 2FA and locale columns
ALTER TABLE users ADD COLUMN two_factor_enabled BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE users ADD COLUMN two_factor_secret_encrypted TEXT;
ALTER TABLE users ADD COLUMN two_factor_pending_secret TEXT;
ALTER TABLE users ADD COLUMN two_factor_enabled_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN locale VARCHAR(5) NOT NULL DEFAULT 'de';
