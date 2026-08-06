-- ============================================================================
-- start_date for work tasks (BACKLOG G "g-work-start-date").
--
-- Carried over from loop run 4 (not reached there). Prerequisite for the
-- planned Gantt view: a task needs both ends of its schedule, not just
-- due_date. NULL is a valid state -- an unscheduled task has neither date.
-- ============================================================================

ALTER TABLE tasks
    ADD COLUMN start_date TIMESTAMPTZ NULL,
    -- Defense in depth: the service already rejects start_date > due_date
    -- (task.ErrInvalidDateRange), this CHECK holds the same invariant at the
    -- data layer in case a future write path bypasses the service. Either
    -- side may be NULL (a task can have just one of the two dates set).
    ADD CONSTRAINT chk_tasks_start_before_due
        CHECK (start_date IS NULL OR due_date IS NULL OR start_date <= due_date);

-- Backfill: existing rows get NULL (no start date known), which is a valid,
-- already-handled state throughout the read paths.

-- tasks already carries tenant_id NOT NULL and a tenant_isolation policy;
-- adding a column needs no RLS change.
