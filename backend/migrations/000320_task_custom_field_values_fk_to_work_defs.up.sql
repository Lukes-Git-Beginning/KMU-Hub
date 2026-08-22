-- Migration 000320: repoint task_custom_field_values.field_id to work_custom_field_definitions
--
-- task_custom_field_values was created in 000026 with
--   field_id UUID NOT NULL REFERENCES custom_field_definitions(id)
-- i.e. the generic CRM definition table from 000005, whose valid_entity_type CHECK only
-- admits 'contact', 'company', 'deal', 'activity' — 'task' is not a valid entity type there.
-- Migration 000146 later introduced work_custom_field_definitions as the task-side table,
-- exposed via /api/v1/work/custom-fields. Every id the work custom-field API hands out comes
-- from work_custom_field_definitions, and the task write path
-- (work/task/postgres_repository.go SetCustomFieldValues) inserts exactly that id into
-- task_custom_field_values.field_id — which the old FK rejects with a foreign_key_violation.
-- Setting a custom field value on a task has therefore been impossible since 000146.
--
-- Existing rows: any row still present can only carry a CRM definition id (the work path
-- could never insert). Such rows are orphans of a table that was never wired up for tasks,
-- so they are deleted explicitly rather than blocking the new constraint silently.

BEGIN;

DELETE FROM task_custom_field_values tcfv
WHERE NOT EXISTS (
    SELECT 1 FROM work_custom_field_definitions wcfd WHERE wcfd.id = tcfv.field_id
);

ALTER TABLE task_custom_field_values
    DROP CONSTRAINT IF EXISTS task_custom_field_values_field_id_fkey;

ALTER TABLE task_custom_field_values
    ADD CONSTRAINT task_custom_field_values_field_id_fkey
    FOREIGN KEY (field_id) REFERENCES work_custom_field_definitions(id) ON DELETE CASCADE;

COMMIT;
