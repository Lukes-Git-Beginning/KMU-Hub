package wiki

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// ============================================================================
// Articles
// ============================================================================

func (r *PostgresRepository) CreateArticle(ctx context.Context, article *Article) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO wiki_articles
		    (id, tenant_id, title, slug, content, author_id, category_id, published, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		article.ID, article.TenantID, article.Title, article.Slug,
		article.Content, article.AuthorID, article.CategoryID,
		article.Published, article.CreatedAt, article.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) UpdateArticle(ctx context.Context, article *Article) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE wiki_articles
		 SET title = $1, slug = $2, content = $3, category_id = $4,
		     published = $5, updated_at = $6
		 WHERE id = $7 AND tenant_id = $8`,
		article.Title, article.Slug, article.Content, article.CategoryID,
		article.Published, article.UpdatedAt, article.ID, article.TenantID,
	)
	return err
}

func (r *PostgresRepository) GetArticleByID(ctx context.Context, tenantID, articleID uuid.UUID) (*Article, error) {
	return r.scanArticle(r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, title, slug, content, author_id, category_id, published, created_at, updated_at
		 FROM wiki_articles WHERE id = $1 AND tenant_id = $2`,
		articleID, tenantID,
	))
}

func (r *PostgresRepository) GetArticleBySlug(ctx context.Context, tenantID uuid.UUID, slug string) (*Article, error) {
	return r.scanArticle(r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, title, slug, content, author_id, category_id, published, created_at, updated_at
		 FROM wiki_articles WHERE slug = $1 AND tenant_id = $2`,
		slug, tenantID,
	))
}

func (r *PostgresRepository) ListArticles(ctx context.Context, tenantID uuid.UUID, filter ListArticlesFilter, offset, limit int) ([]*Article, int, error) {
	var conditions []string
	var args []any
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argNum))
	args = append(args, tenantID)
	argNum++

	if filter.CategoryID != nil {
		conditions = append(conditions, fmt.Sprintf("category_id = $%d", argNum))
		args = append(args, *filter.CategoryID)
		argNum++
	}

	if filter.AuthorID != nil {
		conditions = append(conditions, fmt.Sprintf("author_id = $%d", argNum))
		args = append(args, *filter.AuthorID)
		argNum++
	}

	if filter.Published != nil {
		conditions = append(conditions, fmt.Sprintf("published = $%d", argNum))
		args = append(args, *filter.Published)
		argNum++
	}

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("LOWER(title) LIKE $%d", argNum))
		args = append(args, "%"+strings.ToLower(filter.Search)+"%")
		argNum++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM wiki_articles %s", whereClause)
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count articles: %w", err)
	}

	orderBy := "created_at"
	switch filter.SortBy {
	case "title":
		orderBy = "LOWER(title)"
	case "updated_at":
		orderBy = "updated_at"
	}
	direction := "DESC"
	if !filter.SortDesc {
		direction = "ASC"
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, title, slug, content, author_id, category_id, published, created_at, updated_at
		FROM wiki_articles %s
		ORDER BY %s %s NULLS LAST
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, direction, argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list articles: %w", err)
	}
	defer rows.Close()

	var articles []*Article
	for rows.Next() {
		a, scanErr := r.scanArticleFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		articles = append(articles, a)
	}

	return articles, total, rows.Err()
}

func (r *PostgresRepository) DeleteArticle(ctx context.Context, tenantID, articleID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM wiki_articles WHERE id = $1 AND tenant_id = $2`,
		articleID, tenantID,
	)
	return err
}

