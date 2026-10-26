package wiki

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// MockRepository
// ============================================================================

type mockRepository struct {
	articles    map[uuid.UUID]*Article
	versions    map[uuid.UUID]*Version
	attachments map[uuid.UUID]*Attachment
	categories  map[uuid.UUID]*Category
	tokens      map[string]*ShareToken

	// Slug -> articleID for uniqueness checks (tenant-scoped)
	slugs map[string]uuid.UUID

	createArticleErr error
	updateArticleErr error
	getArticleErr    error
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		articles:    make(map[uuid.UUID]*Article),
		versions:    make(map[uuid.UUID]*Version),
		attachments: make(map[uuid.UUID]*Attachment),
		categories:  make(map[uuid.UUID]*Category),
		tokens:      make(map[string]*ShareToken),
		slugs:       make(map[string]uuid.UUID),
	}
}

func (m *mockRepository) CreateArticle(ctx context.Context, article *Article) error {
	if m.createArticleErr != nil {
		return m.createArticleErr
	}
	m.articles[article.ID] = article
	key := article.TenantID.String() + ":" + article.Slug
	m.slugs[key] = article.ID
	return nil
}

func (m *mockRepository) UpdateArticle(ctx context.Context, article *Article) error {
	if m.updateArticleErr != nil {
		return m.updateArticleErr
	}
	m.articles[article.ID] = article
	return nil
}

func (m *mockRepository) GetArticleByID(ctx context.Context, tenantID, articleID uuid.UUID) (*Article, error) {
	if m.getArticleErr != nil {
		return nil, m.getArticleErr
	}
	a, ok := m.articles[articleID]
	if !ok || a.TenantID != tenantID {
		return nil, ErrArticleNotFound
	}
	return a, nil
}

func (m *mockRepository) GetArticleBySlug(ctx context.Context, tenantID uuid.UUID, slug string) (*Article, error) {
	for _, a := range m.articles {
		if a.TenantID == tenantID && a.Slug == slug {
			return a, nil
		}
	}
	return nil, ErrArticleNotFound
}

func (m *mockRepository) ListArticles(ctx context.Context, tenantID uuid.UUID, filter ListArticlesFilter, offset, limit int) ([]*Article, int, error) {
	var result []*Article
	for _, a := range m.articles {
		if a.TenantID != tenantID {
			continue
		}
		if filter.Published != nil && a.Published != *filter.Published {
			continue
		}
		result = append(result, a)
	}
	total := len(result)
	if offset >= total {
		return []*Article{}, total, nil
	}
	end := min(offset+limit, total)
	return result[offset:end], total, nil
}

func (m *mockRepository) DeleteArticle(ctx context.Context, tenantID, articleID uuid.UUID) error {
	delete(m.articles, articleID)
	return nil
}

