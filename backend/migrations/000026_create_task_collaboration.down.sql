-- Rollback task collaboration tables (reverse dependency order)

DROP TABLE IF EXISTS task_custom_field_values;
DROP TABLE IF EXISTS user_project_preferences;
DROP TABLE IF EXISTS task_files;
DROP TABLE IF EXISTS task_activities;
DROP TABLE IF EXISTS task_entity_links;
DROP TABLE IF EXISTS task_comments;
