-- Migration 000163 DOWN

BEGIN;

DROP TABLE IF EXISTS report_templates;
DROP TABLE IF EXISTS measurement_positions;
DROP TABLE IF EXISTS measurements;

COMMIT;
