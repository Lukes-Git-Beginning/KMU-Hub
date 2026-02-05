-- Pipeline stages for deals
-- Each stage represents a step in the sales pipeline

CREATE TABLE pipeline_stages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    color VARCHAR(7) NOT NULL DEFAULT '#3b82f6',
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_won BOOLEAN NOT NULL DEFAULT FALSE,
    is_lost BOOLEAN NOT NULL DEFAULT FALSE,
    probability DECIMAL(5,2) NOT NULL DEFAULT 0.00,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_stage_color_format CHECK (color ~ '^#[0-9a-fA-F]{6}$'),
    CONSTRAINT valid_probability CHECK (probability >= 0 AND probability <= 100)
);

-- Index for ordering stages
CREATE INDEX idx_pipeline_stages_sort_order ON pipeline_stages (sort_order);

-- Unique index to ensure only one won and one lost stage
CREATE UNIQUE INDEX idx_pipeline_stages_won ON pipeline_stages (is_won) WHERE is_won = TRUE;
CREATE UNIQUE INDEX idx_pipeline_stages_lost ON pipeline_stages (is_lost) WHERE is_lost = TRUE;

-- Insert default pipeline stages
INSERT INTO pipeline_stages (name, color, sort_order, is_won, is_lost, probability) VALUES
    ('Lead', '#3b82f6', 1, FALSE, FALSE, 10.00),
    ('Qualified', '#14b8a6', 2, FALSE, FALSE, 30.00),
    ('Proposal', '#eab308', 3, FALSE, FALSE, 50.00),
    ('Negotiation', '#f97316', 4, FALSE, FALSE, 70.00),
    ('Won', '#22c55e', 5, TRUE, FALSE, 100.00),
    ('Lost', '#6b7280', 6, FALSE, TRUE, 0.00);

-- Add pipeline stages permissions (idempotent)
INSERT INTO permissions (name, resource, action) VALUES
    ('pipeline_stages:read', 'pipeline_stages', 'read'),
    ('pipeline_stages:write', 'pipeline_stages', 'write'),
    ('pipeline_stages:delete', 'pipeline_stages', 'delete')
ON CONFLICT (name) DO NOTHING;

-- Assign pipeline_stages permissions to admin (all)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin' AND p.resource = 'pipeline_stages'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Assign read + write to manager
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'manager' AND p.resource = 'pipeline_stages' AND p.action IN ('read', 'write')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Assign read to member
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'member' AND p.resource = 'pipeline_stages' AND p.action = 'read'
ON CONFLICT (role_id, permission_id) DO NOTHING;
