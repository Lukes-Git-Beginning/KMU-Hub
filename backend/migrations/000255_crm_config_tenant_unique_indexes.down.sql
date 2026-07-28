DROP INDEX IF EXISTS idx_pipeline_stages_lost;
DROP INDEX IF EXISTS idx_pipeline_stages_won;
CREATE UNIQUE INDEX idx_pipeline_stages_lost ON pipeline_stages (is_lost) WHERE is_lost = TRUE;
CREATE UNIQUE INDEX idx_pipeline_stages_won ON pipeline_stages (is_won) WHERE is_won = TRUE;

DROP INDEX IF EXISTS idx_custom_field_definitions_entity_name;
CREATE UNIQUE INDEX idx_custom_field_definitions_entity_name
    ON custom_field_definitions (entity_type, field_name);

DROP INDEX IF EXISTS idx_tags_entity_name;
CREATE UNIQUE INDEX idx_tags_entity_name ON tags (entity_type, LOWER(name));
