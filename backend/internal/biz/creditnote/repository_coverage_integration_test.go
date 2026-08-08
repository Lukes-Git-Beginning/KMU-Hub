//go:build integration

// External test package (like send_atomic_integration_test.go) so it can import
// the invoice package as a real InvoiceReader without imposing that import on
// package creditnote itself.
package creditnote_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/biz/creditnote"
	"github.com/kmuhub/kmuhub/internal/biz/invoice"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testsupport/pgtc"
)

// TestList_FiltersPaginationAndTenantScoping exercises Repository.List end to
// end (0% covered before this test — no mock ever hits the real SQL). Filters
// (status, original_invoice_id), pagination and tenant scoping are all built
// directly into the query string, so only a real DB proves they compose
// correctly.
func TestList_FiltersPaginationAndTenantScoping(t *testing.T) {
	appPool, _ := pgtc.StartPostgres(t)
	tenantA := uuid.New()
	tenantB := uuid.New()
	ctxA := pgtc.TenantCtx(context.Background(), tenantA)
	ctxB := pgtc.TenantCtx(context.Background(), tenantB)
	seedTenantExt(t, appPool, ctxA, tenantA)
	seedTenantExt(t, appPool, ctxB, tenantB)

	invRepo := invoice.NewPostgresRepository(appPool)
	origA := seedInvoice(t, ctxA, invRepo, tenantA)
	origA2 := seedInvoice(t, ctxA, invRepo, tenantA)
	origB := seedInvoice(t, ctxB, invRepo, tenantB)

	repo := creditnote.NewPostgresRepository(appPool)

	cnDraft := draftCreditNote(tenantA, origA, oneValidLine())
	require.NoError(t, repo.Create(ctxA, cnDraft))

	cnSent := draftCreditNote(tenantA, origA, oneValidLine())
	cnSent.Status = models.CreditNoteStatusSent
	cnSent.CreditNoteNumber = "GS-TEST-" + uuid.New().String()[:8]
	require.NoError(t, repo.Create(ctxA, cnSent))

	cnOther := draftCreditNote(tenantA, origA2, oneValidLine())
	require.NoError(t, repo.Create(ctxA, cnOther))

	// Tenant B has its own credit note — must never leak into tenant A's results.
	cnB := draftCreditNote(tenantB, origB, oneValidLine())
	require.NoError(t, repo.Create(ctxB, cnB))

	// Unfiltered list for tenant A: exactly its 3 rows.
	all, total, err := repo.List(ctxA, tenantA, creditnote.ListFilter{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	require.Len(t, all, 3)
	for _, cn := range all {
		assert.Equal(t, tenantA, cn.TenantID)
		assert.NotEqual(t, cnB.ID, cn.ID)
	}

	// Status filter.
	sentOnly, sentTotal, err := repo.List(ctxA, tenantA, creditnote.ListFilter{Status: models.CreditNoteStatusSent, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, sentTotal)
	require.Len(t, sentOnly, 1)
	assert.Equal(t, cnSent.ID, sentOnly[0].ID)

	// OriginalInvID filter.
	byInvoice, invTotal, err := repo.List(ctxA, tenantA, creditnote.ListFilter{OriginalInvID: &origA2, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, invTotal)
	require.Len(t, byInvoice, 1)
	assert.Equal(t, cnOther.ID, byInvoice[0].ID)

	// Pagination: page size 2 over 3 rows — total stays 3 on both pages, and
	// the two pages never repeat a row.
	page1, totalP1, err := repo.List(ctxA, tenantA, creditnote.ListFilter{Limit: 2, Offset: 0})
	require.NoError(t, err)
	assert.Equal(t, 3, totalP1)
	require.Len(t, page1, 2)

	page2, totalP2, err := repo.List(ctxA, tenantA, creditnote.ListFilter{Limit: 2, Offset: 2})
	require.NoError(t, err)
	assert.Equal(t, 3, totalP2)
	require.Len(t, page2, 1)
	assert.NotEqual(t, page1[0].ID, page2[0].ID)
	assert.NotEqual(t, page1[1].ID, page2[0].ID)

	// Cross-tenant: tenant B's own list contains only its own row.
	bList, bTotal, err := repo.List(ctxB, tenantB, creditnote.ListFilter{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, bTotal)
	require.Len(t, bList, 1)
	assert.Equal(t, cnB.ID, bList[0].ID)
}

// TestGetByInvoiceID_TenantScopedAndOrderedDescending exercises
// Repository.GetByInvoiceID (0% covered before this test). This is the exact
// query Service.StornoInvoice relies on to refuse over-crediting an invoice
// that already carries a sent credit note — a wrong WHERE or ORDER BY here is
// a silent double-storno risk, not just a listing bug.
func TestGetByInvoiceID_TenantScopedAndOrderedDescending(t *testing.T) {
	appPool, _ := pgtc.StartPostgres(t)
	tenantA := uuid.New()
	tenantB := uuid.New()
	ctxA := pgtc.TenantCtx(context.Background(), tenantA)
	ctxB := pgtc.TenantCtx(context.Background(), tenantB)
	seedTenantExt(t, appPool, ctxA, tenantA)
	seedTenantExt(t, appPool, ctxB, tenantB)

	invRepo := invoice.NewPostgresRepository(appPool)
	origA := seedInvoice(t, ctxA, invRepo, tenantA)
	origB := seedInvoice(t, ctxB, invRepo, tenantB)

	repo := creditnote.NewPostgresRepository(appPool)

	cn1 := draftCreditNote(tenantA, origA, oneValidLine())
	require.NoError(t, repo.Create(ctxA, cn1))

	cn2 := draftCreditNote(tenantA, origA, oneValidLine())
	cn2.CreatedAt = cn1.CreatedAt.Add(time.Second)
	cn2.UpdatedAt = cn2.CreatedAt
	require.NoError(t, repo.Create(ctxA, cn2))

	cnB := draftCreditNote(tenantB, origB, oneValidLine())
	require.NoError(t, repo.Create(ctxB, cnB))

	got, err := repo.GetByInvoiceID(ctxA, tenantA, origA)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, cn2.ID, got[0].ID, "newest credit note first (created_at DESC)")
	assert.Equal(t, cn1.ID, got[1].ID)

	// An invoice ID with no credit notes yields an empty slice, not an error.
	gotNone, err := repo.GetByInvoiceID(ctxA, tenantA, uuid.New())
	require.NoError(t, err)
	assert.Empty(t, gotNone)

	// Tenant B sees only its own credit note for its own invoice.
	gotB, err := repo.GetByInvoiceID(ctxB, tenantB, origB)
	require.NoError(t, err)
	require.Len(t, gotB, 1)
	assert.Equal(t, cnB.ID, gotB[0].ID)

	// Cross-tenant: tenant B's context against tenant A's invoice ID must not
	// see tenant A's credit notes — this is the guard StornoInvoice depends on.
	gotWrongTenant, err := repo.GetByInvoiceID(ctxB, tenantB, origA)
	require.NoError(t, err)
	assert.Empty(t, gotWrongTenant)
}

// TestListForDATEVExport_DateRangeStatusAndKeysetPaging exercises
// Repository.ListForDATEVExport (0% covered before this test): the sent-only
// filter, the inclusive-end-of-day date boundary, and keyset pagination by
// (created_at, id) that the DATEV export streams pages through.
func TestListForDATEVExport_DateRangeStatusAndKeysetPaging(t *testing.T) {
	appPool, _ := pgtc.StartPostgres(t)
	tenantID := uuid.New()
	ctx := pgtc.TenantCtx(context.Background(), tenantID)
	seedTenantExt(t, appPool, ctx, tenantID)

	invRepo := invoice.NewPostgresRepository(appPool)
	origInvID := seedInvoice(t, ctx, invRepo, tenantID)

	repo := creditnote.NewPostgresRepository(appPool)

	base := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)

	mkSent := func(offsetDays int) *models.CreditNote {
		cn := draftCreditNote(tenantID, origInvID, oneValidLine())
		cn.Status = models.CreditNoteStatusSent
		cn.CreditNoteNumber = "GS-TEST-" + uuid.New().String()[:8]
		cn.CreatedAt = base.AddDate(0, 0, offsetDays)
		cn.UpdatedAt = cn.CreatedAt
		return cn
	}

	inRange1 := mkSent(0) // 2026-03-10 12:00
	inRange2 := mkSent(1) // 2026-03-11 12:00
	inRange3 := mkSent(2) // 2026-03-12 12:00 — still inside an inclusive end-of-day boundary
	beforeRange := mkSent(-1)
	afterRange := mkSent(5)

	draftInRange := draftCreditNote(tenantID, origInvID, oneValidLine()) // status draft — excluded regardless of date
	draftInRange.CreatedAt = base
	draftInRange.UpdatedAt = base

	for _, cn := range []*models.CreditNote{inRange1, inRange2, inRange3, beforeRange, afterRange, draftInRange} {
		require.NoError(t, repo.Create(ctx, cn))
	}

	fromDate := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC)

	// Page 1: oldest first, limit 2.
	page1, err := repo.ListForDATEVExport(ctx, tenantID, fromDate, toDate, nil, nil, 2)
	require.NoError(t, err)
	require.Len(t, page1, 2)
	assert.Equal(t, inRange1.ID, page1[0].ID)
	assert.Equal(t, inRange2.ID, page1[1].ID)

	// Page 2 via the keyset cursor from the last row of page 1.
	lastCreatedAt := page1[1].CreatedAt
	lastID := page1[1].ID
	page2, err := repo.ListForDATEVExport(ctx, tenantID, fromDate, toDate, &lastCreatedAt, &lastID, 2)
	require.NoError(t, err)
	require.Len(t, page2, 1)
	assert.Equal(t, inRange3.ID, page2[0].ID)

	// Whole range in one call: draft and out-of-range rows never leak through.
	allInRange, err := repo.ListForDATEVExport(ctx, tenantID, fromDate, toDate, nil, nil, 100)
	require.NoError(t, err)
	gotIDs := make(map[uuid.UUID]bool)
	for _, cn := range allInRange {
		gotIDs[cn.ID] = true
		assert.Equal(t, models.CreditNoteStatusSent, cn.Status)
	}
	assert.True(t, gotIDs[inRange1.ID])
	assert.True(t, gotIDs[inRange2.ID])
	assert.True(t, gotIDs[inRange3.ID])
	assert.False(t, gotIDs[beforeRange.ID], "created_at before fromDate must be excluded")
	assert.False(t, gotIDs[afterRange.ID], "created_at after the inclusive toDate day must be excluded")
	assert.False(t, gotIDs[draftInRange.ID], "draft credit notes must never appear in the sent-only DATEV export")
}

// TestCreate_AmountNotValidatedAgainstInvoiceOpenBalance pins down current
// behavior rather than a requirement: Service.Create performs no check that a
// credit note's gross total stays within the original invoice's amount, so a
// credit note larger than the invoice it credits is accepted today. This is a
// documented gap (see JOURNAL.md), not a regression this test guards against —
// it exists so a future intentional over-credit guard shows up as an expected
// test change instead of a silent behavior shift nobody notices.
func TestCreate_AmountNotValidatedAgainstInvoiceOpenBalance(t *testing.T) {
	appPool, _ := pgtc.StartPostgres(t)
	tenantID := uuid.New()
	ctx := pgtc.TenantCtx(context.Background(), tenantID)
	seedTenantExt(t, appPool, ctx, tenantID)

	invRepo := invoice.NewPostgresRepository(appPool)
	// seedInvoice creates a 100/19/119 invoice (see send_atomic_integration_test.go).
	origInvID := seedInvoice(t, ctx, invRepo, tenantID)

	repo := creditnote.NewPostgresRepository(appPool)
	svc := creditnote.NewService(repo, invRepo, nil, nil)

	// Credit note gross (10 * 100 = 1000, plus tax) is far beyond the 119
	// gross of the invoice it credits.
	overLines := []models.LineItem{{
		ID: "l1", Position: 1, Description: "Ueberzogene Position",
		Quantity: decimal.NewFromInt(10), UnitPrice: decimal.NewFromInt(100),
		TaxRate: decimal.NewFromInt(19),
	}}

	cn, err := svc.Create(ctx, creditnote.CreateInput{
		TenantID: tenantID, OriginalInvoiceID: origInvID, LineItems: overLines,
		TaxMode: models.TaxModeStandard, Reason: "Testet fehlende Betragspruefung", UserID: uuid.New(),
	})

	require.NoError(t, err, "current behavior: Create does not reject an over-credit")
	assert.True(t, cn.GrossTotal.GreaterThan(decimal.NewFromInt(119)),
		"credit note gross total exceeds the original invoice's gross total, and is accepted anyway")
}

// TestCreate_DecimalAmountsSurviveDBRoundTripAsExactStrings proves amounts are
// computed and persisted via shopspring/decimal end to end and never pass
// through float64, which would corrupt exactly this class of value:
// 0.10 + 0.20 == 0.30000000000000004 in IEEE-754 float64, not 0.30.
func TestCreate_DecimalAmountsSurviveDBRoundTripAsExactStrings(t *testing.T) {
	appPool, superURL := pgtc.StartPostgres(t)
	superPool := pgtc.SuperPool(t, superURL)
	t.Cleanup(superPool.Close)

	tenantID := uuid.New()
	ctx := pgtc.TenantCtx(context.Background(), tenantID)
	seedTenantExt(t, appPool, ctx, tenantID)

	invRepo := invoice.NewPostgresRepository(appPool)
	origInvID := seedInvoice(t, ctx, invRepo, tenantID)

	repo := creditnote.NewPostgresRepository(appPool)
	svc := creditnote.NewService(repo, invRepo, nil, nil)

	lines := []models.LineItem{
		{ID: "l1", Position: 1, Description: "A", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.RequireFromString("0.10"), TaxRate: decimal.Zero},
		{ID: "l2", Position: 2, Description: "B", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.RequireFromString("0.20"), TaxRate: decimal.Zero},
	}

	cn, err := svc.Create(ctx, creditnote.CreateInput{
		TenantID: tenantID, OriginalInvoiceID: origInvID, LineItems: lines,
		TaxMode: models.TaxModeStandard, Reason: "Float-Falle", UserID: uuid.New(),
	})
	require.NoError(t, err)
	assert.True(t, decimal.RequireFromString("0.30").Equal(cn.Subtotal),
		"in-memory subtotal must be the exact decimal 0.30, got %s", cn.Subtotal)

	got, err := repo.GetByID(ctx, tenantID, cn.ID)
	require.NoError(t, err)
	assert.True(t, decimal.RequireFromString("0.30").Equal(got.Subtotal),
		"DB round-trip must preserve the exact decimal 0.30, got %s", got.Subtotal)

	// Read the raw NUMERIC column as text — bypasses Go-side decimal formatting
	// and proves the value stored on disk is exact, not a float artifact.
	var rawSubtotal string
	require.NoError(t, superPool.QueryRow(context.Background(),
		"SELECT subtotal::text FROM finance_credit_notes WHERE id = $1", cn.ID,
	).Scan(&rawSubtotal))
	assert.Equal(t, "0.30", rawSubtotal)
}
