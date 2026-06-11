-- Migration 000145: work_labels + task_labels
-- Adds label taxonomy for tasks: tenant-scoped labels with color coding,
-- and a many-to-many join table associating labels with tasks.

BEGIN;

-- Labels owned by a tenant
CREATE TABLE work_labels (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name       VARCHAR(100) NOT NULL,
    color      VARCHAR(7)  NOT NULL DEFAULT '#6b7280'
                           CHECK (color ~ '^#[0-9A-Fa-f]{6}$'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, name)
);

CREATE INDEX idx_work_labels_tenant_name ON work_labels (tenant_id, name);

-- RLS: standard tenant isolation
ALTER TABLE work_labels ENABLE ROW LEVEL SECURITY;
ALTER TABLE work_labels FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON work_labels
    USING  (tenant_id = current_tenant_id() OR is_system_context())
    WITH CHECK (tenant_id = current_tenant_id() OR is_system_context());

-- Join table: task <-> label (many-to-many)
CREATE TABLE task_labels (
    task_id   UUID NOT NULL REFERENCES tasks(id)       ON DELETE CASCADE,
    label_id  UUID NOT NULL REFERENCES work_labels(id) ON DELETE CASCADE,
    PRIMARY KEY (task_id, label_id)
);

CREATE INDEX idx_task_labels_task_id  ON task_labels (task_id);
CREATE INDEX idx_task_labels_label_id ON task_labels (label_id);

-- RLS on task_labels: allow access if the label belongs to the caller's tenant
ALTER TABLE task_labels ENABLE ROW LEVEL SECURITY;
ALTER TABLE task_labels FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON task_labels
    USING (
        is_system_context()
        OR EXISTS (
            SELECT 1 FROM work_labels wl
            WHERE wl.id = label_id
              AND wl.tenant_id = current_tenant_id()
        )
    )
    WITH CHECK (
        is_system_context()
        OR EXISTS (
            SELECT 1 FROM work_labels wl
            WHERE wl.id = label_id
              AND wl.tenant_id = current_tenant_id()
        )
    );

COMMIT;