func (r *PostgresRepository) SearchArticles(ctx context.Context, tenantID uuid.UUID, tsquery string, limit int) ([]*Article, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, title, slug, content, author_id, category_id, published, created_at, updated_at
		 FROM wiki_articles
		 WHERE tenant_id = $1 AND published = TRUE AND search_vector @@ to_tsquery('german', $2)
		 ORDER BY ts_rank(search_vector, to_tsquery('german', $2)) DESC
		 LIMIT $3`,
		tenantID, tsquery, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("search articles: %w", err)
	}
	defer rows.Close()

	var articles []*Article
	for rows.Next() {
		a, scanErr := r.scanArticleFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		articles = append(articles, a)
	}

	return articles, rows.Err()
}

func (r *PostgresRepository) SlugExists(ctx context.Context, tenantID uuid.UUID, slug string, excludeID *uuid.UUID) (bool, error) {
	var exists bool
	if excludeID != nil {
		err := r.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM wiki_articles WHERE tenant_id = $1 AND slug = $2 AND id <> $3)`,
			tenantID, slug, *excludeID,
		).Scan(&exists)
		return exists, err
	}
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM wiki_articles WHERE tenant_id = $1 AND slug = $2)`,
		tenantID, slug,
	).Scan(&exists)
	return exists, err
}

// ============================================================================
// Versions
// ============================================================================

func (r *PostgresRepository) CreateVersion(ctx context.Context, version *Version) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO wiki_versions (id, article_id, version_number, content, changed_by, changed_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		version.ID, version.ArticleID, version.VersionNumber,
		version.Content, version.ChangedBy, version.ChangedAt,
	)
	return err
}

func (r *PostgresRepository) ListVersions(ctx context.Context, articleID uuid.UUID) ([]*Version, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, article_id, version_number, content, changed_by, changed_at
		 FROM wiki_versions WHERE article_id = $1
		 ORDER BY version_number DESC`,
		articleID,
	)
	if err != nil {
		return nil, fmt.Errorf("list versions: %w", err)
	}
	defer rows.Close()

	var versions []*Version
	for rows.Next() {
		v, scanErr := r.scanVersion(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		versions = append(versions, v)
	}

	return versions, rows.Err()
}

func (r *PostgresRepository) GetVersion(ctx context.Context, versionID uuid.UUID) (*Version, error) {
	var v Version
	err := r.pool.QueryRow(ctx,
		`SELECT id, article_id, version_number, content, changed_by, changed_at
		 FROM wiki_versions WHERE id = $1`,
		versionID,
	).Scan(&v.ID, &v.ArticleID, &v.VersionNumber, &v.Content, &v.ChangedBy, &v.ChangedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrVersionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get version: %w", err)
	}
	return &v, nil
}

func (r *PostgresRepository) GetLatestVersionNumber(ctx context.Context, articleID uuid.UUID) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(version_number), 0) FROM wiki_versions WHERE article_id = $1`,
		articleID,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("get latest version number: %w", err)
	}
	return n, nil
}

// ============================================================================
// Attachments
// ============================================================================

func (r *PostgresRepository) CreateAttachment(ctx context.Context, attachment *Attachment) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO wiki_attachments (id, article_id, file_ref, mime, size, uploaded_by, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		attachment.ID, attachment.ArticleID, attachment.FileRef,
		attachment.Mime, attachment.Size, attachment.UploadedBy, attachment.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) ListAttachments(ctx context.Context, articleID uuid.UUID) ([]*Attachment, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, article_id, file_ref, mime, size, uploaded_by, created_at
		 FROM wiki_attachments WHERE article_id = $1
		 ORDER BY created_at DESC`,
		articleID,
	)
	if err != nil {
		return nil, fmt.Errorf("list attachments: %w", err)
	}
	defer rows.Close()

	var attachments []*Attachment
	for rows.Next() {
		a, scanErr := r.scanAttachment(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		attachments = append(attachments, a)
	}

	return attachments, rows.Err()
}

func (r *PostgresRepository) GetAttachment(ctx context.Context, attachmentID uuid.UUID) (*Attachment, error) {
	var a Attachment
	err := r.pool.QueryRow(ctx,
		`SELECT id, article_id, file_ref, mime, size, uploaded_by, created_at
		 FROM wiki_attachments WHERE id = $1`,
		attachmentID,
	).Scan(&a.ID, &a.ArticleID, &a.FileRef, &a.Mime, &a.Size, &a.UploadedBy, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrArticleNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get attachment: %w", err)
	}
	return &a, nil
}