func (m *mockRepository) SearchArticles(ctx context.Context, tenantID uuid.UUID, tsquery string, limit int) ([]*Article, error) {
	var result []*Article
	for _, a := range m.articles {
		if a.TenantID == tenantID && a.Published {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *mockRepository) SlugExists(ctx context.Context, tenantID uuid.UUID, slug string, excludeID *uuid.UUID) (bool, error) {
	key := tenantID.String() + ":" + slug
	existingID, ok := m.slugs[key]
	if !ok {
		return false, nil
	}
	if excludeID != nil && existingID == *excludeID {
		return false, nil
	}
	return true, nil
}

func (m *mockRepository) CreateVersion(ctx context.Context, version *Version) error {
	m.versions[version.ID] = version
	return nil
}

func (m *mockRepository) ListVersions(ctx context.Context, tenantID, articleID uuid.UUID) ([]*Version, error) {
	var result []*Version
	for _, v := range m.versions {
		if v.ArticleID == articleID {
			result = append(result, v)
		}
	}
	return result, nil
}

func (m *mockRepository) GetVersion(ctx context.Context, tenantID, versionID uuid.UUID) (*Version, error) {
	v, ok := m.versions[versionID]
	if !ok {
		return nil, ErrVersionNotFound
	}
	return v, nil
}

func (m *mockRepository) GetLatestVersionNumber(ctx context.Context, tenantID, articleID uuid.UUID) (int, error) {
	max := 0
	for _, v := range m.versions {
		if v.ArticleID == articleID && v.VersionNumber > max {
			max = v.VersionNumber
		}
	}
	return max, nil
}

func (m *mockRepository) CreateAttachment(ctx context.Context, attachment *Attachment) error {
	m.attachments[attachment.ID] = attachment
	return nil
}

func (m *mockRepository) ListAttachments(ctx context.Context, tenantID, articleID uuid.UUID) ([]*Attachment, error) {
	var result []*Attachment
	for _, a := range m.attachments {
		if a.ArticleID == articleID {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *mockRepository) DeleteAttachment(ctx context.Context, tenantID, attachmentID uuid.UUID) error {
	a, ok := m.attachments[attachmentID]
	if !ok || a.TenantID != tenantID {
		return ErrAttachmentNotFound
	}
	delete(m.attachments, attachmentID)
	return nil
}

func (m *mockRepository) GetAttachmentArticleID(ctx context.Context, tenantID, attachmentID uuid.UUID) (uuid.UUID, error) {
	a, ok := m.attachments[attachmentID]
	if !ok || a.TenantID != tenantID {
		return uuid.Nil, ErrAttachmentNotFound
	}
	return a.ArticleID, nil
}

func (m *mockRepository) CreateCategory(ctx context.Context, category *Category) error {
	m.categories[category.ID] = category
	return nil
}

func (m *mockRepository) ListCategories(ctx context.Context, tenantID uuid.UUID) ([]*Category, error) {
	var result []*Category
	for _, c := range m.categories {
		if c.TenantID == tenantID {
			result = append(result, c)
		}
	}
	return result, nil
}

func (m *mockRepository) GetCategory(ctx context.Context, tenantID, categoryID uuid.UUID) (*Category, error) {
	c, ok := m.categories[categoryID]
	if !ok || c.TenantID != tenantID {
		return nil, ErrCategoryNotFound
	}
	return c, nil
}

func (m *mockRepository) DeleteCategory(ctx context.Context, tenantID, categoryID uuid.UUID) error {
	delete(m.categories, categoryID)
	return nil
}

func (m *mockRepository) UpdateCategory(ctx context.Context, category *Category) error {
	m.categories[category.ID] = category
	return nil
}

func (m *mockRepository) CreateShareToken(ctx context.Context, token *ShareToken) error {
	m.tokens[token.Token] = token
	return nil
}

func (m *mockRepository) RevokeShareToken(ctx context.Context, tenantID, tokenID uuid.UUID, now time.Time) error {
	for _, t := range m.tokens {
		if t.ID == tokenID {
			if t.TenantID != tenantID {
				return ErrShareTokenNotFound
			}
			// Soft: the row stays and keeps its first revocation timestamp,
			// mirroring the COALESCE in the real UPDATE.
			if t.RevokedAt == nil {
				t.RevokedAt = &now
			}
			return nil
		}
	}
	return ErrShareTokenNotFound
}

func (m *mockRepository) ListShareTokensByArticle(ctx context.Context, tenantID, articleID uuid.UUID) ([]*ShareToken, error) {
	var result []*ShareToken
	for _, t := range m.tokens {
		if t.ArticleID == articleID && t.TenantID == tenantID {
			result = append(result, t)
		}
	}
	return result, nil
}

func (m *mockRepository) GetShareTokenArticleID(ctx context.Context, tenantID, tokenID uuid.UUID) (uuid.UUID, error) {
	for _, tok := range m.tokens {
		if tok.ID == tokenID && tok.TenantID == tenantID {
			return tok.ArticleID, nil
		}
	}
	return uuid.Nil, ErrShareTokenNotFound
}

func (m *mockRepository) GetShareTokenByToken(ctx context.Context, token string) (*ShareToken, error) {
	t, ok := m.tokens[token]
	if !ok {
		return nil, ErrShareTokenNotFound
	}
	return t, nil
}

// compile-time check
var _ Repository = (*mockRepository)(nil)

// ============================================================================
// Helpers
// ============================================================================

func validContent() json.RawMessage {
	return json.RawMessage(`{"type":"doc","content":[]}`)
}

func addArticle(repo *mockRepository, tenantID uuid.UUID, title, slug string, published bool) *Article {
	a := &Article{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Title:     title,
		Slug:      slug,
		Content:   []byte(`{}`),
		AuthorID:  uuid.New(),
		Published: published,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.articles[a.ID] = a
	key := tenantID.String() + ":" + slug
	repo.slugs[key] = a.ID
	return a
}

// ============================================================================
// CreateArticle Tests
// ============================================================================

func TestService_CreateArticle_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	input := CreateArticleInput{
		TenantID:  tenantID,
		Title:     "Handbuch",
		Slug:      "handbuch",
		Content:   validContent(),
		AuthorID:  uuid.New(),
		Published: false,
	}

	article, err := svc.CreateArticle(context.Background(), input)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, article.ID)
	assert.Equal(t, "Handbuch", article.Title)
	assert.Equal(t, "handbuch", article.Slug)
	assert.Equal(t, tenantID, article.TenantID)
}

func TestService_CreateArticle_SlugAutoGenerated(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	input := CreateArticleInput{
		TenantID: uuid.New(),
		Title:    "Mein Artikel",
		Slug:     "", // empty -> auto-generated
		Content:  validContent(),
		AuthorID: uuid.New(),
	}

	article, err := svc.CreateArticle(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, "mein-artikel", article.Slug)
}

func TestService_CreateArticle_EmptyTitle(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.CreateArticle(context.Background(), CreateArticleInput{
		TenantID: uuid.New(),
		Title:    "   ",
		Content:  validContent(),
		AuthorID: uuid.New(),
	})

	assert.ErrorIs(t, err, ErrInvalidContent)
}

func TestService_CreateArticle_InvalidJSON(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.CreateArticle(context.Background(), CreateArticleInput{
		TenantID: uuid.New(),
		Title:    "Artikel",
		Slug:     "artikel",
		Content:  json.RawMessage(`not-json`),
		AuthorID: uuid.New(),
	})

	assert.ErrorIs(t, err, ErrInvalidContent)
}

func TestService_CreateArticle_SlugTaken(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	addArticle(repo, tenantID, "Existing", "handbuch", false)

	_, err := svc.CreateArticle(context.Background(), CreateArticleInput{
		TenantID: tenantID,
		Title:    "New Artikel",
		Slug:     "handbuch",
		Content:  validContent(),
		AuthorID: uuid.New(),
	})

	assert.ErrorIs(t, err, ErrSlugTaken)
}

func TestService_CreateArticle_CategoryNotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	catID := uuid.New() // does not exist

	_, err := svc.CreateArticle(context.Background(), CreateArticleInput{
		TenantID:   uuid.New(),
		Title:      "Artikel",
		Slug:       "artikel",
		Content:    validContent(),
		AuthorID:   uuid.New(),
		CategoryID: &catID,
	})

	assert.ErrorIs(t, err, ErrCategoryNotFound)
}

func TestService_CreateArticle_WithCategory(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	cat := &Category{
		ID:       uuid.New(),
		TenantID: tenantID,
		Name:     "FAQ",
	}
	repo.categories[cat.ID] = cat

	article, err := svc.CreateArticle(context.Background(), CreateArticleInput{
		TenantID:   tenantID,
		Title:      "FAQ Artikel",
		Slug:       "faq-artikel",
		Content:    validContent(),
		AuthorID:   uuid.New(),
		CategoryID: &cat.ID,
	})

	require.NoError(t, err)
	require.NotNil(t, article.CategoryID)
	assert.Equal(t, cat.ID, *article.CategoryID)
}

func TestService_CreateArticle_RepoError(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	repo.createArticleErr = ErrInvalidContent

	_, err := svc.CreateArticle(context.Background(), CreateArticleInput{
		TenantID: uuid.New(),
		Title:    "Artikel",
		Slug:     "artikel",
		Content:  validContent(),
		AuthorID: uuid.New(),
	})

	assert.Error(t, err)
}

// ============================================================================
// UpdateArticle Tests
// ============================================================================

func TestService_UpdateArticle_UpdatesTitle(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	a := addArticle(repo, tenantID, "Old Title", "old-title", false)

	newTitle := "New Title"
	updated, err := svc.UpdateArticle(context.Background(), UpdateArticleInput{
		TenantID:  tenantID,
		ArticleID: a.ID,
		Title:     &newTitle,
		EditorID:  uuid.New(),
	})

	require.NoError(t, err)
	assert.Equal(t, "New Title", updated.Title)
}

func TestService_UpdateArticle_CreatesVersion(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	a := addArticle(repo, tenantID, "Article", "article", false)
	originalContent := a.Content

	newContent := validContent()
	_, err := svc.UpdateArticle(context.Background(), UpdateArticleInput{
		TenantID:  tenantID,
		ArticleID: a.ID,
		Content:   newContent,
		EditorID:  uuid.New(),
	})

	require.NoError(t, err)
	versions, listErr := repo.ListVersions(context.Background(), tenantID, a.ID)
	require.NoError(t, listErr)
	require.Len(t, versions, 1)
	assert.Equal(t, originalContent, versions[0].Content)
	assert.Equal(t, 1, versions[0].VersionNumber)
}

func TestService_UpdateArticle_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.UpdateArticle(context.Background(), UpdateArticleInput{
		TenantID:  uuid.New(),
		ArticleID: uuid.New(),
		EditorID:  uuid.New(),
	})

	assert.ErrorIs(t, err, ErrArticleNotFound)
}

func TestService_UpdateArticle_SlugTaken(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	_ = addArticle(repo, tenantID, "Existing", "taken-slug", false)
	a := addArticle(repo, tenantID, "Mine", "my-slug", false)

	takenSlug := "taken-slug"
	_, err := svc.UpdateArticle(context.Background(), UpdateArticleInput{
		TenantID:  tenantID,
		ArticleID: a.ID,
		Slug:      &takenSlug,
		EditorID:  uuid.New(),
	})

	assert.ErrorIs(t, err, ErrSlugTaken)
}

// ============================================================================
// DeleteArticle Tests
// ============================================================================

func TestService_DeleteArticle_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	a := addArticle(repo, tenantID, "Article", "article", false)

	err := svc.DeleteArticle(context.Background(), tenantID, a.ID, nil)

	require.NoError(t, err)
	assert.NotContains(t, repo.articles, a.ID)
}

