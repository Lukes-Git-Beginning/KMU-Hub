-- Sprint 1 S1.2 Berichte — rollback

DROP INDEX IF EXISTS idx_report_runs_definition;
DROP INDEX IF EXISTS idx_report_runs_schedule;
DROP INDEX IF EXISTS idx_report_runs_tenant_started;
DROP TABLE IF EXISTS report_runs;

DROP INDEX IF EXISTS idx_report_schedules_definition;
DROP INDEX IF EXISTS idx_report_schedules_active;
DROP INDEX IF EXISTS idx_report_schedules_tenant_id;
DROP TABLE IF EXISTS report_schedules;

DROP INDEX IF EXISTS idx_report_cache_tenant_id;
DROP INDEX IF EXISTS idx_report_cache_definition;
DROP INDEX IF EXISTS idx_report_cache_expires;
DROP TABLE IF EXISTS report_cache;

DROP INDEX IF EXISTS idx_report_definitions_kind;
DROP INDEX IF EXISTS idx_report_definitions_module;
DROP INDEX IF EXISTS idx_report_definitions_tenant_id;
DROP TABLE IF EXISTS report_definitions;
