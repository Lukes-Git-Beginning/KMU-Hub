package wiki

// The public redemption path is the only one in this service that starts
// without a tenant, so it is also the only one where a wrong context is a
// cross-tenant read rather than a bug. These tests drive the repository and
// service methods the handler actually calls (never raw SQL), against a real
// database with RLS on, and pin the two properties that matter:
//
//   - the token lookup resolves WITHOUT a tenant in the session (that is the
//     one read allowed to escape RLS — which tenant may be seen is exactly
//     what it answers), and
//   - nothing past that lookup escapes: a token whose article belongs to
//     another tenant reads nothing, because the article read runs under the
//     tenant the token itself named.
//
// Own tenants rather than the shared TenantA/B, so parallel tests cannot
// disturb the counts.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestWikiShareRedeem_ResolvesTenantWithoutOne(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Wiki Share Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Wiki Share Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	repo := NewPostgresRepository(pool)
	svc := NewService(repo)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	article := &Article{
		ID:        uuid.New(),
		TenantID:  tenantOwn,
		Title:     "Share-Test-" + uuid.New().String()[:8],
		Slug:      "share-test-" + uuid.New().String()[:8],
		Content:   []byte(`{"type":"doc"}`),
		AuthorID:  uuid.New(),
		Published: true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.CreateArticle(ctxOwn, article); err != nil {
		t.Fatalf("CreateArticle: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "wiki_articles", article.ID)

	share, err := svc.CreateShareToken(ctxOwn, CreateShareTokenInput{
		TenantID:  tenantOwn,
		ArticleID: article.ID,
	})
	if err != nil {
		t.Fatalf("CreateShareToken: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "wiki_share_tokens", share.ID)

	// A visitor's context carries no tenant at all. Without the system context
	// inside the lookup, RLS would filter away the one row that answers which
	// tenant this request belongs to, and the whole path would 404.
	visitor := context.Background()

	resolved, err := repo.GetShareTokenByToken(visitor, share.Token)
	if err != nil {
		t.Fatalf("GetShareTokenByToken (no tenant in ctx): %v", err)
	}
	if resolved.TenantID != tenantOwn {
		t.Fatalf("resolved tenant = %s; want %s", resolved.TenantID, tenantOwn)
	}

	got, err := svc.RedeemShareToken(visitor, share.Token)
	if err != nil {
		t.Fatalf("RedeemShareToken (no tenant in ctx): %v", err)
	}
	if got.Title != article.Title {
		t.Fatalf("redeemed title = %q; want %q", got.Title, article.Title)
	}

	// A foreign tenant in the context must not redirect the read either: the
	// tenant comes from the token, not from whoever is calling.
	got, err = svc.RedeemShareToken(ctxOther, share.Token)
	if err != nil {
		t.Fatalf("RedeemShareToken (foreign ctx): %v", err)
	}
	if got.Title != article.Title {
		t.Fatalf("redeemed title under foreign ctx = %q; want %q", got.Title, article.Title)
	}

	// Unpublishing must kill a link already handed out.
	article.Published = false
	if err := repo.UpdateArticle(ctxOwn, article); err != nil {
		t.Fatalf("UpdateArticle: %v", err)
	}
	if _, err := svc.RedeemShareToken(visitor, share.Token); !errors.Is(err, ErrShareTokenNotFound) {
		t.Fatalf("RedeemShareToken (unpublished): got %v, want ErrShareTokenNotFound", err)
	}
}

// A token whose tenant and article disagree must read nothing. This is the
// shape a tenant bypass would take if the system context were carried past the
// lookup: the token row resolves, and the article read that follows would then
// see every tenant's rows instead of one.
func TestWikiShareRedeem_TokenCannotReachForeignArticle(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Wiki Share Cross Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Wiki Share Cross Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	repo := NewPostgresRepository(pool)
	svc := NewService(repo)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	foreign := &Article{
		ID:        uuid.New(),
		TenantID:  tenantOther,
		Title:     "Fremd-" + uuid.New().String()[:8],
		Slug:      "fremd-" + uuid.New().String()[:8],
		Content:   []byte(`{"type":"doc"}`),
		AuthorID:  uuid.New(),
		Published: true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.CreateArticle(ctxOther, foreign); err != nil {
		t.Fatalf("CreateArticle (other tenant): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "wiki_articles", foreign.ID)

	// The token belongs to tenantOwn but names tenantOther's article — the FK
	// permits it, so only the tenant-scoped read stands in the way.
	crossed := &ShareToken{
		ID:          uuid.New(),
		TenantID:    tenantOwn,
		ArticleID:   foreign.ID,
		Token:       uuid.New().String(),
		Permissions: []string{sharePermissionRead},
		CreatedAt:   now,
	}
	if err := repo.CreateShareToken(ctxOwn, crossed); err != nil {
		t.Fatalf("CreateShareToken (crossed): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "wiki_share_tokens", crossed.ID)

	if _, err := svc.RedeemShareToken(context.Background(), crossed.Token); !errors.Is(err, ErrShareTokenNotFound) {
		t.Fatalf("RedeemShareToken (crossed token): got %v, want ErrShareTokenNotFound", err)
	}
}

// Revocation against the real schema (000297). The soft cut is the whole point
// here: the row survives, still resolves by its secret in the system context,
// and is refused only because Usable() says so. A test that deleted the row
// would prove nothing about that check.
func TestWikiShareRedeem_RevokedTokenIsRefused(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Wiki Share Revoke Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Wiki Share Revoke Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	repo := NewPostgresRepository(pool)
	svc := NewService(repo)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	article := &Article{
		ID:        uuid.New(),
		TenantID:  tenantOwn,
		Title:     "Revoke-Test-" + uuid.New().String()[:8],
		Slug:      "revoke-test-" + uuid.New().String()[:8],
		Content:   []byte(`{"type":"doc"}`),
		AuthorID:  uuid.New(),
		Published: true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.CreateArticle(ctxOwn, article); err != nil {
		t.Fatalf("CreateArticle: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "wiki_articles", article.ID)

	share, err := svc.CreateShareToken(ctxOwn, CreateShareTokenInput{
		TenantID:  tenantOwn,
		ArticleID: article.ID,
	})
	if err != nil {
		t.Fatalf("CreateShareToken: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "wiki_share_tokens", share.ID)

	visitor := context.Background()
	if _, err := svc.RedeemShareToken(visitor, share.Token); err != nil {
		t.Fatalf("RedeemShareToken before revocation: %v", err)
	}

	// A foreign tenant must not be able to cut somebody else's link, and the
	// failed attempt must leave no mark.
	if err := svc.RevokeShareToken(ctxOther, tenantOwn, share.ID); !errors.Is(err, ErrShareTokenNotFound) {
		t.Fatalf("RevokeShareToken (foreign ctx): got %v, want ErrShareTokenNotFound", err)
	}
	if _, err := svc.RedeemShareToken(visitor, share.Token); err != nil {
		t.Fatalf("a refused cross-tenant revoke killed the link anyway: %v", err)
	}

	if err := svc.RevokeShareToken(ctxOwn, tenantOwn, share.ID); err != nil {
		t.Fatalf("RevokeShareToken (own ctx): %v", err)
	}

	// The row is still there and still resolves; only Usable() refuses it.
	resolved, err := repo.GetShareTokenByToken(visitor, share.Token)
	if err != nil {
		t.Fatalf("GetShareTokenByToken after revocation: %v", err)
	}
	if resolved.RevokedAt == nil {
		t.Fatal("revoked_at did not come back on the read")
	}
	if resolved.Usable(time.Now().UTC()) {
		t.Fatal("a revoked token still reports itself as usable")
	}

	// Same 404 an invented token gets.
	if _, err := svc.RedeemShareToken(visitor, share.Token); !errors.Is(err, ErrShareTokenNotFound) {
		t.Fatalf("RedeemShareToken after revocation: got %v, want ErrShareTokenNotFound", err)
	}
	if _, err := svc.RedeemShareToken(visitor, uuid.New().String()); !errors.Is(err, ErrShareTokenNotFound) {
		t.Fatalf("RedeemShareToken (invented token): got %v, want ErrShareTokenNotFound", err)
	}
}
