package payment

// postgres_repository_db_test.go exercises PostgresRepository against the
// real schema instead of a mock. This package has never run a single query
// against Postgres before (no DB test at all, and outside the "Finance & HR
// Integration Tests" CI job): payment is the point where an open invoice
// becomes a paid one, so a wrong tenant filter or a broken In-Tx coupling
// changes balances silently. Untagged (no //go:build integration) so it runs
// in the normal local gate and the coverage job — see BACKLOG.yml Befund 2.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// seedPaymentInvoice inserts a minimal finance_invoices row under the given
// tenant so payment rows (fk_finance_payments_invoice) have a parent to point
// at. Mirrors the pattern in hr/timetracking/postgres_invoice_reservation_test.go.
func seedPaymentInvoice(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	id := testutil.SeedRow(t, pool, "finance_invoices", map[string]any{
		"id":           uuid.New(),
		"tenant_id":    tenantID,
		"created_by":   uuid.New(),
		"invoice_date": time.Now(),
		"due_date":     time.Now().AddDate(0, 0, 30),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_invoices", id) })
	return id
}

func newTestPayment(tenantID, invoiceID uuid.UUID, amount string, paymentDate time.Time) *models.Payment {
	return &models.Payment{
		ID:          uuid.New(),
		TenantID:    tenantID,
		InvoiceID:   invoiceID,
		Amount:      decimal.RequireFromString(amount),
		PaymentDate: paymentDate,
		Method:      "bank_transfer",
		Reference:   "REF-" + uuid.New().String()[:8],
		CreatedBy:   uuid.New(),
		CreatedAt:   time.Now(),
	}
}

// TestPostgresRepository_Create_GetByID_RoundTripsExactDecimal proves the
// scan-as-string-then-decimal.NewFromString path in GetByID never routes the
// amount through float64: a value a naive float round-trip could perturb
// must come back byte-for-byte identical, both through the repo and read
// directly off the NUMERIC column.
func TestPostgresRepository_Create_GetByID_RoundTripsExactDecimal(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Payment Core Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	invoiceID := seedPaymentInvoice(t, pool, tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	p := newTestPayment(tenantID, invoiceID, "1029.33", time.Now())
	require.NoError(t, repo.Create(ctx, p))
	defer testutil.CleanupRow(t, pool, "finance_payments", p.ID)

	got, err := repo.GetByID(ctx, tenantID, p.ID)
	require.NoError(t, err)
	assert.True(t, decimal.RequireFromString("1029.33").Equal(got.Amount),
		"amount must survive the round trip as an exact decimal, got %s", got.Amount)
	assert.Equal(t, p.Method, got.Method)
	assert.Equal(t, p.Reference, got.Reference)

	var rawAmount string
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT amount::text FROM finance_payments WHERE id = $1", p.ID,
	).Scan(&rawAmount))
	assert.Equal(t, "1029.33", rawAmount, "on-disk NUMERIC value must not have passed through float64")
}

// TestPostgresRepository_GetByID_CrossTenant_ReturnsNotFound belongs to the
// class of test a mock cannot give: it proves the explicit tenant_id
// parameter (backed by RLS) actually refuses a foreign tenant, not just that
// the SQL string contains a WHERE clause.
func TestPostgresRepository_GetByID_CrossTenant_ReturnsNotFound(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Payment CrossTenant A")
	testutil.EnsureTenant(t, pool, tenantB, "Payment CrossTenant B")
	defer testutil.CleanupRow(t, pool, "tenants", tenantA)
	defer testutil.CleanupRow(t, pool, "tenants", tenantB)
	invoiceID := seedPaymentInvoice(t, pool, tenantA)

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	p := newTestPayment(tenantA, invoiceID, "50.00", time.Now())
	require.NoError(t, repo.Create(ctxA, p))
	defer testutil.CleanupRow(t, pool, "finance_payments", p.ID)

	_, err := repo.GetByID(ctxB, tenantB, p.ID)
	assert.ErrorIs(t, err, ErrNotFound, "a foreign tenant must never see another tenant's payment")

	got, err := repo.GetByID(ctxA, tenantA, p.ID)
	require.NoError(t, err)
	assert.Equal(t, p.ID, got.ID)
}

// TestPostgresRepository_List_TenantScopedAndOrdered proves List's WHERE
// (tenant_id, invoice_id) and its ORDER BY payment_date DESC, created_at DESC —
// neither is exercised by a mock, and a wrong tenant filter here would leak
// one customer's payment history into another's invoice view.
func TestPostgresRepository_List_TenantScopedAndOrdered(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Payment List Tenant A")
	testutil.EnsureTenant(t, pool, tenantB, "Payment List Tenant B")
	defer testutil.CleanupRow(t, pool, "tenants", tenantA)
	defer testutil.CleanupRow(t, pool, "tenants", tenantB)
	invoiceA := seedPaymentInvoice(t, pool, tenantA)
	invoiceB := seedPaymentInvoice(t, pool, tenantB)

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	base := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	older := newTestPayment(tenantA, invoiceA, "10.00", base)
	newer := newTestPayment(tenantA, invoiceA, "20.00", base.AddDate(0, 0, 1))
	other := newTestPayment(tenantB, invoiceB, "999.00", base)

	require.NoError(t, repo.Create(ctxA, older))
	defer testutil.CleanupRow(t, pool, "finance_payments", older.ID)
	require.NoError(t, repo.Create(ctxA, newer))
	defer testutil.CleanupRow(t, pool, "finance_payments", newer.ID)
	require.NoError(t, repo.Create(ctxB, other))
	defer testutil.CleanupRow(t, pool, "finance_payments", other.ID)

	list, err := repo.List(ctxA, tenantA, invoiceA)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, newer.ID, list[0].ID, "payment_date DESC: the later payment must come first")
	assert.Equal(t, older.ID, list[1].ID)
	for _, p := range list {
		assert.NotEqual(t, other.ID, p.ID, "tenant B's payment must never appear in tenant A's list")
	}

	listB, err := repo.List(ctxB, tenantB, invoiceB)
	require.NoError(t, err)
	require.Len(t, listB, 1)
	assert.Equal(t, other.ID, listB[0].ID)
}

