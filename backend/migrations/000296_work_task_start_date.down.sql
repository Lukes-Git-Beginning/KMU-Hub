ALTER TABLE tasks
    DROP CONSTRAINT chk_tasks_start_before_due,
    DROP COLUMN start_date;
