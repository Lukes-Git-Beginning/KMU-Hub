package invoice

// postgres_repository_gobd_export_db_test.go exercises ListForGoBDExport
// (postgres_repository.go:574-620) against real Postgres. It feeds the GoBD
// Belegarchiv and the gap-free-journal check (GetJournalSummary) — a wrong
// status filter or a broken invoice_number ordering surfaces only when a
// tax audit compares the export against the sequence, not before. Untagged
// (testutil.SkipIfNoDB) so it runs in the local gate and counts toward this
// package's coverage number, unlike integration_test.go.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// newGoBDExportTestInvoice creates and persists an invoice with the given
// status, invoice_number and invoice_date via the real Create path.
func newGoBDExportTestInvoice(t *testing.T, repo *PostgresRepository, pool *pgxpool.Pool, ctx context.Context, tenantID uuid.UUID, status, invoiceNumber string, invoiceDate time.Time) *models.Invoice {
	t.Helper()
	inv := newListTestInvoice(tenantID, invoiceDate, invoiceDate.AddDate(0, 0, 30))
	inv.Status = status
	inv.InvoiceNumber = invoiceNumber
	createListTestInvoice(t, repo, pool, ctx, inv)
	return inv
}

// TestPostgresRepository_ListForGoBDExport_StatusFilter proves every
// non-draft status is included — draft is the only status excluded, in
// particular cancelled invoices ARE included. This is a deliberate
// divergence from ListForDATEVExport (which additionally excludes
// cancelled): GoBD requires a gap-free journal of every number ever
// assigned, and a storno consumes a number just like an issued invoice —
// leaving it out of the archive would hide the cancellation from the
// journal. ListForDATEVExport, by contrast, feeds the accounting export,
// where a cancelled invoice does not itself carry a booking (the reversal
// is booked separately via the credit note path, see biz_grpc.go
// GenerateGoBDExport). Both are correct for their own purpose; the
// divergence is intentional, not a bug — see JOURNAL.md iteration 25 for
// the production-wiring finding this test also surfaced.
func TestPostgresRepository_ListForGoBDExport_StatusFilter(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Invoice GoBD Export Status Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	from := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	mid := from.AddDate(0, 0, 15)

	draft := newGoBDExportTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusDraft, "GOBD-DRAFT-0001", mid)
	sent := newGoBDExportTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusSent, "GOBD-SENT-0001", mid)
	paid := newGoBDExportTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusPaid, "GOBD-PAID-0001", mid)
	overdue := newGoBDExportTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusOverdue, "GOBD-OVERDUE-0001", mid)
	cancelled := newGoBDExportTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusCancelled, "GOBD-CANCELLED-0001", mid)

	got, err := repo.ListForGoBDExport(ctx, tenantID, from, to)
	require.NoError(t, err)

	ids := map[uuid.UUID]bool{}
	for _, inv := range got {
		ids[inv.ID] = true
	}
	assert.False(t, ids[draft.ID], "draft invoice must never appear in the GoBD export")
	assert.True(t, ids[sent.ID], "sent invoice must appear in the GoBD export")
	assert.True(t, ids[paid.ID], "paid invoice must appear in the GoBD export")
	assert.True(t, ids[overdue.ID], "overdue invoice must appear in the GoBD export")
	assert.True(t, ids[cancelled.ID], "cancelled invoice must appear in the GoBD export (gap-free journal, GoBD §146)")
	assert.Len(t, got, 4, "exactly the four non-draft invoices must be returned")
}

// TestPostgresRepository_ListForGoBDExport_EmptyInvoiceNumberExcluded proves
// the invoice_number != '' guard actually filters — a non-draft invoice
// without an assigned number (should not occur via the normal issuing
// path, but the query defends against it explicitly) must not appear.
func TestPostgresRepository_ListForGoBDExport_EmptyInvoiceNumberExcluded(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Invoice GoBD Export Empty Number Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	from := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
	mid := from.AddDate(0, 0, 10)

	unnumbered := newGoBDExportTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusSent, "", mid)
	numbered := newGoBDExportTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusSent, "GOBD-NUM-0001", mid)

	got, err := repo.ListForGoBDExport(ctx, tenantID, from, to)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, numbered.ID, got[0].ID)
	for _, inv := range got {
		assert.NotEqual(t, unnumbered.ID, inv.ID, "invoice without an assigned number must never appear in the GoBD export")
	}
}

