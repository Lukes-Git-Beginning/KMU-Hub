-- Normalize task priority 'medium' -> 'normal' to match the FE/API contract
-- (low | normal | high | urgent). The automation layer passes a task's priority
-- through verbatim (internal/automation/action/work_actions.go), so the column
-- CHECK, the DEFAULT, existing rows AND the seeded industry-template
-- workflow_rules JSONB must all move together — otherwise template-driven task
-- creation would fail the new CHECK.

-- 1. Drop the old CHECK + DEFAULT first (the old CHECK forbids 'normal', so the
--    backfill below would violate it while the constraint is still active).
ALTER TABLE tasks ALTER COLUMN priority DROP DEFAULT;
ALTER TABLE tasks DROP CONSTRAINT tasks_priority_check;

-- 2. Backfill existing rows (no priority CHECK active at this point).
UPDATE tasks SET priority = 'normal' WHERE priority = 'medium';

-- 3. Re-add the DEFAULT + CHECK with the normalized value set.
ALTER TABLE tasks ALTER COLUMN priority SET DEFAULT 'normal';
ALTER TABLE tasks
    ADD CONSTRAINT tasks_priority_check
    CHECK (priority IN ('urgent', 'high', 'normal', 'low'));

-- 4. Normalize the seeded industry-template workflow_rules (Migr.058) so that
--    template instantiation produces a CHECK-valid priority. Scoped to the slugs
--    seeded by Migr.058; whitespace-insensitive match on the create_task config.
UPDATE industry_templates
SET workflow_rules = regexp_replace(
        workflow_rules::text,
        '"priority"\s*:\s*"medium"',
        '"priority": "normal"',
        'g'
    )::jsonb
WHERE slug IN ('handwerk', 'beratung', 'handel')
  AND workflow_rules::text ~ '"priority"\s*:\s*"medium"';
