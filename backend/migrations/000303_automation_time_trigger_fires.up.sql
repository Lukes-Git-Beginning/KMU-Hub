-- ============================================================================
-- Migration 000303: automation_time_trigger_fires -- per-entity dedup for the
-- time-based automation poller.
--
-- 000284 gave trigger.TimeTriggerPoller an optimistic-concurrency claim on
-- automations.last_polled_at. That claim answers "may THIS instance handle
-- this automation on THIS tick" -- it does not answer "has this automation
-- already fired for this invoice". Without the second answer the poller
-- re-fires every due automation on every five-minute tick forever: an invoice
-- that went overdue once would create a dunning 288 times a day.
--
-- The unique key is (automation_id, entity_key), and entity_key's granularity
-- is the resolver's decision, not this table's:
--   * biz.invoice.overdue uses "invoice:<id>:<yyyy-mm-dd>" -- once per invoice
--     per day. Once-per-invoice-ever would be wrong: the shipped template
--     "invoice-overdue-dunning" fires on invoice.days_overdue >= 14, so a
--     single fire on day 1 (condition false) would consume the only chance
--     that invoice ever had and the dunning would never be created.
--   * calendar.event.upcoming uses "calendar_event:<id>" -- an event starts
--     once, so a reminder that fires twice is a bug.
--
-- INSERT ... ON CONFLICT DO NOTHING makes the claim atomic across instances,
-- the same shape as the last_polled_at claim one level up: whoever's INSERT
-- affects a row owns the fire, everyone else observes 0 rows and skips.
-- ============================================================================

BEGIN;

CREATE TABLE automation_time_trigger_fires (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID        NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    automation_id UUID        NOT NULL REFERENCES automations (id) ON DELETE CASCADE,
    entity_key    TEXT        NOT NULL,
    fired_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (automation_id, entity_key)
);

-- Retention scan only. The claim itself rides the UNIQUE constraint's index.
-- lean: no retention job yet -- the table grows by one row per real fire, which
-- for a per-day-per-overdue-invoice key is single digits per tenant per day.
-- Add a cleanup alongside ExecutionRepository.CleanupOldExecutions when a
-- tenant's row count here passes six figures.
CREATE INDEX idx_automation_time_trigger_fires_age
    ON automation_time_trigger_fires (tenant_id, fired_at);

CALL enable_tenant_rls('automation_time_trigger_fires');

COMMIT;
