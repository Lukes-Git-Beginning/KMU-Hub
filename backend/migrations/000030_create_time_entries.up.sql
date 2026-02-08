-- Time entries for task time tracking (PM-17)
CREATE TABLE time_entries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    started_at TIMESTAMPTZ NOT NULL,
    ended_at TIMESTAMPTZ,
    duration_seconds INTEGER,
    description TEXT,
    is_manual BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_time_entries_task ON time_entries (task_id);
CREATE INDEX idx_time_entries_user ON time_entries (user_id);
CREATE INDEX idx_time_entries_started ON time_entries (started_at DESC);
CREATE INDEX idx_time_entries_active ON time_entries (user_id, ended_at) WHERE ended_at IS NULL;
