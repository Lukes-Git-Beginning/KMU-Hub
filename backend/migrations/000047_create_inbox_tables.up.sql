-- Unified inbox messages (normalized from all channels)
CREATE TABLE inbox_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel VARCHAR(20) NOT NULL CHECK (channel IN ('email', 'chat', 'notification')),
    source_id VARCHAR(255) NOT NULL,
    sender_name VARCHAR(255) NOT NULL DEFAULT '',
    sender_id UUID,
    sender_email VARCHAR(255),
    subject VARCHAR(500) NOT NULL DEFAULT '',
    preview TEXT NOT NULL DEFAULT '',
    is_read BOOLEAN NOT NULL DEFAULT false,
    is_starred BOOLEAN NOT NULL DEFAULT false,
    is_archived BOOLEAN NOT NULL DEFAULT false,
    snoozed_until TIMESTAMPTZ,
    assigned_to UUID REFERENCES users(id),
    team_inbox_id UUID,
    tags TEXT[] NOT NULL DEFAULT '{}',
    deep_link VARCHAR(1000) NOT NULL DEFAULT '',
    crm_contact_id UUID,
    metadata JSONB,
    received_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Fast lookup: user's unread inbox (non-archived, non-snoozed)
CREATE INDEX idx_inbox_messages_user_unread ON inbox_messages(user_id, received_at DESC)
    WHERE is_read = false AND is_archived = false AND snoozed_until IS NULL;

-- Fast lookup: user's inbox filtered by channel (non-archived)
CREATE INDEX idx_inbox_messages_user_channel ON inbox_messages(user_id, channel, received_at DESC)
    WHERE is_archived = false;

-- Fast lookup: team inbox unassigned items
CREATE INDEX idx_inbox_messages_team_unassigned ON inbox_messages(team_inbox_id, received_at DESC)
    WHERE assigned_to IS NULL AND is_archived = false;

-- Fast lookup: snoozed items due for requeue
CREATE INDEX idx_inbox_messages_snoozed ON inbox_messages(snoozed_until)
    WHERE snoozed_until IS NOT NULL AND is_archived = false;

-- Unique constraint: prevent duplicate source items per user per channel
CREATE UNIQUE INDEX idx_inbox_messages_user_source ON inbox_messages(user_id, channel, source_id);

-- Starred items lookup
CREATE INDEX idx_inbox_messages_user_starred ON inbox_messages(user_id, received_at DESC)
    WHERE is_starred = true AND is_archived = false;

-- Team inboxes
CREATE TABLE team_inboxes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    description TEXT,
    assignment_mode VARCHAR(20) NOT NULL DEFAULT 'manual'
        CHECK (assignment_mode IN ('manual', 'round_robin')),
    visibility VARCHAR(20) NOT NULL DEFAULT 'open'
        CHECK (visibility IN ('open', 'private')),
    next_assignee_index INTEGER NOT NULL DEFAULT 0,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Team inbox members (composite PK)
CREATE TABLE team_inbox_members (
    team_inbox_id UUID NOT NULL REFERENCES team_inboxes(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL DEFAULT 'member'
        CHECK (role IN ('admin', 'member')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (team_inbox_id, user_id)
);

-- FK for inbox_messages.team_inbox_id (added after team_inboxes exists)
ALTER TABLE inbox_messages
    ADD CONSTRAINT fk_inbox_messages_team_inbox
    FOREIGN KEY (team_inbox_id) REFERENCES team_inboxes(id) ON DELETE SET NULL;

-- Routing rules for automatic inbox message routing
CREATE TABLE routing_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    channel VARCHAR(20),
    conditions JSONB NOT NULL,
    actions JSONB NOT NULL,
    priority INTEGER NOT NULL DEFAULT 100,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Active routing rules ordered by priority
CREATE INDEX idx_routing_rules_active ON routing_rules(priority ASC)
    WHERE is_active = true;
