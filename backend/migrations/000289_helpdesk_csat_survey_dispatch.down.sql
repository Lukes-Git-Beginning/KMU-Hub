BEGIN;

DROP INDEX IF EXISTS idx_ticket_csat_responses_due;

ALTER TABLE ticket_csat_responses
    DROP COLUMN IF EXISTS survey_dispatch_attempts,
    DROP COLUMN IF EXISTS survey_sent_at,
    DROP COLUMN IF EXISTS survey_send_after;

COMMIT;
