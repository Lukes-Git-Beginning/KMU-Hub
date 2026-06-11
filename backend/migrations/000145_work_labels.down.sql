BEGIN;
DROP INDEX IF EXISTS idx_task_labels_label_id;
DROP INDEX IF EXISTS idx_task_labels_task_id;
DROP TABLE IF EXISTS task_labels;
DROP INDEX IF EXISTS idx_work_labels_tenant_name;
DROP TABLE IF EXISTS work_labels;
COMMIT;
