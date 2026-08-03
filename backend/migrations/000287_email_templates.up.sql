-- Email templates / Quicktexte: tenant-scoped, personal-or-shared reusable
-- message bodies with server-side placeholder substitution (fixed allow-list,
-- see internal/email/template.AllowedPlaceholders — not a general template
-- engine, to avoid turning a template into a data leak across fields).
CREATE TABLE IF NOT EXISTS email_templates (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    -- NULL for shared templates; the creating user for personal ones.
    owner_id    UUID REFERENCES users(id),
    visibility  VARCHAR(20) NOT NULL DEFAULT 'personal',
    name        VARCHAR(255) NOT NULL,
    subject     TEXT NOT NULL DEFAULT '',
    body_html   TEXT NOT NULL DEFAULT '',
    body_text   TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_email_templates_visibility CHECK (visibility IN ('personal', 'shared'))
);

CREATE INDEX idx_email_templates_tenant ON email_templates(tenant_id);
CREATE INDEX idx_email_templates_owner ON email_templates(owner_id) WHERE owner_id IS NOT NULL;

CALL enable_tenant_rls('email_templates');
