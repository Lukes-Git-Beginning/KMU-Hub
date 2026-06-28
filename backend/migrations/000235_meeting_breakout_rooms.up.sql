-- Migration 000235: Breakout rooms for in-progress meetings (Wave 6A).
--
-- meeting_breakout_rooms: tracks active breakout rooms spawned from a main meeting.
-- meeting_breakout_assignments: which user is assigned to which breakout room.

CREATE TABLE meeting_breakout_rooms (
    id          UUID        NOT NULL DEFAULT gen_random_uuid(),
    tenant_id   UUID        NOT NULL,
    meeting_id  UUID        NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    room_name   TEXT        NOT NULL,
    label       TEXT        NOT NULL,
    sort_index  INT         NOT NULL DEFAULT 0,
    status      TEXT        NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
    created_by  UUID        NOT NULL,
    closed_at   TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT meeting_breakout_rooms_pkey PRIMARY KEY (id),
    CONSTRAINT meeting_breakout_rooms_room_name_unique UNIQUE (room_name)
);

CREATE INDEX meeting_breakout_rooms_meeting_idx ON meeting_breakout_rooms (meeting_id, tenant_id);

CALL enable_tenant_rls('meeting_breakout_rooms');

CREATE TABLE meeting_breakout_assignments (
    id               UUID        NOT NULL DEFAULT gen_random_uuid(),
    tenant_id        UUID        NOT NULL,
    meeting_id       UUID        NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    breakout_room_id UUID        NOT NULL REFERENCES meeting_breakout_rooms(id) ON DELETE CASCADE,
    user_id          UUID        NOT NULL,
    assigned_by      UUID        NOT NULL,
    assigned_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT meeting_breakout_assignments_pkey PRIMARY KEY (id),
    CONSTRAINT meeting_breakout_assignments_unique UNIQUE (meeting_id, user_id)
);

CREATE INDEX meeting_breakout_assignments_meeting_user_idx ON meeting_breakout_assignments (meeting_id, user_id, tenant_id);

CALL enable_tenant_rls('meeting_breakout_assignments');