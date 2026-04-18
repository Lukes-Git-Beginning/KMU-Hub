-- Sprint 1 S1.1 Wiki — rollback

DROP INDEX IF EXISTS idx_wiki_share_tokens_token;
DROP INDEX IF EXISTS idx_wiki_share_tokens_article_id;
DROP TABLE IF EXISTS wiki_share_tokens;

DROP INDEX IF EXISTS idx_wiki_attachments_article_id;
DROP TABLE IF EXISTS wiki_attachments;

DROP INDEX IF EXISTS idx_wiki_versions_article_id;
DROP TABLE IF EXISTS wiki_versions;

DROP TRIGGER  IF EXISTS wiki_articles_search_update ON wiki_articles;
DROP FUNCTION IF EXISTS wiki_articles_search_vector_update();

DROP INDEX IF EXISTS idx_wiki_articles_search;
DROP INDEX IF EXISTS idx_wiki_articles_published;
DROP INDEX IF EXISTS idx_wiki_articles_category_id;
DROP INDEX IF EXISTS idx_wiki_articles_author_id;
DROP INDEX IF EXISTS idx_wiki_articles_tenant_id;
DROP TABLE IF EXISTS wiki_articles;

DROP INDEX IF EXISTS idx_wiki_categories_tenant_id;
DROP TABLE IF EXISTS wiki_categories;
