-- Deal Stage History table for tracking stage transitions
-- Used for conversion reports

CREATE TABLE deal_stage_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    deal_id UUID NOT NULL REFERENCES deals(id) ON DELETE CASCADE,
    from_stage_id UUID REFERENCES pipeline_stages(id) ON DELETE SET NULL,
    to_stage_id UUID REFERENCES pipeline_stages(id) ON DELETE SET NULL,
    changed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_deal_stage_history_deal ON deal_stage_history (deal_id);
CREATE INDEX idx_deal_stage_history_changed_at ON deal_stage_history (changed_at);
CREATE INDEX idx_deal_stage_history_from_stage ON deal_stage_history (from_stage_id) WHERE from_stage_id IS NOT NULL;
CREATE INDEX idx_deal_stage_history_to_stage ON deal_stage_history (to_stage_id) WHERE to_stage_id IS NOT NULL;

-- Trigger function for automatic recording of stage changes
CREATE OR REPLACE FUNCTION record_deal_stage_change() RETURNS TRIGGER AS $$
BEGIN
    IF OLD.stage_id IS DISTINCT FROM NEW.stage_id THEN
        INSERT INTO deal_stage_history (deal_id, from_stage_id, to_stage_id, changed_at)
        VALUES (NEW.id, OLD.stage_id, NEW.stage_id, NOW());
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger on deals table
CREATE TRIGGER trigger_deal_stage_change
    AFTER UPDATE OF stage_id ON deals
    FOR EACH ROW EXECUTE FUNCTION record_deal_stage_change();
