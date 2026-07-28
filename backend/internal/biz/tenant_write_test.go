package biz

// internal/biz is the largest write surface in the repo (44 INSERTs), but its
// three existing DB tests (tenant_isolation_phase2_test.go, ..._open_items_test.go,
// ..._recurring_test.go) all seed exclusively through testutil.SeedRow (a raw
// INSERT under system context) and never call the real repositories -- exactly
// the shape of gap Nacht 1 found in internal/auth (invitations shipped without a
// tenant_id column entirely, migration 000249). This file closes that gap for
// the finance-core write surface this unit was scoped to: invoices, quotes,
// credit notes, payments, dunning records. HR sits in wp-biz-hr; the
// bexio/lexware/datev/gobdarchive/banking packages have their own units.
//
// No dead write found: every Create/Update/Delete below already carries the
// right tenant_id, and every explicit tenant_id parameter is still backstopped
// by RLS. Each write is exercised once from a foreign-tenant ctx, passing the
// row's *real* tenantID as the explicit parameter -- so only RLS, not the WHERE
// clause, can be stopping it -- before repeating the same call from the owning
// ctx and confirming it lands.

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/biz/creditnote"
	"github.com/kmuhub/kmuhub/internal/biz/dunning"
	"github.com/kmuhub/kmuhub/internal/biz/invoice"
	"github.com/kmuhub/kmuhub/internal/biz/payment"
	"github.com/kmuhub/kmuhub/internal/biz/quote"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func testFinanceLineItems() []models.LineItem {
	return []models.LineItem{{
		ID: "1", Position: 1, Description: "Write Test Item",
		Quantity: decimal.NewFromInt(2), UnitPrice: decimal.NewFromInt(100),
		TaxRate: decimal.NewFromInt(19), LineTotal: decimal.NewFromInt(238),
	}}
}

func TestInvoiceWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Biz Invoice Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Biz Invoice Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	repo := invoice.NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	lineItemsJSON, err := json.Marshal(testFinanceLineItems())
	if err != nil {
		t.Fatalf("marshal line items: %v", err)
	}

	inv := &models.Invoice{
		ID:           uuid.New(),
		TenantID:     tenantOwn,
		Status:       models.InvoiceStatusDraft,
		CustomerName: "Write Test Customer",
		TaxMode:      models.TaxModeStandard,
		LineItems:    lineItemsJSON,
		Subtotal:     decimal.NewFromInt(200),
		TotalTax:     decimal.NewFromInt(38),
		GrossTotal:   decimal.NewFromInt(238),
		Currency:     models.DefaultCurrency,
		InvoiceDate:  now,
		DueDate:      now.Add(30 * 24 * time.Hour),
		CreatedBy:    uuid.New(),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := repo.Create(ctxOwn, inv); err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "finance_invoices", inv.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "finance_invoices", inv.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "finance_invoices", inv.ID, 0)

	// Update/UpdateStatus/SetLock all carry an explicit tenant_id predicate, but
	// pass the row's real tenantID from a foreign ctx so only RLS -- not the
	// WHERE clause -- can be stopping the write. Update replaces line items
	// inside a transaction (DELETE then re-INSERT); the header UPDATE alone
	// would silently affect 0 rows under RLS, but the re-INSERT's WITH CHECK
	// rejects a line tagged with a foreign tenant_id outright, so the whole
	// transaction errors and rolls back -- there is no partial state to check.
	foreign := *inv
	foreign.CustomerName = "Hacked"
	foreign.UpdatedAt = time.Now().UTC()
	if err := repo.Update(ctxOther, &foreign); err == nil {
		t.Fatalf("Update (foreign ctx): expected an RLS error, got nil")
	}
	if err := repo.UpdateStatus(ctxOther, tenantOwn, inv.ID, models.InvoiceStatusSent); err != nil {
		t.Fatalf("UpdateStatus (foreign ctx): %v", err)
	}
	lockedAt := time.Now().UTC()
	if err := repo.SetLock(ctxOther, tenantOwn, inv.ID, lockedAt, uuid.New()); err != nil {
		t.Fatalf("SetLock (foreign ctx): %v", err)
	}
	got, err := repo.GetByID(sysCtx, tenantOwn, inv.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.CustomerName != "Write Test Customer" || got.Status != models.InvoiceStatusDraft || got.LockedAt != nil {
		t.Fatalf("a foreign-tenant write reached the invoice: customer_name=%q status=%q locked_at=%v",
			got.CustomerName, got.Status, got.LockedAt)
	}

	// The owning tenant's own writes do land.
	foreign.CustomerName = "Renamed"
	if err := repo.Update(ctxOwn, &foreign); err != nil {
		t.Fatalf("Update (own ctx): %v", err)
	}
	if err := repo.UpdateStatus(ctxOwn, tenantOwn, inv.ID, models.InvoiceStatusSent); err != nil {
		t.Fatalf("UpdateStatus (own ctx): %v", err)
	}
	if err := repo.SetLock(ctxOwn, tenantOwn, inv.ID, lockedAt, uuid.New()); err != nil {
		t.Fatalf("SetLock (own ctx): %v", err)
	}
	got, err = repo.GetByID(ctxOwn, tenantOwn, inv.ID)
	if err != nil {
		t.Fatalf("GetByID (own ctx): %v", err)
	}
	if got.CustomerName != "Renamed" || got.Status != models.InvoiceStatusSent || got.LockedAt == nil {
		t.Fatalf("own-tenant write did not land: customer_name=%q status=%q locked_at=%v",
			got.CustomerName, got.Status, got.LockedAt)
	}
}

func TestQuoteWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Biz Quote Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Biz Quote Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	repo := quote.NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	lineItemsJSON, err := json.Marshal(testFinanceLineItems())
	if err != nil {
		t.Fatalf("marshal line items: %v", err)
	}

	q := &models.Quote{
		ID:           uuid.New(),
		TenantID:     tenantOwn,
		Status:       models.QuoteStatusDraft,
		CustomerName: "Write Test Quote Customer",
		TaxMode:      models.TaxModeStandard,
		LineItems:    lineItemsJSON,
		Subtotal:     decimal.NewFromInt(200),
		TotalTax:     decimal.NewFromInt(38),
		GrossTotal:   decimal.NewFromInt(238),
		Currency:     models.DefaultCurrency,
		CreatedBy:    uuid.New(),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := repo.Create(ctxOwn, q); err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "finance_quotes", q.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "finance_quotes", q.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "finance_quotes", q.ID, 0)

	// Update replaces line items inside a transaction; a foreign-ctx call fails
	// outright (RLS rejects the re-INSERT of a foreign-tenant line) rather than
	// silently affecting 0 rows -- see the invoice test above for the same shape.
	foreign := *q
	foreign.CustomerName = "Hacked"
	foreign.UpdatedAt = time.Now().UTC()
	if err := repo.Update(ctxOther, &foreign); err == nil {
		t.Fatalf("Update (foreign ctx): expected an RLS error, got nil")
	}
	if err := repo.UpdateStatus(ctxOther, tenantOwn, q.ID, models.QuoteStatusSent); err != nil {
		t.Fatalf("UpdateStatus (foreign ctx): %v", err)
	}
	got, err := repo.GetByID(sysCtx, tenantOwn, q.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.CustomerName != "Write Test Quote Customer" || got.Status != models.QuoteStatusDraft {
		t.Fatalf("a foreign-tenant write reached the quote: customer_name=%q status=%q", got.CustomerName, got.Status)
	}

	foreign.CustomerName = "Renamed"
	if err := repo.Update(ctxOwn, &foreign); err != nil {
		t.Fatalf("Update (own ctx): %v", err)
	}
	if err := repo.UpdateStatus(ctxOwn, tenantOwn, q.ID, models.QuoteStatusSent); err != nil {
		t.Fatalf("UpdateStatus (own ctx): %v", err)
	}
	got, err = repo.GetByID(ctxOwn, tenantOwn, q.ID)
	if err != nil {
		t.Fatalf("GetByID (own ctx): %v", err)
	}
	if got.CustomerName != "Renamed" || got.Status != models.QuoteStatusSent {
		t.Fatalf("own-tenant write did not land: customer_name=%q status=%q", got.CustomerName, got.Status)
	}

	// Delete: foreign ctx must not remove the row, own ctx must.
	if err := repo.Delete(ctxOther, tenantOwn, q.ID); err != nil {
		t.Fatalf("Delete (foreign ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "finance_quotes", q.ID, 1)
	if err := repo.Delete(ctxOwn, tenantOwn, q.ID); err != nil {
		t.Fatalf("Delete (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "finance_quotes", q.ID, 0)
}

func TestCreditNotePaymentDunningWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Biz CN/Payment/Dunning Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Biz CN/Payment/Dunning Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	lineItemsJSON, err := json.Marshal(testFinanceLineItems())
	if err != nil {
		t.Fatalf("marshal line items: %v", err)
	}

	// Parent invoice via the real repository (also exercises invoice.Create a
	// second time under a different fixture, cheap given it's a prerequisite here).
	invRepo := invoice.NewPostgresRepository(pool)
	inv := &models.Invoice{
		ID:           uuid.New(),
		TenantID:     tenantOwn,
		Status:       models.InvoiceStatusSent,
		CustomerName: "CN/Payment/Dunning Parent Invoice",
		TaxMode:      models.TaxModeStandard,
		LineItems:    lineItemsJSON,
		Subtotal:     decimal.NewFromInt(200),
		TotalTax:     decimal.NewFromInt(38),
		GrossTotal:   decimal.NewFromInt(238),
		Currency:     models.DefaultCurrency,
		InvoiceDate:  now,
		DueDate:      now.Add(30 * 24 * time.Hour),
		CreatedBy:    uuid.New(),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := invRepo.Create(ctxOwn, inv); err != nil {
		t.Fatalf("Create parent invoice: %v", err)
	}
	// finance_credit_notes/finance_payments/finance_dunning_records all FK to
	// finance_invoices ON DELETE RESTRICT -- their cleanups must run (LIFO) before
	// this one, so this defer is registered first.
	defer testutil.CleanupRow(t, pool, "finance_invoices", inv.ID)

	// --- Credit note ---
	cnRepo := creditnote.NewPostgresRepository(pool)
	cn := &models.CreditNote{
		ID:                uuid.New(),
		TenantID:          tenantOwn,
		OriginalInvoiceID: inv.ID,
		Status:            models.CreditNoteStatusDraft,
		CustomerName:      "CN Write Test Customer",
		TaxMode:           models.TaxModeStandard,
		LineItems:         lineItemsJSON,
		Subtotal:          decimal.NewFromInt(200),
		TotalTax:          decimal.NewFromInt(38),
		GrossTotal:        decimal.NewFromInt(238),
		Currency:          models.DefaultCurrency,
		Reason:            "Write test",
		CreatedBy:         uuid.New(),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := cnRepo.Create(ctxOwn, cn); err != nil {
		t.Fatalf("CreditNote Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "finance_credit_notes", cn.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "finance_credit_notes", cn.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "finance_credit_notes", cn.ID, 0)

	// Update replaces line items inside a transaction; a foreign-ctx call fails
	// outright (RLS rejects the re-INSERT of a foreign-tenant line) rather than
	// silently affecting 0 rows -- see the invoice test above for the same shape.
	foreignCN := *cn
	foreignCN.CustomerName = "Hacked"
	foreignCN.UpdatedAt = time.Now().UTC()
	if err := cnRepo.Update(ctxOther, &foreignCN); err == nil {
		t.Fatalf("CreditNote Update (foreign ctx): expected an RLS error, got nil")
	}
	gotCN, err := cnRepo.GetByID(sysCtx, tenantOwn, cn.ID)
	if err != nil {
		t.Fatalf("CreditNote GetByID: %v", err)
	}
	if gotCN.CustomerName != "CN Write Test Customer" {
		t.Fatalf("a foreign-tenant write reached the credit note: customer_name=%q", gotCN.CustomerName)
	}
	foreignCN.CustomerName = "Renamed"
	if err := cnRepo.Update(ctxOwn, &foreignCN); err != nil {
		t.Fatalf("CreditNote Update (own ctx): %v", err)
	}
	gotCN, err = cnRepo.GetByID(ctxOwn, tenantOwn, cn.ID)
	if err != nil {
		t.Fatalf("CreditNote GetByID (own ctx): %v", err)
	}
	if gotCN.CustomerName != "Renamed" {
		t.Fatalf("own-tenant credit note write did not land: customer_name=%q", gotCN.CustomerName)
	}

	// --- Payment ---
	payRepo := payment.NewPostgresRepository(pool)
	pay := &models.Payment{
		ID:          uuid.New(),
		TenantID:    tenantOwn,
		InvoiceID:   inv.ID,
		Amount:      decimal.NewFromInt(50),
		PaymentDate: now,
		Method:      "bank_transfer",
		CreatedBy:   uuid.New(),
		CreatedAt:   now,
	}
	if err := payRepo.Create(ctxOwn, pay); err != nil {
		t.Fatalf("Payment Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "finance_payments", pay.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "finance_payments", pay.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "finance_payments", pay.ID, 0)

	if err := payRepo.Delete(ctxOther, tenantOwn, pay.ID); err != nil {
		t.Fatalf("Payment Delete (foreign ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "finance_payments", pay.ID, 1)
	if err := payRepo.Delete(ctxOwn, tenantOwn, pay.ID); err != nil {
		t.Fatalf("Payment Delete (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "finance_payments", pay.ID, 0)

	// --- Dunning ---
	dunRepo := dunning.NewPostgresRepository(pool)
	dun := &models.DunningRecord{
		ID:        uuid.New(),
		TenantID:  tenantOwn,
		InvoiceID: inv.ID,
		Level:     1,
		Status:    models.DunningStatusDraft,
		Fee:       decimal.NewFromInt(5),
		Interest:  decimal.Zero,
		CreatedBy: uuid.New(),
		CreatedAt: now,
	}
	if err := dunRepo.Create(ctxOwn, dun); err != nil {
		t.Fatalf("Dunning Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "finance_dunning_records", dun.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "finance_dunning_records", dun.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "finance_dunning_records", dun.ID, 0)

	sentAt := time.Now().UTC()
	if err := dunRepo.UpdateStatus(ctxOther, tenantOwn, dun.ID, models.DunningStatusSent, &sentAt); err != nil {
		t.Fatalf("Dunning UpdateStatus (foreign ctx): %v", err)
	}
	gotDun, err := dunRepo.GetByID(sysCtx, tenantOwn, dun.ID)
	if err != nil {
		t.Fatalf("Dunning GetByID: %v", err)
	}
	if gotDun.Status != models.DunningStatusDraft || gotDun.SentAt != nil {
		t.Fatalf("a foreign-tenant write reached the dunning record: status=%q sent_at=%v", gotDun.Status, gotDun.SentAt)
	}
	if err := dunRepo.UpdateStatus(ctxOwn, tenantOwn, dun.ID, models.DunningStatusSent, &sentAt); err != nil {
		t.Fatalf("Dunning UpdateStatus (own ctx): %v", err)
	}
	gotDun, err = dunRepo.GetByID(ctxOwn, tenantOwn, dun.ID)
	if err != nil {
		t.Fatalf("Dunning GetByID (own ctx): %v", err)
	}
	if gotDun.Status != models.DunningStatusSent || gotDun.SentAt == nil {
		t.Fatalf("own-tenant dunning write did not land: status=%q sent_at=%v", gotDun.Status, gotDun.SentAt)
	}
}
