package wiki

// wp-document-wiki: the existing tenant_isolation_phase2_test.go in this
// package seeds exclusively through testutil.SeedRow and never calls the real
// repository write methods. This file closes that gap for the write surface,
// and pins down the concrete bugs found while auditing it:
//
//   - DeleteAttachment and DeleteShareToken used to DELETE by ID alone, with
//     no tenant_id predicate at all — RLS's WITH CHECK still blocked a
//     cross-tenant delete at the database, but the repo never inspected
//     RowsAffected, so a foreign tenant's revoke/delete request against a
//     real (but not-theirs) ID silently reported success without deleting
//     anything. Both now carry an explicit tenant_id predicate and return
//     ErrAttachmentNotFound / ErrShareTokenNotFound when nothing matched.
//   - GetShareToken and GetAttachment were dead code (zero callers anywhere
//     in the repo) and GetShareToken never even scanned tenant_id into the
//     returned struct — removed rather than fixed, per the same precedent as
//     the dialer iteration's dead CampaignRepository.GetByID.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestWikiWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Wiki Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Wiki Write Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	article := &Article{
		ID:        uuid.New(),
		TenantID:  tenantOwn,
		Title:     "Write-Test-" + uuid.New().String()[:8],
		Slug:      "write-test-" + uuid.New().String()[:8],
		Content:   []byte(`{}`),
		AuthorID:  uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Create with the victim tenant's real TenantID, called from the
	// attacker session — only RLS's WITH CHECK can be stopping this insert.
	if err := repo.CreateArticle(ctxOther, article); err == nil {
		t.Fatalf("CreateArticle (foreign ctx): expected an RLS error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "wiki_articles", article.ID, 0)

	if err := repo.CreateArticle(ctxOwn, article); err != nil {
		t.Fatalf("CreateArticle (own ctx): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "wiki_articles", article.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "wiki_articles", article.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "wiki_articles", article.ID, 0)

	// --- Versions ---------------------------------------------------------

	version := &Version{
		ID:            uuid.New(),
		TenantID:      tenantOwn,
		ArticleID:     article.ID,
		VersionNumber: 1,
		Content:       []byte(`{"old":true}`),
		ChangedAt:     now,
	}
	if err := repo.CreateVersion(ctxOther, version); err == nil {
		t.Fatalf("CreateVersion (foreign ctx): expected an RLS error, got nil")
	}
	if err := repo.CreateVersion(ctxOwn, version); err != nil {
		t.Fatalf("CreateVersion (own ctx): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "wiki_versions", version.ID)

	if _, err := repo.GetVersion(ctxOther, tenantOwn, version.ID); err == nil {
		t.Fatalf("GetVersion (foreign ctx, victim tenantID): expected RLS to hide the row, got nil error")
	}
	got, err := repo.GetVersion(ctxOwn, tenantOwn, version.ID)
	if err != nil {
		t.Fatalf("GetVersion (own ctx): %v", err)
	}
	if got.ArticleID != article.ID {
		t.Fatalf("GetVersion returned wrong article: %v", got.ArticleID)
	}

	versions, err := repo.ListVersions(ctxOther, tenantOwn, article.ID)
	if err != nil {
		t.Fatalf("ListVersions (foreign ctx): %v", err)
	}
	if len(versions) != 0 {
		t.Fatalf("ListVersions (foreign ctx): expected 0 versions visible, got %d", len(versions))
	}
	versions, err = repo.ListVersions(ctxOwn, tenantOwn, article.ID)
	if err != nil {
		t.Fatalf("ListVersions (own ctx): %v", err)
	}
	if len(versions) != 1 {
		t.Fatalf("ListVersions (own ctx): expected 1 version, got %d", len(versions))
	}

	// --- Attachments --------------------------------------------------------

	attachment := &Attachment{
		ID:        uuid.New(),
		TenantID:  tenantOwn,
		ArticleID: article.ID,
		FileRef:   "uploads/" + uuid.New().String() + ".pdf",
		Mime:      "application/pdf",
		Size:      1024,
		CreatedAt: now,
	}
	if err := repo.CreateAttachment(ctxOther, attachment); err == nil {
		t.Fatalf("CreateAttachment (foreign ctx): expected an RLS error, got nil")
	}
	if err := repo.CreateAttachment(ctxOwn, attachment); err != nil {
		t.Fatalf("CreateAttachment (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, ctxOwn, "wiki_attachments", attachment.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "wiki_attachments", attachment.ID, 0)

	// A foreign session deleting the victim's real attachment ID (with the
	// victim's real tenantID passed explicitly) must be refused: RLS hides
	// the row from the DELETE entirely, and the repo must report
	// ErrAttachmentNotFound rather than silently claiming success.
	if err := repo.DeleteAttachment(ctxOther, tenantOwn, attachment.ID); !errors.Is(err, ErrAttachmentNotFound) {
		t.Fatalf("DeleteAttachment (foreign ctx): expected ErrAttachmentNotFound, got %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "wiki_attachments", attachment.ID, 1)

	if err := repo.DeleteAttachment(ctxOwn, tenantOwn, attachment.ID); err != nil {
		t.Fatalf("DeleteAttachment (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "wiki_attachments", attachment.ID, 0)

	// --- Share tokens --------------------------------------------------------

	token := &ShareToken{
		ID:          uuid.New(),
		TenantID:    tenantOwn,
		ArticleID:   article.ID,
		Token:       uuid.New().String(),
		Permissions: []string{"read"},
		CreatedAt:   now,
	}
	if err := repo.CreateShareToken(ctxOther, token); err == nil {
		t.Fatalf("CreateShareToken (foreign ctx): expected an RLS error, got nil")
	}
	if err := repo.CreateShareToken(ctxOwn, token); err != nil {
		t.Fatalf("CreateShareToken (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, ctxOwn, "wiki_share_tokens", token.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "wiki_share_tokens", token.ID, 0)

	tokens, err := repo.ListShareTokensByArticle(ctxOther, tenantOwn, article.ID)
	if err != nil {
		t.Fatalf("ListShareTokensByArticle (foreign ctx): %v", err)
	}
	if len(tokens) != 0 {
		t.Fatalf("ListShareTokensByArticle (foreign ctx): expected 0 tokens visible, got %d", len(tokens))
	}

	// Same story as attachments: a foreign session revoking the victim's
	// real token ID (even naming the victim's real tenantID) must be
	// refused rather than silently no-op-succeed.
	if err := repo.RevokeShareToken(ctxOther, tenantOwn, token.ID, now); !errors.Is(err, ErrShareTokenNotFound) {
		t.Fatalf("RevokeShareToken (foreign ctx): expected ErrShareTokenNotFound, got %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "wiki_share_tokens", token.ID, 1)
	if revokedAt := readRevokedAt(t, pool, sysCtx, token.ID); revokedAt != nil {
		t.Fatalf("a refused cross-tenant revoke still stamped revoked_at = %v", revokedAt)
	}

	// Revocation is soft since 000297: the row survives, carrying the moment
	// it was cut.
	if err := repo.RevokeShareToken(ctxOwn, tenantOwn, token.ID, now); err != nil {
		t.Fatalf("RevokeShareToken (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "wiki_share_tokens", token.ID, 1)
	firstCut := readRevokedAt(t, pool, sysCtx, token.ID)
	if firstCut == nil {
		t.Fatal("RevokeShareToken (own ctx): revoked_at was not stamped")
	}

	// Cutting an already cut link succeeds without rewriting when it happened.
	if err := repo.RevokeShareToken(ctxOwn, tenantOwn, token.ID, now.Add(time.Hour)); err != nil {
		t.Fatalf("RevokeShareToken (repeat): %v", err)
	}
	if secondCut := readRevokedAt(t, pool, sysCtx, token.ID); secondCut == nil || !secondCut.Equal(*firstCut) {
		t.Fatalf("a repeated revoke moved revoked_at from %v to %v", firstCut, secondCut)
	}
}

// readRevokedAt reads one token's revoked_at directly, bypassing the
// repository, so the assertion cannot be satisfied by the code under test.
func readRevokedAt(t *testing.T, pool *pgxpool.Pool, ctx context.Context, tokenID uuid.UUID) *time.Time {
	t.Helper()
	var revokedAt *time.Time
	if err := pool.QueryRow(ctx,
		`SELECT revoked_at FROM wiki_share_tokens WHERE id = $1`, tokenID,
	).Scan(&revokedAt); err != nil {
		t.Fatalf("read revoked_at: %v", err)
	}
	return revokedAt
}
