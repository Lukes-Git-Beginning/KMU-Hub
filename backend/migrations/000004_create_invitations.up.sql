-- Create invitations table for user onboarding
CREATE TABLE invitations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL,
    token_hash VARCHAR(64) NOT NULL,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Prevent duplicate pending invitations for the same email
CREATE UNIQUE INDEX idx_invitations_email_pending ON invitations (email) WHERE accepted_at IS NULL;

-- Fast lookup by token hash
CREATE UNIQUE INDEX idx_invitations_token_hash ON invitations (token_hash);

-- Index for cleanup of expired invitations
CREATE INDEX idx_invitations_expires_at ON invitations (expires_at);
