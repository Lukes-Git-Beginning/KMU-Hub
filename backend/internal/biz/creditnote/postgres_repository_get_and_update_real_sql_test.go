package creditnote

// postgres_repository_get_and_update_real_sql_test.go exercises GetByID's
// not-found path and UpdateInTx's persistence, rollback and cross-tenant
// behavior against real Postgres — the parts of the repository the existing
// //go:build integration tests (integration_test.go, repository_coverage_
// integration_test.go, send_atomic_integration_test.go — all testcontainer-
// backed via internal/testsupport/pgtc) do not cover:
//
//   - integration_test.go: TestCreditNoteLinesRelationalRoundtrip (GetByID with
//     line items), TestCreditNoteUpdateReplacesLines (Update deletes+reinserts
//     lines), TestCreditNoteTenantIsolation (RLS blocks a cross-tenant GetByID).
//   - repository_coverage_integration_test.go: TestList_FiltersPaginationAnd-
//     TenantScoping, TestGetByInvoiceID_TenantScopedAndOrderedDescending,
//     TestListForDATEVExport_DateRangeStatusAndKeysetPaging,
//     TestCreate_AmountNotValidatedAgainstInvoiceOpenBalance (the "credit over
//     the invoice amount" question — answered there: Create does not reject
//     it; see also the comment on PostgresRepository.Update),
//     TestCreate_DecimalAmountsSurviveDBRoundTripAsExactStrings (line-level
//     decimals, via Create).
//   - send_atomic_integration_test.go: TestCreditNoteSend_AtomicRollback_
//     NumberNotConsumed (Service.Send's number assignment + UpdateInTx coupled
//     in one tx — the number sequence itself lives in quote.NewPostgresNumber-
//     SequenceRepo, outside this package; Delete does not exist on this
//     repository at all — there is no hard-delete path for credit notes).
//
// None of the above cover: GetByID's not-found error mapping, UpdateInTx used
// standalone with an explicit rollback, a cross-tenant UpdateInTx call, or
// whether the header-level totals (subtotal/total_tax/gross_total — set
// directly by the caller, not derived from the lines by the DB) survive a
// round trip byte-exact, including a value the tax calculator would never
// produce (negative). This file covers exactly those.
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

func oneCreditNoteLine() []models.LineItem {
	return []models.LineItem{{
		Position: 1, Description: "Gutschrift Position",
		Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(50),
		TaxRate: decimal.NewFromInt(19), LineTotal: decimal.NewFromInt(50),
	}}
}

