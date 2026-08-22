-- Revert 000320: point task_custom_field_values.field_id back at custom_field_definitions.
-- Rows referencing work_custom_field_definitions cannot satisfy the old constraint, so they
-- are removed first — same reasoning as in the up migration, mirrored.

BEGIN;

DELETE FROM task_custom_field_values tcfv
WHERE NOT EXISTS (
    SELECT 1 FROM custom_field_definitions cfd WHERE cfd.id = tcfv.field_id
);

ALTER TABLE task_custom_field_values
    DROP CONSTRAINT IF EXISTS task_custom_field_values_field_id_fkey;

ALTER TABLE task_custom_field_values
    ADD CONSTRAINT task_custom_field_values_field_id_fkey
    FOREIGN KEY (field_id) REFERENCES custom_field_definitions(id) ON DELETE CASCADE;

COMMIT;
