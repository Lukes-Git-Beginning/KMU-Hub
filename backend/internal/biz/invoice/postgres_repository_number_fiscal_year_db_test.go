package invoice

// postgres_repository_number_fiscal_year_db_test.go exercises
// InvoiceNumberExists and CountByFiscalYear (postgres_repository.go:494-526)
// against real Postgres. Both back the GoBD gap-free/unique numbering
// guarantee: InvoiceNumberExists prevents a duplicate number within a
// tenant (ValidateInvoiceNumber, service_gobd.go), CountByFiscalYear feeds
// GetJournalSummary's gap detection. Untagged (testutil.SkipIfNoDB) so they
// run in the local gate and count toward this package's coverage number,
// unlike integration_test.go.

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

// newNumberFiscalYearTestInvoice creates and persists an invoice with the
// given status, invoice_number and invoice_date via the real Create path.
func newNumberFiscalYearTestInvoice(t *testing.T, repo *PostgresRepository, pool *pgxpool.Pool, ctx context.Context, tenantID uuid.UUID, status, invoiceNumber string, invoiceDate time.Time) *models.Invoice {
	t.Helper()
	inv := newListTestInvoice(tenantID, invoiceDate, invoiceDate.AddDate(0, 0, 30))
	inv.Status = status
	inv.InvoiceNumber = invoiceNumber
	createListTestInvoice(t, repo, pool, ctx, inv)
	return inv
}

// TestPostgresRepository_InvoiceNumberExists_IsTenantScoped proves the same
// invoice number assigned in tenant A does not "exist" for tenant B — a
// forgotten tenant_id condition here would couple numbering across tenants,
// a data leak in the numbering scheme (unique index idx_finance_invoices_number
// is already tenant-scoped, but the read side must honour that too).
func TestPostgresRepository_InvoiceNumberExists_IsTenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Invoice Number Exists Tenant A")
	testutil.EnsureTenant(t, pool, tenantB, "Invoice Number Exists Tenant B")
	defer testutil.CleanupRow(t, pool, "tenants", tenantA)
	defer testutil.CleanupRow(t, pool, "tenants", tenantB)

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	shared := "RE-2026-NUMEXIST-0001"
	when := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	newNumberFiscalYearTestInvoice(t, repo, pool, ctxA, tenantA, models.InvoiceStatusSent, shared, when)

	existsInA, err := repo.InvoiceNumberExists(ctxA, tenantA, shared)
	require.NoError(t, err)
	assert.True(t, existsInA, "number must exist for the tenant that owns it")

	existsInB, err := repo.InvoiceNumberExists(ctxB, tenantB, shared)
	require.NoError(t, err)
	assert.False(t, existsInB, "the same number in a foreign tenant must not be reported as existing (numbering must not be coupled across tenants)")
}

// TestPostgresRepository_InvoiceNumberExists_FalseForUnassignedNumber covers
// the negative case directly, independent of tenant isolation.
func TestPostgresRepository_InvoiceNumberExists_FalseForUnassignedNumber(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Invoice Number Exists Unassigned Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	exists, err := repo.InvoiceNumberExists(testutil.WithTenantCtx(context.Background(), tenantID), tenantID, "RE-2026-NEVER-ISSUED")
	require.NoError(t, err)
	assert.False(t, exists)
}

// TestPostgresRepository_CountByFiscalYear_YearBoundary proves the year
// filter is exact — an invoice dated 31 December belongs to that year, one
// dated 1 January the following day belongs to the next year.
// invoice_date is a plain DATE column (Migration 000045_create_finance_tables,
// not timestamptz), so this boundary is not subject to session-timezone
// drift the way a timestamptz column would be — confirmed by reading the
// column definition, not assumed.
func TestPostgresRepository_CountByFiscalYear_YearBoundary(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Invoice Fiscal Year Boundary Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	newNumberFiscalYearTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusSent, "RE-2030-BOUND-0001", time.Date(2030, 12, 31, 0, 0, 0, 0, time.UTC))
	newNumberFiscalYearTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusSent, "RE-2031-BOUND-0001", time.Date(2031, 1, 1, 0, 0, 0, 0, time.UTC))

	count2030, err := repo.CountByFiscalYear(ctx, tenantID, 2030)
	require.NoError(t, err)
	assert.Equal(t, 1, count2030, "31 December invoice must count toward its own calendar year")

	count2031, err := repo.CountByFiscalYear(ctx, tenantID, 2031)
	require.NoError(t, err)
	assert.Equal(t, 1, count2031, "1 January invoice must count toward the new calendar year, not the previous one")
}

// TestPostgresRepository_CountByFiscalYear_ExcludesDraftAndUnnumbered proves
// the status and invoice_number guards actually filter: a draft never has a
// gap-free-journal claim to make, and a non-draft invoice without an
// assigned number (defensive case, should not occur via the normal issuing
// path) must not inflate the count either.
func TestPostgresRepository_CountByFiscalYear_ExcludesDraftAndUnnumbered(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Invoice Fiscal Year Exclusion Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	when := time.Date(2032, 5, 1, 0, 0, 0, 0, time.UTC)

	newNumberFiscalYearTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusDraft, "RE-2032-DRAFT-0001", when)
	newNumberFiscalYearTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusSent, "", when)
	newNumberFiscalYearTestInvoice(t, repo, pool, ctx, tenantID, models.InvoiceStatusSent, "RE-2032-COUNTED-0001", when)

	count, err := repo.CountByFiscalYear(ctx, tenantID, 2032)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "only the sent invoice with an assigned number must be counted")
}

// TestPostgresRepository_CountByFiscalYear_IsTenantScoped proves a tenant's
// fiscal-year count never includes another tenant's invoices — a forgotten
// tenant_id condition here would shift GetJournalSummary's gap detection for
// every tenant sharing a year.
func TestPostgresRepository_CountByFiscalYear_IsTenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Invoice Fiscal Year CrossTenant A")
	testutil.EnsureTenant(t, pool, tenantB, "Invoice Fiscal Year CrossTenant B")
	defer testutil.CleanupRow(t, pool, "tenants", tenantA)
	defer testutil.CleanupRow(t, pool, "tenants", tenantB)

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)
	when := time.Date(2033, 3, 3, 0, 0, 0, 0, time.UTC)

	newNumberFiscalYearTestInvoice(t, repo, pool, ctxA, tenantA, models.InvoiceStatusSent, "RE-2033-CT-A-0001", when)
	newNumberFiscalYearTestInvoice(t, repo, pool, ctxB, tenantB, models.InvoiceStatusSent, "RE-2033-CT-B-0001", when)
	newNumberFiscalYearTestInvoice(t, repo, pool, ctxB, tenantB, models.InvoiceStatusSent, "RE-2033-CT-B-0002", when)

	countA, err := repo.CountByFiscalYear(ctxA, tenantA, 2033)
	require.NoError(t, err)
	assert.Equal(t, 1, countA, "tenant A must not see tenant B's invoices in its fiscal-year count")
}
