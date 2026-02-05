-- Activities table for CRM
-- Tracks calls, meetings, notes, emails, and tasks

-- Activity type enum
CREATE TYPE activity_type AS ENUM ('call', 'meeting', 'note', 'email', 'task');

CREATE TABLE activities (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    activity_type activity_type NOT NULL,
    subject VARCHAR(255) NOT NULL,
    description TEXT,
    contact_id UUID REFERENCES contacts(id) ON DELETE SET NULL,
    company_id UUID REFERENCES companies(id) ON DELETE SET NULL,
    deal_id UUID REFERENCES deals(id) ON DELETE SET NULL,
    assigned_to UUID REFERENCES users(id) ON DELETE SET NULL,
    due_date TIMESTAMPTZ,
    is_completed BOOLEAN NOT NULL DEFAULT FALSE,
    completed_at TIMESTAMPTZ,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for activities
CREATE INDEX idx_activities_type ON activities (activity_type);
CREATE INDEX idx_activities_contact ON activities (contact_id) WHERE contact_id IS NOT NULL;
CREATE INDEX idx_activities_company ON activities (company_id) WHERE company_id IS NOT NULL;
CREATE INDEX idx_activities_deal ON activities (deal_id) WHERE deal_id IS NOT NULL;
CREATE INDEX idx_activities_assigned_to ON activities (assigned_to) WHERE assigned_to IS NOT NULL;
CREATE INDEX idx_activities_created_by ON activities (created_by);
CREATE INDEX idx_activities_due_date ON activities (due_date) WHERE due_date IS NOT NULL;
CREATE INDEX idx_activities_is_completed ON activities (is_completed);
CREATE INDEX idx_activities_created_at ON activities (created_at DESC);
CREATE INDEX idx_activities_subject ON activities (LOWER(subject));

-- Custom field values for activities
CREATE TABLE activity_custom_field_values (
    activity_id UUID NOT NULL REFERENCES activities(id) ON DELETE CASCADE,
    field_id UUID NOT NULL REFERENCES custom_field_definitions(id) ON DELETE CASCADE,
    value JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (activity_id, field_id)
);

CREATE INDEX idx_activity_custom_field_values_field ON activity_custom_field_values (field_id);

-- Add FK constraint to existing activity_tags junction table
ALTER TABLE activity_tags
    ADD CONSTRAINT fk_activity_tags_activity
    FOREIGN KEY (activity_id) REFERENCES activities(id) ON DELETE CASCADE;

-- Add activities permissions (idempotent)
INSERT INTO permissions (name, resource, action) VALUES
    ('activities:read', 'activities', 'read'),
    ('activities:write', 'activities', 'write'),
    ('activities:delete', 'activities', 'delete')
ON CONFLICT (name) DO NOTHING;

-- Assign activities permissions to admin (all)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin' AND p.resource = 'activities'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Assign read + write to manager
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'manager' AND p.resource = 'activities' AND p.action IN ('read', 'write')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Assign read + write to member (members can manage their own activities)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'member' AND p.resource = 'activities' AND p.action IN ('read', 'write')
ON CONFLICT (role_id, permission_id) DO NOTHING;
