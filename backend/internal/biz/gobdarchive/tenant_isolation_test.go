package gobdarchive

// Cross-tenant isolation tests for gobd_documents and gobd_document_events.
// Both tables have tenant_id NOT NULL and RLS enabled in migration 000139.
//
// These tests run only when DATABASE_URL is set (local Compose / CI with DB).
// In CI without a Postgres sidecar they are skipped via SkipIfNoDB.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_GobdDocuments(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A GoBD")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B GoBD")

	// Seed a gobd_document for Tenant A.
	docID := testutil.SeedRow(t, pool, "gobd_documents", map[string]any{
		"tenant_id":         testutil.TenantA,
		"doc_type":          "invoice",
		"storage_key":       "gobd/test/tenant-a/doc.pdf",
		"sha256":            "aabbccdd" + "00000000" + "00000000" + "00000000" + "00000000" + "00000000" + "00000000" + "00000000",
		"original_filename": "invoice.pdf",
		"mime_type":         "application/pdf",
		"file_size_bytes":   int64(1024),
		"archived_by":       uuid.New(),
		"retention_until":   time.Date(2034, 12, 31, 0, 0, 0, 0, time.UTC),
	})
	defer testutil.CleanupRow(t, pool, "gobd_documents", docID)

	// Tenant A can see the document.
	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	testutil.AssertRowCount(t, pool, ctxA, "gobd_documents", docID, 1)

	// Tenant B cannot see Tenant A's document.
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)
	testutil.AssertRowCount(t, pool, ctxB, "gobd_documents", docID, 0)
}

func TestTenantIsolation_GobdDocumentEvents(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A GoBD Events")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B GoBD Events")

	// Seed parent document for Tenant A.
	docID := testutil.SeedRow(t, pool, "gobd_documents", map[string]any{
		"tenant_id":         testutil.TenantA,
		"doc_type":          "receipt",
		"storage_key":       "gobd/test/events/doc.pdf",
		"sha256":            "11223344" + "00000000" + "00000000" + "00000000" + "00000000" + "00000000" + "00000000" + "00000000",
		"original_filename": "receipt.pdf",
		"mime_type":         "application/pdf",
		"file_size_bytes":   int64(512),
		"archived_by":       uuid.New(),
		"retention_until":   time.Date(2034, 12, 31, 0, 0, 0, 0, time.UTC),
	})
	defer testutil.CleanupRow(t, pool, "gobd_documents", docID)

	// Seed an event for Tenant A's document.
	eventID := testutil.SeedRow(t, pool, "gobd_document_events", map[string]any{
		"tenant_id":   testutil.TenantA,
		"document_id": docID,
		"event_type":  "archived",
		"created_by":  uuid.New(),
	})
	defer testutil.CleanupRow(t, pool, "gobd_document_events", eventID)

	// Tenant A can see the event.
	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	testutil.AssertRowCount(t, pool, ctxA, "gobd_document_events", eventID, 1)

	// Tenant B cannot see Tenant A's event.
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)
	testutil.AssertRowCount(t, pool, ctxB, "gobd_document_events", eventID, 0)
}
