-- Rollback 000236: drop the counter table and the new ticket columns.

BEGIN;

DROP TABLE IF EXISTS helpdesk_ticket_counters;

ALTER TABLE tickets
    DROP COLUMN IF EXISTS ticket_number,
    DROP COLUMN IF EXISTS category,
    DROP COLUMN IF EXISTS description;

COMMIT;