func baseCreditNote(tenantID, origInvoiceID uuid.UUID, lines []models.LineItem) *models.CreditNote {
	raw, _ := json.Marshal(lines)
	now := time.Now().UTC().Truncate(time.Second)
	return &models.CreditNote{
		ID:                uuid.New(),
		TenantID:          tenantID,
		CreditNoteNumber:  "",
		Status:            models.CreditNoteStatusDraft,
		OriginalInvoiceID: origInvoiceID,
		CustomerName:      "Real-SQL Test GmbH",
		CustomerAddress:   "Teststraße 1, 10115 Berlin",
		CustomerEmail:     "cn-realsql@example.com",
		TaxMode:           models.TaxModeStandard,
		LineItems:         raw,
		Subtotal:          decimal.NewFromInt(50),
		TotalTax:          decimal.NewFromFloat(9.5),
		GrossTotal:        decimal.NewFromFloat(59.5),
		Currency:          models.DefaultCurrency,
		Reason:            "Real-SQL Coverage Test",
		CreatedBy:         uuid.New(),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

func TestPostgresRepository_GetByID_NotFound(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "CreditNote GetByID NotFound Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	_, err := repo.GetByID(ctx, tenantID, uuid.New())
	assert.ErrorIs(t, err, ErrCreditNoteNotFound)
}

func TestPostgresRepository_UpdateInTx_RolledBackTransactionPersistsNothing(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "CreditNote UpdateInTx Rollback Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	invoiceID := testutil.SeedRow(t, pool, "finance_invoices", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID,
		"invoice_date": time.Now().UTC(), "due_date": time.Now().UTC().AddDate(0, 0, 14),
		"created_by": uuid.New(),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_invoices", invoiceID) })

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	cn := baseCreditNote(tenantID, invoiceID, oneCreditNoteLine())
	require.NoError(t, repo.Create(ctx, cn))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_credit_notes", cn.ID) })

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)

	cn.Status = models.CreditNoteStatusSent
	cn.CreditNoteNumber = "GS-ROLLBACK-TEST"
	cn.UpdatedAt = time.Now().UTC()
	require.NoError(t, repo.UpdateInTx(ctx, tx, cn))

	require.NoError(t, tx.Rollback(ctx))

	got, err := repo.GetByID(ctx, tenantID, cn.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CreditNoteStatusDraft, got.Status, "a rolled-back UpdateInTx must not leave the status change behind")
	assert.Empty(t, got.CreditNoteNumber, "a rolled-back UpdateInTx must not leave the number assignment behind")
}

// TestPostgresRepository_Update_CrossTenantIsNoop mirrors the same
// silent-noop shape LinkTimeTracking/SetLock/UpdateStatus have on the invoice
// repository: a foreign tenant's Update call affects zero rows and returns no
// error — the WHERE tenant_id=$14 clause in UpdateInTx is a second, explicit
// barrier on top of RLS.
func TestPostgresRepository_Update_CrossTenantIsNoop(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	ownerTenant := uuid.New()
	testutil.EnsureTenant(t, pool, ownerTenant, "CreditNote Update Owner Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", ownerTenant)

	foreignTenant := uuid.New()
	testutil.EnsureTenant(t, pool, foreignTenant, "CreditNote Update Foreign Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", foreignTenant)

	invoiceID := testutil.SeedRow(t, pool, "finance_invoices", map[string]any{
		"id": uuid.New(), "tenant_id": ownerTenant,
		"invoice_date": time.Now().UTC(), "due_date": time.Now().UTC().AddDate(0, 0, 14),
		"created_by": uuid.New(),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_invoices", invoiceID) })

	repo := NewPostgresRepository(pool)
	ownerCtx := testutil.WithTenantCtx(context.Background(), ownerTenant)

	cn := baseCreditNote(ownerTenant, invoiceID, oneCreditNoteLine())
	require.NoError(t, repo.Create(ownerCtx, cn))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_credit_notes", cn.ID) })

	foreignCtx := testutil.WithTenantCtx(context.Background(), foreignTenant)
	foreignView := *cn
	foreignView.TenantID = foreignTenant
	foreignView.Status = models.CreditNoteStatusSent
	foreignView.CreditNoteNumber = "GS-FOREIGN-TEST"
	require.NoError(t, repo.Update(foreignCtx, &foreignView), "a cross-tenant Update must not error, it must match zero rows")

	got, err := repo.GetByID(ownerCtx, ownerTenant, cn.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CreditNoteStatusDraft, got.Status, "a cross-tenant Update must not affect the owner's row")
	assert.Empty(t, got.CreditNoteNumber)
}

// TestPostgresRepository_UpdateInTx_HeaderTotalsSurviveRoundTripAsExactStrings
// proves subtotal/total_tax/gross_total — set directly by the caller from
// tax.Calculate, not derived by the DB — survive a round trip byte-exact via
// shopspring/decimal, including a negative total_tax the tax calculator would
// never itself produce: the repository performs no sign/range validation on
// these columns (same finding as TestCreate_AmountNotValidatedAgainstInvoice-
// OpenBalance — the repo trusts the caller).
func TestPostgresRepository_UpdateInTx_HeaderTotalsSurviveRoundTripAsExactStrings(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "CreditNote UpdateInTx Decimal Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	invoiceID := testutil.SeedRow(t, pool, "finance_invoices", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID,
		"invoice_date": time.Now().UTC(), "due_date": time.Now().UTC().AddDate(0, 0, 14),
		"created_by": uuid.New(),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_invoices", invoiceID) })

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	cn := baseCreditNote(tenantID, invoiceID, oneCreditNoteLine())
	require.NoError(t, repo.Create(ctx, cn))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_credit_notes", cn.ID) })

	subtotal, _ := decimal.NewFromString("1234.56")
	totalTax, _ := decimal.NewFromString("-7.89")
	grossTotal, _ := decimal.NewFromString("1226.67")

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	cn.Subtotal, cn.TotalTax, cn.GrossTotal = subtotal, totalTax, grossTotal
	cn.UpdatedAt = time.Now().UTC()
	require.NoError(t, repo.UpdateInTx(ctx, tx, cn))
	require.NoError(t, tx.Commit(ctx))

	got, err := repo.GetByID(ctx, tenantID, cn.ID)
	require.NoError(t, err)
	assert.Equal(t, subtotal.String(), got.Subtotal.String())
	assert.Equal(t, totalTax.String(), got.TotalTax.String(), "the repository does not reject a negative total_tax")
	assert.Equal(t, grossTotal.String(), got.GrossTotal.String())
}
