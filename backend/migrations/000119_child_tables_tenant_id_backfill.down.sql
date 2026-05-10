-- Rollback 000119 — drop FK constraints, indices, and tenant_id columns
-- on the four child tables; relax consent_records.tenant_id back to NULLABLE.
-- Backfilled column data is lost on the four ADD-COLUMN tables; consent_records
-- retains its data (only the NOT NULL constraint is reverted).

BEGIN;

-- consent_records: only relax NOT NULL, keep column + data.
ALTER TABLE consent_records ALTER COLUMN tenant_id DROP NOT NULL;

-- recording_consents
DROP INDEX IF EXISTS idx_recording_consents_tenant_id;
ALTER TABLE recording_consents DROP CONSTRAINT IF EXISTS fk_recording_consents_tenant;
ALTER TABLE recording_consents DROP COLUMN IF EXISTS tenant_id;

-- dialer_call_events
DROP INDEX IF EXISTS idx_dialer_call_events_tenant_id;
ALTER TABLE dialer_call_events DROP CONSTRAINT IF EXISTS fk_dialer_call_events_tenant;
ALTER TABLE dialer_call_events DROP COLUMN IF EXISTS tenant_id;

-- dialer_agent_status_log
DROP INDEX IF EXISTS idx_dialer_agent_status_log_tenant_id;
ALTER TABLE dialer_agent_status_log DROP CONSTRAINT IF EXISTS fk_dialer_agent_status_log_tenant;
ALTER TABLE dialer_agent_status_log DROP COLUMN IF EXISTS tenant_id;

-- dialer_campaign_contacts
DROP INDEX IF EXISTS idx_dialer_campaign_contacts_tenant_id;
ALTER TABLE dialer_campaign_contacts DROP CONSTRAINT IF EXISTS fk_dialer_campaign_contacts_tenant;
ALTER TABLE dialer_campaign_contacts DROP COLUMN IF EXISTS tenant_id;

COMMIT;
