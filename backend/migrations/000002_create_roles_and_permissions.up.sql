CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(50) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_roles_name ON roles (name);

CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    resource VARCHAR(50) NOT NULL,
    action VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_permissions_name ON permissions (name);
CREATE UNIQUE INDEX idx_permissions_resource_action ON permissions (resource, action);

CREATE TABLE role_permissions (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, role_id)
);

-- Seed default roles
INSERT INTO roles (name, description) VALUES
    ('admin', 'Full system access'),
    ('manager', 'Team management and reporting'),
    ('member', 'Standard user access');

-- Seed default permissions
INSERT INTO permissions (name, resource, action) VALUES
    ('contacts:read', 'contacts', 'read'),
    ('contacts:write', 'contacts', 'write'),
    ('contacts:delete', 'contacts', 'delete'),
    ('deals:read', 'deals', 'read'),
    ('deals:write', 'deals', 'write'),
    ('deals:delete', 'deals', 'delete'),
    ('users:read', 'users', 'read'),
    ('users:write', 'users', 'write'),
    ('users:delete', 'users', 'delete'),
    ('roles:manage', 'roles', 'manage');

-- Assign all permissions to admin
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r CROSS JOIN permissions p WHERE r.name = 'admin';

-- Assign read + write to manager
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'manager' AND p.action IN ('read', 'write');

-- Assign read to member
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'member' AND p.action = 'read';
