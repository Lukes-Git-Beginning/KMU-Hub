-- Guest chat support: session management + channel config

-- Guest sessions: one per guest visitor per chat conversation
CREATE TABLE guest_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    display_name VARCHAR(200) NOT NULL,
    email VARCHAR(255),
    ip_address INET,
    user_agent TEXT,
    last_activity_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_guest_sessions_token_hash ON guest_sessions(token_hash);
CREATE INDEX idx_guest_sessions_channel ON guest_sessions(channel_id);
CREATE INDEX idx_guest_sessions_active ON guest_sessions(channel_id) WHERE is_active = true;
CREATE INDEX idx_guest_sessions_cleanup ON guest_sessions(expires_at) WHERE is_active = true;

-- Per-channel guest chat settings
CREATE TABLE guest_channel_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id UUID NOT NULL UNIQUE REFERENCES channels(id) ON DELETE CASCADE,
    welcome_message TEXT DEFAULT 'Willkommen! Wie können wir Ihnen helfen?',
    logo_url VARCHAR(500),
    primary_color VARCHAR(7) DEFAULT '#3b82f6',
    token_expiry_hours INTEGER NOT NULL DEFAULT 168,
    max_file_size_mb INTEGER NOT NULL DEFAULT 10,
    allowed_file_types VARCHAR(100) DEFAULT 'image/png,image/jpeg,image/gif,image/webp,application/pdf',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Extend channels with guest support flag
ALTER TABLE channels ADD COLUMN is_guest_enabled BOOLEAN NOT NULL DEFAULT false;

-- Extend messages with guest session reference
ALTER TABLE messages ADD COLUMN guest_session_id UUID REFERENCES guest_sessions(id) ON DELETE SET NULL;
ALTER TABLE messages ALTER COLUMN created_by DROP NOT NULL;
ALTER TABLE messages ADD CONSTRAINT chk_message_sender CHECK (
    (created_by IS NOT NULL AND guest_session_id IS NULL) OR
    (created_by IS NULL AND guest_session_id IS NOT NULL)
);
CREATE INDEX idx_messages_guest_session ON messages(guest_session_id) WHERE guest_session_id IS NOT NULL;
