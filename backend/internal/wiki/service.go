package wiki

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/linksecret"
)

// Service handles wiki business logic.
type Service struct {
	repo Repository
}

// NewService creates a new wiki service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ============================================================================
// Input types
// ============================================================================

// CreateArticleInput contains the data needed to create a wiki article.
type CreateArticleInput struct {
	TenantID   uuid.UUID
	Title      string
	Slug       string
	Content    json.RawMessage
	AuthorID   uuid.UUID
	CategoryID *uuid.UUID
	Published  bool
}

// UpdateArticleInput contains the data that can be updated on an article.
type UpdateArticleInput struct {
	TenantID   uuid.UUID
	ArticleID  uuid.UUID
	Title      *string
	Slug       *string
	Content    json.RawMessage
	CategoryID *uuid.UUID // uuid.Nil to clear
	Published  *bool
	EditorID   uuid.UUID // user making the change (saved as version author)
	// OwnerID narrows the edit to articles this user wrote (scope "own");
	// nil leaves it tenant-wide. A foreign article answers ErrArticleNotFound.
	OwnerID *uuid.UUID
}

// ListArticlesInput contains filtering options for listing articles.
type ListArticlesInput struct {
	TenantID   uuid.UUID
	CategoryID *uuid.UUID
	AuthorID   *uuid.UUID
	Published  *bool
	Search     string
	Page       int
	PageSize   int
	SortBy     string
	SortDesc   bool
}

// UploadInput contains data for uploading an attachment.
type UploadInput struct {
	TenantID   uuid.UUID
	ArticleID  uuid.UUID
	FileRef    string
	Mime       string
	Size       int64
	UploadedBy *uuid.UUID
	OwnerID    *uuid.UUID // see UpdateArticleInput.OwnerID
}

// CreateCategoryInput contains data for creating a category.
type CreateCategoryInput struct {
	TenantID uuid.UUID
	Name     string
	ParentID *uuid.UUID
	Position int
}

// UpdateCategoryInput contains data for updating a category.
type UpdateCategoryInput struct {
	TenantID   uuid.UUID
	CategoryID uuid.UUID
	Name       *string
	ParentID   *uuid.UUID // uuid.Nil to clear parent
	Position   *int
}

// ============================================================================
// Article methods
// ============================================================================

// CreateArticle creates a new wiki article.
func (s *Service) CreateArticle(ctx context.Context, input CreateArticleInput) (*Article, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, ErrInvalidContent
	}

	slug := strings.TrimSpace(input.Slug)
	if slug == "" {
		slug = slugify(title)
	}

	// Validate content is valid JSON
	if len(input.Content) > 0 {
		if !json.Valid(input.Content) {
			return nil, ErrInvalidContent
		}
	}

	// Check slug uniqueness
	exists, err := s.repo.SlugExists(ctx, input.TenantID, slug, nil)
	if err != nil {
		return nil, fmt.Errorf("check slug: %w", err)
	}
	if exists {
		return nil, ErrSlugTaken
	}

	// Validate category exists if provided
	if input.CategoryID != nil {
		if _, catErr := s.repo.GetCategory(ctx, input.TenantID, *input.CategoryID); catErr != nil {
			return nil, ErrCategoryNotFound
		}
	}

	content := []byte("{}")
	if len(input.Content) > 0 {
		content = input.Content
	}

	now := time.Now()
	article := &Article{
		ID:         uuid.New(),
		TenantID:   input.TenantID,
		Title:      title,
		Slug:       slug,
		Content:    content,
		AuthorID:   input.AuthorID,
		CategoryID: input.CategoryID,
		Published:  input.Published,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if createErr := s.repo.CreateArticle(ctx, article); createErr != nil {
		return nil, fmt.Errorf("create article: %w", createErr)
	}

	slog.Info("wiki article created",
		"article_id", article.ID,
		"tenant_id", article.TenantID,
		"title", article.Title,
	)

	return article, nil
}

