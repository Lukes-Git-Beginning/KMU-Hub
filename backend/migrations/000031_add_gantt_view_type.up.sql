-- Add 'gantt' to the allowed view_type values in user_project_preferences
ALTER TABLE user_project_preferences DROP CONSTRAINT IF EXISTS user_project_preferences_view_type_check;
ALTER TABLE user_project_preferences ADD CONSTRAINT user_project_preferences_view_type_check CHECK (view_type IN ('list', 'kanban', 'gantt'));