func (r *PostgresRepository) DeleteAttachment(ctx context.Context, attachmentID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM wiki_attachments WHERE id = $1`, attachmentID)
	return err
}

// ============================================================================
// Categories
// ============================================================================

func (r *PostgresRepository) CreateCategory(ctx context.Context, category *Category) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO wiki_categories (id, tenant_id, name, parent_id, position, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		category.ID, category.TenantID, category.Name,
		category.ParentID, category.Position, category.CreatedAt, category.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) ListCategories(ctx context.Context, tenantID uuid.UUID) ([]*Category, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, parent_id, position, created_at, updated_at
		 FROM wiki_categories WHERE tenant_id = $1
		 ORDER BY position ASC, name ASC`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	var categories []*Category
	for rows.Next() {
		c, scanErr := r.scanCategory(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		categories = append(categories, c)
	}

	return categories, rows.Err()
}

func (r *PostgresRepository) GetCategory(ctx context.Context, tenantID, categoryID uuid.UUID) (*Category, error) {
	var c Category
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, parent_id, position, created_at, updated_at
		 FROM wiki_categories WHERE id = $1 AND tenant_id = $2`,
		categoryID, tenantID,
	).Scan(&c.ID, &c.TenantID, &c.Name, &c.ParentID, &c.Position, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCategoryNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get category: %w", err)
	}
	return &c, nil
}

func (r *PostgresRepository) DeleteCategory(ctx context.Context, tenantID, categoryID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM wiki_categories WHERE id = $1 AND tenant_id = $2`,
		categoryID, tenantID,
	)
	return err
}

// ============================================================================
// Share tokens
// ============================================================================

func (r *PostgresRepository) CreateShareToken(ctx context.Context, token *ShareToken) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO wiki_share_tokens (id, article_id, token, expires_at, permissions, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		token.ID, token.ArticleID, token.Token,
		token.ExpiresAt, token.Permissions, token.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) GetShareToken(ctx context.Context, token string) (*ShareToken, error) {
	var t ShareToken
	err := r.pool.QueryRow(ctx,
		`SELECT id, article_id, token, expires_at, permissions, created_at
		 FROM wiki_share_tokens WHERE token = $1`,
		token,
	).Scan(&t.ID, &t.ArticleID, &t.Token, &t.ExpiresAt, &t.Permissions, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrArticleNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get share token: %w", err)
	}
	return &t, nil
}

func (r *PostgresRepository) DeleteShareToken(ctx context.Context, tokenID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM wiki_share_tokens WHERE id = $1`, tokenID)
	return err
}

// ============================================================================
// Scan helpers
// ============================================================================

func (r *PostgresRepository) scanArticle(row pgx.Row) (*Article, error) {
	var a Article
	err := row.Scan(
		&a.ID, &a.TenantID, &a.Title, &a.Slug, &a.Content,
		&a.AuthorID, &a.CategoryID, &a.Published, &a.CreatedAt, &a.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrArticleNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan article: %w", err)
	}
	return &a, nil
}

func (r *PostgresRepository) scanArticleFromRows(rows pgx.Rows) (*Article, error) {
	var a Article
	err := rows.Scan(
		&a.ID, &a.TenantID, &a.Title, &a.Slug, &a.Content,
		&a.AuthorID, &a.CategoryID, &a.Published, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan article row: %w", err)
	}
	return &a, nil
}

func (r *PostgresRepository) scanVersion(rows pgx.Rows) (*Version, error) {
	var v Version
	err := rows.Scan(&v.ID, &v.ArticleID, &v.VersionNumber, &v.Content, &v.ChangedBy, &v.ChangedAt)
	if err != nil {
		return nil, fmt.Errorf("scan version row: %w", err)
	}
	return &v, nil
}

func (r *PostgresRepository) scanAttachment(rows pgx.Rows) (*Attachment, error) {
	var a Attachment
	err := rows.Scan(&a.ID, &a.ArticleID, &a.FileRef, &a.Mime, &a.Size, &a.UploadedBy, &a.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan attachment row: %w", err)
	}
	return &a, nil
}

func (r *PostgresRepository) scanCategory(rows pgx.Rows) (*Category, error) {
	var c Category
	err := rows.Scan(&c.ID, &c.TenantID, &c.Name, &c.ParentID, &c.Position, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan category row: %w", err)
	}
	return &c, nil
}

// compile-time interface check
var _ Repository = (*PostgresRepository)(nil)