// UpdateArticle updates an existing article and saves a version snapshot.
func (s *Service) UpdateArticle(ctx context.Context, input UpdateArticleInput) (*Article, error) {
	article, err := s.loadEditableArticle(ctx, input.TenantID, input.ArticleID, input.OwnerID)
	if err != nil {
		return nil, err
	}

	if input.Title != nil {
		t := strings.TrimSpace(*input.Title)
		if t == "" {
			return nil, ErrInvalidContent
		}
		article.Title = t
	}

	if input.Slug != nil {
		slug := strings.TrimSpace(*input.Slug)
		if slug != "" && slug != article.Slug {
			exists, slugErr := s.repo.SlugExists(ctx, input.TenantID, slug, &article.ID)
			if slugErr != nil {
				return nil, fmt.Errorf("check slug: %w", slugErr)
			}
			if exists {
				return nil, ErrSlugTaken
			}
			article.Slug = slug
		}
	}

	if len(input.Content) > 0 {
		if !json.Valid(input.Content) {
			return nil, ErrInvalidContent
		}
		// Save current content as version before overwriting
		latestNum, vErr := s.repo.GetLatestVersionNumber(ctx, input.TenantID, article.ID)
		if vErr != nil {
			return nil, fmt.Errorf("get version number: %w", vErr)
		}
		version := &Version{
			ID:            uuid.New(),
			TenantID:      input.TenantID,
			ArticleID:     article.ID,
			VersionNumber: latestNum + 1,
			Content:       article.Content,
			ChangedBy:     &input.EditorID,
			ChangedAt:     time.Now(),
		}
		if vCreateErr := s.repo.CreateVersion(ctx, version); vCreateErr != nil {
			return nil, fmt.Errorf("create version: %w", vCreateErr)
		}
		article.Content = input.Content
	}

	if input.CategoryID != nil {
		if *input.CategoryID == uuid.Nil {
			article.CategoryID = nil
		} else {
			if _, catErr := s.repo.GetCategory(ctx, input.TenantID, *input.CategoryID); catErr != nil {
				return nil, ErrCategoryNotFound
			}
			article.CategoryID = input.CategoryID
		}
	}

	if input.Published != nil {
		article.Published = *input.Published
	}

	article.UpdatedAt = time.Now()

	if updateErr := s.repo.UpdateArticle(ctx, article); updateErr != nil {
		return nil, fmt.Errorf("update article: %w", updateErr)
	}

	slog.Info("wiki article updated",
		"article_id", article.ID,
		"tenant_id", article.TenantID,
	)

	return article, nil
}

// DeleteArticle removes an article. With ownerID set (scope "own") only an
// article the caller wrote is removed; any other answers ErrArticleNotFound.
func (s *Service) DeleteArticle(ctx context.Context, tenantID, articleID uuid.UUID, ownerID *uuid.UUID) error {
	article, err := s.loadEditableArticle(ctx, tenantID, articleID, ownerID)
	if err != nil {
		return err
	}

	if delErr := s.repo.DeleteArticle(ctx, tenantID, articleID); delErr != nil {
		return fmt.Errorf("delete article: %w", delErr)
	}

	slog.Info("wiki article deleted",
		"article_id", article.ID,
		"tenant_id", tenantID,
	)

	return nil
}

// GetArticle retrieves an article by ID.
func (s *Service) GetArticle(ctx context.Context, tenantID, articleID uuid.UUID) (*Article, error) {
	return s.repo.GetArticleByID(ctx, tenantID, articleID)
}

// ListArticles retrieves articles with optional filtering and pagination.
func (s *Service) ListArticles(ctx context.Context, input ListArticlesInput) ([]*Article, int, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 100 {
		input.PageSize = 20
	}

	offset := (input.Page - 1) * input.PageSize
	filter := ListArticlesFilter{
		CategoryID: input.CategoryID,
		AuthorID:   input.AuthorID,
		Published:  input.Published,
		Search:     input.Search,
		SortBy:     input.SortBy,
		SortDesc:   input.SortDesc,
	}

	return s.repo.ListArticles(ctx, input.TenantID, filter, offset, input.PageSize)
}

// SearchArticles performs full-text search on published articles.
func (s *Service) SearchArticles(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]*Article, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	tsquery := BuildSearchQuery(query)
	if tsquery == "" {
		return nil, nil
	}

	return s.repo.SearchArticles(ctx, tenantID, tsquery, limit)
}

