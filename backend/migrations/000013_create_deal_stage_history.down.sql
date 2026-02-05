-- Drop trigger and function
DROP TRIGGER IF EXISTS trigger_deal_stage_change ON deals;
DROP FUNCTION IF EXISTS record_deal_stage_change();

-- Drop table
DROP TABLE IF EXISTS deal_stage_history;