func TestService_DeleteArticle_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	err := svc.DeleteArticle(context.Background(), uuid.New(), uuid.New(), nil)

	assert.ErrorIs(t, err, ErrArticleNotFound)
}

// ============================================================================
// SearchArticles Tests
// ============================================================================

func TestService_SearchArticles_ReturnsPublished(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	addArticle(repo, tenantID, "Published Article", "pub", true)
	addArticle(repo, tenantID, "Draft Article", "draft", false)

	results, err := svc.SearchArticles(context.Background(), tenantID, "article", 10)

	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.True(t, results[0].Published)
}

func TestService_SearchArticles_EmptyQuery(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	results, err := svc.SearchArticles(context.Background(), uuid.New(), "   ", 10)

	require.NoError(t, err)
	assert.Nil(t, results)
}

func TestService_SearchArticles_LimitClamped(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	addArticle(repo, tenantID, "Pub", "pub", true)

	// limit 0 should default to 20 — mock returns all published, but no panic
	results, err := svc.SearchArticles(context.Background(), tenantID, "pub", 0)

	require.NoError(t, err)
	assert.Len(t, results, 1)
}

// ============================================================================
// Version Flow Tests
// ============================================================================

func TestService_ListVersions_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	a := addArticle(repo, tenantID, "Article", "art", false)

	// manually add two versions
	v1 := &Version{ID: uuid.New(), ArticleID: a.ID, VersionNumber: 1, Content: []byte(`{}`), ChangedAt: time.Now()}
	v2 := &Version{ID: uuid.New(), ArticleID: a.ID, VersionNumber: 2, Content: []byte(`{}`), ChangedAt: time.Now()}
	repo.versions[v1.ID] = v1
	repo.versions[v2.ID] = v2

	versions, err := svc.ListVersions(context.Background(), tenantID, a.ID)

	require.NoError(t, err)
	assert.Len(t, versions, 2)
}

