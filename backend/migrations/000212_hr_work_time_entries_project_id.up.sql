ALTER TABLE hr_work_time_entries
    ADD COLUMN IF NOT EXISTS project_id UUID NULL
        REFERENCES hr_time_projects(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_hr_work_time_entries_project_id
    ON hr_work_time_entries(project_id)
    WHERE project_id IS NOT NULL;
