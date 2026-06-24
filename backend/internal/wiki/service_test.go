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

func (m *mockRepository) ListVersions(ctx context.Context, articleID uuid.UUID) ([]*Version, error) {
	var result []*Version
	for _, v := range m.versions {
		if v.ArticleID == articleID {
			result = append(result, v)
		}
	}
	return result, nil
}

func (m *mockRepository) GetVersion(ctx context.Context, versionID uuid.UUID) (*Version, error) {
	v, ok := m.versions[versionID]
	if !ok {
		return nil, ErrVersionNotFound
	}
	return v, nil
}

func (m *mockRepository) GetLatestVersionNumber(ctx context.Context, articleID uuid.UUID) (int, error) {
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

func (m *mockRepository) ListAttachments(ctx context.Context, articleID uuid.UUID) ([]*Attachment, error) {
	var result []*Attachment
	for _, a := range m.attachments {
		if a.ArticleID == articleID {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *mockRepository) GetAttachment(ctx context.Context, attachmentID uuid.UUID) (*Attachment, error) {
	a, ok := m.attachments[attachmentID]
	if !ok {
		return nil, ErrArticleNotFound
	}
	return a, nil
}

func (m *mockRepository) DeleteAttachment(ctx context.Context, attachmentID uuid.UUID) error {
	delete(m.attachments, attachmentID)
	return nil
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

func (m *mockRepository) GetShareToken(ctx context.Context, token string) (*ShareToken, error) {
	t, ok := m.tokens[token]
	if !ok {
		return nil, ErrArticleNotFound
	}
	return t, nil
}

func (m *mockRepository) DeleteShareToken(ctx context.Context, tokenID uuid.UUID) error {
	for k, t := range m.tokens {
		if t.ID == tokenID {
			delete(m.tokens, k)
			break
		}
	}
	return nil
}

func (m *mockRepository) ListShareTokensByArticle(ctx context.Context, articleID uuid.UUID) ([]*ShareToken, error) {
	var result []*ShareToken
	for _, t := range m.tokens {
		if t.ArticleID == articleID {
			result = append(result, t)
		}
	}
	return result, nil
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
	versions, listErr := repo.ListVersions(context.Background(), a.ID)
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

	err := svc.DeleteArticle(context.Background(), tenantID, a.ID)

	require.NoError(t, err)
	assert.NotContains(t, repo.articles, a.ID)
}

func TestService_DeleteArticle_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	err := svc.DeleteArticle(context.Background(), uuid.New(), uuid.New())

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

	versions, err := svc.ListVersions(context.Background(), a.ID)

	require.NoError(t, err)
	assert.Len(t, versions, 2)
}

func TestService_GetVersion_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.GetVersion(context.Background(), uuid.New())

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

	restored, err := svc.RestoreVersion(context.Background(), tenantID, a.ID, vID, uuid.New())

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

	attachments, err := svc.ListAttachments(context.Background(), articleID)

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
