-- feat-retention-handler-guest-sessions: the existing idx_guest_sessions_cleanup
-- covers expires_at WHERE is_active = true, which is the wrong clock for retention
-- (a guest session past its 90-day support relevance can still be marked active)
-- and excludes inactive rows entirely. The retention handler filters on
-- tenant_id + last_activity_at across all rows, which neither existing index serves.
CREATE INDEX idx_guest_sessions_tenant_last_activity ON guest_sessions(tenant_id, last_activity_at);
