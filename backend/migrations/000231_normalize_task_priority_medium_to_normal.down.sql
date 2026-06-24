-- Reverse the priority normalization: 'normal' -> 'medium'. Mirrors the up
-- migration step order so the backfill never runs against an incompatible CHECK.

-- 1. Drop the new CHECK + DEFAULT (the new CHECK forbids 'medium').
ALTER TABLE tasks ALTER COLUMN priority DROP DEFAULT;
ALTER TABLE tasks DROP CONSTRAINT tasks_priority_check;

-- 2. Backfill existing rows back to the legacy value.
UPDATE tasks SET priority = 'medium' WHERE priority = 'normal';

-- 3. Re-add the legacy DEFAULT + CHECK.
ALTER TABLE tasks ALTER COLUMN priority SET DEFAULT 'medium';
ALTER TABLE tasks
    ADD CONSTRAINT tasks_priority_check
    CHECK (priority IN ('urgent', 'high', 'medium', 'low'));

-- 4. Revert the seeded industry-template workflow_rules. Scoped to the same
--    Migr.058 slugs so unrelated templates that legitimately use 'normal' are
--    untouched.
UPDATE industry_templates
SET workflow_rules = regexp_replace(
        workflow_rules::text,
        '"priority"\s*:\s*"normal"',
        '"priority": "medium"',
        'g'
    )::jsonb
WHERE slug IN ('handwerk', 'beratung', 'handel')
  AND workflow_rules::text ~ '"priority"\s*:\s*"normal"';
