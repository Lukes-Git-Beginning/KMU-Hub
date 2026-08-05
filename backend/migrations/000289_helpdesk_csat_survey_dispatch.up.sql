-- Migration 000289: CSAT survey dispatch bookkeeping
--
-- 000288 gave a pending survey a token and an expiry. What it could not answer
-- is *when* the survey mail should go out and whether it already did, so the
-- poller that sends the mail gets its three columns here:
--
--   survey_send_after        when the mail becomes due (close + tenant delay).
--                            Stored explicitly instead of derived from
--                            token_expires_at: expiry is delay + TTL, and
--                            reconstructing the delay out of it would break
--                            silently the moment the TTL changes.
--   survey_sent_at           doubles as the dispatch claim. A tick claims a row
--                            by setting it (RowsAffected()==1 wins), and clears
--                            it again if the send fails, exactly like
--                            berichte/scheduler.ClaimSchedule.
--   survey_dispatch_attempts bounds those retries. Without it a permanently
--                            undeliverable address would be retried every tick
--                            until the token expires 30 days later.
--
-- No RLS work: the columns land on ticket_csat_responses, whose policy came
-- with 000288.

BEGIN;

ALTER TABLE ticket_csat_responses
    ADD COLUMN survey_send_after        TIMESTAMPTZ NULL,
    ADD COLUMN survey_sent_at           TIMESTAMPTZ NULL,
    ADD COLUMN survey_dispatch_attempts SMALLINT    NOT NULL DEFAULT 0;

-- Rows that already carry a token were issued before this migration and have
-- no send time. Their delay is gone, so they become due immediately rather
-- than never -- an unsent survey is the lesser of the two errors.
UPDATE ticket_csat_responses
   SET survey_send_after = created_at
 WHERE token IS NOT NULL
   AND survey_send_after IS NULL;

-- The poller's due query in one index: only pending, unsent surveys are ever
-- scanned, and the table stays cheap as rated rows accumulate.
CREATE INDEX idx_ticket_csat_responses_due
    ON ticket_csat_responses (survey_send_after)
    WHERE token IS NOT NULL
      AND survey_sent_at IS NULL
      AND submitted_at IS NULL;

COMMIT;
