-- Sprint 4 Welle 1b — backfill tenant_id on Pilot-0 child tables.
-- 2026-05-10
--
-- Migrations 000114/000115 retrofitted tenant_id onto most application tables
-- but skipped four child tables that derive their tenant transitively from a
-- parent FK: dialer_campaign_contacts, dialer_agent_status_log,
-- dialer_call_events, recording_consents. RLS policies prefer same-table
-- columns over joins, so this migration adds the column with a JOIN backfill,
-- enforces NOT NULL + FK + index, and promotes consent_records.tenant_id
-- (already added in 000111 but left nullable) to NOT NULL.
--
-- Pattern follows 000115: ADD COLUMN IF NOT EXISTS, UPDATE via JOIN, ALTER
-- COLUMN SET NOT NULL, FK NOT VALID + VALIDATE separately so the row scan
-- runs under SHARE UPDATE EXCLUSIVE without blocking writers.

BEGIN;

-- ============================================================================
-- 1) dialer_campaign_contacts (campaign_id -> dialer_campaigns.tenant_id)
-- ============================================================================

ALTER TABLE dialer_campaign_contacts ADD COLUMN IF NOT EXISTS tenant_id UUID;

UPDATE dialer_campaign_contacts dcc
   SET tenant_id = dc.tenant_id
  FROM dialer_campaigns dc
 WHERE dcc.campaign_id = dc.id
   AND dcc.tenant_id IS NULL;

DO $$
BEGIN
    IF (SELECT COUNT(*) FROM dialer_campaign_contacts WHERE tenant_id IS NULL) > 0 THEN
        RAISE EXCEPTION 'dialer_campaign_contacts has rows with NULL tenant_id after backfill — orphan campaign_id?';
    END IF;
END$$;

ALTER TABLE dialer_campaign_contacts ALTER COLUMN tenant_id SET NOT NULL;

ALTER TABLE dialer_campaign_contacts
    ADD CONSTRAINT fk_dialer_campaign_contacts_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) NOT VALID;

ALTER TABLE dialer_campaign_contacts VALIDATE CONSTRAINT fk_dialer_campaign_contacts_tenant;

CREATE INDEX IF NOT EXISTS idx_dialer_campaign_contacts_tenant_id
    ON dialer_campaign_contacts(tenant_id);

-- ============================================================================
-- 2) dialer_agent_status_log (campaign_id -> dialer_campaigns.tenant_id,
--    fallback user_id -> users.tenant_id when campaign_id is NULL)
-- ============================================================================

ALTER TABLE dialer_agent_status_log ADD COLUMN IF NOT EXISTS tenant_id UUID;

UPDATE dialer_agent_status_log dasl
   SET tenant_id = dc.tenant_id
  FROM dialer_campaigns dc
 WHERE dasl.campaign_id = dc.id
   AND dasl.tenant_id IS NULL;

UPDATE dialer_agent_status_log dasl
   SET tenant_id = u.tenant_id
  FROM users u
 WHERE dasl.user_id = u.id
   AND dasl.tenant_id IS NULL;

DO $$
BEGIN
    IF (SELECT COUNT(*) FROM dialer_agent_status_log WHERE tenant_id IS NULL) > 0 THEN
        RAISE EXCEPTION 'dialer_agent_status_log has rows with NULL tenant_id after backfill — neither campaign nor user resolved';
    END IF;
END$$;

ALTER TABLE dialer_agent_status_log ALTER COLUMN tenant_id SET NOT NULL;

ALTER TABLE dialer_agent_status_log
    ADD CONSTRAINT fk_dialer_agent_status_log_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) NOT VALID;

ALTER TABLE dialer_agent_status_log VALIDATE CONSTRAINT fk_dialer_agent_status_log_tenant;

CREATE INDEX IF NOT EXISTS idx_dialer_agent_status_log_tenant_id
    ON dialer_agent_status_log(tenant_id);

-- ============================================================================
-- 3) dialer_call_events (dialer_call_session_id -> dialer_call_sessions.tenant_id)
-- ============================================================================

ALTER TABLE dialer_call_events ADD COLUMN IF NOT EXISTS tenant_id UUID;

UPDATE dialer_call_events dce
   SET tenant_id = dcs.tenant_id
  FROM dialer_call_sessions dcs
 WHERE dce.dialer_call_session_id = dcs.id
   AND dce.tenant_id IS NULL;

DO $$
BEGIN
    IF (SELECT COUNT(*) FROM dialer_call_events WHERE tenant_id IS NULL) > 0 THEN
        RAISE EXCEPTION 'dialer_call_events has rows with NULL tenant_id after backfill — orphan dialer_call_session_id?';
    END IF;
END$$;

ALTER TABLE dialer_call_events ALTER COLUMN tenant_id SET NOT NULL;

ALTER TABLE dialer_call_events
    ADD CONSTRAINT fk_dialer_call_events_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) NOT VALID;

ALTER TABLE dialer_call_events VALIDATE CONSTRAINT fk_dialer_call_events_tenant;

CREATE INDEX IF NOT EXISTS idx_dialer_call_events_tenant_id
    ON dialer_call_events(tenant_id);

-- ============================================================================
-- 4) recording_consents (recording_id -> recordings.tenant_id)
-- ============================================================================

ALTER TABLE recording_consents ADD COLUMN IF NOT EXISTS tenant_id UUID;

UPDATE recording_consents rc
   SET tenant_id = r.tenant_id
  FROM recordings r
 WHERE rc.recording_id = r.id
   AND rc.tenant_id IS NULL;

DO $$
BEGIN
    IF (SELECT COUNT(*) FROM recording_consents WHERE tenant_id IS NULL) > 0 THEN
        RAISE EXCEPTION 'recording_consents has rows with NULL tenant_id after backfill — orphan recording_id?';
    END IF;
END$$;

ALTER TABLE recording_consents ALTER COLUMN tenant_id SET NOT NULL;

ALTER TABLE recording_consents
    ADD CONSTRAINT fk_recording_consents_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) NOT VALID;

ALTER TABLE recording_consents VALIDATE CONSTRAINT fk_recording_consents_tenant;

CREATE INDEX IF NOT EXISTS idx_recording_consents_tenant_id
    ON recording_consents(tenant_id);

-- ============================================================================
-- 5) consent_records.tenant_id NULLABLE -> NOT NULL
--    Column was added in 000111 but left nullable. Backfill via existing
--    contact_id -> contacts.tenant_id, then promote.
-- ============================================================================

UPDATE consent_records cr
   SET tenant_id = c.tenant_id
  FROM contacts c
 WHERE cr.contact_id = c.id
   AND cr.tenant_id IS NULL;

DO $$
BEGIN
    IF (SELECT COUNT(*) FROM consent_records WHERE tenant_id IS NULL) > 0 THEN
        RAISE EXCEPTION 'consent_records has rows with NULL tenant_id after backfill — orphan contact_id?';
    END IF;
END$$;

ALTER TABLE consent_records ALTER COLUMN tenant_id SET NOT NULL;

COMMIT;
