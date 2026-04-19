-- Sprint 1 S1.3 Formulare — schema

-- ============================================================================
-- form_schemas
-- ============================================================================
CREATE TYPE form_schema_status AS ENUM ('draft', 'active', 'archived');

CREATE TABLE form_schemas (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL,
    title            TEXT NOT NULL,
    description      TEXT NOT NULL DEFAULT '',
    fields           JSONB NOT NULL DEFAULT '[]',
    status           form_schema_status NOT NULL DEFAULT 'draft',
    is_template      BOOLEAN NOT NULL DEFAULT FALSE,
    is_public        BOOLEAN NOT NULL DEFAULT FALSE,
    page_count       INTEGER NOT NULL DEFAULT 1,
    submission_count INTEGER NOT NULL DEFAULT 0,
    created_by       UUID NULL REFERENCES users (id) ON DELETE SET NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ NULL
);

CREATE INDEX idx_form_schemas_tenant_active ON form_schemas (tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_form_schemas_tenant_status ON form_schemas (tenant_id, status) WHERE deleted_at IS NULL;

-- ============================================================================
-- form_submissions
-- ============================================================================
CREATE TYPE form_submission_status AS ENUM ('new', 'read', 'archived');

CREATE TABLE form_submissions (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    form_schema_id UUID NULL REFERENCES form_schemas (id) ON DELETE SET NULL,
    tenant_id      UUID NOT NULL,
    answers        JSONB NOT NULL,
    status         form_submission_status NOT NULL DEFAULT 'new',
    submitted_by   TEXT NULL,
    -- DSGVO: ip_address nur bei explizitem Consent speichern, ansonsten NULL lassen
    ip_address     INET NULL,
    submitted_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_form_submissions_schema_time   ON form_submissions (form_schema_id, submitted_at DESC);
CREATE INDEX idx_form_submissions_tenant_status ON form_submissions (tenant_id, status);

-- ============================================================================
-- form_webhooks
-- ============================================================================
CREATE TABLE form_webhooks (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    form_schema_id    UUID NOT NULL REFERENCES form_schemas (id) ON DELETE CASCADE,
    tenant_id         UUID NOT NULL,
    url               TEXT NOT NULL,
    secret            TEXT NULL,
    events            TEXT[] NOT NULL DEFAULT ARRAY['submission.created'],
    active            BOOLEAN NOT NULL DEFAULT TRUE,
    last_triggered_at TIMESTAMPTZ NULL,
    last_status       TEXT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_form_webhooks_schema ON form_webhooks (form_schema_id);
CREATE INDEX idx_form_webhooks_tenant ON form_webhooks (tenant_id);

-- ============================================================================
-- form_webhook_deliveries
-- ============================================================================
CREATE TYPE form_webhook_delivery_status AS ENUM ('pending', 'delivered', 'failed', 'dead');

CREATE TABLE form_webhook_deliveries (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_id         UUID NOT NULL REFERENCES form_webhooks (id) ON DELETE CASCADE,
    submission_id      UUID NOT NULL REFERENCES form_submissions (id) ON DELETE CASCADE,
    tenant_id          UUID NOT NULL,
    payload            JSONB NOT NULL,
    status             form_webhook_delivery_status NOT NULL DEFAULT 'pending',
    attempt_count      INTEGER NOT NULL DEFAULT 0,
    max_attempts       INTEGER NOT NULL DEFAULT 5,
    next_attempt_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_error         TEXT NULL,
    last_response_code INTEGER NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    delivered_at       TIMESTAMPTZ NULL
);

CREATE INDEX idx_form_webhook_deliveries_pending    ON form_webhook_deliveries (next_attempt_at) WHERE status = 'pending';
CREATE INDEX idx_form_webhook_deliveries_webhook    ON form_webhook_deliveries (webhook_id);
CREATE INDEX idx_form_webhook_deliveries_submission ON form_webhook_deliveries (submission_id);

-- ============================================================================
-- Trigger: submission_count automatisch pflegen
-- ============================================================================
CREATE OR REPLACE FUNCTION update_form_submission_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE form_schemas
        SET submission_count = submission_count + 1
        WHERE id = NEW.form_schema_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE form_schemas
        SET submission_count = GREATEST(submission_count - 1, 0)
        WHERE id = OLD.form_schema_id;
    END IF;
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_form_submissions_count
    AFTER INSERT OR DELETE ON form_submissions
    FOR EACH ROW EXECUTE FUNCTION update_form_submission_count();
