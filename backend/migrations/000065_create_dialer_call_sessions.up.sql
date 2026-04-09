CREATE TABLE dialer_call_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_contact_id UUID NOT NULL REFERENCES dialer_campaign_contacts(id),
    call_session_id UUID REFERENCES call_sessions(id),
    agent_id UUID NOT NULL REFERENCES users(id),
    outcome_id UUID,
    duration_seconds INT,
    notes TEXT,
    next_action TEXT,
    appointment_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    wrap_up_completed_at TIMESTAMPTZ
);

CREATE INDEX idx_dcs_campaign_contact ON dialer_call_sessions(campaign_contact_id);
CREATE INDEX idx_dcs_agent ON dialer_call_sessions(agent_id);
CREATE INDEX idx_dcs_call_session ON dialer_call_sessions(call_session_id) WHERE call_session_id IS NOT NULL;

CREATE TABLE dialer_call_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dialer_call_session_id UUID NOT NULL REFERENCES dialer_call_sessions(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_dce_session ON dialer_call_events(dialer_call_session_id, occurred_at);
