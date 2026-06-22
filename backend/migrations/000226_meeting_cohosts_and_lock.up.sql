-- Migration 000226: Co-hosts table and meeting lock column for server-authoritative
-- host controls (Wave 3 of the meeting-parity roadmap).
--
-- meeting_cohosts: tracks which users the organizer has granted co-host rights to.
-- meetings.locked:  when true, non-host/non-cohost participants cannot join.

CREATE TABLE meeting_cohosts (
    id          UUID        NOT NULL DEFAULT gen_random_uuid(),
    tenant_id   UUID        NOT NULL,
    meeting_id  UUID        NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    user_id     UUID        NOT NULL,
    granted_by  UUID        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT meeting_cohosts_pkey PRIMARY KEY (id),
    CONSTRAINT meeting_cohosts_unique UNIQUE (meeting_id, user_id)
);

CREATE INDEX meeting_cohosts_meeting_idx ON meeting_cohosts (meeting_id);

CALL enable_tenant_rls('meeting_cohosts');

ALTER TABLE meetings ADD COLUMN locked BOOLEAN NOT NULL DEFAULT false;
