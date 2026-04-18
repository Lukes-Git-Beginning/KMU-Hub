-- Sprint 1 S1.4 Helpdesk — schema (filled by Welle 2 Agent MOD-Helpdesk)

-- SLA policies must exist before ticket_queues references them.
CREATE TABLE sla_policies (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID        NOT NULL,
    name                 TEXT        NOT NULL,
    first_response_mins  INT         NOT NULL,
    resolution_mins      INT         NOT NULL,
    business_hours       JSONB       NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE ticket_queues (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID        NOT NULL,
    name                TEXT        NOT NULL,
    default_assignee_id UUID        NULL,
    sla_policy_id       UUID        NULL REFERENCES sla_policies(id) ON DELETE SET NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, name)
);

CREATE TABLE tickets (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID        NOT NULL,
    subject            TEXT        NOT NULL,
    status             TEXT        NOT NULL DEFAULT 'open'
                           CHECK (status IN ('open','pending','solved','closed','merged')),
    priority           TEXT        NOT NULL DEFAULT 'normal'
                           CHECK (priority IN ('low','normal','high','urgent')),
    assignee_id        UUID        NULL,
    requester_id       UUID        NOT NULL,
    queue_id           UUID        NULL REFERENCES ticket_queues(id) ON DELETE SET NULL,
    due_at             TIMESTAMPTZ NULL,
    merged_into_id     UUID        NULL REFERENCES tickets(id) ON DELETE SET NULL,
    first_response_at  TIMESTAMPTZ NULL,
    resolved_at        TIMESTAMPTZ NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE ticket_messages (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id   UUID        NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    author_id   UUID        NOT NULL,
    body        TEXT        NOT NULL,
    internal    BOOL        NOT NULL DEFAULT false,
    attachments JSONB       NOT NULL DEFAULT '[]'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE canned_responses (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID        NOT NULL,
    name       TEXT        NOT NULL,
    body       TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, name)
);

-- Indexes
CREATE INDEX idx_tickets_tenant_status   ON tickets(tenant_id, status);
CREATE INDEX idx_tickets_tenant_assignee ON tickets(tenant_id, assignee_id);
CREATE INDEX idx_tickets_tenant_due      ON tickets(tenant_id, due_at) WHERE due_at IS NOT NULL;
CREATE INDEX idx_ticket_messages_ticket  ON ticket_messages(ticket_id);
