-- Mention type enum
CREATE TYPE mention_type AS ENUM ('user', 'channel', 'everyone');

-- Junction table for message mentions
CREATE TABLE message_mentions (
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    mention_type mention_type NOT NULL DEFAULT 'user',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (message_id, user_id)
);

CREATE INDEX idx_message_mentions_user ON message_mentions(user_id, created_at DESC);
CREATE INDEX idx_message_mentions_type ON message_mentions(mention_type) WHERE mention_type != 'user';

-- Permissions
INSERT INTO permissions (name, resource, action) VALUES ('mentions:read', 'mentions', 'read') ON CONFLICT (name) DO NOTHING;
