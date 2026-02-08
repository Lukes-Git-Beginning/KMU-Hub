-- Projects foundation: projects, project_members, project_statuses
-- Supports project management with role-based membership and customizable status workflows

-- ============================================================================
-- Projects table
-- ============================================================================

CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    project_key VARCHAR(10) NOT NULL,
    next_task_number INTEGER NOT NULL DEFAULT 1,
    is_template BOOLEAN NOT NULL DEFAULT false,
    template_source_id UUID REFERENCES projects(id) ON DELETE SET NULL,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    archived_at TIMESTAMPTZ
);

-- Unique project key among active (non-archived) projects
CREATE UNIQUE INDEX idx_projects_key ON projects (project_key) WHERE archived_at IS NULL;
CREATE INDEX idx_projects_created_by ON projects (created_by);
CREATE INDEX idx_projects_archived ON projects (archived_at) WHERE archived_at IS NOT NULL;
CREATE INDEX idx_projects_template ON projects (is_template) WHERE is_template = true;

-- ============================================================================
-- Project members table (composite PK)
-- ============================================================================

CREATE TABLE project_members (
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL DEFAULT 'member' CHECK (role IN ('owner', 'member', 'viewer')),
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (project_id, user_id)
);

CREATE INDEX idx_project_members_user ON project_members (user_id);

-- ============================================================================
-- Project statuses table (customizable per project)
-- ============================================================================

CREATE TABLE project_statuses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    color VARCHAR(7),
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_default BOOLEAN NOT NULL DEFAULT false,
    is_closed BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_project_statuses_project ON project_statuses (project_id, sort_order);
CREATE UNIQUE INDEX idx_project_statuses_name ON project_statuses (project_id, LOWER(name));

-- ============================================================================
-- RBAC permissions for projects and tasks
-- ============================================================================

INSERT INTO permissions (name, resource, action) VALUES
    ('projects:read', 'projects', 'read'),
    ('projects:write', 'projects', 'write'),
    ('projects:delete', 'projects', 'delete'),
    ('tasks:read', 'tasks', 'read'),
    ('tasks:write', 'tasks', 'write'),
    ('tasks:delete', 'tasks', 'delete')
ON CONFLICT (name) DO NOTHING;

-- Grant read + write to all roles
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name IN ('admin', 'manager', 'member')
AND p.resource IN ('projects', 'tasks')
AND p.action IN ('read', 'write')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Grant delete only to admin and manager
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name IN ('admin', 'manager')
AND p.resource IN ('projects', 'tasks')
AND p.action = 'delete'
ON CONFLICT (role_id, permission_id) DO NOTHING;
