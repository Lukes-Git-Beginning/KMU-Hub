-- Automation workflows
CREATE TABLE automations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    description TEXT,
    scope VARCHAR(20) NOT NULL DEFAULT 'personal'
        CHECK (scope IN ('personal', 'team', 'organization')),
    owner_id UUID NOT NULL REFERENCES users(id),
    trigger_type VARCHAR(100) NOT NULL,
    trigger_config JSONB NOT NULL DEFAULT '{}',
    conditions JSONB NOT NULL DEFAULT '{}',
    actions JSONB NOT NULL DEFAULT '[]',
    is_active BOOLEAN NOT NULL DEFAULT true,
    max_steps INTEGER NOT NULL DEFAULT 10,
    template_id VARCHAR(100),
    last_triggered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_automations_trigger_active ON automations(trigger_type) WHERE is_active = true;
CREATE INDEX idx_automations_owner ON automations(owner_id, created_at DESC);

-- Execution logs
CREATE TABLE automation_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    automation_id UUID NOT NULL REFERENCES automations(id) ON DELETE CASCADE,
    chain_id UUID NOT NULL,
    trigger_event JSONB NOT NULL,
    condition_result BOOLEAN NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'running'
        CHECK (status IN ('running', 'completed', 'failed', 'skipped', 'aborted')),
    steps JSONB NOT NULL DEFAULT '[]',
    error_message TEXT,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    duration_ms INTEGER
);

CREATE INDEX idx_automation_executions_automation ON automation_executions(automation_id, started_at DESC);
CREATE INDEX idx_automation_executions_status ON automation_executions(status, started_at DESC)
    WHERE status IN ('failed', 'running');
CREATE INDEX idx_automation_executions_cleanup ON automation_executions(started_at)
    WHERE completed_at IS NOT NULL;

-- Pre-built templates (immutable reference)
CREATE TABLE automation_templates (
    id VARCHAR(100) PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    description TEXT NOT NULL,
    category VARCHAR(50) NOT NULL,
    complexity VARCHAR(20) NOT NULL DEFAULT 'einfach'
        CHECK (complexity IN ('einfach', 'mittel', 'fortgeschritten')),
    trigger_type VARCHAR(100) NOT NULL,
    trigger_config JSONB NOT NULL DEFAULT '{}',
    conditions JSONB NOT NULL DEFAULT '{}',
    actions JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
