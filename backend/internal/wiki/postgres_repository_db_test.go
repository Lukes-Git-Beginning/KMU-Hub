package wiki

// Read-path coverage for PostgresRepository against real SQL and real RLS.
// tenant_write_test.go and tenant_isolation_phase2_test.go already prove the
// write paths and the raw-RLS shape; this file closes the gap on the reads
// and on categories, which neither of those touch: GetArticleByID/BySlug,
// ListArticles (filters, pagination, sort), SlugExists, SearchArticles,
// UpdateArticle, DeleteArticle, ListAttachments, and the full category CRUD.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestWikiArticleReads_ScopeToCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Wiki Read Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Wiki Read Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	article := &Article{
		ID:        uuid.New(),
		TenantID:  tenantOwn,
		Title:     "Read-Test-" + uuid.New().String()[:8],
		Slug:      "read-test-" + uuid.New().String()[:8],
		Content:   []byte(`{"plain":"Kundenbericht wichtig"}`),
		AuthorID:  uuid.New(),
		Published: true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.CreateArticle(ctxOwn, article); err != nil {
		t.Fatalf("CreateArticle: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "wiki_articles", article.ID)

	// --- GetArticleByID / GetArticleBySlug ---

	got, err := repo.GetArticleByID(ctxOwn, tenantOwn, article.ID)
	if err != nil {
		t.Fatalf("GetArticleByID (own ctx): %v", err)
	}
	if got.Title != article.Title {
		t.Fatalf("GetArticleByID title = %q, want %q", got.Title, article.Title)
	}
	if _, err := repo.GetArticleByID(ctxOther, tenantOwn, article.ID); err != ErrArticleNotFound {
		t.Fatalf("GetArticleByID (foreign ctx): got %v, want ErrArticleNotFound", err)
	}

	got, err = repo.GetArticleBySlug(ctxOwn, tenantOwn, article.Slug)
	if err != nil {
		t.Fatalf("GetArticleBySlug (own ctx): %v", err)
	}
	if got.ID != article.ID {
		t.Fatalf("GetArticleBySlug returned wrong article: %v", got.ID)
	}
	if _, err := repo.GetArticleBySlug(ctxOther, tenantOwn, article.Slug); err != ErrArticleNotFound {
		t.Fatalf("GetArticleBySlug (foreign ctx): got %v, want ErrArticleNotFound", err)
	}

	// --- SlugExists ---

	exists, err := repo.SlugExists(ctxOwn, tenantOwn, article.Slug, nil)
	if err != nil {
		t.Fatalf("SlugExists: %v", err)
	}
	if !exists {
		t.Fatal("SlugExists = false for a slug that was just created")
	}
	exists, err = repo.SlugExists(ctxOwn, tenantOwn, article.Slug, &article.ID)
	if err != nil {
		t.Fatalf("SlugExists (excludeID): %v", err)
	}
	if exists {
		t.Fatal("SlugExists = true when excluding the article's own id")
	}
	exists, err = repo.SlugExists(ctxOther, tenantOwn, article.Slug, nil)
	if err != nil {
		t.Fatalf("SlugExists (foreign ctx): %v", err)
	}
	if exists {
		t.Fatal("SlugExists = true from a foreign tenant's session")
	}

	// --- ListArticles: filter + pagination + sort ---

	articles, total, err := repo.ListArticles(ctxOwn, tenantOwn, ListArticlesFilter{Search: article.Title[:9]}, 0, 10)
	if err != nil {
		t.Fatalf("ListArticles (search filter): %v", err)
	}
	if total != 1 || len(articles) != 1 || articles[0].ID != article.ID {
		t.Fatalf("ListArticles (search filter): total=%d len=%d, want the one seeded article", total, len(articles))
	}

	published := true
	articles, total, err = repo.ListArticles(ctxOwn, tenantOwn, ListArticlesFilter{Published: &published}, 0, 10)
	if err != nil {
		t.Fatalf("ListArticles (published filter): %v", err)
	}
	if total != 1 || len(articles) != 1 {
		t.Fatalf("ListArticles (published filter): total=%d len=%d, want 1", total, len(articles))
	}

	unpublished := false
	_, total, err = repo.ListArticles(ctxOwn, tenantOwn, ListArticlesFilter{Published: &unpublished}, 0, 10)
	if err != nil {
		t.Fatalf("ListArticles (unpublished filter): %v", err)
	}
	if total != 0 {
		t.Fatalf("ListArticles (unpublished filter): total=%d, want 0", total)
	}

	_, total, err = repo.ListArticles(ctxOther, tenantOwn, ListArticlesFilter{}, 0, 10)
	if err != nil {
		t.Fatalf("ListArticles (foreign ctx): %v", err)
	}
	if total != 0 {
		t.Fatalf("ListArticles (foreign ctx): total=%d, want 0 — RLS must hide the row", total)
	}

	// --- SearchArticles: full-text, published-only, tenant-scoped ---

	tsquery := BuildSearchQuery("Kundenbericht")
	results, err := repo.SearchArticles(ctxOwn, tenantOwn, tsquery, 10)
	if err != nil {
		t.Fatalf("SearchArticles (own ctx): %v", err)
	}
	if len(results) != 1 || results[0].ID != article.ID {
		t.Fatalf("SearchArticles (own ctx): got %d results, want the seeded article", len(results))
	}
	results, err = repo.SearchArticles(ctxOther, tenantOwn, tsquery, 10)
	if err != nil {
		t.Fatalf("SearchArticles (foreign ctx): %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("SearchArticles (foreign ctx): got %d results, want 0 — RLS must hide the row", len(results))
	}

	// --- UpdateArticle ---

	article.Title = "Updated-" + article.Title
	article.Published = false
	article.UpdatedAt = time.Now().UTC()
	if err := repo.UpdateArticle(ctxOwn, article); err != nil {
		t.Fatalf("UpdateArticle (own ctx): %v", err)
	}
	got, err = repo.GetArticleByID(ctxOwn, tenantOwn, article.ID)
	if err != nil {
		t.Fatalf("GetArticleByID after update: %v", err)
	}
	if got.Title != article.Title || got.Published {
		t.Fatalf("UpdateArticle did not persist: title=%q published=%v", got.Title, got.Published)
	}

	// An unpublished article must fall out of SearchArticles even though the
	// tsquery still matches — SearchArticles filters `published = TRUE`.
	results, err = repo.SearchArticles(ctxOwn, tenantOwn, tsquery, 10)
	if err != nil {
		t.Fatalf("SearchArticles after unpublish: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("SearchArticles after unpublish: got %d results, want 0", len(results))
	}

	// --- ListAttachments ---

	attachment := &Attachment{
		ID:        uuid.New(),
		TenantID:  tenantOwn,
		ArticleID: article.ID,
		FileRef:   "uploads/" + uuid.New().String() + ".pdf",
		Mime:      "application/pdf",
		Size:      2048,
		CreatedAt: now,
	}
	if err := repo.CreateAttachment(ctxOwn, attachment); err != nil {
		t.Fatalf("CreateAttachment: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "wiki_attachments", attachment.ID)

	attachments, err := repo.ListAttachments(ctxOwn, tenantOwn, article.ID)
	if err != nil {
		t.Fatalf("ListAttachments (own ctx): %v", err)
	}
	if len(attachments) != 1 || attachments[0].ID != attachment.ID {
		t.Fatalf("ListAttachments (own ctx): got %d, want 1", len(attachments))
	}
	attachments, err = repo.ListAttachments(ctxOther, tenantOwn, article.ID)
	if err != nil {
		t.Fatalf("ListAttachments (foreign ctx): %v", err)
	}
	if len(attachments) != 0 {
		t.Fatalf("ListAttachments (foreign ctx): got %d, want 0", len(attachments))
	}

	// --- DeleteArticle ---

	if err := repo.DeleteArticle(ctxOther, tenantOwn, article.ID); err != nil {
		t.Fatalf("DeleteArticle (foreign ctx) returned an error instead of a silent no-op: %v", err)
	}
	testutil.AssertRowCount(t, pool, testutil.WithSystemCtx(context.Background()), "wiki_articles", article.ID, 1)

	if err := repo.DeleteArticle(ctxOwn, tenantOwn, article.ID); err != nil {
		t.Fatalf("DeleteArticle (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, testutil.WithSystemCtx(context.Background()), "wiki_articles", article.ID, 0)
}

func TestWikiCategories_CRUDScopesToCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Wiki Category Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Wiki Category Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	category := &Category{
		ID:        uuid.New(),
		TenantID:  tenantOwn,
		Name:      "Category-" + uuid.New().String()[:8],
		Position:  1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.CreateCategory(ctxOther, category); err == nil {
		t.Fatal("CreateCategory (foreign ctx): expected an RLS error, got nil")
	}
	if err := repo.CreateCategory(ctxOwn, category); err != nil {
		t.Fatalf("CreateCategory (own ctx): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "wiki_categories", category.ID)

	got, err := repo.GetCategory(ctxOwn, tenantOwn, category.ID)
	if err != nil {
		t.Fatalf("GetCategory (own ctx): %v", err)
	}
	if got.Name != category.Name {
		t.Fatalf("GetCategory name = %q, want %q", got.Name, category.Name)
	}
	if _, err := repo.GetCategory(ctxOther, tenantOwn, category.ID); err != ErrCategoryNotFound {
		t.Fatalf("GetCategory (foreign ctx): got %v, want ErrCategoryNotFound", err)
	}

	categories, err := repo.ListCategories(ctxOwn, tenantOwn)
	if err != nil {
		t.Fatalf("ListCategories (own ctx): %v", err)
	}
	found := false
	for _, c := range categories {
		if c.ID == category.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("ListCategories (own ctx): seeded category not in result")
	}
	categories, err = repo.ListCategories(ctxOther, tenantOwn)
	if err != nil {
		t.Fatalf("ListCategories (foreign ctx): %v", err)
	}
	for _, c := range categories {
		if c.ID == category.ID {
			t.Fatal("ListCategories (foreign ctx): saw another tenant's category")
		}
	}

	category.Name = "Renamed-" + category.Name
	category.Position = 2
	category.UpdatedAt = time.Now().UTC()
	if err := repo.UpdateCategory(ctxOwn, category); err != nil {
		t.Fatalf("UpdateCategory (own ctx): %v", err)
	}
	got, err = repo.GetCategory(ctxOwn, tenantOwn, category.ID)
	if err != nil {
		t.Fatalf("GetCategory after update: %v", err)
	}
	if got.Name != category.Name || got.Position != 2 {
		t.Fatalf("UpdateCategory did not persist: name=%q position=%d", got.Name, got.Position)
	}

	// A foreign tenant's delete must not remove the row — RLS's USING clause
	// hides it from the DELETE entirely.
	if err := repo.DeleteCategory(ctxOther, tenantOwn, category.ID); err != nil {
		t.Fatalf("DeleteCategory (foreign ctx) returned an error instead of a silent no-op: %v", err)
	}
	testutil.AssertRowCount(t, pool, testutil.WithSystemCtx(context.Background()), "wiki_categories", category.ID, 1)

	if err := repo.DeleteCategory(ctxOwn, tenantOwn, category.ID); err != nil {
		t.Fatalf("DeleteCategory (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, testutil.WithSystemCtx(context.Background()), "wiki_categories", category.ID, 0)
}
