package formulare

// DB-backed cross-tenant coverage for the public share-link submission path
// (form_share_db_test.go complements the stub-repo tests in form_share_test.go,
// which prove the token-verdict and quota logic but run against an in-memory
// repo pinned to a single tenant and can't exercise RLS).
//
// The one thing that matters here: a public submission redeemed through
// tenant A's token must land -- and be visible -- ONLY in tenant A, even while
// tenant B has its own schema, link and pending submission open at the same
// time. GetShareLinkByToken resolves the token under
// database.WithSystemContext, the one read in this service allowed to escape
// RLS; everything after it (schema load, RedeemShareLinkTx) is ordinary
// tenant-scoped work. This test seeds two real tenants and proves that
// boundary holds against the real database, not a fixture.

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestSubmitByShareToken_TwoTenantsRedemptionStaysScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Formulare Public Submit Tenant A")
	testutil.EnsureTenant(t, pool, tenantB, "Formulare Public Submit Tenant B")

	repo := NewPostgresRepository(pool)
	svc := NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)
	now := time.Now().UTC()

	mkSchema := func(ctx context.Context, tenantID uuid.UUID, title string) *FormSchema {
		t.Helper()
		s := &FormSchema{
			ID: uuid.New(), TenantID: tenantID, Title: title,
			Fields: []byte(`[]`), Status: FormSchemaStatusActive, IsPublic: true,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := repo.CreateSchema(ctx, s); err != nil {
			t.Fatalf("CreateSchema(%s): %v", title, err)
		}
		return s
	}

	schemaA := mkSchema(ctxA, tenantA, "Public Submit Cross Tenant Schema A")
	defer testutil.CleanupRow(t, pool, "form_schemas", schemaA.ID)
	schemaB := mkSchema(ctxB, tenantB, "Public Submit Cross Tenant Schema B")
	defer testutil.CleanupRow(t, pool, "form_schemas", schemaB.ID)

	linkA, err := svc.CreateShareLink(ctxA, CreateShareLinkInput{TenantID: tenantA, FormSchemaID: schemaA.ID})
	if err != nil {
		t.Fatalf("CreateShareLink (A): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "form_share_tokens", linkA.ID)

	linkB, err := svc.CreateShareLink(ctxB, CreateShareLinkInput{TenantID: tenantB, FormSchemaID: schemaB.ID})
	if err != nil {
		t.Fatalf("CreateShareLink (B): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "form_share_tokens", linkB.ID)

	// Redeem A's token. If the token lookup or the write behind it ever
	// slipped tenant scope, this is where tenant B's link or a phantom
	// submission in tenant B would get touched instead of (or alongside) A's.
	subA, err := svc.SubmitByShareToken(context.Background(), linkA.Token, []byte(`{"f1":"a"}`), "")
	if err != nil {
		t.Fatalf("SubmitByShareToken (A): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "form_submissions", subA.SubmissionID)

	if subA.TenantID != tenantA {
		t.Fatalf("submission resolved to tenant %s, want %s", subA.TenantID, tenantA)
	}
	testutil.AssertRowCount(t, pool, ctxA, "form_submissions", subA.SubmissionID, 1)
	testutil.AssertRowCount(t, pool, ctxB, "form_submissions", subA.SubmissionID, 0)

	// Tenant B's link must be completely untouched by A's redemption: still
	// zero submissions, and still independently redeemable.
	linksB, err := svc.ListShareLinks(ctxB, schemaB.ID, tenantB)
	if err != nil {
		t.Fatalf("ListShareLinks (B): %v", err)
	}
	if len(linksB) != 1 || linksB[0].SubmissionCount != 0 {
		t.Fatalf("tenant B's link was touched by tenant A's redemption: %#v", linksB)
	}

	subB, err := svc.SubmitByShareToken(context.Background(), linkB.Token, []byte(`{"f1":"b"}`), "")
	if err != nil {
		t.Fatalf("SubmitByShareToken (B): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "form_submissions", subB.SubmissionID)

	if subB.TenantID != tenantB {
		t.Fatalf("submission resolved to tenant %s, want %s", subB.TenantID, tenantB)
	}
	testutil.AssertRowCount(t, pool, ctxB, "form_submissions", subB.SubmissionID, 1)
	testutil.AssertRowCount(t, pool, ctxA, "form_submissions", subB.SubmissionID, 0)
}
