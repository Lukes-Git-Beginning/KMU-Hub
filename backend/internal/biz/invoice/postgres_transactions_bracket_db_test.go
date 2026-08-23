package invoice

// postgres_transactions_bracket_db_test.go exercises the transaction bracket
// that couples header, line-item, and (for Send) number-sequence writes into
// one atomic unit -- Create (postgres_repository.go:48) and UpdateInTx
// (postgres_repository.go:328), the same primitive invoice.Service.Send uses
// to couple NextNumberInTx with the status/number update. Untagged
// (testutil.SkipIfNoDB) so it runs in the local gate and counts toward this
// package's coverage number.
//
// The existing send_atomic_integration_test.go (build tag "integration")
// already proves the Send-specific case end-to-end: a failed line re-insert
// during Send does not burn a gap-free invoice number. It is intentionally
// left untouched -- it gates the CI integration job. The tests here cover the
// bracket's other failure sources instead of duplicating that one: a failed
// header insert, a failed line insert, a failed header update (before any
// line touches happen at all), and a transaction aborted after a valid write
// but before commit. Every case checks row counts on ALL tables the bracket
// touches, not just the one that failed -- a rollback that misses a sibling
// table is exactly the kind of bug an aggregate-only check would hide.
import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// TestPostgresRepository_Create_RollbackOnHeaderConstraintViolation proves the
// "Fehler beim Anlegen der Rechnung" case: chk_finance_invoices_status rejects
// the header INSERT before insertInvoiceLines ever runs, and the whole tx
// (header + the lines that were never reached) leaves nothing behind.
func TestPostgresRepository_Create_RollbackOnHeaderConstraintViolation(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Tx Header Rollback Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	now := time.Now().UTC()
	inv := newListTestInvoice(tenantID, now, now.AddDate(0, 0, 14))
	inv.Status = "not-a-real-status" // violates chk_finance_invoices_status

	err := repo.Create(ctx, inv)
	require.Error(t, err)

	var invoiceCount int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM finance_invoices WHERE id = $1`, inv.ID).Scan(&invoiceCount))
	assert.Equal(t, 0, invoiceCount, "header insert must not survive a rolled-back tx")

	var lineCount int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM finance_invoice_lines WHERE invoice_id = $1`, inv.ID).Scan(&lineCount))
	assert.Equal(t, 0, lineCount, "line insert must never run once the header insert already failed")
}

// TestPostgresRepository_Create_RollbackOnLineConstraintViolation proves the
// "Fehler beim Anlegen der Positionen" case: the header insert succeeds
// inside the tx, a later line violates chk_invoice_lines_quantity, and the
// rollback must take the already-inserted header AND the first, valid line
// down with it -- not just the line that failed.
func TestPostgresRepository_Create_RollbackOnLineConstraintViolation(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Tx Line Rollback Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	now := time.Now().UTC()
	inv := newListTestInvoice(tenantID, now, now.AddDate(0, 0, 14))
	lines := []models.LineItem{
		{ID: "l1", Position: 1, Description: "Valid", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(50), TaxRate: decimal.NewFromInt(19), LineTotal: decimal.NewFromInt(50)},
		{ID: "l2", Position: 2, Description: "Invalid quantity", Quantity: decimal.Zero, UnitPrice: decimal.NewFromInt(50), TaxRate: decimal.NewFromInt(19), LineTotal: decimal.Zero},
	}
	raw, err := json.Marshal(lines)
	require.NoError(t, err)
	inv.LineItems = raw

	err = repo.Create(ctx, inv)
	require.Error(t, err)

	var invoiceCount int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM finance_invoices WHERE id = $1`, inv.ID).Scan(&invoiceCount))
	assert.Equal(t, 0, invoiceCount, "header must roll back together with the failed line insert")

	var lineCount int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM finance_invoice_lines WHERE invoice_id = $1`, inv.ID).Scan(&lineCount))
	assert.Equal(t, 0, lineCount, "the first, valid line must not survive either -- the whole insert is one transaction")
}