func TestService_GetVersion_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.GetVersion(context.Background(), uuid.New(), uuid.New())

	assert.ErrorIs(t, err, ErrVersionNotFound)
}

func TestService_RestoreVersion_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	a := addArticle(repo, tenantID, "Article", "art", false)

	oldContent := json.RawMessage(`{"type":"doc","content":[{"text":"old"}]}`)
	vID := uuid.New()
	repo.versions[vID] = &Version{
		ID:            vID,
		ArticleID:     a.ID,
		VersionNumber: 1,
		Content:       oldContent,
		ChangedAt:     time.Now(),
	}

	restored, err := svc.RestoreVersion(context.Background(), tenantID, a.ID, vID, uuid.New(), nil)

	require.NoError(t, err)
	assert.Equal(t, string(oldContent), string(restored.Content))
}

// ============================================================================
// ListCategories Tests
// ============================================================================

func TestService_ListCategories_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	for range 3 {
		repo.categories[uuid.New()] = &Category{
			ID:       uuid.New(),
			TenantID: tenantID,
			Name:     "Cat",
		}
	}
	// another tenant's category — should not appear
	repo.categories[uuid.New()] = &Category{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Name:     "Other",
	}

	categories, err := svc.ListCategories(context.Background(), tenantID)

	require.NoError(t, err)
	assert.Len(t, categories, 3)
}

