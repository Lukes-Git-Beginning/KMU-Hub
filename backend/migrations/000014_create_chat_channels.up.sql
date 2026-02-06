-- Chat channels table
-- Supports public channels, private channels, and direct messages

-- Channel role enum for memberships
CREATE TYPE channel_role AS ENUM ('owner', 'admin', 'member');

CREATE TABLE channels (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    is_private BOOLEAN NOT NULL DEFAULT FALSE,
    is_dm BOOLEAN NOT NULL DEFAULT FALSE,
    is_archived BOOLEAN NOT NULL DEFAULT FALSE,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for channels
CREATE INDEX idx_channels_created_by ON channels (created_by);
CREATE INDEX idx_channels_is_private ON channels (is_private);
CREATE INDEX idx_channels_is_dm ON channels (is_dm) WHERE is_dm = TRUE;
CREATE INDEX idx_channels_is_archived ON channels (is_archived);
CREATE INDEX idx_channels_name ON channels (LOWER(name));
CREATE INDEX idx_channels_created_at ON channels (created_at DESC);

-- Channel memberships junction table
CREATE TABLE channel_memberships (
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role channel_role NOT NULL DEFAULT 'member',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_read_at TIMESTAMPTZ,
    PRIMARY KEY (channel_id, user_id)
);

-- Indexes for channel memberships
CREATE INDEX idx_channel_memberships_user ON channel_memberships (user_id);
CREATE INDEX idx_channel_memberships_role ON channel_memberships (role);
CREATE INDEX idx_channel_memberships_last_read ON channel_memberships (last_read_at) WHERE last_read_at IS NOT NULL;

-- Add channels permissions (idempotent)
INSERT INTO permissions (name, resource, action) VALUES
    ('channels:read', 'channels', 'read'),
    ('channels:write', 'channels', 'write'),
    ('channels:delete', 'channels', 'delete')
ON CONFLICT (name) DO NOTHING;

-- Assign channels permissions to admin (all)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin' AND p.resource = 'channels'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Assign read + write to manager
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'manager' AND p.resource = 'channels' AND p.action IN ('read', 'write')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Assign read + write to member (members can create and join channels)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'member' AND p.resource = 'channels' AND p.action IN ('read', 'write')
ON CONFLICT (role_id, permission_id) DO NOTHING;
