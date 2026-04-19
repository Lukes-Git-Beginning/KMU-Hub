-- Sprint 1 S1.3 Formulare — rollback

DROP TRIGGER IF EXISTS trg_form_submissions_count ON form_submissions;
DROP FUNCTION IF EXISTS update_form_submission_count();

DROP INDEX IF EXISTS idx_form_webhook_deliveries_submission;
DROP INDEX IF EXISTS idx_form_webhook_deliveries_webhook;
DROP INDEX IF EXISTS idx_form_webhook_deliveries_pending;
DROP TABLE IF EXISTS form_webhook_deliveries;
DROP TYPE IF EXISTS form_webhook_delivery_status;

DROP INDEX IF EXISTS idx_form_webhooks_tenant;
DROP INDEX IF EXISTS idx_form_webhooks_schema;
DROP TABLE IF EXISTS form_webhooks;

DROP INDEX IF EXISTS idx_form_submissions_tenant_status;
DROP INDEX IF EXISTS idx_form_submissions_schema_time;
DROP TABLE IF EXISTS form_submissions;
DROP TYPE IF EXISTS form_submission_status;

DROP INDEX IF EXISTS idx_form_schemas_tenant_status;
DROP INDEX IF EXISTS idx_form_schemas_tenant_active;
DROP TABLE IF EXISTS form_schemas;
DROP TYPE IF EXISTS form_schema_status;
