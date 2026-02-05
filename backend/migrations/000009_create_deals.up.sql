-- Deals table for sales pipeline
-- Tracks sales opportunities through pipeline stages

CREATE TABLE deals (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    value DECIMAL(15,2) NOT NULL DEFAULT 0.00,
    currency VARCHAR(3) NOT NULL DEFAULT 'EUR',
    stage_id UUID NOT NULL REFERENCES pipeline_stages(id),
    contact_id UUID REFERENCES contacts(id) ON DELETE SET NULL,
    company_id UUID REFERENCES companies(id) ON DELETE SET NULL,
    owner_id UUID REFERENCES users(id) ON DELETE SET NULL,
    expected_close_date DATE,
    notes TEXT,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMPTZ
);

-- Indexes for deals
CREATE INDEX idx_deals_stage ON deals (stage_id);
CREATE INDEX idx_deals_contact ON deals (contact_id) WHERE contact_id IS NOT NULL;
CREATE INDEX idx_deals_company ON deals (company_id) WHERE company_id IS NOT NULL;
CREATE INDEX idx_deals_owner ON deals (owner_id) WHERE owner_id IS NOT NULL;
CREATE INDEX idx_deals_created_by ON deals (created_by);
CREATE INDEX idx_deals_created_at ON deals (created_at DESC);
CREATE INDEX idx_deals_expected_close ON deals (expected_close_date) WHERE expected_close_date IS NOT NULL;
CREATE INDEX idx_deals_name ON deals (LOWER(name));

-- Custom field values for deals
CREATE TABLE deal_custom_field_values (
    deal_id UUID NOT NULL REFERENCES deals(id) ON DELETE CASCADE,
    field_id UUID NOT NULL REFERENCES custom_field_definitions(id) ON DELETE CASCADE,
    value JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (deal_id, field_id)
);

CREATE INDEX idx_deal_custom_field_values_field ON deal_custom_field_values (field_id);

-- Add FK constraint to existing deal_tags junction table
ALTER TABLE deal_tags
    ADD CONSTRAINT fk_deal_tags_deal
    FOREIGN KEY (deal_id) REFERENCES deals(id) ON DELETE CASCADE;

-- Add deals permissions
INSERT INTO permissions (name, resource, action) VALUES
    ('deals:read', 'deals', 'read'),
    ('deals:write', 'deals', 'write'),
    ('deals:delete', 'deals', 'delete');

-- Assign deals permissions to admin (all)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin' AND p.resource = 'deals';

-- Assign read + write to manager
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'manager' AND p.resource = 'deals' AND p.action IN ('read', 'write');

-- Assign read to member
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'member' AND p.resource = 'deals' AND p.action = 'read';
