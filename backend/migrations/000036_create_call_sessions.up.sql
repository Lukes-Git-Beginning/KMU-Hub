-- Call sessions and participants for video/voice calls
CREATE TABLE call_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    call_type TEXT NOT NULL CHECK (call_type IN ('one_to_one', 'group')),
    status TEXT NOT NULL DEFAULT 'ringing' CHECK (status IN ('ringing', 'active', 'ended')),
    room_name TEXT NOT NULL UNIQUE,
    initiator_id UUID NOT NULL REFERENCES users(id),
    channel_id UUID REFERENCES channels(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    duration_seconds INT
);

CREATE INDEX idx_call_sessions_initiator ON call_sessions(initiator_id);
CREATE INDEX idx_call_sessions_status ON call_sessions(status) WHERE status != 'ended';
CREATE INDEX idx_call_sessions_channel ON call_sessions(channel_id) WHERE channel_id IS NOT NULL;

CREATE TABLE call_participants (
    call_id UUID NOT NULL REFERENCES call_sessions(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    left_at TIMESTAMPTZ,
    has_video BOOLEAN NOT NULL DEFAULT true,
    has_audio BOOLEAN NOT NULL DEFAULT true,
    PRIMARY KEY (call_id, user_id)
);
