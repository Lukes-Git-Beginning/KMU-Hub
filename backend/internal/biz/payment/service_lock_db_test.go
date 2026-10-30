package payment

// service_lock_db_test.go proves Record/Delete serialize on the invoice row.
// Without the lock each of two concurrent payments sums only its own
// uncommitted row (READ COMMITTED), so two payments that together cover the
// invoice leave it "sent" and the dunning run keeps chasing a paid invoice.
// The first writer is held open on a second connection; no goroutine timing
// decides the outcome, the test only waits for the second one to be blocked.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/biz/invoice"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func seedLockInvoice(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, status, gross string) uuid.UUID {
	t.Helper()
	id := testutil.SeedRow(t, pool, "finance_invoices", map[string]any{
		"id":           uuid.New(),
		"tenant_id":    tenantID,
		"created_by":   uuid.New(),
		"invoice_date": time.Now(),
		"due_date":     time.Now().AddDate(0, 0, 30),
		"status":       status,
		"gross_total":  gross,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_invoices", id) })
	return id
}

func invoiceStatus(t *testing.T, pool *pgxpool.Pool, ctx context.Context, id uuid.UUID) string {
	t.Helper()
	var status string
	require.NoError(t, pool.QueryRow(ctx, `SELECT status FROM finance_invoices WHERE id = $1`, id).Scan(&status))
	return status
}

// waitForBlockedBackend polls until some other backend waits on a lock, or the
// deadline passes. Returns whether one was seen.
func waitForBlockedBackend(t *testing.T, pool *pgxpool.Pool) bool {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var n int
		require.NoError(t, pool.QueryRow(context.Background(),
			`SELECT count(*) FROM pg_stat_activity WHERE wait_event_type = 'Lock' AND datname = current_database()`).Scan(&n))
		if n > 0 {
			return true
		}
		time.Sleep(25 * time.Millisecond)
	}
	return false
}

func TestService_Record_SecondPaymentWaitsForInvoiceLockAndPaysInvoice(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Payment Lock")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	invoiceID := seedLockInvoice(t, pool, tenantID, models.InvoiceStatusSent, "1000.00")

	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	repo := NewPostgresRepository(pool)
	invRepo := invoice.NewPostgresRepository(pool)
	svc := NewService(repo, invRepo, invRepo, pool)

	// First writer: 500 recorded but uncommitted. Its insert only holds the FK
	// share lock on the invoice row, which is exactly what a second Record that
	// locks the row FOR UPDATE has to wait for.
	txA, err := pool.Begin(ctx)
	require.NoError(t, err)
	first := newTestPayment(tenantID, invoiceID, "500.00", time.Now())
	defer testutil.CleanupRow(t, pool, "finance_payments", first.ID)
	require.NoError(t, repo.CreateInTx(ctx, txA, first))

	// Second writer: must queue behind the lock.
	type result struct {
		p   *models.Payment
		err error
	}
	done := make(chan result, 1)
	go func() {
		p, recErr := svc.Record(ctx, RecordInput{
			TenantID: tenantID, InvoiceID: invoiceID, UserID: uuid.New(),
			Amount: decimal.RequireFromString("500.00"), Method: "bank_transfer",
		})
		done <- result{p, recErr}
	}()

	blocked := waitForBlockedBackend(t, pool)
	require.NoError(t, txA.Commit(ctx))
	res := <-done
	require.NoError(t, res.err)
	defer testutil.CleanupRow(t, pool, "finance_payments", res.p.ID)

	assert.True(t, blocked, "second Record must wait for the invoice row lock")
	assert.Equal(t, models.InvoiceStatusPaid, invoiceStatus(t, pool, ctx, invoiceID),
		"two payments covering the gross total must leave the invoice paid")
}

func TestService_Delete_WaitsForInvoiceLockAndUsesLockedStatus(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Payment Lock Delete")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	invoiceID := seedLockInvoice(t, pool, tenantID, models.InvoiceStatusPaid, "1000.00")

	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	repo := NewPostgresRepository(pool)
	invRepo := invoice.NewPostgresRepository(pool)
	svc := NewService(repo, invRepo, invRepo, pool)

	pay := newTestPayment(tenantID, invoiceID, "1000.00", time.Now())
	require.NoError(t, repo.Create(ctx, pay))
	defer testutil.CleanupRow(t, pool, "finance_payments", pay.ID)

	// A concurrent writer cancels the invoice and has not committed yet.
	txA, err := pool.Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, invRepo.UpdateStatusInTx(ctx, txA, tenantID, invoiceID, models.InvoiceStatusCancelled))

	done := make(chan error, 1)
	go func() { done <- svc.Delete(ctx, tenantID, pay.ID) }()

	blocked := waitForBlockedBackend(t, pool)
	require.NoError(t, txA.Commit(ctx))
	delErr := <-done

	assert.True(t, blocked, "Delete must wait for the invoice row lock")
	require.Error(t, delErr, "Delete must see the cancellation committed under the lock")
	got, getErr := repo.GetByID(ctx, tenantID, pay.ID)
	require.NoError(t, getErr)
	assert.Equal(t, pay.ID, got.ID, "payment on the cancelled invoice must survive")
}
