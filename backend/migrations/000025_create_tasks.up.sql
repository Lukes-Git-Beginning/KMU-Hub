-- Tasks and task dependencies
-- Supports hierarchical tasks with subtasks, priorities, full-text search, and dependency tracking

-- ============================================================================
-- Tasks table
-- ============================================================================

CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    task_number INTEGER NOT NULL,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    status_id UUID REFERENCES project_statuses(id) ON DELETE SET NULL,
    priority VARCHAR(10) NOT NULL DEFAULT 'medium' CHECK (priority IN ('urgent', 'high', 'medium', 'low')),
    assignee_id UUID REFERENCES users(id) ON DELETE SET NULL,
    parent_task_id UUID REFERENCES tasks(id) ON DELETE CASCADE,
    depth INTEGER NOT NULL DEFAULT 0,
    sort_order DOUBLE PRECISION NOT NULL DEFAULT 0,
    due_date DATE,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    search_vector TSVECTOR
);

-- Core query indexes
CREATE INDEX idx_tasks_project ON tasks (project_id);
CREATE INDEX idx_tasks_assignee ON tasks (assignee_id) WHERE assignee_id IS NOT NULL;
CREATE INDEX idx_tasks_parent ON tasks (parent_task_id) WHERE parent_task_id IS NOT NULL;
CREATE INDEX idx_tasks_status ON tasks (status_id) WHERE status_id IS NOT NULL;
CREATE INDEX idx_tasks_due_date ON tasks (due_date) WHERE due_date IS NOT NULL;
CREATE INDEX idx_tasks_created_by ON tasks (created_by);
CREATE UNIQUE INDEX idx_tasks_project_number ON tasks (project_id, task_number);
CREATE INDEX idx_tasks_priority ON tasks (priority);
CREATE INDEX idx_tasks_created_at ON tasks (created_at DESC);
CREATE INDEX idx_tasks_search ON tasks USING GIN (search_vector);

-- Full-text search trigger (German config, matching existing CRM pattern)
CREATE OR REPLACE FUNCTION tasks_search_vector_update() RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('german', COALESCE(NEW.title, '')), 'A') ||
        setweight(to_tsvector('german', COALESCE(NEW.description, '')), 'B');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tasks_search_update
    BEFORE INSERT OR UPDATE ON tasks
    FOR EACH ROW EXECUTE FUNCTION tasks_search_vector_update();

-- ============================================================================
-- Task dependencies table
-- ============================================================================

CREATE TABLE task_dependencies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    target_task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    dependency_type VARCHAR(20) NOT NULL CHECK (dependency_type IN ('blocks', 'blocked_by', 'relates_to', 'duplicates')),
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (source_task_id, target_task_id, dependency_type)
);

CREATE INDEX idx_task_deps_source ON task_dependencies (source_task_id);
CREATE INDEX idx_task_deps_target ON task_dependencies (target_task_id);
