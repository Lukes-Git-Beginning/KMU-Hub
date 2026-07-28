package gdpr

// wp-security: closes the write-surface gap for GDPR export requests.
// UpdateExportStatus, StoreExportResult and MarkDownloaded took a bare row ID
// with no tenant_id predicate (RLS was the only backstop); GetExportRequest
// and GetExportByToken ran without a tenant filter at all, so
// ApproveExport/DenyExport/ExecuteExportAsync/GetExportDownload resolved rows
// through an RLS-only lookup instead of an explicit tenant scope. All five now
// carry an explicit tenant_id predicate, fixed in the same commit as this test.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func seedGDPRUser(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("wp-security-gdpr-%s@test.invalid", uuid.New()),
		"password_hash": "x",
	})
}

func TestGDPRExportRequestWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "GDPR Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "GDPR Write Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := seedGDPRUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", userOwn)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())

	// CreateExportRequest derives tenant_id from the target user's row (not
	// from the caller's ctx), so filing a request for a foreign user from the
	// attacker's own tenant session must fail RLS's WITH CHECK.
	foreignReq := &models.GDPRExportRequest{
		ID:          uuid.New(),
		UserID:      userOwn,
		Status:      models.ExportStatusPending,
		RequestedAt: time.Now().UTC(),
	}
	if err := repo.CreateExportRequest(ctxOther, foreignReq); err == nil {
		t.Fatalf("CreateExportRequest (foreign ctx): expected an RLS error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "gdpr_export_requests", foreignReq.ID, 0)

	req := &models.GDPRExportRequest{
		ID:          uuid.New(),
		UserID:      userOwn,
		Status:      models.ExportStatusPending,
		RequestedAt: time.Now().UTC(),
	}
	if err := repo.CreateExportRequest(ctxOwn, req); err != nil {
		t.Fatalf("CreateExportRequest (own ctx): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "gdpr_export_requests", req.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "gdpr_export_requests", req.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "gdpr_export_requests", req.ID, 0)

	// GetExportRequest is explicitly tenant-scoped.
	if _, err := repo.GetExportRequest(ctxOther, tenantOther, req.ID); err == nil {
		t.Fatalf("GetExportRequest (foreign ctx): expected not-found, got nil")
	}
	got, err := repo.GetExportRequest(ctxOwn, tenantOwn, req.ID)
	if err != nil {
		t.Fatalf("GetExportRequest (own ctx): %v", err)
	}
	if got.ID != req.ID {
		t.Fatalf("GetExportRequest (own ctx): got wrong request %s", got.ID)
	}

	// UpdateExportStatus carries an explicit tenant_id predicate — a
	// foreign-tenant call must come back an error, not a silent no-op.
	foreignUpdate := *req
	foreignUpdate.Status = models.ExportStatusApproved
	foreignUpdate.ReviewNote = "hacked"
	if err := repo.UpdateExportStatus(ctxOther, tenantOther, &foreignUpdate); err == nil {
		t.Fatalf("UpdateExportStatus (foreign ctx): expected an error, got nil")
	}
	got, err = repo.GetExportRequest(ctxOwn, tenantOwn, req.ID)
	if err != nil {
		t.Fatalf("GetExportRequest after foreign update: %v", err)
	}
	if got.Status != models.ExportStatusPending {
		t.Fatalf("a foreign-tenant write reached the export request: status=%q", got.Status)
	}

	ownUpdate := *req
	ownUpdate.Status = models.ExportStatusApproved
	ownUpdate.ReviewNote = "approved by owner"
	if err := repo.UpdateExportStatus(ctxOwn, tenantOwn, &ownUpdate); err != nil {
		t.Fatalf("UpdateExportStatus (own ctx): %v", err)
	}
	got, err = repo.GetExportRequest(ctxOwn, tenantOwn, req.ID)
	if err != nil {
		t.Fatalf("GetExportRequest after own update: %v", err)
	}
	if got.Status != models.ExportStatusApproved {
		t.Fatalf("own-tenant write did not land: status=%q", got.Status)
	}

	// StoreExportResult carries an explicit tenant_id predicate.
	expiresAt := time.Now().UTC().Add(7 * 24 * time.Hour)
	if err := repo.StoreExportResult(ctxOther, tenantOther, req.ID, []byte("hacked"), "hacked-token", expiresAt); err == nil {
		t.Fatalf("StoreExportResult (foreign ctx): expected an error, got nil")
	}
	got, err = repo.GetExportRequest(ctxOwn, tenantOwn, req.ID)
	if err != nil {
		t.Fatalf("GetExportRequest after foreign StoreExportResult: %v", err)
	}
	if got.DownloadToken == "hacked-token" {
		t.Fatalf("a foreign-tenant write reached the export result")
	}

	token := "own-token-" + uuid.New().String()
	if err := repo.StoreExportResult(ctxOwn, tenantOwn, req.ID, []byte("zip-bytes"), token, expiresAt); err != nil {
		t.Fatalf("StoreExportResult (own ctx): %v", err)
	}

	// GetExportByToken is explicitly tenant-scoped: a foreign tenant must not
	// resolve the caller's download token.
	if _, err := repo.GetExportByToken(ctxOther, tenantOther, token); err == nil {
		t.Fatalf("GetExportByToken (foreign ctx): expected not-found, got nil")
	}
	byToken, err := repo.GetExportByToken(ctxOwn, tenantOwn, token)
	if err != nil {
		t.Fatalf("GetExportByToken (own ctx): %v", err)
	}
	if byToken.ID != req.ID {
		t.Fatalf("GetExportByToken (own ctx): got wrong request %s", byToken.ID)
	}

	// MarkDownloaded carries an explicit tenant_id predicate.
	if err := repo.MarkDownloaded(ctxOther, tenantOther, req.ID); err == nil {
		t.Fatalf("MarkDownloaded (foreign ctx): expected an error, got nil")
	}
	got, err = repo.GetExportRequest(ctxOwn, tenantOwn, req.ID)
	if err != nil {
		t.Fatalf("GetExportRequest after foreign MarkDownloaded: %v", err)
	}
	if got.DownloadedAt != nil {
		t.Fatalf("a foreign-tenant write marked the export as downloaded")
	}

	if err := repo.MarkDownloaded(ctxOwn, tenantOwn, req.ID); err != nil {
		t.Fatalf("MarkDownloaded (own ctx): %v", err)
	}
	got, err = repo.GetExportRequest(ctxOwn, tenantOwn, req.ID)
	if err != nil {
		t.Fatalf("GetExportRequest after own MarkDownloaded: %v", err)
	}
	if got.DownloadedAt == nil {
		t.Fatalf("own-tenant MarkDownloaded did not land")
	}
}
