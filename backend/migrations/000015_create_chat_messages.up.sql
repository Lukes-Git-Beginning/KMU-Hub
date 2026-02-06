-- Chat messages table
-- Stores all messages in channels

CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    edited_at TIMESTAMPTZ,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for messages
-- Composite index for efficient pagination (channel + time)
CREATE INDEX idx_messages_channel_created ON messages (channel_id, created_at DESC);
CREATE INDEX idx_messages_created_by ON messages (created_by);
CREATE INDEX idx_messages_is_deleted ON messages (is_deleted) WHERE is_deleted = TRUE;

-- Add messages permissions (idempotent)
INSERT INTO permissions (name, resource, action) VALUES
    ('messages:read', 'messages', 'read'),
    ('messages:write', 'messages', 'write'),
    ('messages:delete', 'messages', 'delete')
ON CONFLICT (name) DO NOTHING;

-- Assign messages permissions to admin (all)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin' AND p.resource = 'messages'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Assign read + write to manager
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'manager' AND p.resource = 'messages' AND p.action IN ('read', 'write')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Assign read + write to member (members can send messages)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'member' AND p.resource = 'messages' AND p.action IN ('read', 'write')
ON CONFLICT (role_id, permission_id) DO NOTHING;
