package server

import (
	"context"
	"errors"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/wiki"
	wikiv1 "github.com/kmuhub/kmuhub/proto/wiki/v1"
)

// ============================================================================
// Stub repository (server-package copy — wiki has no internal fakeRepo to reuse)
// ============================================================================

var errStubWikiRepoFailure = errors.New("stub wiki repo: forced failure")

type stubWikiRepo struct {
	mu          sync.Mutex
	articles    map[uuid.UUID]*wiki.Article
	versions    map[uuid.UUID]*wiki.Version
	attachments map[uuid.UUID]*wiki.Attachment
	categories  map[uuid.UUID]*wiki.Category
	shareTokens map[uuid.UUID]*wiki.ShareToken

	forceErr error
}

func newStubWikiRepo() *stubWikiRepo {
	return &stubWikiRepo{
		articles:    make(map[uuid.UUID]*wiki.Article),
		versions:    make(map[uuid.UUID]*wiki.Version),
		attachments: make(map[uuid.UUID]*wiki.Attachment),
		categories:  make(map[uuid.UUID]*wiki.Category),
		shareTokens: make(map[uuid.UUID]*wiki.ShareToken),
	}
}

// --- Articles ---

func (r *stubWikiRepo) CreateArticle(_ context.Context, a *wiki.Article) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.forceErr != nil {
		return r.forceErr
	}
	cp := *a
	r.articles[a.ID] = &cp
	return nil
}

func (r *stubWikiRepo) UpdateArticle(_ context.Context, a *wiki.Article) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.forceErr != nil {
		return r.forceErr
	}
	existing, ok := r.articles[a.ID]
	if !ok || existing.TenantID != a.TenantID {
		return wiki.ErrArticleNotFound
	}
	cp := *a
	r.articles[a.ID] = &cp
	return nil
}

func (r *stubWikiRepo) GetArticleByID(_ context.Context, tenantID, articleID uuid.UUID) (*wiki.Article, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	a, ok := r.articles[articleID]
	if !ok || a.TenantID != tenantID {
		return nil, wiki.ErrArticleNotFound
	}
	cp := *a
	return &cp, nil
}

func (r *stubWikiRepo) GetArticleBySlug(_ context.Context, tenantID uuid.UUID, slug string) (*wiki.Article, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, a := range r.articles {
		if a.TenantID == tenantID && a.Slug == slug {
			cp := *a
			return &cp, nil
		}
	}
	return nil, wiki.ErrArticleNotFound
}

func (r *stubWikiRepo) ListArticles(_ context.Context, tenantID uuid.UUID, filter wiki.ListArticlesFilter, offset, limit int) ([]*wiki.Article, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.forceErr != nil {
		return nil, 0, r.forceErr
	}
	var all []*wiki.Article
	for _, a := range r.articles {
		if a.TenantID != tenantID {
			continue
		}
		if filter.CategoryID != nil && (a.CategoryID == nil || *a.CategoryID != *filter.CategoryID) {
			continue
		}
		if filter.AuthorID != nil && a.AuthorID != *filter.AuthorID {
			continue
		}
		if filter.Published != nil && a.Published != *filter.Published {
			continue
		}
		cp := *a
		all = append(all, &cp)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].CreatedAt.Before(all[j].CreatedAt) })
	total := len(all)
	offset = min(offset, total)
	end := min(offset+limit, total)
	return all[offset:end], total, nil
}

func (r *stubWikiRepo) DeleteArticle(_ context.Context, tenantID, articleID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.articles[articleID]
	if !ok || a.TenantID != tenantID {
		return wiki.ErrArticleNotFound
	}
	delete(r.articles, articleID)
	return nil
}

func (r *stubWikiRepo) SearchArticles(_ context.Context, tenantID uuid.UUID, _ string, limit int) ([]*wiki.Article, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	var out []*wiki.Article
	for _, a := range r.articles {
		if a.TenantID != tenantID {
			continue
		}
		cp := *a
		out = append(out, &cp)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (r *stubWikiRepo) SlugExists(_ context.Context, tenantID uuid.UUID, slug string, excludeID *uuid.UUID) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, a := range r.articles {
		if a.TenantID == tenantID && a.Slug == slug {
			if excludeID != nil && a.ID == *excludeID {
				continue
			}
			return true, nil
		}
	}
	return false, nil
}

// --- Versions ---

func (r *stubWikiRepo) CreateVersion(_ context.Context, v *wiki.Version) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.forceErr != nil {
		return r.forceErr
	}
	cp := *v
	r.versions[v.ID] = &cp
	return nil
}

func (r *stubWikiRepo) ListVersions(_ context.Context, tenantID, articleID uuid.UUID) ([]*wiki.Version, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*wiki.Version
	for _, v := range r.versions {
		if v.TenantID == tenantID && v.ArticleID == articleID {
			cp := *v
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].VersionNumber < out[j].VersionNumber })
	return out, nil
}

func (r *stubWikiRepo) GetVersion(_ context.Context, tenantID, versionID uuid.UUID) (*wiki.Version, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.versions[versionID]
	if !ok || v.TenantID != tenantID {
		return nil, wiki.ErrVersionNotFound
	}
	cp := *v
	return &cp, nil
}

func (r *stubWikiRepo) GetLatestVersionNumber(_ context.Context, tenantID, articleID uuid.UUID) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	max := 0
	for _, v := range r.versions {
		if v.TenantID == tenantID && v.ArticleID == articleID && v.VersionNumber > max {
			max = v.VersionNumber
		}
	}
	return max, nil
}

