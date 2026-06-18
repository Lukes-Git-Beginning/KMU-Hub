-- Migration 000182: extend hr_work_time_entries with category, location, soft-delete

ALTER TABLE hr_work_time_entries
    ADD COLUMN IF NOT EXISTS category_id UUID NULL REFERENCES hr_time_categories(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS location_lat NUMERIC(9,6) NULL,
    ADD COLUMN IF NOT EXISTS location_lng NUMERIC(9,6) NULL,
    ADD COLUMN IF NOT EXISTS location_address TEXT NULL,
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ NULL;