func TestService_CreateCategory_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	cat, err := svc.CreateCategory(context.Background(), CreateCategoryInput{
		TenantID: tenantID,
		Name:     "Dokumentation",
		Position: 1,
	})

	require.NoError(t, err)
	assert.Equal(t, "Dokumentation", cat.Name)
	assert.Equal(t, tenantID, cat.TenantID)
}

func TestService_CreateCategory_EmptyName(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.CreateCategory(context.Background(), CreateCategoryInput{
		TenantID: uuid.New(),
		Name:     "  ",
	})

	assert.ErrorIs(t, err, ErrInvalidContent)
}

func TestService_CreateCategory_ParentNotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	parentID := uuid.New() // does not exist

	_, err := svc.CreateCategory(context.Background(), CreateCategoryInput{
		TenantID: uuid.New(),
		Name:     "Sub",
		ParentID: &parentID,
	})

	assert.ErrorIs(t, err, ErrCategoryNotFound)
}

// ============================================================================
// Attachment Tests
// ============================================================================

func TestService_UploadAttachment_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	articleID := uuid.New()
	uploaderID := uuid.New()
	att, err := svc.UploadAttachment(context.Background(), UploadInput{
		TenantID:   tenantID,
		ArticleID:  articleID,
		FileRef:    "uploads/doc.pdf",
		Mime:       "application/pdf",
		Size:       1024,
		UploadedBy: &uploaderID,
	})

	require.NoError(t, err)
	assert.Equal(t, articleID, att.ArticleID)
	assert.Equal(t, tenantID, att.TenantID)
	assert.Equal(t, "uploads/doc.pdf", att.FileRef)
}

func TestService_UploadAttachment_EmptyFileRef(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.UploadAttachment(context.Background(), UploadInput{
		ArticleID: uuid.New(),
		FileRef:   "  ",
	})

	assert.ErrorIs(t, err, ErrInvalidContent)
}

func TestService_ListAttachments_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	articleID := uuid.New()
	repo.attachments[uuid.New()] = &Attachment{ID: uuid.New(), ArticleID: articleID, FileRef: "a.pdf", CreatedAt: time.Now()}
	repo.attachments[uuid.New()] = &Attachment{ID: uuid.New(), ArticleID: articleID, FileRef: "b.docx", CreatedAt: time.Now()}

	attachments, err := svc.ListAttachments(context.Background(), uuid.New(), articleID)

	require.NoError(t, err)
	assert.Len(t, attachments, 2)
}

// ============================================================================
// GetArticle Tests
// ============================================================================

func TestService_GetArticle_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	a := addArticle(repo, tenantID, "Test", "test", true)

	result, err := svc.GetArticle(context.Background(), tenantID, a.ID)

	require.NoError(t, err)
	assert.Equal(t, a.ID, result.ID)
}

func TestService_GetArticle_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.GetArticle(context.Background(), uuid.New(), uuid.New())

	assert.ErrorIs(t, err, ErrArticleNotFound)
}

// ============================================================================
// ListArticles Tests
// ============================================================================

func TestService_ListArticles_DefaultPagination(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	for range 25 {
		slug := uuid.New().String()
		addArticle(repo, tenantID, "Article", slug, true)
	}

	articles, total, err := svc.ListArticles(context.Background(), ListArticlesInput{
		TenantID: tenantID,
	})

	require.NoError(t, err)
	assert.Equal(t, 25, total)
	assert.Len(t, articles, 20)
}

// ============================================================================
// FTS unit tests
// ============================================================================

