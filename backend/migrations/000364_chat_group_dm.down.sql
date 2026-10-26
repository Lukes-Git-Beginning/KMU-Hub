-- Group DMs cannot exist under the old constraint, so they go first. Their
-- memberships and messages follow via ON DELETE CASCADE. call_sessions has no
-- cascade on channel_id: a group DM with call history makes this down fail
-- loudly instead of dropping that history.
DELETE FROM channels WHERE dm_group_key IS NOT NULL;

DROP INDEX IF EXISTS idx_channels_dm_group_key;

ALTER TABLE channels DROP CONSTRAINT chk_dm_users_set;
ALTER TABLE channels ADD CONSTRAINT chk_dm_users_set
    CHECK (
        (is_dm = FALSE AND dm_user1 IS NULL AND dm_user2 IS NULL)
        OR (is_dm = TRUE AND dm_user1 IS NOT NULL AND dm_user2 IS NOT NULL)
    );

ALTER TABLE channels DROP COLUMN dm_group_key;
