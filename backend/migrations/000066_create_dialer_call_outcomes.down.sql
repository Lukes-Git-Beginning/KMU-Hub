ALTER TABLE dialer_call_sessions DROP CONSTRAINT IF EXISTS fk_dcs_outcome;
ALTER TABLE dialer_campaign_contacts DROP CONSTRAINT IF EXISTS fk_dcc_outcome;
DROP TABLE IF EXISTS dialer_call_outcomes;