// TestPostgresRepository_UpdateInTx_RollbackOnHeaderConstraintViolation proves
// a header-only failure inside the bracket UpdateInTx uses for Send's coupled
// number+status write: the header UPDATE fails before the DELETE of the old
// lines even runs, so the original line must be completely untouched, not
// merely restored.
func TestPostgresRepository_UpdateInTx_RollbackOnHeaderConstraintViolation(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Tx UpdateInTx Header Rollback Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	now := time.Now().UTC()
	inv := newListTestInvoice(tenantID, now, now.AddDate(0, 0, 14))
	createListTestInvoice(t, repo, pool, ctx, inv)

	mutated := *inv
	mutated.Status = "not-a-real-status" // violates chk_finance_invoices_status
	mutated.CustomerName = "Should Never Persist"

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx) //nolint:errcheck // safety net if an assertion below fails before the explicit rollback
	updateErr := repo.UpdateInTx(ctx, tx, &mutated)
	require.Error(t, updateErr)
	require.NoError(t, tx.Rollback(ctx))

	got, err := repo.GetByID(ctx, tenantID, inv.ID)
	require.NoError(t, err)
	assert.Equal(t, models.InvoiceStatusDraft, got.Status)
	assert.Equal(t, inv.CustomerName, got.CustomerName)

	var lineCount int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM finance_invoice_lines WHERE invoice_id = $1`, inv.ID).Scan(&lineCount))
	assert.Equal(t, 1, lineCount, "the header UPDATE failed before the DELETE ran -- the original line must never have been touched")
}

// TestPostgresRepository_UpdateInTx_RollbackOnLineReinsertRestoresDeletedLine
// proves the case send_atomic_integration_test.go demonstrates only through
// the full Send flow (number assignment + update), isolated to the
// replace-strategy line rewrite itself: the header UPDATE succeeds inside the
// tx, the old line is DELETEd, the re-INSERT then violates a check
// constraint, and the rollback must restore both the header AND the deleted
// line -- proving the delete-then-reinsert step is not two separate
// commitment points.
func TestPostgresRepository_UpdateInTx_RollbackOnLineReinsertRestoresDeletedLine(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Tx UpdateInTx Line Rollback Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	now := time.Now().UTC()
	inv := newListTestInvoice(tenantID, now, now.AddDate(0, 0, 14))
	createListTestInvoice(t, repo, pool, ctx, inv)

	mutated := *inv
	mutated.CustomerName = "Should Also Never Persist"
	badLines := []models.LineItem{
		{ID: "bad", Position: 1, Description: "Invalid", Quantity: decimal.Zero, UnitPrice: decimal.NewFromInt(10), TaxRate: decimal.NewFromInt(19), LineTotal: decimal.Zero},
	}
	raw, err := json.Marshal(badLines)
	require.NoError(t, err)
	mutated.LineItems = raw

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx) //nolint:errcheck // safety net if an assertion below fails before the explicit rollback
	updateErr := repo.UpdateInTx(ctx, tx, &mutated)
	require.Error(t, updateErr, "the header UPDATE inside this tx succeeds; the line re-insert must fail and take the whole tx down")
	require.NoError(t, tx.Rollback(ctx))

	got, err := repo.GetByID(ctx, tenantID, inv.ID)
	require.NoError(t, err)
	assert.Equal(t, inv.CustomerName, got.CustomerName, "the header UPDATE that ran inside the aborted tx must not persist")

	var lineCount int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM finance_invoice_lines WHERE invoice_id = $1`, inv.ID).Scan(&lineCount))
	assert.Equal(t, 1, lineCount, "the original line, deleted mid-tx before the failed re-insert, must be restored by the rollback")
}

// TestPostgresRepository_UpdateInTx_AbortedTxDoesNotPersistPriorWrite proves
// the "Fehler beim Commit" case: a fully valid UpdateInTx call succeeds
// inside the tx, but a later statement on the same connection (standing in
// for a concurrent serialization failure or a sibling write elsewhere in a
// larger caller-owned transaction, as Send couples with NextNumberInTx)
// aborts the transaction before commit. pgx surfaces this as a Commit error
// ("commit unexpectedly resulted in rollback"), and the already-valid write
// must not have reached the table either way.
func TestPostgresRepository_UpdateInTx_AbortedTxDoesNotPersistPriorWrite(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Tx Commit Rollback Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	now := time.Now().UTC()
	inv := newListTestInvoice(tenantID, now, now.AddDate(0, 0, 14))
	createListTestInvoice(t, repo, pool, ctx, inv)

	mutated := *inv
	mutated.CustomerName = "Committed Only If Tx Survives"

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx) //nolint:errcheck // safety net if an assertion below fails before the explicit rollback
	require.NoError(t, repo.UpdateInTx(ctx, tx, &mutated), "the update itself is valid and must succeed inside the tx")

	_, execErr := tx.Exec(ctx, `SELECT 1/0`)
	require.Error(t, execErr, "the division by zero must abort the transaction")

	commitErr := tx.Commit(ctx)
	require.Error(t, commitErr, "committing an already-aborted tx must fail")
	_ = tx.Rollback(ctx) // best-effort: Commit already closed the tx

	got, err := repo.GetByID(ctx, tenantID, inv.ID)
	require.NoError(t, err)
	assert.Equal(t, inv.CustomerName, got.CustomerName, "a transaction aborted before commit must not leave the otherwise-valid update applied")
}
