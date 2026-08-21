package gobdarchive

// PostgresRepository is the write path for an immutable, append-only archive
// (§147 AO). service_test.go proves the business rules against a mock; these
// tests prove the SQL itself: tenant scoping on every read, filter/pagination
// correctness in List, event ordering, and — the one behavior that matters
// most for a "revisionssicheres" archive — that there is no way to overwrite
// an already-archived document. Create is the only write method the
// Repository interface exposes, so a second Create for the same document ID
// is the only "update attempt" that can be made through this API at all, and
// it must fail on the primary key.
//
// No test cleans up the rows it archives. Migration 000315 revokes DELETE on
// both archive tables from kmuhub_app, and that is the point of the archive --
// what lands in it stays. The fixtures use a throwaway tenant per test, so the
// leftovers are invisible to everything else. Calling testutil.CleanupRow here
// would only log a permission error.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newGobdFixture(t *testing.T) (*pgxpool.Pool, uuid.UUID, context.Context) {
	t.Helper()
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "GoBD Repository Tenant")

	return pool, tenantID, testutil.WithTenantCtx(context.Background(), tenantID)
}

func fakeSHA256(seed string) string {
	sum := sha256.Sum256([]byte(seed))
	return hex.EncodeToString(sum[:])
}

func newTestDocument(tenantID uuid.UUID, seed string) *models.GobdDocument {
	now := time.Now().UTC().Truncate(time.Second)
	return &models.GobdDocument{
		ID:               uuid.New(),
		TenantID:         tenantID,
		DocType:          models.GobdDocTypeReceipt,
		StorageKey:       "gobd/test/" + seed + ".pdf",
		SHA256:           fakeSHA256(seed),
		OriginalFilename: seed + ".pdf",
		MimeType:         "application/pdf",
		FileSizeBytes:    1024,
		ArchivedAt:       now,
		ArchivedBy:       uuid.New(),
		RetentionUntil:   computeRetentionUntil(now),
	}
}

// ============================================================================
// Create + GetByID round trip
// ============================================================================

func TestPostgresRepository_CreateAndGetByID_RoundTrip(t *testing.T) {
	t.Parallel()
	pool, tenantID, ctx := newGobdFixture(t)
	repo := NewPostgresRepository(pool)

	doc := newTestDocument(tenantID, "roundtrip")
	require.NoError(t, repo.Create(ctx, doc))

	got, err := repo.GetByID(ctx, tenantID, doc.ID)
	require.NoError(t, err)
	assert.Equal(t, doc.SHA256, got.SHA256)
	assert.Equal(t, doc.OriginalFilename, got.OriginalFilename)
	assert.Equal(t, doc.RetentionUntil, got.RetentionUntil)
	assert.Nil(t, got.SourceInvoiceID, "unset source_invoice_id must round-trip as nil, not uuid.Nil")
}

// ============================================================================
// Create — the only "update attempt" this API allows is re-Create under the
// same ID, and that must fail on the primary key. The original row must
// survive untouched.
// ============================================================================

func TestPostgresRepository_Create_DuplicateIDRejected(t *testing.T) {
	t.Parallel()
	pool, tenantID, ctx := newGobdFixture(t)
	repo := NewPostgresRepository(pool)

	original := newTestDocument(tenantID, "immutable-original")
	require.NoError(t, repo.Create(ctx, original))

	tampered := newTestDocument(tenantID, "tampered-payload")
	tampered.ID = original.ID // same primary key — this is the "update attempt"

	err := repo.Create(ctx, tampered)
	require.Error(t, err, "re-archiving under an existing document ID must be rejected")

	// The original record must be entirely unaffected by the rejected attempt.
	got, getErr := repo.GetByID(ctx, tenantID, original.ID)
	require.NoError(t, getErr)
	assert.Equal(t, original.SHA256, got.SHA256)
	assert.Equal(t, original.OriginalFilename, got.OriginalFilename)
}

// ============================================================================
// GetByID — tenant scoping and not-found
// ============================================================================

func TestPostgresRepository_GetByID_CrossTenantReturnsNotFound(t *testing.T) {
	t.Parallel()
	pool, tenantA, ctxA := newGobdFixture(t)
	tenantB := uuid.New()
	testutil.EnsureTenant(t, pool, tenantB, "GoBD Repository Tenant B")
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	repo := NewPostgresRepository(pool)
	doc := newTestDocument(tenantA, "cross-tenant")
	require.NoError(t, repo.Create(ctxA, doc))

	_, err := repo.GetByID(ctxB, tenantB, doc.ID)
	assert.ErrorIs(t, err, ErrDocumentNotFound)

	// Sanity: the owning tenant can still read it.
	_, err = repo.GetByID(ctxA, tenantA, doc.ID)
	require.NoError(t, err)
}

func TestPostgresRepository_GetByID_UnknownIDReturnsNotFound(t *testing.T) {
	t.Parallel()
	pool, tenantID, ctx := newGobdFixture(t)
	repo := NewPostgresRepository(pool)

	_, err := repo.GetByID(ctx, tenantID, uuid.New())
	assert.ErrorIs(t, err, ErrDocumentNotFound)
}

// ============================================================================
// List — filters run in SQL, not in Go
// ============================================================================

func TestPostgresRepository_List_FiltersByDocType(t *testing.T) {
	t.Parallel()
	pool, tenantID, ctx := newGobdFixture(t)
	repo := NewPostgresRepository(pool)

	invoice := newTestDocument(tenantID, "list-invoice")
	invoice.DocType = models.GobdDocTypeInvoice
	receipt := newTestDocument(tenantID, "list-receipt")
	receipt.DocType = models.GobdDocTypeReceipt
	require.NoError(t, repo.Create(ctx, invoice))
	require.NoError(t, repo.Create(ctx, receipt))

	docs, total, err := repo.List(ctx, ListFilter{TenantID: tenantID, DocType: models.GobdDocTypeInvoice})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, docs, 1)
	assert.Equal(t, invoice.ID, docs[0].ID)
}