// ============================================================================
// Version methods
// ============================================================================

// ListVersions returns all versions for an article.
func (s *Service) ListVersions(ctx context.Context, tenantID, articleID uuid.UUID) ([]*Version, error) {
	return s.repo.ListVersions(ctx, tenantID, articleID)
}

// GetVersion returns a specific version.
func (s *Service) GetVersion(ctx context.Context, tenantID, versionID uuid.UUID) (*Version, error) {
	return s.repo.GetVersion(ctx, tenantID, versionID)
}

// RestoreVersion restores an article to a previous version's content. ownerID
// narrows it like UpdateArticleInput.OwnerID; the article is checked before the
// version, so a foreign article never confirms that a version id exists. A
// version of another article answers ErrVersionNotFound.
func (s *Service) RestoreVersion(ctx context.Context, tenantID, articleID, versionID uuid.UUID, editorID uuid.UUID, ownerID *uuid.UUID) (*Article, error) {
	if _, err := s.loadEditableArticle(ctx, tenantID, articleID, ownerID); err != nil {
		return nil, err
	}
	version, err := s.repo.GetVersion(ctx, tenantID, versionID)
	if err != nil {
		return nil, err
	}
	if version.ArticleID != articleID {
		return nil, ErrVersionNotFound
	}

	return s.UpdateArticle(ctx, UpdateArticleInput{
		TenantID:  tenantID,
		ArticleID: articleID,
		Content:   version.Content,
		EditorID:  editorID,
		OwnerID:   ownerID,
	})
}

// loadEditableArticle loads an article the caller may change. With ownerID set
// (scope "own") an article written by someone else answers ErrArticleNotFound
// rather than a permission error, so the caller learns nothing about the id.
func (s *Service) loadEditableArticle(ctx context.Context, tenantID, articleID uuid.UUID, ownerID *uuid.UUID) (*Article, error) {
	article, err := s.repo.GetArticleByID(ctx, tenantID, articleID)
	if err != nil {
		return nil, err
	}
	if ownerID != nil && article.AuthorID != *ownerID {
		return nil, ErrArticleNotFound
	}
	return article, nil
}

// ============================================================================
// Attachment methods
// ============================================================================

// UploadAttachment records an uploaded attachment.
func (s *Service) UploadAttachment(ctx context.Context, input UploadInput) (*Attachment, error) {
	if strings.TrimSpace(input.FileRef) == "" {
		return nil, ErrInvalidContent
	}
	if input.OwnerID != nil {
		if _, err := s.loadEditableArticle(ctx, input.TenantID, input.ArticleID, input.OwnerID); err != nil {
			return nil, err
		}
	}

	now := time.Now()
	attachment := &Attachment{
		ID:         uuid.New(),
		TenantID:   input.TenantID,
		ArticleID:  input.ArticleID,
		FileRef:    input.FileRef,
		Mime:       input.Mime,
		Size:       input.Size,
		UploadedBy: input.UploadedBy,
		CreatedAt:  now,
	}

	if createErr := s.repo.CreateAttachment(ctx, attachment); createErr != nil {
		return nil, fmt.Errorf("create attachment: %w", createErr)
	}

	slog.Info("wiki attachment uploaded",
		"attachment_id", attachment.ID,
		"article_id", attachment.ArticleID,
	)

	return attachment, nil
}

// ListAttachments returns all attachments for an article.
func (s *Service) ListAttachments(ctx context.Context, tenantID, articleID uuid.UUID) ([]*Attachment, error) {
	return s.repo.ListAttachments(ctx, tenantID, articleID)
}

