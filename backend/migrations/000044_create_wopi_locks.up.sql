-- WOPI lock table for collaborative document editing

CREATE TABLE wopi_locks (
    file_id UUID PRIMARY KEY REFERENCES document_files(id) ON DELETE CASCADE,
    lock_id VARCHAR(1024) NOT NULL,
    locked_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '30 minutes'
);