func TestBuildSearchQuery_SingleWord(t *testing.T) {
	assert.Equal(t, "Handbuch:*", BuildSearchQuery("Handbuch"))
}

func TestBuildSearchQuery_MultiWord(t *testing.T) {
	assert.Equal(t, "Kunden & report:*", BuildSearchQuery("Kunden report"))
}

func TestBuildSearchQuery_Empty(t *testing.T) {
	assert.Equal(t, "", BuildSearchQuery(""))
	assert.Equal(t, "", BuildSearchQuery("   "))
}

func TestBuildSearchQuery_StripsSpecialChars(t *testing.T) {
	// Semicolons and SQL punctuation are stripped; regular words pass through.
	// Because the tsquery is always a parameterised argument, SQL injection is
	// not a concern — the test validates that special characters are removed.
	result := BuildSearchQuery("hello; DROP--")
	assert.NotContains(t, result, ";")
	assert.NotContains(t, result, "--")
	assert.Contains(t, result, "hello")
}

// ============================================================================
// Own-scope edit (wiki:article:edit at scope own)
// ============================================================================

func TestService_OwnScopeEdit(t *testing.T) {
	tenantID := uuid.New()
	content := json.RawMessage(`{"type":"doc","content":[{"text":"new"}]}`)

	setup := func() (*Service, *mockRepository, *Article, uuid.UUID) {
		repo := newMockRepository()
		a := addArticle(repo, tenantID, "Article", "art", true)
		vID := uuid.New()
		repo.versions[vID] = &Version{ID: vID, ArticleID: a.ID, VersionNumber: 1, Content: content, ChangedAt: time.Now()}
		return NewService(repo), repo, a, vID
	}

	t.Run("a stranger at own scope gets not found on update, restore and upload", func(t *testing.T) {
		svc, repo, a, vID := setup()
		stranger := uuid.New()
		title := "Hijacked"

		_, err := svc.UpdateArticle(context.Background(), UpdateArticleInput{
			TenantID: tenantID, ArticleID: a.ID, Title: &title, Content: content, EditorID: stranger, OwnerID: &stranger,
		})
		require.ErrorIs(t, err, ErrArticleNotFound)

		_, err = svc.RestoreVersion(context.Background(), tenantID, a.ID, vID, stranger, &stranger)
		require.ErrorIs(t, err, ErrArticleNotFound)

		_, err = svc.UploadAttachment(context.Background(), UploadInput{
			TenantID: tenantID, ArticleID: a.ID, FileRef: "uploads/x.pdf", UploadedBy: &stranger, OwnerID: &stranger,
		})
		require.ErrorIs(t, err, ErrArticleNotFound)

		assert.Equal(t, "Article", repo.articles[a.ID].Title, "title must stay untouched")
		assert.Len(t, repo.versions, 1, "no version snapshot for a refused edit")
		assert.Empty(t, repo.attachments, "no attachment for a refused upload")
	})

	t.Run("the author at own scope edits, restores and uploads", func(t *testing.T) {
		svc, repo, a, vID := setup()
		author := a.AuthorID
		title := "Renamed"

		updated, err := svc.UpdateArticle(context.Background(), UpdateArticleInput{
			TenantID: tenantID, ArticleID: a.ID, Title: &title, EditorID: author, OwnerID: &author,
		})
		require.NoError(t, err)
		assert.Equal(t, "Renamed", updated.Title)

		restored, err := svc.RestoreVersion(context.Background(), tenantID, a.ID, vID, author, &author)
		require.NoError(t, err)
		assert.Equal(t, string(content), string(restored.Content))

		_, err = svc.UploadAttachment(context.Background(), UploadInput{
			TenantID: tenantID, ArticleID: a.ID, FileRef: "uploads/x.pdf", UploadedBy: &author, OwnerID: &author,
		})
		require.NoError(t, err)
		assert.Len(t, repo.attachments, 1)
	})

	t.Run("without an owner the edit stays tenant-wide", func(t *testing.T) {
		svc, _, a, vID := setup()
		other := uuid.New()
		title := "Manager edit"

		_, err := svc.UpdateArticle(context.Background(), UpdateArticleInput{
			TenantID: tenantID, ArticleID: a.ID, Title: &title, EditorID: other,
		})
		require.NoError(t, err)
		_, err = svc.RestoreVersion(context.Background(), tenantID, a.ID, vID, other, nil)
		require.NoError(t, err)
	})

	t.Run("an unknown article at own scope is not found before the version is looked up", func(t *testing.T) {
		svc, _, _, _ := setup()
		me := uuid.New()
		_, err := svc.RestoreVersion(context.Background(), tenantID, uuid.New(), uuid.New(), me, &me)
		require.ErrorIs(t, err, ErrArticleNotFound)
	})
}

