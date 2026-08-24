package invoice

// postgres_repository_datev_export_db_test.go exercises ListForDATEVExport
// (postgres_repository.go:626-682) against real Postgres. It is the read
// path of the DATEV export — a bug in the keyset paging loses invoices
// silently: the export looks complete and isn't, and nobody notices until
// the tax advisor asks. Untagged (testutil.SkipIfNoDB) so it runs in the
// local gate and counts toward this package's coverage, unlike
// integration_test.go.

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// newDatevExportTestInvoice creates and persists an invoice with the given
// status and invoice_date via the real Create path (source defaults to
// 'cosmi' at the DB level, per migration 000243).
func newDatevExportTestInvoice(t *testing.T, repo *PostgresRepository, pool *pgxpool.Pool, ctx context.Context, tenantID uuid.UUID, status string, invoiceDate time.Time) *models.Invoice {
	t.Helper()
	inv := newListTestInvoice(tenantID, invoiceDate, invoiceDate.AddDate(0, 0, 30))
	inv.Status = status
	createListTestInvoice(t, repo, pool, ctx, inv)
	return inv
}

// TestPostgresRepository_ListForDATEVExport_PagingCoversFullSet proves that
// paging through with a small limit returns exactly the same set of
// invoices as a single call with a limit large enough for one page — no
// duplicates, no gaps — across a mix of matching and non-matching rows
// (wrong status, outside the date range).
func TestPostgresRepository_ListForDATEVExport_PagingCoversFullSet(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Invoice DATEV Export Paging Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	from := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC)

	statuses := []string{
		models.InvoiceStatusSent, models.InvoiceStatusPaid, models.InvoiceStatusOverdue,
		models.InvoiceStatusSent, models.InvoiceStatusPaid, models.InvoiceStatusOverdue,
		models.InvoiceStatusSent,
	}
	expected := map[uuid.UUID]bool{}
	for i, status := range statuses {
		inv := newDatevExportTestInvoice(t, repo, pool, ctx, tenantID, status, from.AddDate(0, 0, i))
		expected[inv.ID] = true
	}

	// Non-matching: wrong status inside the range, and right status outside it.
	draft := newDatevExportTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusDraft, from.AddDate(0, 0, 2))
	_ = draft
	beforeRange := newDatevExportTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusSent, from.AddDate(0, 0, -1))
	_ = beforeRange
	afterRange := newDatevExportTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusSent, to.AddDate(0, 0, 1))
	_ = afterRange

	baseline, err := repo.ListForDATEVExport(ctx, tenantID, from, to, nil, nil, 100)
	require.NoError(t, err)
	baselineIDs := map[uuid.UUID]bool{}
	for _, inv := range baseline {
		baselineIDs[inv.ID] = true
	}
	assert.Equal(t, expected, baselineIDs, "single-page baseline must match exactly the matching rows")

	pagedIDs := map[uuid.UUID]bool{}
	var seenOrder []uuid.UUID
	var cursorDate *time.Time
	var cursorID *uuid.UUID
	const pageSize = 2
	for pages := 0; ; pages++ {
		require.Less(t, pages, 20, "paging did not terminate — possible infinite loop")
		page, pageErr := repo.ListForDATEVExport(ctx, tenantID, from, to, cursorDate, cursorID, pageSize)
		require.NoError(t, pageErr)
		if len(page) == 0 {
			break
		}
		for _, inv := range page {
			require.False(t, pagedIDs[inv.ID], "duplicate invoice %s returned across pages", inv.ID)
			pagedIDs[inv.ID] = true
			seenOrder = append(seenOrder, inv.ID)
		}
		last := page[len(page)-1]
		d := last.InvoiceDate
		id := last.ID
		cursorDate, cursorID = &d, &id
		if len(page) < pageSize {
			break
		}
	}

	assert.Equal(t, expected, pagedIDs, "paged traversal must cover exactly the matching rows, no gap")
	assert.Len(t, seenOrder, len(expected), "no duplicates across pages")
}