// TestPostgresRepository_Delete_TenantScoped proves a foreign tenant's Delete
// call is a silent no-op (RLS + explicit WHERE both admit nothing) rather than
// an error that happens to leave the row alone for the wrong reason.
func TestPostgresRepository_Delete_TenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Payment Delete Tenant A")
	testutil.EnsureTenant(t, pool, tenantB, "Payment Delete Tenant B")
	defer testutil.CleanupRow(t, pool, "tenants", tenantA)
	defer testutil.CleanupRow(t, pool, "tenants", tenantB)
	invoiceA := seedPaymentInvoice(t, pool, tenantA)

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	p := newTestPayment(tenantA, invoiceA, "42.00", time.Now())
	require.NoError(t, repo.Create(ctxA, p))
	defer testutil.CleanupRow(t, pool, "finance_payments", p.ID)

	require.NoError(t, repo.Delete(ctxB, tenantB, p.ID))
	_, err := repo.GetByID(ctxA, tenantA, p.ID)
	require.NoError(t, err, "payment must survive a cross-tenant delete attempt")

	require.NoError(t, repo.Delete(ctxA, tenantA, p.ID))
	_, err = repo.GetByID(ctxA, tenantA, p.ID)
	assert.ErrorIs(t, err, ErrNotFound)
}

// TestPostgresRepository_CreateInTx_RollbackLeavesNoRow proves CreateInTx
// participates in the caller's transaction rather than committing on its own
// connection — the InTx variant exists specifically so a payment write and
// the coupled invoice status transition either both land or neither does.
func TestPostgresRepository_CreateInTx_RollbackLeavesNoRow(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Payment CreateInTx Rollback")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	invoiceID := seedPaymentInvoice(t, pool, tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	p := newTestPayment(tenantID, invoiceID, "77.70", time.Now())

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, repo.CreateInTx(ctx, tx, p))
	require.NoError(t, tx.Rollback(ctx))

	_, err = repo.GetByID(ctx, tenantID, p.ID)
	assert.ErrorIs(t, err, ErrNotFound, "a rolled-back CreateInTx must leave no row behind")
}

// TestPostgresRepository_CreateInTx_CommitPersistsRow is the positive half:
// a committed CreateInTx must actually persist, with the same decimal
// fidelity as the standalone Create path.
func TestPostgresRepository_CreateInTx_CommitPersistsRow(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Payment CreateInTx Commit")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	invoiceID := seedPaymentInvoice(t, pool, tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	p := newTestPayment(tenantID, invoiceID, "88.80", time.Now())
	defer testutil.CleanupRow(t, pool, "finance_payments", p.ID)

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, repo.CreateInTx(ctx, tx, p))
	require.NoError(t, tx.Commit(ctx))

	got, err := repo.GetByID(ctx, tenantID, p.ID)
	require.NoError(t, err)
	assert.True(t, decimal.RequireFromString("88.80").Equal(got.Amount))
}

// TestPostgresRepository_DeleteInTx_RollbackKeepsRow is the delete-side
// counterpart: a rolled-back DeleteInTx must not have removed anything,
// proving the delete really runs on the caller's transaction and not on a
// separate pool connection that would commit independently.
func TestPostgresRepository_DeleteInTx_RollbackKeepsRow(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Payment DeleteInTx Rollback")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	invoiceID := seedPaymentInvoice(t, pool, tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	p := newTestPayment(tenantID, invoiceID, "15.15", time.Now())
	require.NoError(t, repo.Create(ctx, p))
	defer testutil.CleanupRow(t, pool, "finance_payments", p.ID)

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, repo.DeleteInTx(ctx, tx, tenantID, p.ID))
	require.NoError(t, tx.Rollback(ctx))

	got, err := repo.GetByID(ctx, tenantID, p.ID)
	require.NoError(t, err, "a rolled-back DeleteInTx must leave the row intact")
	assert.Equal(t, p.ID, got.ID)
}

// TestPostgresRepository_DeleteInTx_CommitRemovesRow is DeleteInTx's positive
// path: once committed, the row is actually gone.
func TestPostgresRepository_DeleteInTx_CommitRemovesRow(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Payment DeleteInTx Commit")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	invoiceID := seedPaymentInvoice(t, pool, tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	p := newTestPayment(tenantID, invoiceID, "16.16", time.Now())
	require.NoError(t, repo.Create(ctx, p))
	defer testutil.CleanupRow(t, pool, "finance_payments", p.ID)

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, repo.DeleteInTx(ctx, tx, tenantID, p.ID))
	require.NoError(t, tx.Commit(ctx))

	_, err = repo.GetByID(ctx, tenantID, p.ID)
	assert.ErrorIs(t, err, ErrNotFound)
}
