CREATE EXTENSION IF NOT EXISTS btree_gist;

-- Bookable resources (rooms, equipment, vehicles)
CREATE TABLE resources (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    resource_type VARCHAR(20) NOT NULL,  -- 'room', 'equipment', 'vehicle'
    capacity INTEGER,
    floor VARCHAR(50),
    location VARCHAR(255),
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_resources_type ON resources (resource_type) WHERE is_active = true;

-- Resource equipment tags (beamer, whiteboard, etc.)
CREATE TABLE resource_tags (
    resource_id UUID NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
    tag VARCHAR(50) NOT NULL,
    PRIMARY KEY (resource_id, tag)
);

CREATE INDEX idx_resource_tags_tag ON resource_tags (tag);

-- Resource bookings with double-booking prevention
CREATE TABLE resource_bookings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resource_id UUID NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
    event_id UUID NOT NULL REFERENCES calendar_events(id) ON DELETE CASCADE,
    booked_by UUID NOT NULL REFERENCES users(id),
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    cancelled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Database-level double-booking prevention
    EXCLUDE USING GIST (
        resource_id WITH =,
        tstzrange(start_time, end_time) WITH &&
    ) WHERE (cancelled_at IS NULL)
);

CREATE INDEX idx_resource_bookings_resource ON resource_bookings (resource_id, start_time, end_time);
CREATE INDEX idx_resource_bookings_event ON resource_bookings (event_id);

-- Add FK constraints for resource_id columns deferred from 000033
ALTER TABLE calendar_events
    ADD CONSTRAINT fk_events_resource
    FOREIGN KEY (resource_id) REFERENCES resources(id) ON DELETE SET NULL;

ALTER TABLE event_exceptions
    ADD CONSTRAINT fk_event_exceptions_resource
    FOREIGN KEY (resource_id) REFERENCES resources(id) ON DELETE SET NULL;