// TestPostgresRepository_ListForDATEVExport_SameDateCursorTiebreak proves the
// id is the second sort key: three invoices sharing the same invoice_date
// are still paged correctly across a page boundary that falls between them.
func TestPostgresRepository_ListForDATEVExport_SameDateCursorTiebreak(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Invoice DATEV Export Tiebreak Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	from := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)
	sameDate := from.AddDate(0, 0, 10)

	var ids []uuid.UUID
	for range 3 {
		inv := newDatevExportTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusSent, sameDate)
		ids = append(ids, inv.ID)
	}
	// Postgres orders uuid by raw bytes, which matches lexicographic order of
	// the canonical hyphenated string form (same hex digits, same positions).
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })

	// First page: afterDate/afterID nil, limit smaller than the tied group.
	firstPage, err := repo.ListForDATEVExport(ctx, tenantID, from, to, nil, nil, 2)
	require.NoError(t, err)
	require.Len(t, firstPage, 2)
	assert.Equal(t, ids[0], firstPage[0].ID)
	assert.Equal(t, ids[1], firstPage[1].ID)

	// Second page: cursor set to the last row of the first page, same date —
	// must return the remaining tied row, not repeat or skip it.
	last := firstPage[len(firstPage)-1]
	cursorDate := last.InvoiceDate
	cursorID := last.ID
	secondPage, err := repo.ListForDATEVExport(ctx, tenantID, from, to, &cursorDate, &cursorID, 2)
	require.NoError(t, err)
	require.Len(t, secondPage, 1)
	assert.Equal(t, ids[2], secondPage[0].ID)
}

// TestPostgresRepository_ListForDATEVExport_DateBoundariesInclusive proves
// fromDate and toDate are both inclusive.
func TestPostgresRepository_ListForDATEVExport_DateBoundariesInclusive(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Invoice DATEV Export Boundary Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	from := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)

	onFrom := newDatevExportTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusSent, from)
	onTo := newDatevExportTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusSent, to)
	beforeFrom := newDatevExportTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusSent, from.AddDate(0, 0, -1))
	afterTo := newDatevExportTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusSent, to.AddDate(0, 0, 1))

	got, err := repo.ListForDATEVExport(ctx, tenantID, from, to, nil, nil, 100)
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

// TestPostgresRepository_ListForDATEVExport_StatusFilter proves only
// sent/paid/overdue invoices are returned — a draft invoice inside the date
// range must never appear on any page.
func TestPostgresRepository_ListForDATEVExport_StatusFilter(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Invoice DATEV Export Status Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	from := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	mid := from.AddDate(0, 0, 15)

	draft := newDatevExportTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusDraft, mid)
	sent := newDatevExportTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusSent, mid)

	got, err := repo.ListForDATEVExport(ctx, tenantID, from, to, nil, nil, 100)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, sent.ID, got[0].ID)
	for _, inv := range got {
		assert.NotEqual(t, draft.ID, inv.ID, "draft invoice must never appear in the DATEV export")
	}
}

// TestPostgresRepository_ListForDATEVExport_CrossTenantIsolation proves a
// tenant's export never includes another tenant's invoices, even though
// both fall inside the same date range and status.
func TestPostgresRepository_ListForDATEVExport_CrossTenantIsolation(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Invoice DATEV Export CrossTenant A")
	testutil.EnsureTenant(t, pool, tenantB, "Invoice DATEV Export CrossTenant B")
	defer testutil.CleanupRow(t, pool, "tenants", tenantA)
	defer testutil.CleanupRow(t, pool, "tenants", tenantB)

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	from := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)
	mid := from.AddDate(0, 0, 5)

	invA := newDatevExportTestInvoice(t, repo, pool, ctxA, tenantA, models.InvoiceStatusSent, mid)
	newDatevExportTestInvoice(t, repo, pool, ctxB, tenantB, models.InvoiceStatusSent, mid)

	got, err := repo.ListForDATEVExport(ctxA, tenantA, from, to, nil, nil, 100)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, invA.ID, got[0].ID)
}
