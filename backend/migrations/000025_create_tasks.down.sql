-- Rollback tasks and dependencies

-- Drop dependencies first (references tasks)
DROP TABLE IF EXISTS task_dependencies;

-- Drop search trigger and function
DROP TRIGGER IF EXISTS tasks_search_update ON tasks;
DROP FUNCTION IF EXISTS tasks_search_vector_update();

-- Drop tasks table
DROP TABLE IF EXISTS tasks;
