CREATE TABLE IF NOT EXISTS helpdesk_kb_articles (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID        NOT NULL,
    title       TEXT        NOT NULL,
    content     TEXT        NOT NULL DEFAULT '',
    category    TEXT        NOT NULL DEFAULT '',
    status      TEXT        NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published')),
    author_id   UUID        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS helpdesk_kb_articles_tenant_idx ON helpdesk_kb_articles (tenant_id);
CREATE INDEX IF NOT EXISTS helpdesk_kb_articles_category_idx ON helpdesk_kb_articles (tenant_id, category);

CALL enable_tenant_rls('helpdesk_kb_articles');