func TestPostgresRepository_List_FiltersByDateRange(t *testing.T) {
	t.Parallel()
	pool, tenantID, ctx := newGobdFixture(t)
	repo := NewPostgresRepository(pool)

	old := newTestDocument(tenantID, "list-old")
	old.ArchivedAt = time.Date(2020, 1, 15, 12, 0, 0, 0, time.UTC)
	recent := newTestDocument(tenantID, "list-recent")
	recent.ArchivedAt = time.Now().UTC().Truncate(time.Second)
	require.NoError(t, repo.Create(ctx, old))
	require.NoError(t, repo.Create(ctx, recent))

	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	docs, total, err := repo.List(ctx, ListFilter{TenantID: tenantID, DateFrom: &from})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, docs, 1)
	assert.Equal(t, recent.ID, docs[0].ID)
}

func TestPostgresRepository_List_FiltersBySourceInvoiceID(t *testing.T) {
	t.Parallel()
	pool, tenantID, ctx := newGobdFixture(t)
	repo := NewPostgresRepository(pool)

	invoiceID := testutil.SeedRow(t, pool, "finance_invoices", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID,
		"invoice_number": "RE-GOBD-" + uuid.NewString()[:8],
		"status":         models.InvoiceStatusSent,
		"customer_name":  "GoBD Test GmbH",
		"gross_total":    "0",
		"invoice_date":   time.Now().UTC().Format("2006-01-02"),
		"due_date":       time.Now().UTC().AddDate(0, 0, 30).Format("2006-01-02"),
		"created_by":     tenantID,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_invoices", invoiceID) })

	linked := newTestDocument(tenantID, "list-linked")
	linked.SourceInvoiceID = &invoiceID
	unlinked := newTestDocument(tenantID, "list-unlinked")
	require.NoError(t, repo.Create(ctx, linked))
	require.NoError(t, repo.Create(ctx, unlinked))

	docs, total, err := repo.List(ctx, ListFilter{TenantID: tenantID, SourceInvoiceID: &invoiceID})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, docs, 1)
	assert.Equal(t, linked.ID, docs[0].ID)
	require.NotNil(t, docs[0].SourceInvoiceID)
	assert.Equal(t, invoiceID, *docs[0].SourceInvoiceID)
}

func TestPostgresRepository_List_OrderedNewestFirstAndPaginated(t *testing.T) {
	t.Parallel()
	pool, tenantID, ctx := newGobdFixture(t)
	repo := NewPostgresRepository(pool)

	base := time.Now().UTC().Truncate(time.Second)
	var ids []uuid.UUID
	for i := range 3 {
		doc := newTestDocument(tenantID, "list-page")
		doc.ArchivedAt = base.Add(time.Duration(i) * time.Minute)
		require.NoError(t, repo.Create(ctx, doc))
		ids = append(ids, doc.ID)
	}

	// Page 1 of 2 must return the two newest documents, newest first.
	page1, total, err := repo.List(ctx, ListFilter{TenantID: tenantID, Page: 1, PerPage: 2})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	require.Len(t, page1, 2)
	assert.Equal(t, ids[2], page1[0].ID)
	assert.Equal(t, ids[1], page1[1].ID)

	page2, total, err := repo.List(ctx, ListFilter{TenantID: tenantID, Page: 2, PerPage: 2})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	require.Len(t, page2, 1)
	assert.Equal(t, ids[0], page2[0].ID)
}

// ============================================================================
// AppendEvent + ListEvents — append-only audit trail
// ============================================================================

func TestPostgresRepository_AppendEventAndListEvents_OrderedNewestFirst(t *testing.T) {
	t.Parallel()
	pool, tenantID, ctx := newGobdFixture(t)
	repo := NewPostgresRepository(pool)

	doc := newTestDocument(tenantID, "events-doc")
	require.NoError(t, repo.Create(ctx, doc))

	base := time.Now().UTC().Truncate(time.Second)
	ev1 := &models.GobdDocumentEvent{
		ID: uuid.New(), TenantID: tenantID, DocumentID: doc.ID,
		EventType: models.GobdEventTypeArchived, CreatedBy: uuid.New(),
		CreatedAt: base, Metadata: []byte("{}"),
	}
	ev2 := &models.GobdDocumentEvent{
		ID: uuid.New(), TenantID: tenantID, DocumentID: doc.ID,
		EventType: models.GobdEventTypeAccess, CreatedBy: uuid.New(),
		CreatedAt: base.Add(time.Minute), Metadata: []byte("{}"),
	}
	require.NoError(t, repo.AppendEvent(ctx, ev1))
	require.NoError(t, repo.AppendEvent(ctx, ev2))

	events, err := repo.ListEvents(ctx, tenantID, doc.ID)
	require.NoError(t, err)
	require.Len(t, events, 2)
	assert.Equal(t, ev2.ID, events[0].ID, "newest event must come first")
	assert.Equal(t, ev1.ID, events[1].ID)
}

func TestPostgresRepository_ListEvents_UnknownDocumentReturnsEmptyNotError(t *testing.T) {
	t.Parallel()
	pool, tenantID, ctx := newGobdFixture(t)
	repo := NewPostgresRepository(pool)

	events, err := repo.ListEvents(ctx, tenantID, uuid.New())
	require.NoError(t, err)
	assert.Empty(t, events)
}
