-- Rollback 000288: drop the CSAT table and the denormalised ticket columns.

BEGIN;

DROP TABLE IF EXISTS ticket_csat_responses;

ALTER TABLE tickets
    DROP COLUMN IF EXISTS csat_comment,
    DROP COLUMN IF EXISTS csat_rating;

COMMIT;
