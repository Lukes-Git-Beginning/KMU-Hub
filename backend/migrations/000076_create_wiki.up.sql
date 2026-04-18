-- Sprint 1 S1.1 Wiki — schema

-- ============================================================================
-- wiki_categories (before articles for FK)
-- ============================================================================
CREATE TABLE wiki_categories (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    name        TEXT NOT NULL,
    parent_id   UUID NULL REFERENCES wiki_categories (id) ON DELETE SET NULL,
    position    INT  NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, parent_id, name)
);

CREATE INDEX idx_wiki_categories_tenant_id ON wiki_categories (tenant_id);

-- ============================================================================
-- wiki_articles
-- ============================================================================
CREATE TABLE wiki_articles (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL,
    title          TEXT NOT NULL,
    slug           TEXT NOT NULL,
    content        JSONB NOT NULL DEFAULT '{}',
    search_vector  TSVECTOR,
    author_id      UUID NOT NULL,
    category_id    UUID NULL REFERENCES wiki_categories (id) ON DELETE SET NULL,
    published      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, slug)
);

CREATE INDEX idx_wiki_articles_tenant_id   ON wiki_articles (tenant_id);
CREATE INDEX idx_wiki_articles_author_id   ON wiki_articles (author_id);
CREATE INDEX idx_wiki_articles_category_id ON wiki_articles (category_id);
CREATE INDEX idx_wiki_articles_published   ON wiki_articles (tenant_id, published);
CREATE INDEX idx_wiki_articles_search      ON wiki_articles USING GIN (search_vector);

-- FTS trigger
CREATE OR REPLACE FUNCTION wiki_articles_search_vector_update() RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('german', COALESCE(NEW.title, '')), 'A') ||
        setweight(to_tsvector('german', COALESCE(NEW.content->>'plain', '')), 'B');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER wiki_articles_search_update
    BEFORE INSERT OR UPDATE ON wiki_articles
    FOR EACH ROW EXECUTE FUNCTION wiki_articles_search_vector_update();

-- ============================================================================
-- wiki_versions
-- ============================================================================
CREATE TABLE wiki_versions (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    article_id     UUID NOT NULL REFERENCES wiki_articles (id) ON DELETE CASCADE,
    version_number INT  NOT NULL,
    content        JSONB NOT NULL DEFAULT '{}',
    changed_by     UUID NULL,
    changed_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (article_id, version_number)
);

CREATE INDEX idx_wiki_versions_article_id ON wiki_versions (article_id);

-- ============================================================================
-- wiki_attachments
-- ============================================================================
CREATE TABLE wiki_attachments (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    article_id   UUID NOT NULL REFERENCES wiki_articles (id) ON DELETE CASCADE,
    file_ref     TEXT NOT NULL,
    mime         TEXT NOT NULL DEFAULT '',
    size         BIGINT NOT NULL DEFAULT 0,
    uploaded_by  UUID NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_wiki_attachments_article_id ON wiki_attachments (article_id);

-- ============================================================================
-- wiki_share_tokens
-- ============================================================================
CREATE TABLE wiki_share_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    article_id  UUID NOT NULL REFERENCES wiki_articles (id) ON DELETE CASCADE,
    token       TEXT NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NULL,
    permissions TEXT[] NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_wiki_share_tokens_article_id ON wiki_share_tokens (article_id);
CREATE INDEX idx_wiki_share_tokens_token      ON wiki_share_tokens (token);
