CREATE TABLE chat_files (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    message_id      UUID REFERENCES messages(id) ON DELETE SET NULL,
    channel_id      UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    filename        TEXT NOT NULL,
    mime_type       TEXT NOT NULL,
    file_size       BIGINT NOT NULL CHECK (file_size > 0),
    storage_key     TEXT NOT NULL,
    thumbnail_key   TEXT,
    uploaded_by     UUID NOT NULL REFERENCES users(id),
    is_deleted      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_chat_files_message ON chat_files(message_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_chat_files_channel ON chat_files(channel_id, created_at DESC) WHERE is_deleted = FALSE;
CREATE INDEX idx_chat_files_uploaded_by ON chat_files(uploaded_by);

-- Storage Quotas (org-level)
CREATE TABLE storage_quotas (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    max_bytes       BIGINT NOT NULL DEFAULT 10737418240,  -- 10 GB default
    used_bytes      BIGINT NOT NULL DEFAULT 0,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Insert default quota row (single-tenant for now, extendable to per-org later)
INSERT INTO storage_quotas (id, max_bytes, used_bytes)
VALUES ('00000000-0000-0000-0000-000000000001', 10737418240, 0);

-- Permissions
INSERT INTO permissions (name, resource, action) VALUES
    ('files:read', 'files', 'read'),
    ('files:write', 'files', 'write'),
    ('files:delete', 'files', 'delete')
ON CONFLICT (name) DO NOTHING;

-- Role assignments (admin: all, manager+member: read+write)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin' AND p.resource = 'files'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name IN ('manager', 'member') AND p.resource = 'files' AND p.action IN ('read', 'write')
ON CONFLICT (role_id, permission_id) DO NOTHING;