// --- Attachments ---

func (r *stubWikiRepo) CreateAttachment(_ context.Context, a *wiki.Attachment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.forceErr != nil {
		return r.forceErr
	}
	cp := *a
	r.attachments[a.ID] = &cp
	return nil
}

func (r *stubWikiRepo) ListAttachments(_ context.Context, tenantID, articleID uuid.UUID) ([]*wiki.Attachment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*wiki.Attachment
	for _, a := range r.attachments {
		if a.TenantID == tenantID && a.ArticleID == articleID {
			cp := *a
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *stubWikiRepo) DeleteAttachment(_ context.Context, tenantID, attachmentID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.attachments[attachmentID]
	if !ok || a.TenantID != tenantID {
		return wiki.ErrAttachmentNotFound
	}
	delete(r.attachments, attachmentID)
	return nil
}

// --- Categories ---

func (r *stubWikiRepo) CreateCategory(_ context.Context, c *wiki.Category) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.forceErr != nil {
		return r.forceErr
	}
	cp := *c
	r.categories[c.ID] = &cp
	return nil
}

func (r *stubWikiRepo) ListCategories(_ context.Context, tenantID uuid.UUID) ([]*wiki.Category, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*wiki.Category
	for _, c := range r.categories {
		if c.TenantID == tenantID {
			cp := *c
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *stubWikiRepo) GetCategory(_ context.Context, tenantID, categoryID uuid.UUID) (*wiki.Category, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.categories[categoryID]
	if !ok || c.TenantID != tenantID {
		return nil, wiki.ErrCategoryNotFound
	}
	cp := *c
	return &cp, nil
}

func (r *stubWikiRepo) DeleteCategory(_ context.Context, tenantID, categoryID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.categories[categoryID]
	if !ok || c.TenantID != tenantID {
		return wiki.ErrCategoryNotFound
	}
	delete(r.categories, categoryID)
	return nil
}

func (r *stubWikiRepo) UpdateCategory(_ context.Context, c *wiki.Category) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.categories[c.ID]
	if !ok || existing.TenantID != c.TenantID {
		return wiki.ErrCategoryNotFound
	}
	cp := *c
	r.categories[c.ID] = &cp
	return nil
}

// --- Share tokens ---

func (r *stubWikiRepo) CreateShareToken(_ context.Context, t *wiki.ShareToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.forceErr != nil {
		return r.forceErr
	}
	cp := *t
	r.shareTokens[t.ID] = &cp
	return nil
}

func (r *stubWikiRepo) RevokeShareToken(_ context.Context, tenantID, tokenID uuid.UUID, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.shareTokens[tokenID]
	if !ok || t.TenantID != tenantID {
		return wiki.ErrShareTokenNotFound
	}
	if t.RevokedAt == nil {
		cp := now
		t.RevokedAt = &cp
	}
	return nil
}

func (r *stubWikiRepo) ListShareTokensByArticle(_ context.Context, tenantID, articleID uuid.UUID) ([]*wiki.ShareToken, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*wiki.ShareToken
	for _, t := range r.shareTokens {
		if t.TenantID == tenantID && t.ArticleID == articleID {
			cp := *t
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *stubWikiRepo) GetShareTokenByToken(_ context.Context, token string) (*wiki.ShareToken, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range r.shareTokens {
		if t.Token == token {
			cp := *t
			return &cp, nil
		}
	}
	return nil, wiki.ErrShareTokenNotFound
}

// ============================================================================
// Test helpers
// ============================================================================

func newTestWikiServer() *WikiGRPCServer {
	return NewWikiGRPCServer(nil)
}

func newWikiServerWithRepo(repo wiki.Repository) (*WikiGRPCServer, *wiki.Service) {
	svc := wiki.NewService(repo)
	return NewWikiGRPCServer(svc), svc
}

const (
	blockDocContent   = `{"type":"doc","content":[{"type":"paragraph"}]}`
	legacyHTMLContent = `"<p>Legacy content</p>"` // a JSON string literal wrapping raw HTML
)

// ============================================================================
// UUID validation — every handler rejects a malformed id before ever touching
// a nil service, so these run against newTestWikiServer().
// ============================================================================

func TestWikiHandlers_UUIDValidation(t *testing.T) {
	srv := newTestWikiServer()
	ctx := context.Background()
	validID := uuid.New().String()

	cases := []struct {
		name string
		call func() error
	}{
		{"CreateArticle invalid tenant_id", func() error {
			_, err := srv.CreateArticle(ctx, &wikiv1.CreateArticleRequest{TenantId: "bad", AuthorId: validID, Title: "T"})
			return err
		}},
		{"CreateArticle invalid author_id", func() error {
			_, err := srv.CreateArticle(ctx, &wikiv1.CreateArticleRequest{TenantId: validID, AuthorId: "bad", Title: "T"})
			return err
		}},
		{"CreateArticle invalid category_id", func() error {
			cat := "bad"
			_, err := srv.CreateArticle(ctx, &wikiv1.CreateArticleRequest{TenantId: validID, AuthorId: validID, Title: "T", CategoryId: &cat})
			return err
		}},
		{"GetArticle invalid tenant_id", func() error {
			_, err := srv.GetArticle(ctx, &wikiv1.GetArticleRequest{TenantId: "bad", ArticleId: validID})
			return err
		}},
		{"GetArticle invalid article_id", func() error {
			_, err := srv.GetArticle(ctx, &wikiv1.GetArticleRequest{TenantId: validID, ArticleId: "bad"})
			return err
		}},
		{"UpdateArticle invalid tenant_id", func() error {
			_, err := srv.UpdateArticle(ctx, &wikiv1.UpdateArticleRequest{TenantId: "bad", ArticleId: validID, EditorId: validID})
			return err
		}},
		{"UpdateArticle invalid article_id", func() error {
			_, err := srv.UpdateArticle(ctx, &wikiv1.UpdateArticleRequest{TenantId: validID, ArticleId: "bad", EditorId: validID})
			return err
		}},
		{"UpdateArticle invalid editor_id", func() error {
			_, err := srv.UpdateArticle(ctx, &wikiv1.UpdateArticleRequest{TenantId: validID, ArticleId: validID, EditorId: "bad"})
			return err
		}},
		{"UpdateArticle invalid category_id", func() error {
			cat := "bad"
			_, err := srv.UpdateArticle(ctx, &wikiv1.UpdateArticleRequest{TenantId: validID, ArticleId: validID, EditorId: validID, CategoryId: &cat})
			return err
		}},
		{"DeleteArticle invalid tenant_id", func() error {
			_, err := srv.DeleteArticle(ctx, &wikiv1.DeleteArticleRequest{TenantId: "bad", ArticleId: validID})
			return err
		}},
		{"DeleteArticle invalid article_id", func() error {
			_, err := srv.DeleteArticle(ctx, &wikiv1.DeleteArticleRequest{TenantId: validID, ArticleId: "bad"})
			return err
		}},
		{"ListArticles invalid tenant_id", func() error {
			_, err := srv.ListArticles(ctx, &wikiv1.ListArticlesRequest{TenantId: "bad"})
			return err
		}},
		{"ListArticles invalid category_id", func() error {
			cat := "bad"
			_, err := srv.ListArticles(ctx, &wikiv1.ListArticlesRequest{TenantId: validID, CategoryId: &cat})
			return err
		}},
		{"ListArticles invalid author_id", func() error {
			author := "bad"
			_, err := srv.ListArticles(ctx, &wikiv1.ListArticlesRequest{TenantId: validID, AuthorId: &author})
			return err
		}},
		{"SearchArticles invalid tenant_id", func() error {
			_, err := srv.SearchArticles(ctx, &wikiv1.SearchArticlesRequest{TenantId: "bad"})
			return err
		}},
		{"ListVersions invalid tenant_id", func() error {
			_, err := srv.ListVersions(ctx, &wikiv1.ListVersionsRequest{TenantId: "bad", ArticleId: validID})
			return err
		}},
		{"ListVersions invalid article_id", func() error {
			_, err := srv.ListVersions(ctx, &wikiv1.ListVersionsRequest{TenantId: validID, ArticleId: "bad"})
			return err
		}},
		{"GetVersion invalid tenant_id", func() error {
			_, err := srv.GetVersion(ctx, &wikiv1.GetVersionRequest{TenantId: "bad", VersionId: validID})
			return err
		}},
		{"GetVersion invalid version_id", func() error {
			_, err := srv.GetVersion(ctx, &wikiv1.GetVersionRequest{TenantId: validID, VersionId: "bad"})
			return err
		}},
		{"RestoreVersion invalid tenant_id", func() error {
			_, err := srv.RestoreVersion(ctx, &wikiv1.RestoreVersionRequest{TenantId: "bad", ArticleId: validID, VersionId: validID, EditorId: validID})
			return err
		}},
		{"RestoreVersion invalid article_id", func() error {
			_, err := srv.RestoreVersion(ctx, &wikiv1.RestoreVersionRequest{TenantId: validID, ArticleId: "bad", VersionId: validID, EditorId: validID})
			return err
		}},
		{"RestoreVersion invalid version_id", func() error {
			_, err := srv.RestoreVersion(ctx, &wikiv1.RestoreVersionRequest{TenantId: validID, ArticleId: validID, VersionId: "bad", EditorId: validID})
			return err
		}},
		{"RestoreVersion invalid editor_id", func() error {
			_, err := srv.RestoreVersion(ctx, &wikiv1.RestoreVersionRequest{TenantId: validID, ArticleId: validID, VersionId: validID, EditorId: "bad"})
			return err
		}},
		{"UploadAttachment invalid tenant_id", func() error {
			_, err := srv.UploadAttachment(ctx, &wikiv1.UploadAttachmentRequest{TenantId: "bad", ArticleId: validID})
			return err
		}},
		{"UploadAttachment invalid article_id", func() error {
			_, err := srv.UploadAttachment(ctx, &wikiv1.UploadAttachmentRequest{TenantId: validID, ArticleId: "bad"})
			return err
		}},
		{"UploadAttachment invalid uploaded_by", func() error {
			by := "bad"
			_, err := srv.UploadAttachment(ctx, &wikiv1.UploadAttachmentRequest{TenantId: validID, ArticleId: validID, UploadedBy: &by})
			return err
		}},
		{"ListAttachments invalid tenant_id", func() error {
			_, err := srv.ListAttachments(ctx, &wikiv1.ListAttachmentsRequest{TenantId: "bad", ArticleId: validID})
			return err
		}},
		{"ListAttachments invalid article_id", func() error {
			_, err := srv.ListAttachments(ctx, &wikiv1.ListAttachmentsRequest{TenantId: validID, ArticleId: "bad"})
			return err
		}},
		{"DeleteAttachment invalid tenant_id", func() error {
			_, err := srv.DeleteAttachment(ctx, &wikiv1.DeleteAttachmentRequest{TenantId: "bad", AttachmentId: validID})
			return err
		}},
		{"DeleteAttachment invalid attachment_id", func() error {
			_, err := srv.DeleteAttachment(ctx, &wikiv1.DeleteAttachmentRequest{TenantId: validID, AttachmentId: "bad"})
			return err
		}},
		{"ListCategories invalid tenant_id", func() error {
			_, err := srv.ListCategories(ctx, &wikiv1.ListCategoriesRequest{TenantId: "bad"})
			return err
		}},
		{"CreateCategory invalid tenant_id", func() error {
			_, err := srv.CreateCategory(ctx, &wikiv1.CreateCategoryRequest{TenantId: "bad"})
			return err
		}},
		{"CreateCategory invalid parent_id", func() error {
			parent := "bad"
			_, err := srv.CreateCategory(ctx, &wikiv1.CreateCategoryRequest{TenantId: validID, ParentId: &parent})
			return err
		}},
		{"DeleteCategory invalid tenant_id", func() error {
			_, err := srv.DeleteCategory(ctx, &wikiv1.DeleteCategoryRequest{TenantId: "bad", CategoryId: validID})
			return err
		}},
		{"DeleteCategory invalid category_id", func() error {
			_, err := srv.DeleteCategory(ctx, &wikiv1.DeleteCategoryRequest{TenantId: validID, CategoryId: "bad"})
			return err
		}},
		{"UpdateCategory invalid tenant_id", func() error {
			_, err := srv.UpdateCategory(ctx, &wikiv1.UpdateCategoryRequest{TenantId: "bad", CategoryId: validID})
			return err
		}},
		{"UpdateCategory invalid category_id", func() error {
			_, err := srv.UpdateCategory(ctx, &wikiv1.UpdateCategoryRequest{TenantId: validID, CategoryId: "bad"})
			return err
		}},
		{"UpdateCategory invalid parent_id", func() error {
			parent := "bad"
			_, err := srv.UpdateCategory(ctx, &wikiv1.UpdateCategoryRequest{TenantId: validID, CategoryId: validID, ParentId: &parent})
			return err
		}},
		{"CreateShareToken invalid tenant_id", func() error {
			_, err := srv.CreateShareToken(ctx, &wikiv1.CreateShareTokenRequest{TenantId: "bad", ArticleId: validID})
			return err
		}},
		{"CreateShareToken invalid article_id", func() error {
			_, err := srv.CreateShareToken(ctx, &wikiv1.CreateShareTokenRequest{TenantId: validID, ArticleId: "bad"})
			return err
		}},
		{"CreateShareToken invalid expires_at", func() error {
			exp := "not-a-date"
			_, err := srv.CreateShareToken(ctx, &wikiv1.CreateShareTokenRequest{TenantId: validID, ArticleId: validID, ExpiresAt: &exp})
			return err
		}},
		{"CreateShareToken invalid created_by", func() error {
			_, err := srv.CreateShareToken(ctx, &wikiv1.CreateShareTokenRequest{TenantId: validID, ArticleId: validID, CreatedBy: strPtr("bad")})
			return err
		}},
		{"ListShareTokens invalid tenant_id", func() error {
			_, err := srv.ListShareTokens(ctx, &wikiv1.ListShareTokensRequest{TenantId: "bad", ArticleId: validID})
			return err
		}},
		{"ListShareTokens invalid article_id", func() error {
			_, err := srv.ListShareTokens(ctx, &wikiv1.ListShareTokensRequest{TenantId: validID, ArticleId: "bad"})
			return err
		}},
		{"RevokeShareToken invalid tenant_id", func() error {
			_, err := srv.RevokeShareToken(ctx, &wikiv1.RevokeShareTokenRequest{TenantId: "bad", TokenId: validID})
			return err
		}},
		{"RevokeShareToken invalid token_id", func() error {
			_, err := srv.RevokeShareToken(ctx, &wikiv1.RevokeShareTokenRequest{TenantId: validID, TokenId: "bad"})
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertGRPCCode(t, tc.call(), codes.InvalidArgument)
		})
	}
}

// ============================================================================
// Article cluster — happy path plus both content forms
// ============================================================================

func TestCreateArticle_BothContentFormsPassThrough(t *testing.T) {
	repo := newStubWikiRepo()
	srv, _ := newWikiServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	authorID := uuid.New().String()

	tests := []struct {
		name    string
		content string
	}{
		{"block-doc JSON object", blockDocContent},
		{"legacy HTML wrapped as a JSON string", legacyHTMLContent},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := srv.CreateArticle(ctx, &wikiv1.CreateArticleRequest{
				TenantId: tenantID,
				AuthorId: authorID,
				Title:    "Article " + tc.name,
				Content:  []byte(tc.content),
			})
			requireGRPCOK(t, err)
			if string(resp.Article.Content) != tc.content {
				t.Fatalf("content mismatch: got %q want %q", resp.Article.Content, tc.content)
			}

			got, err := srv.GetArticle(ctx, &wikiv1.GetArticleRequest{TenantId: tenantID, ArticleId: resp.Article.Id})
			requireGRPCOK(t, err)
			if string(got.Article.Content) != tc.content {
				t.Fatalf("round-tripped content mismatch: got %q want %q", got.Article.Content, tc.content)
			}
		})
	}
}

func TestGetArticle_NotFound(t *testing.T) {
	repo := newStubWikiRepo()
	srv, _ := newWikiServerWithRepo(repo)
	_, err := srv.GetArticle(context.Background(), &wikiv1.GetArticleRequest{
		TenantId:  uuid.New().String(),
		ArticleId: uuid.New().String(),
	})
	assertGRPCCode(t, err, codes.NotFound)
}

func TestListArticles_EmptyResultIsEmptySliceNotNull(t *testing.T) {
	repo := newStubWikiRepo()
	srv, _ := newWikiServerWithRepo(repo)
	resp, err := srv.ListArticles(context.Background(), &wikiv1.ListArticlesRequest{TenantId: uuid.New().String()})
	requireGRPCOK(t, err)
	if resp.Articles == nil {
		t.Fatal("expected an empty slice, got nil")
	}
	if len(resp.Articles) != 0 {
		t.Fatalf("expected 0 articles, got %d", len(resp.Articles))
	}
	if resp.Total != 0 {
		t.Fatalf("expected total 0, got %d", resp.Total)
	}
}

func TestUpdateArticle_ClearCategoryWithEmptyString(t *testing.T) {
	repo := newStubWikiRepo()
	srv, _ := newWikiServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	authorID := uuid.New().String()

	catResp, err := srv.CreateCategory(ctx, &wikiv1.CreateCategoryRequest{TenantId: tenantID, Name: "Category"})
	requireGRPCOK(t, err)

	created, err := srv.CreateArticle(ctx, &wikiv1.CreateArticleRequest{
		TenantId:   tenantID,
		AuthorId:   authorID,
		Title:      "Article",
		CategoryId: &catResp.Category.Id,
	})
	requireGRPCOK(t, err)
	if created.Article.CategoryId == nil || *created.Article.CategoryId != catResp.Category.Id {
		t.Fatalf("expected category to be set on create")
	}

	empty := ""
	updated, err := srv.UpdateArticle(ctx, &wikiv1.UpdateArticleRequest{
		TenantId:   tenantID,
		ArticleId:  created.Article.Id,
		EditorId:   authorID,
		CategoryId: &empty,
	})
	requireGRPCOK(t, err)
	if updated.Article.CategoryId != nil {
		t.Fatalf("expected category to be cleared, got %v", *updated.Article.CategoryId)
	}
}

func TestUpdateArticle_InvalidContentJSON(t *testing.T) {
	repo := newStubWikiRepo()
	srv, _ := newWikiServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	authorID := uuid.New().String()

	created, err := srv.CreateArticle(ctx, &wikiv1.CreateArticleRequest{TenantId: tenantID, AuthorId: authorID, Title: "Article"})
	requireGRPCOK(t, err)

	_, err = srv.UpdateArticle(ctx, &wikiv1.UpdateArticleRequest{
		TenantId:  tenantID,
		ArticleId: created.Article.Id,
		EditorId:  authorID,
		Content:   []byte("not json"),
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestDeleteArticle_UnknownID(t *testing.T) {
	repo := newStubWikiRepo()
	srv, _ := newWikiServerWithRepo(repo)
	_, err := srv.DeleteArticle(context.Background(), &wikiv1.DeleteArticleRequest{
		TenantId:  uuid.New().String(),
		ArticleId: uuid.New().String(),
	})
	assertGRPCCode(t, err, codes.NotFound)
}

func TestSearchArticles_HappyPath(t *testing.T) {
	repo := newStubWikiRepo()
	srv, _ := newWikiServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	authorID := uuid.New().String()

	_, err := srv.CreateArticle(ctx, &wikiv1.CreateArticleRequest{TenantId: tenantID, AuthorId: authorID, Title: "Findable Article"})
	requireGRPCOK(t, err)

	resp, err := srv.SearchArticles(ctx, &wikiv1.SearchArticlesRequest{TenantId: tenantID, Query: "Findable", Limit: 10})
	requireGRPCOK(t, err)
	if len(resp.Articles) != 1 {
		t.Fatalf("expected 1 match, got %d", len(resp.Articles))
	}
}

func TestSearchArticles_EmptyQueryReturnsNoArticles(t *testing.T) {
	repo := newStubWikiRepo()
	srv, _ := newWikiServerWithRepo(repo)
	resp, err := srv.SearchArticles(context.Background(), &wikiv1.SearchArticlesRequest{TenantId: uuid.New().String(), Query: ""})
	requireGRPCOK(t, err)
	if len(resp.Articles) != 0 {
		t.Fatalf("expected no matches for an empty query, got %d", len(resp.Articles))
	}
}

func TestListCategories_HappyPath(t *testing.T) {
	repo := newStubWikiRepo()
	srv, _ := newWikiServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()

	_, err := srv.CreateCategory(ctx, &wikiv1.CreateCategoryRequest{TenantId: tenantID, Name: "Category A"})
	requireGRPCOK(t, err)

	resp, err := srv.ListCategories(ctx, &wikiv1.ListCategoriesRequest{TenantId: tenantID})
	requireGRPCOK(t, err)
	if len(resp.Categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(resp.Categories))
	}
}

func TestCreateArticle_DuplicateSlugConflicts(t *testing.T) {
	repo := newStubWikiRepo()
	srv, _ := newWikiServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	authorID := uuid.New().String()

	_, err := srv.CreateArticle(ctx, &wikiv1.CreateArticleRequest{TenantId: tenantID, AuthorId: authorID, Title: "First", Slug: "same-slug"})
	requireGRPCOK(t, err)

	_, err = srv.CreateArticle(ctx, &wikiv1.CreateArticleRequest{TenantId: tenantID, AuthorId: authorID, Title: "Second", Slug: "same-slug"})
	assertGRPCCode(t, err, codes.AlreadyExists)
}

// ============================================================================
// Version cluster
// ============================================================================

func TestUpdateArticle_CreatesVersionSnapshot(t *testing.T) {
	repo := newStubWikiRepo()
	srv, _ := newWikiServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	authorID := uuid.New().String()

	created, err := srv.CreateArticle(ctx, &wikiv1.CreateArticleRequest{
		TenantId: tenantID, AuthorId: authorID, Title: "Article", Content: []byte(`{"v":1}`),
	})
	requireGRPCOK(t, err)

	_, err = srv.UpdateArticle(ctx, &wikiv1.UpdateArticleRequest{
		TenantId: tenantID, ArticleId: created.Article.Id, EditorId: authorID, Content: []byte(`{"v":2}`),
	})
	requireGRPCOK(t, err)

	versions, err := srv.ListVersions(ctx, &wikiv1.ListVersionsRequest{TenantId: tenantID, ArticleId: created.Article.Id})
	requireGRPCOK(t, err)
	if len(versions.Versions) != 1 {
		t.Fatalf("expected 1 version snapshot, got %d", len(versions.Versions))
	}
	if string(versions.Versions[0].Content) != `{"v":1}` {
		t.Fatalf("expected the snapshot to hold the pre-update content, got %q", versions.Versions[0].Content)
	}
}

func TestRestoreVersion_RevertsContentAndSnapshotsAgain(t *testing.T) {
	repo := newStubWikiRepo()
	srv, _ := newWikiServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	authorID := uuid.New().String()

	created, err := srv.CreateArticle(ctx, &wikiv1.CreateArticleRequest{
		TenantId: tenantID, AuthorId: authorID, Title: "Article", Content: []byte(`{"v":1}`),
	})
	requireGRPCOK(t, err)
	_, err = srv.UpdateArticle(ctx, &wikiv1.UpdateArticleRequest{
		TenantId: tenantID, ArticleId: created.Article.Id, EditorId: authorID, Content: []byte(`{"v":2}`),
	})
	requireGRPCOK(t, err)

	versions, err := srv.ListVersions(ctx, &wikiv1.ListVersionsRequest{TenantId: tenantID, ArticleId: created.Article.Id})
	requireGRPCOK(t, err)
	restored, err := srv.RestoreVersion(ctx, &wikiv1.RestoreVersionRequest{
		TenantId: tenantID, ArticleId: created.Article.Id, VersionId: versions.Versions[0].Id, EditorId: authorID,
	})
	requireGRPCOK(t, err)
	if string(restored.Article.Content) != `{"v":1}` {
		t.Fatalf("expected restored content %q, got %q", `{"v":1}`, restored.Article.Content)
	}
}

func TestGetVersion_NotFound(t *testing.T) {
	repo := newStubWikiRepo()
	srv, _ := newWikiServerWithRepo(repo)
	_, err := srv.GetVersion(context.Background(), &wikiv1.GetVersionRequest{
		TenantId: uuid.New().String(), VersionId: uuid.New().String(),
	})
	assertGRPCCode(t, err, codes.NotFound)
}

// ============================================================================
// Attachment cluster
// ============================================================================

func TestUploadAttachment_EmptyFileRefRejected(t *testing.T) {
	repo := newStubWikiRepo()
	srv, _ := newWikiServerWithRepo(repo)
	_, err := srv.UploadAttachment(context.Background(), &wikiv1.UploadAttachmentRequest{
		TenantId: uuid.New().String(), ArticleId: uuid.New().String(), FileRef: "",
	})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestUploadAttachment_ListAndDelete(t *testing.T) {
	repo := newStubWikiRepo()
	srv, _ := newWikiServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	articleID := uuid.New().String()

	up, err := srv.UploadAttachment(ctx, &wikiv1.UploadAttachmentRequest{
		TenantId: tenantID, ArticleId: articleID, FileRef: "s3://bucket/key", Mime: "image/png", Size: 1024,
	})
	requireGRPCOK(t, err)
	if up.Attachment.UploadedBy != nil {
		t.Fatalf("expected no uploader when uploaded_by is unset, got %v", *up.Attachment.UploadedBy)
	}

	list, err := srv.ListAttachments(ctx, &wikiv1.ListAttachmentsRequest{TenantId: tenantID, ArticleId: articleID})
	requireGRPCOK(t, err)
	if len(list.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(list.Attachments))
	}

	_, err = srv.DeleteAttachment(ctx, &wikiv1.DeleteAttachmentRequest{TenantId: tenantID, AttachmentId: up.Attachment.Id})
	requireGRPCOK(t, err)

	_, err = srv.DeleteAttachment(ctx, &wikiv1.DeleteAttachmentRequest{TenantId: tenantID, AttachmentId: up.Attachment.Id})
	assertGRPCCode(t, err, codes.NotFound)
}

// ============================================================================
// Category cluster
// ============================================================================

func TestCreateCategory_EmptyNameRejected(t *testing.T) {
	repo := newStubWikiRepo()
	srv, _ := newWikiServerWithRepo(repo)
	_, err := srv.CreateCategory(context.Background(), &wikiv1.CreateCategoryRequest{TenantId: uuid.New().String(), Name: "  "})
	assertGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateCategory_UnknownParentNotFound(t *testing.T) {
	repo := newStubWikiRepo()
	srv, _ := newWikiServerWithRepo(repo)
	parent := uuid.New().String()
	_, err := srv.CreateCategory(context.Background(), &wikiv1.CreateCategoryRequest{
		TenantId: uuid.New().String(), Name: "Child", ParentId: &parent,
	})
	assertGRPCCode(t, err, codes.NotFound)
}

func TestUpdateCategory_ClearParentWithEmptyString(t *testing.T) {
	repo := newStubWikiRepo()
	srv, _ := newWikiServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()

	parent, err := srv.CreateCategory(ctx, &wikiv1.CreateCategoryRequest{TenantId: tenantID, Name: "Parent"})
	requireGRPCOK(t, err)
	child, err := srv.CreateCategory(ctx, &wikiv1.CreateCategoryRequest{TenantId: tenantID, Name: "Child", ParentId: &parent.Category.Id})
	requireGRPCOK(t, err)
	if child.Category.ParentId == nil {
		t.Fatal("expected parent to be set on create")
	}

	empty := ""
	updated, err := srv.UpdateCategory(ctx, &wikiv1.UpdateCategoryRequest{
		TenantId: tenantID, CategoryId: child.Category.Id, ParentId: &empty,
	})
	requireGRPCOK(t, err)
	if updated.Category.ParentId != nil {
		t.Fatalf("expected parent to be cleared, got %v", *updated.Category.ParentId)
	}
}

func TestDeleteCategory_UnknownIDNotFound(t *testing.T) {
	repo := newStubWikiRepo()
	srv, _ := newWikiServerWithRepo(repo)
	_, err := srv.DeleteCategory(context.Background(), &wikiv1.DeleteCategoryRequest{
		TenantId: uuid.New().String(), CategoryId: uuid.New().String(),
	})
	assertGRPCCode(t, err, codes.NotFound)
}

// ============================================================================
// Share token cluster — this is the load-bearing part of this unit: a
// revoked token and an unknown token must be indistinguishable to the caller.
// ============================================================================

func setupPublishedArticleWithToken(t *testing.T, srv *WikiGRPCServer, permissions []string) (tenantID, articleID string, token *wikiv1.ShareToken) {
	t.Helper()
	ctx := context.Background()
	tenantID = uuid.New().String()
	authorID := uuid.New().String()

	created, err := srv.CreateArticle(ctx, &wikiv1.CreateArticleRequest{
		TenantId: tenantID, AuthorId: authorID, Title: "Shared", Content: []byte(blockDocContent), Published: true,
	})
	requireGRPCOK(t, err)

	resp, err := srv.CreateShareToken(ctx, &wikiv1.CreateShareTokenRequest{
		TenantId: tenantID, ArticleId: created.Article.Id, Permissions: permissions,
	})
	requireGRPCOK(t, err)
	return tenantID, created.Article.Id, resp.Token
}

func TestRedeemShareToken_UnknownTokenIsNotFound(t *testing.T) {
	repo := newStubWikiRepo()
	srv, _ := newWikiServerWithRepo(repo)
	_, err := srv.RedeemShareToken(context.Background(), &wikiv1.RedeemShareTokenRequest{Token: "does-not-exist"})
	assertGRPCCode(t, err, codes.NotFound)
}

func TestRedeemShareToken_RevokedAndUnknownProduceTheSameAnswer(t *testing.T) {
	repo := newStubWikiRepo()
	srv, _ := newWikiServerWithRepo(repo)
	ctx := context.Background()

	tenantID, _, revokable := setupPublishedArticleWithToken(t, srv, []string{"read"})
	_, err := srv.RevokeShareToken(ctx, &wikiv1.RevokeShareTokenRequest{TenantId: tenantID, TokenId: revokable.Id})
	requireGRPCOK(t, err)

	_, revokedErr := srv.RedeemShareToken(ctx, &wikiv1.RedeemShareTokenRequest{Token: revokable.Token})
	_, unknownErr := srv.RedeemShareToken(ctx, &wikiv1.RedeemShareTokenRequest{Token: "some-token-that-was-never-minted"})

	assertGRPCCode(t, revokedErr, codes.NotFound)
	assertGRPCCode(t, unknownErr, codes.NotFound)
	if revokedErr.Error() != unknownErr.Error() {
		t.Fatalf("revoked and unknown tokens must be indistinguishable, got %q vs %q", revokedErr.Error(), unknownErr.Error())
	}
}

func TestRedeemShareToken_ExpiredTokenMatchesUnknownAnswer(t *testing.T) {
	repo := newStubWikiRepo()
	srv, _ := newWikiServerWithRepo(repo)
	ctx := context.Background()
	tenantID, articleID, _ := setupPublishedArticleWithToken(t, srv, nil)

	past := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	resp, err := srv.CreateShareToken(ctx, &wikiv1.CreateShareTokenRequest{
		TenantId: tenantID, ArticleId: articleID, ExpiresAt: &past,
	})
	requireGRPCOK(t, err)

	_, expiredErr := srv.RedeemShareToken(ctx, &wikiv1.RedeemShareTokenRequest{Token: resp.Token.Token})
	_, unknownErr := srv.RedeemShareToken(ctx, &wikiv1.RedeemShareTokenRequest{Token: "never-minted"})

	assertGRPCCode(t, expiredErr, codes.NotFound)
	if expiredErr.Error() != unknownErr.Error() {
		t.Fatalf("expired and unknown tokens must be indistinguishable, got %q vs %q", expiredErr.Error(), unknownErr.Error())
	}
}

func TestRedeemShareToken_NoReadPermissionMatchesUnknownAnswer(t *testing.T) {
	repo := newStubWikiRepo()
	srv, _ := newWikiServerWithRepo(repo)
	_, _, token := setupPublishedArticleWithToken(t, srv, []string{"write"}) // no "read"

	_, err := srv.RedeemShareToken(context.Background(), &wikiv1.RedeemShareTokenRequest{Token: token.Token})
	assertGRPCCode(t, err, codes.NotFound)
}

func TestRedeemShareToken_UnpublishedArticleMatchesUnknownAnswer(t *testing.T) {
	repo := newStubWikiRepo()
	srv, _ := newWikiServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New().String()
	authorID := uuid.New().String()

	created, err := srv.CreateArticle(ctx, &wikiv1.CreateArticleRequest{
		TenantId: tenantID, AuthorId: authorID, Title: "Draft", Published: false,
	})
	requireGRPCOK(t, err)
	tokenResp, err := srv.CreateShareToken(ctx, &wikiv1.CreateShareTokenRequest{TenantId: tenantID, ArticleId: created.Article.Id})
	requireGRPCOK(t, err)

	_, err = srv.RedeemShareToken(ctx, &wikiv1.RedeemShareTokenRequest{Token: tokenResp.Token.Token})
	assertGRPCCode(t, err, codes.NotFound)
}

func TestRedeemShareToken_ValidTokenReturnsNarrowedSharedArticle(t *testing.T) {
	repo := newStubWikiRepo()
	srv, _ := newWikiServerWithRepo(repo)
	_, _, token := setupPublishedArticleWithToken(t, srv, []string{"read"})

	resp, err := srv.RedeemShareToken(context.Background(), &wikiv1.RedeemShareTokenRequest{Token: token.Token})
	requireGRPCOK(t, err)
	if resp.Article.Title != "Shared" {
		t.Fatalf("expected title %q, got %q", "Shared", resp.Article.Title)
	}
	if string(resp.Article.Content) != blockDocContent {
		t.Fatalf("expected content to round-trip, got %q", resp.Article.Content)
	}
}

func TestRevokeShareToken_TwiceIsIdempotentAndKeepsFirstTimestamp(t *testing.T) {
	repo := newStubWikiRepo()
	srv, _ := newWikiServerWithRepo(repo)
	ctx := context.Background()
	tenantID, _, token := setupPublishedArticleWithToken(t, srv, []string{"read"})

	_, err := srv.RevokeShareToken(ctx, &wikiv1.RevokeShareTokenRequest{TenantId: tenantID, TokenId: token.Id})
	requireGRPCOK(t, err)
	firstRevokedAt := repo.shareTokens[uuid.MustParse(token.Id)].RevokedAt

	_, err = srv.RevokeShareToken(ctx, &wikiv1.RevokeShareTokenRequest{TenantId: tenantID, TokenId: token.Id})
	requireGRPCOK(t, err)
	secondRevokedAt := repo.shareTokens[uuid.MustParse(token.Id)].RevokedAt

	if !firstRevokedAt.Equal(*secondRevokedAt) {
		t.Fatalf("expected revoked_at to stay put across a second revoke, got %v then %v", firstRevokedAt, secondRevokedAt)
	}
}

func TestListShareTokens_UnknownArticleNotFound(t *testing.T) {
	repo := newStubWikiRepo()
	srv, _ := newWikiServerWithRepo(repo)
	_, err := srv.ListShareTokens(context.Background(), &wikiv1.ListShareTokensRequest{
		TenantId: uuid.New().String(), ArticleId: uuid.New().String(),
	})
	assertGRPCCode(t, err, codes.NotFound)
}

// ============================================================================
// Error mapping — table test
// ============================================================================

func TestMapWikiError_Table(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want codes.Code
	}{
		{"nil", nil, codes.OK},
		{"article not found", wiki.ErrArticleNotFound, codes.NotFound},
		{"version not found", wiki.ErrVersionNotFound, codes.NotFound},
		{"category not found", wiki.ErrCategoryNotFound, codes.NotFound},
		{"attachment not found", wiki.ErrAttachmentNotFound, codes.NotFound},
		{"share token not found", wiki.ErrShareTokenNotFound, codes.NotFound},
		{"slug taken", wiki.ErrSlugTaken, codes.AlreadyExists},
		{"invalid content", wiki.ErrInvalidContent, codes.InvalidArgument},
		{"generic unmapped error", errors.New("boom"), codes.Internal},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mapWikiError(tc.err)
			if tc.err == nil {
				if got != nil {
					t.Fatalf("expected nil, got %v", got)
				}
				return
			}
			assertGRPCCode(t, got, tc.want)
		})
	}
}

func TestCreateArticle_RepoFailureMapsToInternal(t *testing.T) {
	repo := newStubWikiRepo()
	repo.forceErr = errStubWikiRepoFailure
	srv, _ := newWikiServerWithRepo(repo)
	_, err := srv.CreateArticle(context.Background(), &wikiv1.CreateArticleRequest{
		TenantId: uuid.New().String(), AuthorId: uuid.New().String(), Title: "T",
	})
	assertGRPCCode(t, err, codes.Internal)
}