// DeleteAttachment removes an attachment. With ownerID set (scope "own") the
// attachment must hang on an article the caller wrote, otherwise it answers
// ErrAttachmentNotFound.
func (s *Service) DeleteAttachment(ctx context.Context, tenantID, attachmentID uuid.UUID, ownerID *uuid.UUID) error {
	if ownerID != nil {
		articleID, err := s.repo.GetAttachmentArticleID(ctx, tenantID, attachmentID)
		if err != nil {
			return err
		}
		if _, err := s.loadEditableArticle(ctx, tenantID, articleID, ownerID); err != nil {
			if errors.Is(err, ErrArticleNotFound) {
				return ErrAttachmentNotFound
			}
			return err
		}
	}
	if delErr := s.repo.DeleteAttachment(ctx, tenantID, attachmentID); delErr != nil {
		return fmt.Errorf("delete attachment: %w", delErr)
	}

	slog.Info("wiki attachment deleted", "attachment_id", attachmentID)

	return nil
}

// ============================================================================
// Category methods
// ============================================================================

// ListCategories returns all categories for a tenant.
func (s *Service) ListCategories(ctx context.Context, tenantID uuid.UUID) ([]*Category, error) {
	return s.repo.ListCategories(ctx, tenantID)
}

// CreateCategory creates a new wiki category.
func (s *Service) CreateCategory(ctx context.Context, input CreateCategoryInput) (*Category, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrInvalidContent
	}

	// Validate parent exists if provided
	if input.ParentID != nil {
		if _, parentErr := s.repo.GetCategory(ctx, input.TenantID, *input.ParentID); parentErr != nil {
			return nil, ErrCategoryNotFound
		}
	}

	now := time.Now()
	category := &Category{
		ID:        uuid.New(),
		TenantID:  input.TenantID,
		Name:      name,
		ParentID:  input.ParentID,
		Position:  input.Position,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if createErr := s.repo.CreateCategory(ctx, category); createErr != nil {
		return nil, fmt.Errorf("create category: %w", createErr)
	}

	slog.Info("wiki category created",
		"category_id", category.ID,
		"tenant_id", category.TenantID,
		"name", category.Name,
	)

	return category, nil
}

// DeleteCategory removes a wiki category.
// Articles in this category will have their category_id set to NULL by FK ON DELETE SET NULL.
func (s *Service) DeleteCategory(ctx context.Context, tenantID, categoryID uuid.UUID) error {
	if _, err := s.repo.GetCategory(ctx, tenantID, categoryID); err != nil {
		return err
	}

	if delErr := s.repo.DeleteCategory(ctx, tenantID, categoryID); delErr != nil {
		return fmt.Errorf("delete category: %w", delErr)
	}

	slog.Info("wiki category deleted",
		"category_id", categoryID,
		"tenant_id", tenantID,
	)

	return nil
}

// UpdateCategory updates a wiki category's fields.
func (s *Service) UpdateCategory(ctx context.Context, input UpdateCategoryInput) (*Category, error) {
	category, err := s.repo.GetCategory(ctx, input.TenantID, input.CategoryID)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, ErrInvalidContent
		}
		category.Name = name
	}

	if input.ParentID != nil {
		if *input.ParentID == uuid.Nil {
			category.ParentID = nil
		} else {
			if _, parentErr := s.repo.GetCategory(ctx, input.TenantID, *input.ParentID); parentErr != nil {
				return nil, ErrCategoryNotFound
			}
			category.ParentID = input.ParentID
		}
	}

	if input.Position != nil {
		category.Position = *input.Position
	}

	category.UpdatedAt = time.Now()

	if updateErr := s.repo.UpdateCategory(ctx, category); updateErr != nil {
		return nil, fmt.Errorf("update category: %w", updateErr)
	}

	slog.Info("wiki category updated",
		"category_id", category.ID,
		"tenant_id", category.TenantID,
	)

	return category, nil
}

// ============================================================================
// Share token methods
// ============================================================================

// CreateShareTokenInput contains data for creating a share token.
type CreateShareTokenInput struct {
	TenantID    uuid.UUID
	ArticleID   uuid.UUID
	ExpiresAt   *time.Time
	Permissions []string
	// CreatedBy is who minted the link. Optional -- the gateway fills it from
	// the authenticated caller, and a nil value is recorded as "unknown"
	// rather than refused, so a missing subject never blocks a share.
	CreatedBy *uuid.UUID
	// OwnerID set (scope "own") shares only an article the caller wrote; any
	// other answers ErrArticleNotFound.
	OwnerID *uuid.UUID
}

