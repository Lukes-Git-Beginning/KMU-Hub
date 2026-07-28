package wiki

import (
	"context"

	"github.com/google/uuid"
)

// ListArticlesFilter contains filtering options for listing articles.
type ListArticlesFilter struct {
	CategoryID *uuid.UUID
	AuthorID   *uuid.UUID
	Published  *bool
	Search     string
	SortBy     string // title, created_at, updated_at
	SortDesc   bool
}

// Repository defines the interface for wiki persistence.
type Repository interface {
	// Articles
	CreateArticle(ctx context.Context, article *Article) error
	UpdateArticle(ctx context.Context, article *Article) error
	GetArticleByID(ctx context.Context, tenantID, articleID uuid.UUID) (*Article, error)
	GetArticleBySlug(ctx context.Context, tenantID uuid.UUID, slug string) (*Article, error)
	ListArticles(ctx context.Context, tenantID uuid.UUID, filter ListArticlesFilter, offset, limit int) ([]*Article, int, error)
	DeleteArticle(ctx context.Context, tenantID, articleID uuid.UUID) error
	SearchArticles(ctx context.Context, tenantID uuid.UUID, tsquery string, limit int) ([]*Article, error)
	SlugExists(ctx context.Context, tenantID uuid.UUID, slug string, excludeID *uuid.UUID) (bool, error)

	// Versions. Reads take an explicit tenantID for the same defense-in-depth
	// reason as the article reads above — not just relying on RLS.
	CreateVersion(ctx context.Context, version *Version) error
	ListVersions(ctx context.Context, tenantID, articleID uuid.UUID) ([]*Version, error)
	GetVersion(ctx context.Context, tenantID, versionID uuid.UUID) (*Version, error)
	GetLatestVersionNumber(ctx context.Context, tenantID, articleID uuid.UUID) (int, error)

	// Attachments
	CreateAttachment(ctx context.Context, attachment *Attachment) error
	ListAttachments(ctx context.Context, tenantID, articleID uuid.UUID) ([]*Attachment, error)
	DeleteAttachment(ctx context.Context, tenantID, attachmentID uuid.UUID) error

	// Categories
	CreateCategory(ctx context.Context, category *Category) error
	ListCategories(ctx context.Context, tenantID uuid.UUID) ([]*Category, error)
	GetCategory(ctx context.Context, tenantID, categoryID uuid.UUID) (*Category, error)
	DeleteCategory(ctx context.Context, tenantID, categoryID uuid.UUID) error
	UpdateCategory(ctx context.Context, category *Category) error

	// Share tokens. Delete and list take an explicit tenantID for the same
	// defense-in-depth reason as the reads above.
	CreateShareToken(ctx context.Context, token *ShareToken) error
	DeleteShareToken(ctx context.Context, tenantID, tokenID uuid.UUID) error
	ListShareTokensByArticle(ctx context.Context, tenantID, articleID uuid.UUID) ([]*ShareToken, error)
}
