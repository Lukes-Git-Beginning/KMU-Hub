-- Event types: module-agnostic registry of all notification event types
CREATE TABLE event_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    module_id VARCHAR(50) NOT NULL,
    event_key VARCHAR(100) NOT NULL UNIQUE,
    display_name VARCHAR(200) NOT NULL,
    description TEXT,
    default_priority VARCHAR(10) NOT NULL DEFAULT 'normal'
        CHECK (default_priority IN ('urgent', 'normal', 'low')),
    default_in_app BOOLEAN NOT NULL DEFAULT true,
    default_desktop_push BOOLEAN NOT NULL DEFAULT true,
    default_sound VARCHAR(50) DEFAULT 'default',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_event_types_module_id ON event_types(module_id);

-- Seed initial event types
INSERT INTO event_types (module_id, event_key, display_name, description, default_priority, default_in_app, default_desktop_push, default_sound) VALUES
    ('chat', 'chat.mention', 'Mentioned in message', 'Someone mentioned you in a chat message', 'normal', true, true, 'default'),
    ('chat', 'chat.dm.new', 'New direct message', 'You received a new direct message', 'normal', true, true, 'default'),
    ('chat', 'chat.channel.message', 'New channel message', 'New message posted in a channel', 'low', true, false, ''),
    ('crm', 'crm.deal.stage_changed', 'Deal stage changed', 'A deal you own moved to a new stage', 'normal', true, true, 'default'),
    ('crm', 'crm.deal.assigned', 'Deal assigned to you', 'A deal has been assigned to you', 'normal', true, true, 'default'),
    ('crm', 'crm.contact.assigned', 'Contact assigned to you', 'A contact has been assigned to you', 'normal', true, true, 'default'),
    ('system', 'system.alert', 'System alert', 'Important system notification', 'urgent', true, true, 'urgent');
