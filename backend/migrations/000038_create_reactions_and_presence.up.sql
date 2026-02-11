-- Message reactions for chat and presence configuration
CREATE TABLE message_reactions (
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    emoji TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (message_id, user_id, emoji)
);

CREATE INDEX idx_message_reactions_message ON message_reactions(message_id);

CREATE TABLE presence_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    away_timeout_seconds INT NOT NULL DEFAULT 300,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID REFERENCES users(id)
);

-- Seed default presence config
INSERT INTO presence_config (away_timeout_seconds) VALUES (300);
