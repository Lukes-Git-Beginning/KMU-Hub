-- Group DMs: a DM channel with more than two members.
--
-- A 1:1 DM keeps its canonical pair in dm_user1/dm_user2. A group DM leaves
-- both NULL and carries dm_group_key instead: the hex SHA-256 of the sorted,
-- comma-joined member UUIDs (computed in internal/chat/channel). The unique
-- index makes the same member set map to exactly one channel per tenant.
-- channels already has tenant_id NOT NULL and its RLS policy (000253).

ALTER TABLE channels ADD COLUMN dm_group_key TEXT;

ALTER TABLE channels DROP CONSTRAINT chk_dm_users_set;
ALTER TABLE channels ADD CONSTRAINT chk_dm_users_set
    CHECK (
        (is_dm = FALSE AND dm_user1 IS NULL AND dm_user2 IS NULL AND dm_group_key IS NULL)
        OR (is_dm = TRUE AND dm_user1 IS NOT NULL AND dm_user2 IS NOT NULL AND dm_group_key IS NULL)
        OR (is_dm = TRUE AND dm_user1 IS NULL AND dm_user2 IS NULL AND dm_group_key IS NOT NULL)
    );

CREATE UNIQUE INDEX idx_channels_dm_group_key ON channels (tenant_id, dm_group_key)
    WHERE dm_group_key IS NOT NULL;
