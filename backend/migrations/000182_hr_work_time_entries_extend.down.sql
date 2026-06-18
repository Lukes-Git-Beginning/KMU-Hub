ALTER TABLE hr_work_time_entries
    DROP COLUMN IF EXISTS category_id,
    DROP COLUMN IF EXISTS location_lat,
    DROP COLUMN IF EXISTS location_lng,
    DROP COLUMN IF EXISTS location_address,
    DROP COLUMN IF EXISTS deleted_at;
