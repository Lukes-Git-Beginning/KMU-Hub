CREATE TABLE dialer_agent_status_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    campaign_id UUID REFERENCES dialer_campaigns(id),
    status TEXT NOT NULL
        CHECK (status IN ('available','on_call','wrap_up','break','offline')),
    previous_status TEXT,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    changed_by UUID REFERENCES users(id)
);

CREATE INDEX idx_dasl_user ON dialer_agent_status_log(user_id, changed_at DESC);
CREATE INDEX idx_dasl_campaign ON dialer_agent_status_log(campaign_id) WHERE campaign_id IS NOT NULL;
