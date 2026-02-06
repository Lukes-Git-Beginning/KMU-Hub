-- Revert thread support
DROP INDEX IF EXISTS idx_messages_parent;
ALTER TABLE messages DROP COLUMN IF EXISTS reply_count;
ALTER TABLE messages DROP COLUMN IF EXISTS parent_message_id;

-- Revert DM support
DROP INDEX IF EXISTS idx_channels_dm_user2;
DROP INDEX IF EXISTS idx_channels_dm_user1;
DROP INDEX IF EXISTS idx_channels_dm_pair;
ALTER TABLE channels DROP CONSTRAINT IF EXISTS chk_dm_users_set;
ALTER TABLE channels DROP CONSTRAINT IF EXISTS chk_dm_user_order;
ALTER TABLE channels DROP COLUMN IF EXISTS dm_user2;
ALTER TABLE channels DROP COLUMN IF EXISTS dm_user1;
