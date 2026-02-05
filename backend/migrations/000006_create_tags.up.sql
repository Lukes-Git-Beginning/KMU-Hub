-- Tags system for CRM entities
-- Each entity type has its own tags (contact tags, company tags, deal tags, activity tags)

CREATE TABLE tags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    color VARCHAR(7) NOT NULL DEFAULT '#6b7280',  -- Hex color code
    entity_type VARCHAR(50) NOT NULL,  -- 'contact', 'company', 'deal', 'activity'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_entity_type CHECK (entity_type IN ('contact', 'company', 'deal', 'activity')),
    CONSTRAINT valid_color_format CHECK (color ~ '^#[0-9a-fA-F]{6}$')
);

-- Unique tag name per entity type
CREATE UNIQUE INDEX idx_tags_entity_name ON tags (entity_type, LOWER(name));

-- Index for listing tags by entity type
CREATE INDEX idx_tags_entity_type ON tags (entity_type, name);

-- Junction tables for many-to-many relationships
-- These will be used when we implement contacts, companies, deals, activities

CREATE TABLE contact_tags (
    contact_id UUID NOT NULL,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (contact_id, tag_id)
);

CREATE INDEX idx_contact_tags_contact ON contact_tags (contact_id);
CREATE INDEX idx_contact_tags_tag ON contact_tags (tag_id);

CREATE TABLE company_tags (
    company_id UUID NOT NULL,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (company_id, tag_id)
);

CREATE INDEX idx_company_tags_company ON company_tags (company_id);
CREATE INDEX idx_company_tags_tag ON company_tags (tag_id);

CREATE TABLE deal_tags (
    deal_id UUID NOT NULL,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (deal_id, tag_id)
);

CREATE INDEX idx_deal_tags_deal ON deal_tags (deal_id);
CREATE INDEX idx_deal_tags_tag ON deal_tags (tag_id);

CREATE TABLE activity_tags (
    activity_id UUID NOT NULL,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (activity_id, tag_id)
);

CREATE INDEX idx_activity_tags_activity ON activity_tags (activity_id);
CREATE INDEX idx_activity_tags_tag ON activity_tags (tag_id);

-- Add tags permissions
INSERT INTO permissions (name, resource, action) VALUES
    ('tags:read', 'tags', 'read'),
    ('tags:write', 'tags', 'write'),
    ('tags:delete', 'tags', 'delete');

-- Assign tags permissions to admin
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin' AND p.resource = 'tags';

-- Assign read + write to manager
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'manager' AND p.resource = 'tags' AND p.action IN ('read', 'write');

-- Assign read to member
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'member' AND p.resource = 'tags' AND p.action = 'read';
