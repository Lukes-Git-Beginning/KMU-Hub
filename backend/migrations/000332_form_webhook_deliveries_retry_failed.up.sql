-- fix-formulare-webhook-failed-deliveries-never-retried: ClaimPendingDeliveries now
-- also leases 'failed' rows once their back-off has elapsed (the worker parks a
-- failed attempt as status='failed' + next_attempt_at; until now nothing ever
-- claimed it again).
--
-- Rows that are 'failed' today were abandoned by the old claim and would all
-- become due at once, re-sending old HMAC-signed payloads to customer URLs.
-- Close them out as 'dead' (last_error/last_response_code stay for the record);
-- only attempts that fail from here on are retried.
UPDATE form_webhook_deliveries SET status = 'dead' WHERE status = 'failed';

-- The claim filters on status IN ('pending','failed'); the old partial index
-- (status = 'pending') no longer serves it.
DROP INDEX IF EXISTS idx_form_webhook_deliveries_pending;
CREATE INDEX idx_form_webhook_deliveries_pending
    ON form_webhook_deliveries (next_attempt_at)
    WHERE status IN ('pending', 'failed');
