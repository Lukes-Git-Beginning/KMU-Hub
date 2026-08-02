package file

// DB-level tests for document_share_links: proves the SQL layer persists,
// filters by tenant_id, and that GetShareLinkByToken resolves without a
// tenant in the session (the public redemption path has none yet). Service
// contract tests (password/expiry/collapsed-error behaviour) live in
// service_test.go against the mock repository.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestPostgresRepository_CreateAndListShareLinks(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	fx := seedActivityFixture(t, pool, "sharelink-create-list")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	hash := "bcrypt-hash-placeholder"
	expires := time.Now().Add(24 * time.Hour)
	link := &models.DocumentShareLink{
		ID: uuid.New(), TenantID: fx.tenant, FileID: fx.file,
		Token: uuid.NewString(), PasswordHash: &hash, ExpiresAt: &expires,
		CreatedBy: &fx.user, CreatedAt: time.Now(),
	}
	if err := repo.CreateShareLink(ctx, link); err != nil {
		t.Fatalf("CreateShareLink: %v", err)
	}

	links, err := repo.ListShareLinks(ctx, fx.file, fx.tenant)
	if err != nil {
		t.Fatalf("ListShareLinks: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("ListShareLinks returned %d entries, want 1", len(links))
	}
	got := links[0]
	if got.ID != link.ID || got.TenantID != fx.tenant || got.Token != link.Token {
		t.Errorf("ListShareLinks[0] = %+v, want ID=%s TenantID=%s Token=%s", got, link.ID, fx.tenant, link.Token)
	}
	if got.PasswordHash == nil || *got.PasswordHash != hash {
		t.Errorf("PasswordHash = %v, want %q", got.PasswordHash, hash)
	}
	if got.ExpiresAt == nil {
		t.Error("ExpiresAt is nil, want a value")
	}
	if got.RevokedAt != nil {
		t.Errorf("RevokedAt = %v, want nil for a freshly created link", got.RevokedAt)
	}
}

func TestPostgresRepository_ListShareLinks_TenantIsolation(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	fx := seedActivityFixture(t, pool, "sharelink-tenant-isolation")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	if err := repo.CreateShareLink(ctx, &models.DocumentShareLink{
		ID: uuid.New(), TenantID: fx.tenant, FileID: fx.file,
		Token: uuid.NewString(), CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("CreateShareLink: %v", err)
	}

	otherTenant := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenant, "document share link test other tenant")

	links, err := repo.ListShareLinks(ctx, fx.file, otherTenant)
	if err != nil {
		t.Fatalf("ListShareLinks (foreign tenant): %v", err)
	}
	if len(links) != 0 {
		t.Fatalf("ListShareLinks under a foreign tenant_id returned %d entries, want 0", len(links))
	}
}

