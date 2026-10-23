-- fix-form-webhook-deliveries-unbounded-retention: the new retention handler
-- filters on tenant_id + created_at for final deliveries only. The existing
-- indexes cover next_attempt_at (pending only), webhook_id and submission_id,
-- none of which serves that scan. Partial, because pending/failed rows are
-- still in the retry loop and never retention candidates.
CREATE INDEX idx_form_webhook_deliveries_tenant_final_created
    ON form_webhook_deliveries (tenant_id, created_at)
    WHERE status IN ('delivered', 'dead');
