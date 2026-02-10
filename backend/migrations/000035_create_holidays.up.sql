-- Public holidays (DACH countries with subdivision support)
CREATE TABLE public_holidays (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    date DATE NOT NULL,
    name VARCHAR(255) NOT NULL,
    local_name VARCHAR(255) NOT NULL,
    country_code CHAR(2) NOT NULL,
    is_global BOOLEAN NOT NULL DEFAULT false,
    subdivision_codes TEXT[],
    holiday_type VARCHAR(20) NOT NULL DEFAULT 'public',
    year INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (date, country_code, name)
);

CREATE INDEX idx_holidays_date_range ON public_holidays (date, country_code);
CREATE INDEX idx_holidays_year_country ON public_holidays (year, country_code);
CREATE INDEX idx_holidays_subdivision ON public_holidays USING GIN (subdivision_codes);

-- Seed calendar notification event types
INSERT INTO event_types (module_id, event_key, display_name, description, default_priority, default_in_app, default_desktop_push, default_sound) VALUES
    ('calendar', 'calendar.event.created', 'New calendar event', 'A new event was created on a shared calendar', 'normal', true, false, ''),
    ('calendar', 'calendar.event.updated', 'Calendar event updated', 'An event you attend was updated', 'normal', true, false, ''),
    ('calendar', 'calendar.event.cancelled', 'Calendar event cancelled', 'An event you attend was cancelled', 'normal', true, true, 'default'),
    ('calendar', 'calendar.event.invited', 'Event invitation', 'You were invited to a calendar event', 'normal', true, true, 'default'),
    ('calendar', 'calendar.event.rsvp_changed', 'RSVP response', 'An attendee responded to your event invitation', 'low', true, false, ''),
    ('calendar', 'calendar.reminder.fired', 'Event reminder', 'Reminder for an upcoming event', 'normal', true, true, 'default');

-- Seed RBAC permissions for calendar and resources
INSERT INTO permissions (name, resource, action) VALUES
    ('calendars:read', 'calendars', 'read'),
    ('calendars:write', 'calendars', 'write'),
    ('calendars:admin', 'calendars', 'admin'),
    ('resources:read', 'resources', 'read'),
    ('resources:write', 'resources', 'write'),
    ('resources:admin', 'resources', 'admin')
ON CONFLICT (name) DO NOTHING;

-- Grant calendar and resource permissions to all existing roles
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE p.resource IN ('calendars', 'resources')
ON CONFLICT (role_id, permission_id) DO NOTHING;
