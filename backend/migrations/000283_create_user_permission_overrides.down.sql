-- Drops the table with its policies and indexes. Nothing to undo in the
-- permissions catalogue — the up migration seeded nothing there.
DROP TABLE IF EXISTS user_permission_overrides;
