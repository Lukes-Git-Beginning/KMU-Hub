CREATE TABLE dialer_campaign_contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID NOT NULL REFERENCES dialer_campaigns(id) ON DELETE CASCADE,
    contact_id UUID NOT NULL REFERENCES contacts(id),
    position INT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending','in_progress','completed','skipped','callback')),
    outcome_id UUID,
    notes TEXT,
    callback_at TIMESTAMPTZ,
    last_called_at TIMESTAMPTZ,
    call_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_campaign_contact UNIQUE(campaign_id, contact_id)
);

CREATE INDEX idx_dcc_campaign_status ON dialer_campaign_contacts(campaign_id, status);
CREATE INDEX idx_dcc_campaign_position ON dialer_campaign_contacts(campaign_id, position);
CREATE INDEX idx_dcc_contact ON dialer_campaign_contacts(contact_id);
CREATE INDEX idx_dcc_callback ON dialer_campaign_contacts(callback_at) WHERE status = 'callback';
