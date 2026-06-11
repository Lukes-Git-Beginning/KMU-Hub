BEGIN;
SET LOCAL row_security = off;

ALTER TABLE work_reports
    DROP COLUMN IF EXISTS signature_data,
    DROP COLUMN IF EXISTS signed_at,
    DROP COLUMN IF EXISTS signed_by;

ALTER TABLE rentals
    DROP COLUMN IF EXISTS signature_data,
    DROP COLUMN IF EXISTS signed_at,
    DROP COLUMN IF EXISTS signed_by;

ALTER TABLE contracts
    DROP COLUMN IF EXISTS signature_data,
    DROP COLUMN IF EXISTS signed_at,
    DROP COLUMN IF EXISTS signed_by;

COMMIT;
