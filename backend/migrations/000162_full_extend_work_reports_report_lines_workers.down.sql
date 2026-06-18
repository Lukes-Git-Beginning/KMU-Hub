-- Migration 000162 DOWN

BEGIN;

DROP TABLE IF EXISTS report_workers;

ALTER TABLE report_lines
    DROP COLUMN IF EXISTS category,
    DROP COLUMN IF EXISTS article;

ALTER TABLE work_reports
    DROP COLUMN IF EXISTS weather,
    DROP COLUMN IF EXISTS temperature,
    DROP COLUMN IF EXISTS work_start,
    DROP COLUMN IF EXISTS work_end,
    DROP COLUMN IF EXISTS break_minutes,
    DROP COLUMN IF EXISTS project_name;

COMMIT;
