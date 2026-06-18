CREATE TABLE IF NOT EXISTS helpdesk_routing_rules (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID        NOT NULL,
    name            TEXT        NOT NULL,
    conditions      JSONB       NOT NULL DEFAULT '{}',
    target_queue_id UUID        REFERENCES ticket_queues(id) ON DELETE SET NULL,
    priority        INT         NOT NULL DEFAULT 0,
    enabled         BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS helpdesk_routing_rules_tenant_idx ON helpdesk_routing_rules (tenant_id, priority);

CALL enable_tenant_rls('helpdesk_routing_rules');
