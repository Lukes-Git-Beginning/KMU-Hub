-- Notification preferences: event-type level or module-level delivery settings
CREATE TABLE notification_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type_key VARCHAR(100),
    module_id VARCHAR(50),
    in_app BOOLEAN NOT NULL DEFAULT true,
    desktop_push BOOLEAN NOT NULL DEFAULT true,
    sound VARCHAR(50) DEFAULT 'default',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- One preference per event type per user
CREATE UNIQUE INDEX idx_notification_preferences_event_type
    ON notification_preferences(user_id, event_type_key)
    WHERE event_type_key IS NOT NULL;

-- One module default per user per module
CREATE UNIQUE INDEX idx_notification_preferences_module_default
    ON notification_preferences(user_id, module_id)
    WHERE event_type_key IS NULL AND module_id IS NOT NULL;

CREATE INDEX idx_notification_preferences_user ON notification_preferences(user_id);

-- Resource muting: per-channel, per-pipeline, etc.
CREATE TABLE notification_mutes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    module_id VARCHAR(50) NOT NULL,
    resource_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, module_id, resource_id)
);

CREATE INDEX idx_notification_mutes_user ON notification_mutes(user_id, module_id);

-- Quiet hours configuration: scheduled DND with manual override
CREATE TABLE notification_quiet_hours (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    start_time TIME NOT NULL DEFAULT '18:00',
    end_time TIME NOT NULL DEFAULT '08:00',
    timezone VARCHAR(50) NOT NULL DEFAULT 'Europe/Berlin',
    days_of_week INTEGER[] NOT NULL DEFAULT '{1,2,3,4,5,6,7}',
    enabled BOOLEAN NOT NULL DEFAULT false,
    manual_dnd BOOLEAN NOT NULL DEFAULT false,
    manual_dnd_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id)
);
