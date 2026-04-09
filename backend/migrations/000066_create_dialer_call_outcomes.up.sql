CREATE TABLE dialer_call_outcomes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    label VARCHAR(100) NOT NULL,
    color VARCHAR(20) NOT NULL DEFAULT '#6b7280',
    is_positive BOOLEAN NOT NULL DEFAULT false,
    is_callback BOOLEAN NOT NULL DEFAULT false,
    is_appointment BOOLEAN NOT NULL DEFAULT false,
    sort_order INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_dco_tenant ON dialer_call_outcomes(tenant_id, is_active);

-- Add FK constraints to existing tables
ALTER TABLE dialer_campaign_contacts
    ADD CONSTRAINT fk_dcc_outcome FOREIGN KEY (outcome_id) REFERENCES dialer_call_outcomes(id);

ALTER TABLE dialer_call_sessions
    ADD CONSTRAINT fk_dcs_outcome FOREIGN KEY (outcome_id) REFERENCES dialer_call_outcomes(id);
