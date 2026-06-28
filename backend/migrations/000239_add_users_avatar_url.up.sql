-- Migration 000239 — add avatar_url to users for self-service profile avatars.
-- Stores the presigned-upload object key ({tenant_id}/avatar/{uuid}{ext}); the
-- client resolves it to a viewable URL via /api/v1/files/presign-download.
-- users is a system-global table with a custom RLS policy (user_isolation,
-- migration 000120) — a new column needs no additional policy.
ALTER TABLE users ADD COLUMN avatar_url TEXT NOT NULL DEFAULT '';
