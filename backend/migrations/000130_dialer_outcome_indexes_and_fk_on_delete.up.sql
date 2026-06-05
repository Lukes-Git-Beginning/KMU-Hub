-- =============================================================================
-- Migration 000130: Dialer outcome_id indexes + FK ON DELETE semantics
-- =============================================================================
-- Part 1: Partial indexes on outcome_id FK columns
-- (No CONCURRENTLY — golang-migrate runs in a transaction; tables are small.)
-- =============================================================================

CREATE INDEX IF NOT EXISTS idx_dialer_campaign_contacts_outcome_id
    ON dialer_campaign_contacts(outcome_id)
    WHERE outcome_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_dialer_call_sessions_outcome_id
    ON dialer_call_sessions(outcome_id)
    WHERE outcome_id IS NOT NULL;

-- =============================================================================
-- Part 2: FK ON DELETE semantics — Dialer + closely related history tables
--
-- Decision matrix (GDPR context: erasure.go uses ErasureAnonymize for users,
-- contacts are anonymized not hard-deleted; audit material must survive):
--
--   dialer_campaigns.created_by → users(id)
--     → RESTRICT: a user who created campaigns cannot be hard-deleted while
--       campaigns exist; GDPR erasure anonymizes the user row instead.
--       Keeps campaign authorship traceable for compliance.
--
--   dialer_campaign_contacts.contact_id → contacts(id)
--     → RESTRICT: CRM contacts are anonymized on GDPR erasure, never hard-
--       deleted (CRMErasureHandler uses ErasureAnonymize). Restricting ensures
--       no silent data loss; the anonymized contact row remains as the anchor.
--
--   dialer_call_sessions.campaign_contact_id → dialer_campaign_contacts(id)
--     → RESTRICT: call sessions are audit material (call events, outcomes,
--       duration). A campaign contact must not disappear while sessions exist.
--
--   dialer_call_sessions.call_session_id → call_sessions(id)
--     → SET NULL: the WebRTC/SIP call_session is infrastructure; its deletion
--       (e.g. cleanup after TTL) should not destroy the dialer session record,
--       which holds outcome + notes as business data. NULL means "call log
--       no longer available" — acceptable degradation.
--
--   dialer_call_sessions.agent_id → users(id)
--     → RESTRICT: agents are anonymized not deleted; RESTRICT prevents data
--       loss and keeps call assignment auditable.
--
--   dialer_agent_status_log.user_id → users(id)
--     → RESTRICT: status log is an audit trail; agent rows are anonymized.
--
--   dialer_agent_status_log.campaign_id → dialer_campaigns(id)
--     → SET NULL: if a campaign is deleted (archived then purged), the status
--       log entries lose their campaign reference but must be retained as an
--       agent activity record. NULL is preferable to losing the row.
--
--   dialer_agent_status_log.changed_by → users(id)
--     → SET NULL: the actor who triggered a status change is a soft audit
--       reference; losing it on user anonymization/deletion is acceptable,
--       while the status-change fact itself must survive.
--
--   fk_dcc_outcome (dialer_campaign_contacts.outcome_id → dialer_call_outcomes)
--     → SET NULL: outcome labels are tenant-configurable; deleting an outcome
--       type should NULL the reference, not cascade-delete campaign contact
--       records which are the primary business data.
--
--   fk_dcs_outcome (dialer_call_sessions.outcome_id → dialer_call_outcomes)
--     → SET NULL: same rationale as above — session history survives outcome
--       label deletion.
-- =============================================================================

-- PostgreSQL names inline REFERENCES constraints as <table>_<column>_fkey.
-- Named constraints (fk_dcc_outcome, fk_dcs_outcome) retain their names.

-- ---- dialer_campaigns.created_by ----
ALTER TABLE dialer_campaigns
    DROP CONSTRAINT IF EXISTS dialer_campaigns_created_by_fkey;
ALTER TABLE dialer_campaigns
    ADD CONSTRAINT dialer_campaigns_created_by_fkey
        FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE RESTRICT;

