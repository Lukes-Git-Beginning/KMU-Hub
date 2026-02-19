-- Meetings, attendees, notes, action items, recordings, and consents
CREATE TABLE meetings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    description TEXT,
    agenda TEXT,
    organizer_id UUID NOT NULL REFERENCES users(id),
    status TEXT NOT NULL DEFAULT 'scheduled' CHECK (status IN ('scheduled', 'in_progress', 'completed', 'cancelled')),
    scheduled_start TIMESTAMPTZ NOT NULL,
    scheduled_end TIMESTAMPTZ NOT NULL,
    actual_start TIMESTAMPTZ,
    actual_end TIMESTAMPTZ,
    room_name TEXT,
    calendar_event_id UUID,
    recurring_meeting_id UUID REFERENCES meetings(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_meetings_organizer ON meetings(organizer_id);
CREATE INDEX idx_meetings_status ON meetings(status);
CREATE INDEX idx_meetings_scheduled ON meetings(scheduled_start);
CREATE INDEX idx_meetings_calendar_event ON meetings(calendar_event_id) WHERE calendar_event_id IS NOT NULL;
CREATE INDEX idx_meetings_recurring ON meetings(recurring_meeting_id) WHERE recurring_meeting_id IS NOT NULL;

CREATE TABLE meeting_attendees (
    meeting_id UUID NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    rsvp_status TEXT NOT NULL DEFAULT 'pending' CHECK (rsvp_status IN ('pending', 'accepted', 'declined', 'tentative')),
    PRIMARY KEY (meeting_id, user_id)
);

CREATE TABLE meeting_notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    meeting_id UUID NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    author_id UUID NOT NULL REFERENCES users(id),
    content TEXT NOT NULL DEFAULT '',
    is_private BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_meeting_notes_meeting ON meeting_notes(meeting_id);

CREATE TABLE meeting_action_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    meeting_id UUID NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    description TEXT NOT NULL,
    assignee_id UUID REFERENCES users(id),
    is_completed BOOLEAN NOT NULL DEFAULT false,
    task_id UUID,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_meeting_action_items_meeting ON meeting_action_items(meeting_id);

CREATE TABLE recordings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    call_id UUID REFERENCES call_sessions(id),
    meeting_id UUID REFERENCES meetings(id),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'processing', 'completed', 'failed', 'deleted')),
    egress_id TEXT,
    file_url TEXT,
    file_size_bytes BIGINT,
    duration_seconds INT,
    retention_expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (call_id IS NOT NULL OR meeting_id IS NOT NULL)
);

CREATE INDEX idx_recordings_call ON recordings(call_id) WHERE call_id IS NOT NULL;
CREATE INDEX idx_recordings_meeting ON recordings(meeting_id) WHERE meeting_id IS NOT NULL;
CREATE INDEX idx_recordings_retention ON recordings(retention_expires_at) WHERE status = 'completed';

CREATE TABLE recording_consents (
    recording_id UUID NOT NULL REFERENCES recordings(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    consented BOOLEAN NOT NULL,
    responded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (recording_id, user_id)
);