// TestPostgresRepository_ListForGoBDExport_DateBoundariesInclusive proves
// fromDate and toDate are both inclusive, matching ListForDATEVExport.
func TestPostgresRepository_ListForGoBDExport_DateBoundariesInclusive(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Invoice GoBD Export Boundary Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	from := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)

	onFrom := newGoBDExportTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusSent, "GOBD-BOUND-0001", from)
	onTo := newGoBDExportTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusSent, "GOBD-BOUND-0002", to)
	beforeFrom := newGoBDExportTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusSent, "GOBD-BOUND-0003", from.AddDate(0, 0, -1))
	afterTo := newGoBDExportTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusSent, "GOBD-BOUND-0004", to.AddDate(0, 0, 1))

	got, err := repo.ListForGoBDExport(ctx, tenantID, from, to)
	require.NoError(t, err)
	ids := map[uuid.UUID]bool{}
	for _, inv := range got {
		ids[inv.ID] = true
	}
	assert.True(t, ids[onFrom.ID], "invoice dated exactly on fromDate must be included")
	assert.True(t, ids[onTo.ID], "invoice dated exactly on toDate must be included")
	assert.False(t, ids[beforeFrom.ID], "invoice one day before fromDate must be excluded")
	assert.False(t, ids[afterTo.ID], "invoice one day after toDate must be excluded")
}

// TestPostgresRepository_ListForGoBDExport_OrderedByInvoiceNumber proves the
// result is ordered by invoice_number, not by invoice_date or insertion
// order — the doc comment promises "gap-free journal verification", which
// only holds if the order matches the numbering sequence.
func TestPostgresRepository_ListForGoBDExport_OrderedByInvoiceNumber(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Invoice GoBD Export Order Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	from := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)

	// Insert with invoice_date and invoice_number deliberately out of step,
	// so ordering by the wrong column would be caught.
	third := newGoBDExportTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusSent, "RE-2026-0003", from.AddDate(0, 0, 1))
	first := newGoBDExportTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusSent, "RE-2026-0001", from.AddDate(0, 0, 20))
	second := newGoBDExportTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusSent, "RE-2026-0002", from.AddDate(0, 0, 10))

	got, err := repo.ListForGoBDExport(ctx, tenantID, from, to)
	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.Equal(t, first.ID, got[0].ID, "RE-2026-0001 must sort first")
	assert.Equal(t, second.ID, got[1].ID, "RE-2026-0002 must sort second")
	assert.Equal(t, third.ID, got[2].ID, "RE-2026-0003 must sort third")
}

// TestPostgresRepository_ListForGoBDExport_IncludesLineItems proves the
// returned invoices carry their line items — a GoBD/tax-advisor export
// without positions is not auditable.
func TestPostgresRepository_ListForGoBDExport_IncludesLineItems(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Invoice GoBD Export Lines Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	from := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2027, 1, 31, 0, 0, 0, 0, time.UTC)
	inv := newGoBDExportTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusSent, "GOBD-LINES-0001", from.AddDate(0, 0, 5))

	got, err := repo.ListForGoBDExport(ctx, tenantID, from, to)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, inv.ID, got[0].ID)
	assert.NotEmpty(t, got[0].LineItems, "GoBD export invoice must carry its line items")
}

// TestPostgresRepository_ListForGoBDExport_CrossTenantIsolation proves a
// tenant's export never includes another tenant's invoices.
func TestPostgresRepository_ListForGoBDExport_CrossTenantIsolation(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Invoice GoBD Export CrossTenant A")
	testutil.EnsureTenant(t, pool, tenantB, "Invoice GoBD Export CrossTenant B")
	defer testutil.CleanupRow(t, pool, "tenants", tenantA)
	defer testutil.CleanupRow(t, pool, "tenants", tenantB)

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	from := time.Date(2027, 2, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2027, 2, 28, 0, 0, 0, 0, time.UTC)
	mid := from.AddDate(0, 0, 5)

	invA := newGoBDExportTestInvoice(t, repo, pool, ctxA, tenantA, models.InvoiceStatusSent, "GOBD-CT-A-0001", mid)
	newGoBDExportTestInvoice(t, repo, pool, ctxB, tenantB, models.InvoiceStatusSent, "GOBD-CT-B-0001", mid)

	got, err := repo.ListForGoBDExport(ctxA, tenantA, from, to)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, invA.ID, got[0].ID)
}
