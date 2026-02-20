-- App-specific passwords for CalDAV/CardDAV HTTP Basic Auth
CREATE TABLE app_specific_passwords (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    label VARCHAR(100) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    password_prefix VARCHAR(4) NOT NULL,
    scope VARCHAR(20) NOT NULL DEFAULT 'caldav_carddav',
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ
);

-- All passwords for a user (active + revoked)
CREATE INDEX idx_app_passwords_user ON app_specific_passwords(user_id);

-- Active passwords only (for validation lookups)
CREATE INDEX idx_app_passwords_active ON app_specific_passwords(user_id)
    WHERE revoked_at IS NULL;

-- Add CalDAV toggle to users table
ALTER TABLE users ADD COLUMN caldav_enabled BOOLEAN NOT NULL DEFAULT FALSE;
