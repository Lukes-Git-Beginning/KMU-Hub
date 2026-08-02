-- ============================================================================
-- `events` is the durability table behind the pg_notify event bus. It was the
-- last of the three partitioned log tables without RLS: `automation_executions`
-- and `dialer_call_events` both carry tenant_id + policy since 000242, `events`
-- was carried over from 000021 as a "system-level event bus" and kept that shape.
--
-- The premise no longer holds, and the check is not an opinion — it is what the
-- write path does. models.EventPayload carries TenantID (internal/models/event.go:42)
-- and every emitter fills it; notification/service.go:52 then builds the row and
-- drops the field on the floor. The tenant is known at write time and simply
-- discarded. Two consequences follow from that, and both are real today:
--
--   1. EventBus.ProcessBacklog rebuilds an EventPayload from the stored row
--      (event/bus.go:157) and cannot restore TenantID, so every event replayed
--      after a restart reaches preference.Evaluate with uuid.Nil.
--   2. actor_id and resource_id are tenant-bound by construction — an actor UUID
--      plus a resource id per module is exactly the correlation data RLS exists
--      to keep apart.
--
-- So option (A) from the backlog, not the allowlist entry: add the column, fill
-- it, protect it.
--
-- Backfill in three steps, most precise first:
--   1. actor_id -> users.tenant_id resolves every event that has an actor.
--   2. Rows without an actor (system-emitted) are unambiguous when the database
--      holds exactly one tenant — which is the production shape (single-tenant
--      data, multi-tenant code). The guard keeps this from guessing on a
--      multi-tenant database, where the mapping genuinely is not recoverable.
--   3. Whatever is still NULL was never attributable. Those rows are dropped
--      rather than parked under a sentinel: this is an ephemeral log pruned at
--      90 days by drop_old_partitions(), and an event without a tenant could not
--      have been processed anyway — the notification insert it would drive
--      violates the RLS policy on `notifications`.
--
-- No FK to tenants, unlike dialer_call_events. There it was free inside the
-- CREATE TABLE; adding one here would have to validate against every existing
-- partition, and create_monthly_partition() would carry it into each new one.
-- lean: index + NOT NULL + policy; add the FK if orphaned tenant references ever
-- show up in practice.
-- ============================================================================

ALTER TABLE events ADD COLUMN tenant_id UUID;

-- 1) Events with an actor: the actor's tenant is the event's tenant.
UPDATE events e
SET tenant_id = u.tenant_id
FROM users u
WHERE u.id = e.actor_id
  AND e.tenant_id IS NULL;

-- 2) Actor-less events: unambiguous only on a single-tenant database.
DO $$
DECLARE
    tenant_count integer;
    only_tenant  uuid;
    filled       integer;
BEGIN
    -- Two statements, not one: uuid has no min() aggregate.
    SELECT count(*) INTO tenant_count FROM tenants;

    IF tenant_count = 1 THEN
        SELECT id INTO only_tenant FROM tenants;
        UPDATE events SET tenant_id = only_tenant WHERE tenant_id IS NULL;
        GET DIAGNOSTICS filled = ROW_COUNT;
        RAISE NOTICE 'events: attributed % actor-less row(s) to the single tenant %', filled, only_tenant;
    ELSE
        RAISE NOTICE 'events: % tenants present, actor-less rows are not attributable and will be dropped', tenant_count;
    END IF;
END;
$$;

-- 3) Still unattributable: drop. See the header for why this loses nothing.
DELETE FROM events WHERE tenant_id IS NULL;

ALTER TABLE events ALTER COLUMN tenant_id SET NOT NULL;

-- Index on the parent propagates to every partition, including the ones
-- create_monthly_partition() adds later.
CREATE INDEX idx_events_tenant_id ON events(tenant_id);

-- RLS on the partitioned parent covers access through the parent, which is the
-- only way the application reaches this table — same shape as 000242 chose for
-- dialer_call_events.
CALL enable_tenant_rls('events');
