-- Task collaboration tables: comments, entity links, activities, files,
-- user preferences, and custom field values

-- ============================================================================
-- Task comments (with quote/reply support)
-- ============================================================================

CREATE TABLE task_comments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    author_id UUID NOT NULL REFERENCES users(id),
    content TEXT NOT NULL,
    quoted_comment_id UUID REFERENCES task_comments(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_task_comments_task ON task_comments (task_id, created_at);

-- ============================================================================
-- Task entity links (connect tasks to CRM entities)
-- ============================================================================

CREATE TABLE task_entity_links (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    entity_type VARCHAR(20) NOT NULL CHECK (entity_type IN ('contact', 'company', 'deal', 'channel', 'message')),
    entity_id UUID NOT NULL,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (task_id, entity_type, entity_id)
);

CREATE INDEX idx_task_entity_links_task ON task_entity_links (task_id);
CREATE INDEX idx_task_entity_links_entity ON task_entity_links (entity_type, entity_id);

-- ============================================================================
-- Task activities (audit log for task changes)
-- ============================================================================

CREATE TABLE task_activities (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    actor_id UUID NOT NULL REFERENCES users(id),
    action VARCHAR(50) NOT NULL,
    field_name VARCHAR(50),
    old_value JSONB,
    new_value JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_task_activities_task ON task_activities (task_id, created_at);

-- ============================================================================
-- Task files (file attachments via MinIO)
-- ============================================================================

CREATE TABLE task_files (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    filename VARCHAR(255) NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    file_size BIGINT NOT NULL,
    storage_key VARCHAR(500) NOT NULL,
    thumbnail_key VARCHAR(500),
    uploaded_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_task_files_task ON task_files (task_id);

-- ============================================================================
-- User project preferences (view settings per user per project)
-- ============================================================================

CREATE TABLE user_project_preferences (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    view_type VARCHAR(10) NOT NULL DEFAULT 'list' CHECK (view_type IN ('list', 'kanban')),
    list_group_by VARCHAR(20) DEFAULT 'status',
    list_sort_by VARCHAR(20) DEFAULT 'created_at',
    list_sort_desc BOOLEAN DEFAULT false,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, project_id)
);

-- ============================================================================
-- Task custom field values (reuses existing custom_field_definitions)
-- ============================================================================

CREATE TABLE task_custom_field_values (
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    field_id UUID NOT NULL REFERENCES custom_field_definitions(id) ON DELETE CASCADE,
    value JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (task_id, field_id)
);

CREATE INDEX idx_task_custom_field_values_field ON task_custom_field_values (field_id);
