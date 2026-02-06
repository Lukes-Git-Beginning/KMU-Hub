-- =============================================================================
-- DM Support: canonical user pair columns + unique constraint
-- =============================================================================

-- Add dm_user1 and dm_user2 columns (only populated for DM channels)
ALTER TABLE channels ADD COLUMN dm_user1 UUID REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE channels ADD COLUMN dm_user2 UUID REFERENCES users(id) ON DELETE CASCADE;

-- Ensure dm_user1 < dm_user2 for canonical ordering
ALTER TABLE channels ADD CONSTRAINT chk_dm_user_order
    CHECK (dm_user1 IS NULL OR dm_user1 < dm_user2);

-- Ensure DM channels always have both users set
ALTER TABLE channels ADD CONSTRAINT chk_dm_users_set
    CHECK (
        (is_dm = FALSE AND dm_user1 IS NULL AND dm_user2 IS NULL)
        OR (is_dm = TRUE AND dm_user1 IS NOT NULL AND dm_user2 IS NOT NULL)
    );

-- Unique constraint: only one DM per user pair
CREATE UNIQUE INDEX idx_channels_dm_pair ON channels (dm_user1, dm_user2)
    WHERE is_dm = TRUE;

-- Lookup indexes for finding DMs by user
CREATE INDEX idx_channels_dm_user1 ON channels (dm_user1) WHERE is_dm = TRUE;
CREATE INDEX idx_channels_dm_user2 ON channels (dm_user2) WHERE is_dm = TRUE;

-- =============================================================================
-- Thread Support: parent_message_id + reply_count
-- =============================================================================

-- Add parent_message_id for thread replies
ALTER TABLE messages ADD COLUMN parent_message_id UUID REFERENCES messages(id) ON DELETE SET NULL;

-- Add reply_count (denormalized for efficient thread preview display)
ALTER TABLE messages ADD COLUMN reply_count INT NOT NULL DEFAULT 0;

-- Index for fetching thread replies (oldest first)
CREATE INDEX idx_messages_parent ON messages (parent_message_id, created_at ASC)
    WHERE parent_message_id IS NOT NULL;
