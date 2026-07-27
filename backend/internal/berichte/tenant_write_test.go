package berichte

// Writes into report_documents and report_share_tokens must land in — and
// stay visible only to — the caller's tenant. Both tables were built fresh in
// Nacht 1 (report_documents: migration 000245; report_share_tokens: migration
// 000252) and had no DB-backed test that ever called the repository — the
// existing tenant_isolation_phase2_test.go seeds report_documents via
// testutil.SeedRow and never touches PostgresRepository at all, so a dead
// write here (a column missing from an INSERT list, a WHERE clause missing
// tenant_id) would have gone unnoticed exactly like auth/invitations and the
// hr break table did in earlier iterations.
//
// report_share_tokens carries the module's one unauthenticated read path
// (GetShareTokenBySecret runs under database.WithSystemContext by design —
// see the doc comment on that method). This test additionally proves the
// property that matters for that path: resolving a valid token still cannot
// reach a document belonging to a different tenant than the token itself.
//
// report_definitions/report_cache/report_schedules/report_runs (migration
// 000122) are intentionally out of scope here — they predate Nacht 1, already
// have RLS-level coverage via tenant_isolation_phase2_test.go, and are not
// where the repository-write gap this backlog block is hunting for lives.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestBerichteWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Berichte Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Berichte Write Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	// --- Document: Create / Update / Get / cross-tenant no-access / Delete ---

	doc := &Document{
		ID:        uuid.New(),
		TenantID:  tenantOwn,
		Title:     "Quartalsbericht " + uuid.New().String(),
		Module:    "cross",
		Status:    "draft",
		Rows:      []byte(`[]`),
		Settings:  []byte(`{}`),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.CreateDocument(ctxOwn, doc); err != nil {
		t.Fatalf("CreateDocument: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "report_documents", doc.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "report_documents", doc.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "report_documents", doc.ID, 0)

	doc.Title = "Quartalsbericht (ueberarbeitet)"
	doc.UpdatedAt = time.Now().UTC()
	if err := repo.UpdateDocument(ctxOwn, doc); err != nil {
		t.Fatalf("UpdateDocument: %v", err)
	}
	got, err := repo.GetDocument(ctxOwn, tenantOwn, doc.ID)
	if err != nil {
		t.Fatalf("GetDocument (own tenant): %v", err)
	}
	if got.Title != doc.Title {
		t.Fatalf("GetDocument: title not updated, got %q", got.Title)
	}

	// A foreign tenant context cannot read the row back, even when it passes
	// the correct owning tenantID as the explicit query parameter — RLS, not
	// the WHERE clause alone, is the backstop.
	if _, err := repo.GetDocument(ctxOther, tenantOwn, doc.ID); err != ErrDocumentNotFound {
		t.Fatalf("GetDocument (foreign ctx): expected ErrDocumentNotFound, got %v", err)
	}

	// A foreign tenant's Update/Delete against the row is a no-op, not a
	// cross-tenant mutation.
	stolen := *doc
	stolen.Title = "stolen"
	if err := repo.UpdateDocument(ctxOther, &stolen); err != ErrDocumentNotFound {
		t.Fatalf("UpdateDocument (foreign ctx): expected ErrDocumentNotFound, got %v", err)
	}
	if err := repo.DeleteDocument(ctxOther, tenantOwn, doc.ID); err != ErrDocumentNotFound {
		t.Fatalf("DeleteDocument (foreign ctx): expected ErrDocumentNotFound, got %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "report_documents", doc.ID, 1)

	if err := repo.DeleteDocument(ctxOwn, tenantOwn, doc.ID); err != nil {
		t.Fatalf("DeleteDocument (own tenant): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "report_documents", doc.ID, 0)

	// --- Share tokens: separate document per tenant, one shared -------------

	shareDoc := &Document{
		ID: uuid.New(), TenantID: tenantOwn, Title: "Geteilter Bericht",
		Module: "cross", Status: "final", Rows: []byte(`[]`), Settings: []byte(`{}`),
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateDocument(ctxOwn, shareDoc); err != nil {
		t.Fatalf("CreateDocument (shareDoc): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "report_documents", shareDoc.ID)

	foreignDoc := &Document{
		ID: uuid.New(), TenantID: tenantOther, Title: "Fremder Bericht",
		Module: "cross", Status: "final", Rows: []byte(`[]`), Settings: []byte(`{}`),
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateDocument(ctxOther, foreignDoc); err != nil {
		t.Fatalf("CreateDocument (foreignDoc): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "report_documents", foreignDoc.ID)

	token := &ShareToken{
		ID: uuid.New(), TenantID: tenantOwn, DocumentID: shareDoc.ID,
		Token: uuid.New().String(), ViewCount: 0, CreatedAt: now,
	}
	if err := repo.CreateShareToken(ctxOwn, token); err != nil {
		t.Fatalf("CreateShareToken: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "report_share_tokens", token.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "report_share_tokens", token.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "report_share_tokens", token.ID, 0)

	list, err := repo.ListShareTokens(ctxOwn, tenantOwn, shareDoc.ID)
	if err != nil {
		t.Fatalf("ListShareTokens (own tenant): %v", err)
	}
	if len(list) != 1 || list[0].ID != token.ID {
		t.Fatalf("ListShareTokens (own tenant): expected [%s], got %v", token.ID, list)
	}
	foreignList, err := repo.ListShareTokens(ctxOther, tenantOwn, shareDoc.ID)
	if err != nil {
		t.Fatalf("ListShareTokens (foreign ctx): %v", err)
	}
	if len(foreignList) != 0 {
		t.Fatalf("ListShareTokens (foreign ctx): expected none, got %v", foreignList)
	}

	// IncrementShareView under the wrong tenant context is a silent no-op
	// (the method never inspects RowsAffected) — assert the counter directly
	// instead of trusting a nil error.
	if err := repo.IncrementShareView(ctxOther, tenantOwn, token.ID); err != nil {
		t.Fatalf("IncrementShareView (foreign ctx): %v", err)
	}
	unchanged, err := repo.GetShareTokenBySecret(sysCtx, token.Token)
	if err != nil {
		t.Fatalf("GetShareTokenBySecret (after foreign increment): %v", err)
	}
	if unchanged.ViewCount != 0 {
		t.Fatalf("IncrementShareView (foreign ctx): expected view_count unchanged at 0, got %d", unchanged.ViewCount)
	}
	if err := repo.IncrementShareView(ctxOwn, tenantOwn, token.ID); err != nil {
		t.Fatalf("IncrementShareView (own ctx): %v", err)
	}

	// GetShareTokenBySecret is the module's one unauthenticated read: it runs
	// under system context regardless of the caller's ctx and resolves the
	// token's own tenant, which the service then uses for the follow-up
	// GetDocument. Simulate exactly that two-step resolution here.
	resolved, err := repo.GetShareTokenBySecret(context.Background(), token.Token)
	if err != nil {
		t.Fatalf("GetShareTokenBySecret: %v", err)
	}
	if resolved.TenantID != tenantOwn {
		t.Fatalf("GetShareTokenBySecret: resolved tenant %s, want %s", resolved.TenantID, tenantOwn)
	}
	if resolved.ViewCount != 1 {
		t.Fatalf("GetShareTokenBySecret: expected view_count 1 after one real increment, got %d", resolved.ViewCount)
	}
	resolvedCtx := testutil.WithTenantCtx(context.Background(), resolved.TenantID)
	if _, err := repo.GetDocument(resolvedCtx, resolved.TenantID, resolved.DocumentID); err != nil {
		t.Fatalf("GetDocument (resolved from token): %v", err)
	}

	// The token belongs to tenantOwn. A request that swaps in the other
	// tenant's document id — even while resolving through the same token's
	// tenant — must not reach it: the share path is bounded by DocumentID,
	// not just by tenant.
	if _, err := repo.GetDocument(resolvedCtx, resolved.TenantID, foreignDoc.ID); err != ErrDocumentNotFound {
		t.Fatalf("GetDocument (resolved tenant, foreign document id): expected ErrDocumentNotFound, got %v", err)
	}

	// RevokeShareToken under the wrong tenant context is a no-op, not a
	// cross-tenant revoke.
	if err := repo.RevokeShareToken(ctxOther, tenantOwn, token.ID, time.Now().UTC()); err != ErrShareNotFound {
		t.Fatalf("RevokeShareToken (foreign ctx): expected ErrShareNotFound, got %v", err)
	}
	stillLive, err := repo.GetShareTokenBySecret(sysCtx, token.Token)
	if err != nil {
		t.Fatalf("GetShareTokenBySecret (after foreign revoke attempt): %v", err)
	}
	if stillLive.RevokedAt != nil {
		t.Fatalf("RevokeShareToken (foreign ctx): token was revoked, want untouched")
	}
	if err := repo.RevokeShareToken(ctxOwn, tenantOwn, token.ID, time.Now().UTC()); err != nil {
		t.Fatalf("RevokeShareToken (own ctx): %v", err)
	}
}
