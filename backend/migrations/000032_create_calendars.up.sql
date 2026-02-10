-- Calendars (personal and shared)
CREATE TABLE calendars (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    calendar_type VARCHAR(20) NOT NULL DEFAULT 'personal',  -- 'personal', 'shared', 'resource'
    color VARCHAR(7) NOT NULL DEFAULT '#4285F4',
    owner_id UUID NOT NULL REFERENCES users(id),
    is_default BOOLEAN NOT NULL DEFAULT false,
    timezone VARCHAR(50) NOT NULL DEFAULT 'Europe/Berlin',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_calendars_owner ON calendars (owner_id);
CREATE INDEX idx_calendars_type ON calendars (calendar_type);

-- Calendar membership / subscriptions
CREATE TABLE calendar_members (
    calendar_id UUID NOT NULL REFERENCES calendars(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    permission VARCHAR(10) NOT NULL DEFAULT 'view',  -- 'view', 'edit', 'admin'
    color_override VARCHAR(7),
    is_visible BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (calendar_id, user_id)
);

CREATE INDEX idx_calendar_members_user ON calendar_members (user_id);

-- User-defined event categories
CREATE TABLE event_categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    color VARCHAR(7) NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_event_categories_user ON event_categories (user_id);
CREATE UNIQUE INDEX idx_event_categories_name ON event_categories (user_id, LOWER(name));
