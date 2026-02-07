-- Notifications table: stores all user notifications with grouping support
CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type_key VARCHAR(100) NOT NULL,
    module_id VARCHAR(50) NOT NULL,
    priority VARCHAR(10) NOT NULL DEFAULT 'normal'
        CHECK (priority IN ('urgent', 'normal', 'low')),
    actor_id UUID,
    resource_id VARCHAR(255),
    title VARCHAR(500) NOT NULL,
    body TEXT,
    deep_link VARCHAR(1000),
    group_key VARCHAR(255),
    group_count INTEGER NOT NULL DEFAULT 1,
    is_read BOOLEAN NOT NULL DEFAULT false,
    read_at TIMESTAMPTZ,
    delivered_desktop BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Partial index for fast unread count queries
CREATE INDEX idx_notifications_user_unread ON notifications(user_id, created_at DESC)
    WHERE is_read = false;

-- Index for listing notifications by user
CREATE INDEX idx_notifications_user_created ON notifications(user_id, created_at DESC);

-- Index for smart grouping lookups
CREATE INDEX idx_notifications_group_key ON notifications(user_id, group_key, created_at DESC);

-- Events durability table: ensures no events are lost during service restarts
CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type_key VARCHAR(100) NOT NULL,
    module_id VARCHAR(50) NOT NULL,
    priority VARCHAR(10) NOT NULL DEFAULT 'normal',
    actor_id UUID,
    resource_id VARCHAR(255),
    payload JSONB,
    processed BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for catch-up processing of unprocessed events
CREATE INDEX idx_events_unprocessed ON events(created_at ASC) WHERE processed = false;
