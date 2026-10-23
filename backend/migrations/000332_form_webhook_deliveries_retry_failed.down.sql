-- The failed -> dead close-out in the up migration is not reversible; the rows
-- stay dead. Only the index is restored to its original shape.
DROP INDEX IF EXISTS idx_form_webhook_deliveries_pending;
CREATE INDEX idx_form_webhook_deliveries_pending
    ON form_webhook_deliveries (next_attempt_at)
    WHERE status = 'pending';