func TestService_OwnScopeDeleteAndShare(t *testing.T) {
	tenantID := uuid.New()

	setup := func() (*Service, *mockRepository, *Article, *Attachment, *ShareToken) {
		repo := newMockRepository()
		a := addArticle(repo, tenantID, "Article", "art", true)
		att := &Attachment{ID: uuid.New(), TenantID: tenantID, ArticleID: a.ID, FileRef: "uploads/a.pdf"}
		repo.attachments[att.ID] = att
		tok := &ShareToken{ID: uuid.New(), TenantID: tenantID, ArticleID: a.ID, Token: "tok-" + uuid.NewString()}
		repo.tokens[tok.Token] = tok
		return NewService(repo), repo, a, att, tok
	}

	t.Run("a stranger at own scope gets not found and nothing changes", func(t *testing.T) {
		svc, repo, a, att, tok := setup()
		stranger := uuid.New()
		ctx := context.Background()

		require.ErrorIs(t, svc.DeleteArticle(ctx, tenantID, a.ID, &stranger), ErrArticleNotFound)
		require.ErrorIs(t, svc.DeleteAttachment(ctx, tenantID, att.ID, &stranger), ErrAttachmentNotFound)
		require.ErrorIs(t, svc.RevokeShareToken(ctx, tenantID, tok.ID, &stranger), ErrShareTokenNotFound)
		_, err := svc.CreateShareToken(ctx, CreateShareTokenInput{TenantID: tenantID, ArticleID: a.ID, OwnerID: &stranger})
		require.ErrorIs(t, err, ErrArticleNotFound)

		assert.Contains(t, repo.articles, a.ID, "article must survive")
		assert.Contains(t, repo.attachments, att.ID, "attachment must survive")
		assert.Nil(t, tok.RevokedAt, "link must stay live")
		assert.Len(t, repo.tokens, 1, "no link minted for a stranger")
	})

	t.Run("unknown ids at own scope stay not found", func(t *testing.T) {
		svc, _, _, _, _ := setup()
		me := uuid.New()
		ctx := context.Background()
		require.ErrorIs(t, svc.DeleteAttachment(ctx, tenantID, uuid.New(), &me), ErrAttachmentNotFound)
		require.ErrorIs(t, svc.RevokeShareToken(ctx, tenantID, uuid.New(), &me), ErrShareTokenNotFound)
	})

	t.Run("the author at own scope shares, revokes, deletes the attachment and the article", func(t *testing.T) {
		svc, repo, a, att, tok := setup()
		author := a.AuthorID
		ctx := context.Background()

		_, err := svc.CreateShareToken(ctx, CreateShareTokenInput{TenantID: tenantID, ArticleID: a.ID, OwnerID: &author})
		require.NoError(t, err)
		require.NoError(t, svc.RevokeShareToken(ctx, tenantID, tok.ID, &author))
		assert.NotNil(t, tok.RevokedAt)
		require.NoError(t, svc.DeleteAttachment(ctx, tenantID, att.ID, &author))
		assert.NotContains(t, repo.attachments, att.ID)
		require.NoError(t, svc.DeleteArticle(ctx, tenantID, a.ID, &author))
		assert.NotContains(t, repo.articles, a.ID)
	})

	t.Run("without an owner all four stay tenant-wide", func(t *testing.T) {
		svc, repo, a, att, tok := setup()
		ctx := context.Background()

		_, err := svc.CreateShareToken(ctx, CreateShareTokenInput{TenantID: tenantID, ArticleID: a.ID})
		require.NoError(t, err)
		require.NoError(t, svc.RevokeShareToken(ctx, tenantID, tok.ID, nil))
		require.NoError(t, svc.DeleteAttachment(ctx, tenantID, att.ID, nil))
		require.NoError(t, svc.DeleteArticle(ctx, tenantID, a.ID, nil))
		assert.NotContains(t, repo.articles, a.ID)
	})
}