// TestPostgresRepository_GetShareLinkByToken_NoTenantInSession proves the one
// property the public redemption path depends on: the token lookup resolves
// even when the caller's context carries no tenant at all, because it runs
// in the system context internally (see Repository.GetShareLinkByToken). A
// plain per-tenant RLS read would return zero rows here.
func TestPostgresRepository_GetShareLinkByToken_NoTenantInSession(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	fx := seedActivityFixture(t, pool, "sharelink-token-lookup")
	tenantCtx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	token := uuid.NewString()
	if err := repo.CreateShareLink(tenantCtx, &models.DocumentShareLink{
		ID: uuid.New(), TenantID: fx.tenant, FileID: fx.file,
		Token: token, CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("CreateShareLink: %v", err)
	}

	// No tenant set at all — the exact shape of an unauthenticated public request.
	link, err := repo.GetShareLinkByToken(context.Background(), token)
	if err != nil {
		t.Fatalf("GetShareLinkByToken (no tenant in session): %v", err)
	}
	if link.TenantID != fx.tenant || link.FileID != fx.file {
		t.Errorf("GetShareLinkByToken = %+v, want TenantID=%s FileID=%s", link, fx.tenant, fx.file)
	}
}

func TestPostgresRepository_GetShareLinkByToken_UnknownToken(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	if _, err := repo.GetShareLinkByToken(context.Background(), uuid.NewString()); err != ErrShareLinkInvalid {
		t.Fatalf("GetShareLinkByToken(unknown) error = %v, want ErrShareLinkInvalid", err)
	}
}

func TestPostgresRepository_RevokeShareLink(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	fx := seedActivityFixture(t, pool, "sharelink-revoke")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	linkID := uuid.New()
	if err := repo.CreateShareLink(ctx, &models.DocumentShareLink{
		ID: linkID, TenantID: fx.tenant, FileID: fx.file,
		Token: uuid.NewString(), CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("CreateShareLink: %v", err)
	}

	if err := repo.RevokeShareLink(ctx, linkID, fx.tenant, time.Now()); err != nil {
		t.Fatalf("RevokeShareLink: %v", err)
	}

	links, err := repo.ListShareLinks(ctx, fx.file, fx.tenant)
	if err != nil {
		t.Fatalf("ListShareLinks: %v", err)
	}
	if len(links) != 1 || links[0].RevokedAt == nil {
		t.Fatalf("ListShareLinks after revoke = %+v, want exactly one link with RevokedAt set", links)
	}

	// Revoking an already-revoked link is not found, not a silent no-op —
	// the caller should not believe a second revoke did anything.
	if err := repo.RevokeShareLink(ctx, linkID, fx.tenant, time.Now()); err != ErrShareLinkNotFound {
		t.Fatalf("RevokeShareLink (already revoked) error = %v, want ErrShareLinkNotFound", err)
	}
}

func TestPostgresRepository_RevokeShareLink_WrongTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	fx := seedActivityFixture(t, pool, "sharelink-revoke-wrong-tenant")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	linkID := uuid.New()
	if err := repo.CreateShareLink(ctx, &models.DocumentShareLink{
		ID: linkID, TenantID: fx.tenant, FileID: fx.file,
		Token: uuid.NewString(), CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("CreateShareLink: %v", err)
	}

	otherTenant := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenant, "document share link revoke wrong tenant")

	if err := repo.RevokeShareLink(ctx, linkID, otherTenant, time.Now()); err != ErrShareLinkNotFound {
		t.Fatalf("RevokeShareLink (foreign tenant) error = %v, want ErrShareLinkNotFound", err)
	}

	links, err := repo.ListShareLinks(ctx, fx.file, fx.tenant)
	if err != nil {
		t.Fatalf("ListShareLinks: %v", err)
	}
	if len(links) != 1 || links[0].RevokedAt != nil {
		t.Fatalf("ListShareLinks after foreign-tenant revoke attempt = %+v, want unrevoked link untouched", links)
	}
}

func TestPostgresRepository_IncrementShareLinkView(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	fx := seedActivityFixture(t, pool, "sharelink-view-count")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	linkID := uuid.New()
	if err := repo.CreateShareLink(ctx, &models.DocumentShareLink{
		ID: linkID, TenantID: fx.tenant, FileID: fx.file,
		Token: uuid.NewString(), CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("CreateShareLink: %v", err)
	}

	if err := repo.IncrementShareLinkView(ctx, linkID, fx.tenant); err != nil {
		t.Fatalf("IncrementShareLinkView: %v", err)
	}
	if err := repo.IncrementShareLinkView(ctx, linkID, fx.tenant); err != nil {
		t.Fatalf("IncrementShareLinkView: %v", err)
	}

	links, err := repo.ListShareLinks(ctx, fx.file, fx.tenant)
	if err != nil {
		t.Fatalf("ListShareLinks: %v", err)
	}
	if len(links) != 1 || links[0].ViewCount != 2 {
		t.Fatalf("ListShareLinks after two increments = %+v, want ViewCount=2", links)
	}
}
