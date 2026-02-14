-- Revert gantt view type: set any gantt preferences back to list, then restore constraint
UPDATE user_project_preferences SET view_type = 'list' WHERE view_type = 'gantt';
ALTER TABLE user_project_preferences DROP CONSTRAINT IF EXISTS user_project_preferences_view_type_check;
ALTER TABLE user_project_preferences ADD CONSTRAINT user_project_preferences_view_type_check CHECK (view_type IN ('list', 'kanban'));
