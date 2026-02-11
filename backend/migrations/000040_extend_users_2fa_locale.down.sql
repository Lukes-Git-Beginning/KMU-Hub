-- Remove 2FA and locale columns from users table
ALTER TABLE users DROP COLUMN IF EXISTS locale;
ALTER TABLE users DROP COLUMN IF EXISTS two_factor_enabled_at;
ALTER TABLE users DROP COLUMN IF EXISTS two_factor_pending_secret;
ALTER TABLE users DROP COLUMN IF EXISTS two_factor_secret_encrypted;
ALTER TABLE users DROP COLUMN IF EXISTS two_factor_enabled;