// CreateShareToken creates a new share token for an article.
func (s *Service) CreateShareToken(ctx context.Context, input CreateShareTokenInput) (*ShareToken, error) {
	// Same rules as form share links: a link that is born expired is a mistake
	// the caller should hear about, not a dead row, and "read" is the only
	// permission a redeemed link ever grants ("public" is a display marker).
	now := time.Now()
	if input.ExpiresAt != nil && !input.ExpiresAt.After(now) {
		return nil, fmt.Errorf("expires_at must be in the future: %w", ErrInvalidShareLink)
	}
	expiresAt, err := linksecret.Expiry(input.ExpiresAt, now)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", err, ErrInvalidShareLink)
	}
	public := false
	for _, p := range input.Permissions {
		switch p {
		case sharePermissionRead:
		case sharePermissionPublic:
			public = true
		default:
			return nil, fmt.Errorf("unsupported permission %q: %w", p, ErrInvalidShareLink)
		}
	}

	// Verify article exists in tenant (and is the caller's at scope own)
	if _, err := s.loadEditableArticle(ctx, input.TenantID, input.ArticleID, input.OwnerID); err != nil {
		return nil, err
	}

	permissions := []string{sharePermissionRead}
	if public {
		permissions = append(permissions, sharePermissionPublic)
	}

	secret, err := linksecret.New()
	if err != nil {
		return nil, err
	}

	token := &ShareToken{
		ID:          uuid.New(),
		TenantID:    input.TenantID,
		ArticleID:   input.ArticleID,
		Token:       secret,
		ExpiresAt:   &expiresAt,
		Permissions: permissions,
		CreatedAt:   now,
		CreatedBy:   input.CreatedBy,
	}

	if createErr := s.repo.CreateShareToken(ctx, token); createErr != nil {
		return nil, fmt.Errorf("create share token: %w", createErr)
	}

	slog.Info("wiki share token created",
		"token_id", token.ID,
		"article_id", token.ArticleID,
	)

	return token, nil
}

// ListShareTokens returns all share tokens for an article, verifying the article belongs to the tenant.
func (s *Service) ListShareTokens(ctx context.Context, tenantID, articleID uuid.UUID) ([]*ShareToken, error) {
	if _, err := s.repo.GetArticleByID(ctx, tenantID, articleID); err != nil {
		return nil, err
	}
	return s.repo.ListShareTokensByArticle(ctx, tenantID, articleID)
}

// RevokeShareToken cuts a share link. The row survives with revoked_at set --
// see 000297 for why the revocation is soft. Revoking an already revoked token
// succeeds without moving the original timestamp. With ownerID set (scope
// "own") the link must belong to an article the caller wrote, otherwise it
// answers ErrShareTokenNotFound.
func (s *Service) RevokeShareToken(ctx context.Context, tenantID, tokenID uuid.UUID, ownerID *uuid.UUID) error {
	if ownerID != nil {
		articleID, err := s.repo.GetShareTokenArticleID(ctx, tenantID, tokenID)
		if err != nil {
			return err
		}
		if _, err := s.loadEditableArticle(ctx, tenantID, articleID, ownerID); err != nil {
			if errors.Is(err, ErrArticleNotFound) {
				return ErrShareTokenNotFound
			}
			return err
		}
	}
	if revErr := s.repo.RevokeShareToken(ctx, tenantID, tokenID, time.Now().UTC()); revErr != nil {
		return fmt.Errorf("revoke share token: %w", revErr)
	}

	slog.Info("wiki share token revoked",
		"token_id", tokenID,
		"tenant_id", tenantID,
	)

	return nil
}

// ============================================================================
// Helpers
// ============================================================================

// slugify produces a URL-safe slug from a title.
func slugify(title string) string {
	slug := strings.ToLower(strings.TrimSpace(title))
	var b strings.Builder
	prevHyphen := false
	for _, r := range slug {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevHyphen = false
		case r == ' ' || r == '-' || r == '_':
			if !prevHyphen {
				b.WriteRune('-')
				prevHyphen = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}
