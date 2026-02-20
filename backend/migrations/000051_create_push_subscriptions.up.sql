-- Push subscriptions for WebDAV-Push notification delivery.
-- Clients that support push (primarily macOS Calendar) register a callback URL
-- to receive change notifications instead of polling.

CREATE TABLE caldav_push_subscriptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    collection_type VARCHAR(20) NOT NULL, -- 'calendar', 'addressbook'
    collection_id UUID NOT NULL,
    push_url TEXT NOT NULL,
    push_topic TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Find subscriptions for a collection (used when notifying changes)
CREATE INDEX idx_push_subs_collection ON caldav_push_subscriptions (collection_type, collection_id);

-- Find subscriptions for a user (cleanup on user disable)
CREATE INDEX idx_push_subs_user ON caldav_push_subscriptions (user_id);

-- Expired subscription cleanup queries
CREATE INDEX idx_push_subs_expiry ON caldav_push_subscriptions (expires_at);

-- Unique constraint: one subscription per user+collection+push_url (upsert target)
CREATE UNIQUE INDEX idx_push_subs_upsert ON caldav_push_subscriptions (user_id, collection_type, collection_id, push_url);
