-- The Option-B tenant_id retrofit (migration 000106) added tenant_id to
-- tags, custom_field_definitions and pipeline_stages but never touched their
-- pre-existing UNIQUE indexes, which still enforce uniqueness GLOBALLY across
-- all tenants:
--   - idx_tags_entity_name: a tag name is unique per entity_type for every
--     tenant combined, so the second tenant to create e.g. a "VIP" contact
--     tag hits a raw unique-violation 500.
--   - idx_custom_field_definitions_entity_name: same gap for custom field
--     names.
--   - idx_pipeline_stages_won / idx_pipeline_stages_lost: only the very
--     first tenant to mark a stage "Won"/"Lost" can ever do so — every other
--     tenant's Create/Update fails at the DB, even though
--     pipelinestage.Service already re-implements the same uniqueness check
--     scoped correctly per tenant (HasWonStage/HasLostStage).
-- Safe against current data: production is single-tenant today (all rows
-- carry the sentinel tenant 00000000-0000-0000-0000-000000000001 from the
-- retrofit default), so no existing row can violate the new composite
-- indexes.

DROP INDEX IF EXISTS idx_tags_entity_name;
CREATE UNIQUE INDEX idx_tags_entity_name ON tags (tenant_id, entity_type, LOWER(name));

DROP INDEX IF EXISTS idx_custom_field_definitions_entity_name;
CREATE UNIQUE INDEX idx_custom_field_definitions_entity_name
    ON custom_field_definitions (tenant_id, entity_type, field_name);

DROP INDEX IF EXISTS idx_pipeline_stages_won;
DROP INDEX IF EXISTS idx_pipeline_stages_lost;
CREATE UNIQUE INDEX idx_pipeline_stages_won ON pipeline_stages (tenant_id) WHERE is_won = TRUE;
CREATE UNIQUE INDEX idx_pipeline_stages_lost ON pipeline_stages (tenant_id) WHERE is_lost = TRUE;
