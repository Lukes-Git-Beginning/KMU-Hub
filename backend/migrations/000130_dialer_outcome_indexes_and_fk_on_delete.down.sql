-- =============================================================================
-- Migration 000130 DOWN: Remove outcome_id indexes + revert FK ON DELETE
--
-- Reverts constraints to Postgres default (NO ACTION / equivalent to RESTRICT
-- but deferred-compatible) — i.e. removes the explicit ON DELETE clause by
-- dropping and re-adding each constraint without one.
-- =============================================================================

-- ---- Drop outcome_id partial indexes ----
DROP INDEX IF EXISTS idx_dialer_campaign_contacts_outcome_id;
DROP INDEX IF EXISTS idx_dialer_call_sessions_outcome_id;

-- ---- fk_dcs_outcome: revert to no ON DELETE ----
ALTER TABLE dialer_call_sessions
    DROP CONSTRAINT IF EXISTS fk_dcs_outcome;
ALTER TABLE dialer_call_sessions
    ADD CONSTRAINT fk_dcs_outcome
        FOREIGN KEY (outcome_id) REFERENCES dialer_call_outcomes(id);

-- ---- fk_dcc_outcome: revert to no ON DELETE ----
ALTER TABLE dialer_campaign_contacts
    DROP CONSTRAINT IF EXISTS fk_dcc_outcome;
ALTER TABLE dialer_campaign_contacts
    ADD CONSTRAINT fk_dcc_outcome
        FOREIGN KEY (outcome_id) REFERENCES dialer_call_outcomes(id);

-- ---- dialer_call_sessions.agent_id ----
ALTER TABLE dialer_call_sessions
    DROP CONSTRAINT IF EXISTS dialer_call_sessions_agent_id_fkey;
ALTER TABLE dialer_call_sessions
    ADD CONSTRAINT dialer_call_sessions_agent_id_fkey
        FOREIGN KEY (agent_id) REFERENCES users(id);

-- ---- dialer_call_sessions.call_session_id ----
ALTER TABLE dialer_call_sessions
    DROP CONSTRAINT IF EXISTS dialer_call_sessions_call_session_id_fkey;
ALTER TABLE dialer_call_sessions
    ADD CONSTRAINT dialer_call_sessions_call_session_id_fkey
        FOREIGN KEY (call_session_id) REFERENCES call_sessions(id);

-- ---- dialer_call_sessions.campaign_contact_id ----
ALTER TABLE dialer_call_sessions
    DROP CONSTRAINT IF EXISTS dialer_call_sessions_campaign_contact_id_fkey;
ALTER TABLE dialer_call_sessions
    ADD CONSTRAINT dialer_call_sessions_campaign_contact_id_fkey
        FOREIGN KEY (campaign_contact_id) REFERENCES dialer_campaign_contacts(id);

-- ---- dialer_campaign_contacts.contact_id ----
ALTER TABLE dialer_campaign_contacts
    DROP CONSTRAINT IF EXISTS dialer_campaign_contacts_contact_id_fkey;
ALTER TABLE dialer_campaign_contacts
    ADD CONSTRAINT dialer_campaign_contacts_contact_id_fkey
        FOREIGN KEY (contact_id) REFERENCES contacts(id);

-- ---- dialer_campaigns.created_by ----
ALTER TABLE dialer_campaigns
    DROP CONSTRAINT IF EXISTS dialer_campaigns_created_by_fkey;
ALTER TABLE dialer_campaigns
    ADD CONSTRAINT dialer_campaigns_created_by_fkey
        FOREIGN KEY (created_by) REFERENCES users(id);

-- ---- dialer_agent_status_log.changed_by ----
ALTER TABLE dialer_agent_status_log
    DROP CONSTRAINT IF EXISTS dialer_agent_status_log_changed_by_fkey;
ALTER TABLE dialer_agent_status_log
    ADD CONSTRAINT dialer_agent_status_log_changed_by_fkey
        FOREIGN KEY (changed_by) REFERENCES users(id);

-- ---- dialer_agent_status_log.campaign_id ----
ALTER TABLE dialer_agent_status_log
    DROP CONSTRAINT IF EXISTS dialer_agent_status_log_campaign_id_fkey;
ALTER TABLE dialer_agent_status_log
    ADD CONSTRAINT dialer_agent_status_log_campaign_id_fkey
        FOREIGN KEY (campaign_id) REFERENCES dialer_campaigns(id);

-- ---- dialer_agent_status_log.user_id ----
ALTER TABLE dialer_agent_status_log
    DROP CONSTRAINT IF EXISTS dialer_agent_status_log_user_id_fkey;
ALTER TABLE dialer_agent_status_log
    ADD CONSTRAINT dialer_agent_status_log_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id);
