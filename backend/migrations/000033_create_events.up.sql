-- Calendar events
CREATE TABLE calendar_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    calendar_id UUID NOT NULL REFERENCES calendars(id) ON DELETE CASCADE,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    location TEXT,
    resource_id UUID,  -- FK added in 000034 after resources table exists
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    is_all_day BOOLEAN NOT NULL DEFAULT false,
    timezone VARCHAR(50) NOT NULL DEFAULT 'Europe/Berlin',
    -- Recurrence (RFC 5545)
    rrule TEXT,
    recurrence_end TIMESTAMPTZ,
    -- Video call
    has_video_call BOOLEAN NOT NULL DEFAULT false,
    livekit_room_name VARCHAR(100),
    -- Category
    category_id UUID REFERENCES event_categories(id) ON DELETE SET NULL,
    -- Metadata
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_events_calendar ON calendar_events (calendar_id);
CREATE INDEX idx_events_time_range ON calendar_events (start_time, end_time);
CREATE INDEX idx_events_recurring ON calendar_events (calendar_id) WHERE rrule IS NOT NULL;
CREATE INDEX idx_events_resource ON calendar_events (resource_id) WHERE resource_id IS NOT NULL;
CREATE INDEX idx_events_created_by ON calendar_events (created_by);

-- Event attendees (RSVP)
CREATE TABLE event_attendees (
    event_id UUID NOT NULL REFERENCES calendar_events(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rsvp_status VARCHAR(10) NOT NULL DEFAULT 'pending',
    responded_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (event_id, user_id)
);

CREATE INDEX idx_event_attendees_user ON event_attendees (user_id);

-- Event exceptions (for recurring event modifications/cancellations)
CREATE TABLE event_exceptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id UUID NOT NULL REFERENCES calendar_events(id) ON DELETE CASCADE,
    original_date DATE NOT NULL,
    is_cancelled BOOLEAN NOT NULL DEFAULT false,
    -- Override fields (NULL = use parent event values)
    title VARCHAR(500),
    description TEXT,
    location TEXT,
    start_time TIMESTAMPTZ,
    end_time TIMESTAMPTZ,
    resource_id UUID,  -- FK added in 000034 after resources table exists
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (event_id, original_date)
);

CREATE INDEX idx_event_exceptions_event ON event_exceptions (event_id, original_date);

-- Event reminders
CREATE TABLE event_reminders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id UUID NOT NULL REFERENCES calendar_events(id) ON DELETE CASCADE,
    minutes_before INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_event_reminders_event ON event_reminders (event_id);

-- User calendar preferences (global defaults)
CREATE TABLE user_calendar_preferences (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    default_view VARCHAR(10) NOT NULL DEFAULT 'week',
    week_days INTEGER NOT NULL DEFAULT 5,
    default_reminder_minutes INTEGER NOT NULL DEFAULT 15,
    default_allday_reminder_minutes INTEGER NOT NULL DEFAULT 1080,
    subdivision_code VARCHAR(10),
    show_task_deadlines BOOLEAN NOT NULL DEFAULT true,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
