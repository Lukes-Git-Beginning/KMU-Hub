CREATE TABLE dialer_campaigns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft','active','paused','completed','archived')),
    mode TEXT NOT NULL DEFAULT 'preview'
        CHECK (mode IN ('preview','power','predictive')),
    settings JSONB NOT NULL DEFAULT '{}',
    created_by UUID NOT NULL REFERENCES users(id),
    assigned_agent_ids UUID[] NOT NULL DEFAULT '{}',
    contact_count INT NOT NULL DEFAULT 0,
    completed_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_dialer_campaigns_tenant ON dialer_campaigns(tenant_id);
CREATE INDEX idx_dialer_campaigns_status ON dialer_campaigns(tenant_id, status);
CREATE INDEX idx_dialer_campaigns_created_by ON dialer_campaigns(created_by);