-- ---- dialer_campaign_contacts.contact_id ----
ALTER TABLE dialer_campaign_contacts
    DROP CONSTRAINT IF EXISTS dialer_campaign_contacts_contact_id_fkey;
ALTER TABLE dialer_campaign_contacts
    ADD CONSTRAINT dialer_campaign_contacts_contact_id_fkey
        FOREIGN KEY (contact_id) REFERENCES contacts(id) ON DELETE RESTRICT;

-- ---- dialer_call_sessions.campaign_contact_id ----
ALTER TABLE dialer_call_sessions
    DROP CONSTRAINT IF EXISTS dialer_call_sessions_campaign_contact_id_fkey;
ALTER TABLE dialer_call_sessions
    ADD CONSTRAINT dialer_call_sessions_campaign_contact_id_fkey
        FOREIGN KEY (campaign_contact_id) REFERENCES dialer_campaign_contacts(id) ON DELETE RESTRICT;

-- ---- dialer_call_sessions.call_session_id ----
ALTER TABLE dialer_call_sessions
    DROP CONSTRAINT IF EXISTS dialer_call_sessions_call_session_id_fkey;
ALTER TABLE dialer_call_sessions
    ADD CONSTRAINT dialer_call_sessions_call_session_id_fkey
        FOREIGN KEY (call_session_id) REFERENCES call_sessions(id) ON DELETE SET NULL;

-- ---- dialer_call_sessions.agent_id ----
ALTER TABLE dialer_call_sessions
    DROP CONSTRAINT IF EXISTS dialer_call_sessions_agent_id_fkey;
ALTER TABLE dialer_call_sessions
    ADD CONSTRAINT dialer_call_sessions_agent_id_fkey
        FOREIGN KEY (agent_id) REFERENCES users(id) ON DELETE RESTRICT;

-- ---- dialer_agent_status_log.user_id ----
ALTER TABLE dialer_agent_status_log
    DROP CONSTRAINT IF EXISTS dialer_agent_status_log_user_id_fkey;
ALTER TABLE dialer_agent_status_log
    ADD CONSTRAINT dialer_agent_status_log_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT;

-- ---- dialer_agent_status_log.campaign_id ----
ALTER TABLE dialer_agent_status_log
    DROP CONSTRAINT IF EXISTS dialer_agent_status_log_campaign_id_fkey;
ALTER TABLE dialer_agent_status_log
    ADD CONSTRAINT dialer_agent_status_log_campaign_id_fkey
        FOREIGN KEY (campaign_id) REFERENCES dialer_campaigns(id) ON DELETE SET NULL;

-- ---- dialer_agent_status_log.changed_by ----
ALTER TABLE dialer_agent_status_log
    DROP CONSTRAINT IF EXISTS dialer_agent_status_log_changed_by_fkey;
ALTER TABLE dialer_agent_status_log
    ADD CONSTRAINT dialer_agent_status_log_changed_by_fkey
        FOREIGN KEY (changed_by) REFERENCES users(id) ON DELETE SET NULL;

-- ---- fk_dcc_outcome (outcome_id on dialer_campaign_contacts) ----
ALTER TABLE dialer_campaign_contacts
    DROP CONSTRAINT IF EXISTS fk_dcc_outcome;
ALTER TABLE dialer_campaign_contacts
    ADD CONSTRAINT fk_dcc_outcome
        FOREIGN KEY (outcome_id) REFERENCES dialer_call_outcomes(id) ON DELETE SET NULL;

-- ---- fk_dcs_outcome (outcome_id on dialer_call_sessions) ----
ALTER TABLE dialer_call_sessions
    DROP CONSTRAINT IF EXISTS fk_dcs_outcome;
ALTER TABLE dialer_call_sessions
    ADD CONSTRAINT fk_dcs_outcome
        FOREIGN KEY (outcome_id) REFERENCES dialer_call_outcomes(id) ON DELETE SET NULL;
