-- Contacts and Companies tables for CRM
-- Companies created first since contacts reference them

-- ============================================================================
-- Companies table
-- ============================================================================
CREATE TABLE companies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    domain VARCHAR(255),
    industry VARCHAR(100),
    employee_count INTEGER,
    address TEXT,
    city VARCHAR(100),
    country VARCHAR(100),
    notes TEXT,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for companies
CREATE INDEX idx_companies_name ON companies (LOWER(name));
CREATE INDEX idx_companies_domain ON companies (LOWER(domain)) WHERE domain IS NOT NULL;
CREATE INDEX idx_companies_industry ON companies (industry) WHERE industry IS NOT NULL;
CREATE INDEX idx_companies_created_by ON companies (created_by);
CREATE INDEX idx_companies_created_at ON companies (created_at DESC);

-- ============================================================================
-- Contacts table
-- ============================================================================
CREATE TABLE contacts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    email VARCHAR(255),
    phone VARCHAR(50),
    company_id UUID REFERENCES companies(id) ON DELETE SET NULL,
    position VARCHAR(100),
    notes TEXT,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for contacts
CREATE UNIQUE INDEX idx_contacts_email ON contacts (LOWER(email)) WHERE email IS NOT NULL;
CREATE INDEX idx_contacts_name ON contacts (LOWER(last_name), LOWER(first_name));
CREATE INDEX idx_contacts_company ON contacts (company_id) WHERE company_id IS NOT NULL;
CREATE INDEX idx_contacts_created_by ON contacts (created_by);
CREATE INDEX idx_contacts_created_at ON contacts (created_at DESC);

-- ============================================================================
-- Custom field values tables
-- ============================================================================
CREATE TABLE contact_custom_field_values (
    contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    field_id UUID NOT NULL REFERENCES custom_field_definitions(id) ON DELETE CASCADE,
    value JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (contact_id, field_id)
);

CREATE INDEX idx_contact_custom_field_values_field ON contact_custom_field_values (field_id);

CREATE TABLE company_custom_field_values (
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    field_id UUID NOT NULL REFERENCES custom_field_definitions(id) ON DELETE CASCADE,
    value JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (company_id, field_id)
);

CREATE INDEX idx_company_custom_field_values_field ON company_custom_field_values (field_id);

-- ============================================================================
-- Add FK constraints to existing junction tables
-- ============================================================================
ALTER TABLE contact_tags
    ADD CONSTRAINT fk_contact_tags_contact
    FOREIGN KEY (contact_id) REFERENCES contacts(id) ON DELETE CASCADE;

ALTER TABLE company_tags
    ADD CONSTRAINT fk_company_tags_company
    FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE;

-- ============================================================================
-- Companies permissions (contacts permissions already exist from 000002)
-- ============================================================================
INSERT INTO permissions (name, resource, action) VALUES
    ('companies:read', 'companies', 'read'),
    ('companies:write', 'companies', 'write'),
    ('companies:delete', 'companies', 'delete')
ON CONFLICT (name) DO NOTHING;

-- Assign companies permissions to admin (all)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin' AND p.resource = 'companies'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Assign read + write to manager
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'manager' AND p.resource = 'companies' AND p.action IN ('read', 'write')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Assign read to member
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'member' AND p.resource = 'companies' AND p.action = 'read'
ON CONFLICT (role_id, permission_id) DO NOTHING;
