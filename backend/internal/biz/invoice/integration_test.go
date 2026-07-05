//go:build integration

package invoice

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testsupport/pgtc"
)

// ============================================================================
// Helpers
// ============================================================================

// makeInvoice builds a minimal valid Invoice for the given tenant.
// LineItems must be set by the caller (inv.LineItems = marshalLines(...)).
func makeInvoice(tenantID uuid.UUID, lineItems []models.LineItem) *models.Invoice {
	raw, _ := json.Marshal(lineItems)
	now := time.Now().UTC().Truncate(time.Second)
	return &models.Invoice{
		ID:             uuid.New(),
		TenantID:       tenantID,
		InvoiceNumber:  fmt.Sprintf("RE-TEST-%s", uuid.New().String()[:8]),
		Status:         models.InvoiceStatusDraft,
		CustomerName:   "Test GmbH",
		CustomerAddress: "Musterstraße 1, 10115 Berlin",
		CustomerEmail:  "test@example.com",
		TaxMode:        models.TaxModeStandard,
		LineItems:      raw,
		Subtotal:       decimal.NewFromFloat(300.00),
		TotalTax:       decimal.NewFromFloat(57.00),
		GrossTotal:     decimal.NewFromFloat(357.00),
		InvoiceDate:    now,
		DueDate:        now.Add(30 * 24 * time.Hour),
		PaymentTerms:   "30 days",
		CreatedBy:      uuid.New(),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// twoLines returns two canonical test line items with non-trivial decimals.
func twoLines() []models.LineItem {
	return []models.LineItem{
		{
			ID:          "line-1",
			Position:    1,
			Description: "Beratungsleistung 2.5h",
			Quantity:    decimal.NewFromFloat(2.5),
			UnitPrice:   decimal.NewFromFloat(120.0),
			TaxRate:     decimal.NewFromFloat(19.0),
			LineTotal:   decimal.NewFromFloat(300.0),
		},
		{
			ID:          "line-2",
			Position:    2,
			Description: "Reisekosten pauschal",
			Quantity:    decimal.NewFromFloat(1.0),
			UnitPrice:   decimal.NewFromFloat(45.50),
			TaxRate:     decimal.NewFromFloat(7.0),
			LineTotal:   decimal.NewFromFloat(45.50),
		},
	}
}

// threeLines returns three replacement line items.
func threeLines() []models.LineItem {
	return []models.LineItem{
		{
			ID:          "new-1",
			Position:    1,
			Description: "Software-Lizenz",
			Quantity:    decimal.NewFromFloat(3.0),
			UnitPrice:   decimal.NewFromFloat(99.00),
			TaxRate:     decimal.NewFromFloat(19.0),
			LineTotal:   decimal.NewFromFloat(297.00),
		},
		{
			ID:          "new-2",
			Position:    2,
			Description: "Support-Stunden",
			Quantity:    decimal.NewFromFloat(5.0),
			UnitPrice:   decimal.NewFromFloat(80.00),
			TaxRate:     decimal.NewFromFloat(19.0),
			LineTotal:   decimal.NewFromFloat(400.00),
		},
		{
			ID:          "new-3",
			Position:    3,
			Description: "Dokumentation",
			Quantity:    decimal.NewFromFloat(1.0),
			UnitPrice:   decimal.NewFromFloat(50.00),
			TaxRate:     decimal.NewFromFloat(0.0),
			LineTotal:   decimal.NewFromFloat(50.00),
		},
	}
}

// countLineRows returns how many rows finance_invoice_lines has for invoiceID
// using a direct superuser query (bypasses RLS).
func countLineRows(t *testing.T, superPool *pgxpool.Pool, invoiceID uuid.UUID) int {
	t.Helper()
	var n int
	err := superPool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM finance_invoice_lines WHERE invoice_id = $1",
		invoiceID,
	).Scan(&n)
	if err != nil {
		t.Fatalf("count line rows: %v", err)
	}
	return n
}

// ============================================================================
// Tests
// ============================================================================

// TestUpsertImportedRoundtrip verifies the read-only Bexio import path against real
// Postgres: insert with provenance, read-back, and an idempotent update on re-pull
// (same external_id → one row, replaced lines, no RE-number sequence involved).
func TestUpsertImportedRoundtrip(t *testing.T) {
	appPool, superURL := pgtc.StartPostgres(t)
	superPool := pgtc.SuperPool(t, superURL)
	t.Cleanup(superPool.Close)
	tenantID := uuid.New()
	ctx := pgtc.TenantCtx(context.Background(), tenantID)
	seedTenant(t, appPool, ctx, tenantID)

	repo := NewPostgresRepository(appPool)
	const extID, extNum = "bx-4711", "2026-042"

	build := func(gross float64, lines []models.LineItem) *models.Invoice {
		raw, _ := json.Marshal(lines)
		now := time.Now().UTC().Truncate(time.Second)
		eid, enum := extID, extNum
		return &models.Invoice{
			ID: uuid.New(), TenantID: tenantID, InvoiceNumber: extNum,
			Status: models.InvoiceStatusSent, Source: models.InvoiceSourceBexio,
			ExternalID: &eid, ExternalNumber: &enum, CustomerName: "Bexio Kunde AG",
			TaxMode: models.TaxModeStandard, LineItems: raw,
			Subtotal: decimal.NewFromFloat(gross), TotalTax: decimal.NewFromFloat(0),
			GrossTotal: decimal.NewFromFloat(gross), Currency: "CHF",
			InvoiceDate: now, DueDate: now.Add(30 * 24 * time.Hour),
			CreatedBy: uuid.Nil, CreatedAt: now, UpdatedAt: now,
		}
	}

	// First pull → insert; id is preserved.
	inv1 := build(100.0, twoLines())
	firstID := inv1.ID
	if err := repo.UpsertImported(ctx, inv1); err != nil {
		t.Fatalf("UpsertImported (insert): %v", err)
	}
	if inv1.ID != firstID {
		t.Fatalf("insert must keep id: got %s want %s", inv1.ID, firstID)
	}

	got, err := repo.GetByID(ctx, tenantID, firstID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Source != models.InvoiceSourceBexio {
		t.Fatalf("source: got %q want bexio", got.Source)
	}
	if got.ExternalID == nil || *got.ExternalID != extID || got.ExternalNumber == nil || *got.ExternalNumber != extNum {
		t.Fatalf("provenance mismatch: id=%v num=%v", got.ExternalID, got.ExternalNumber)
	}
	if n := countLineRows(t, superPool, firstID); n != 2 {
		t.Fatalf("line rows after insert: got %d want 2", n)
	}

	// Second pull (same external_id) → idempotent update: one row, replaced lines,
	// canonical id resolved.
	inv2 := build(150.0, threeLines())
	if err := repo.UpsertImported(ctx, inv2); err != nil {
		t.Fatalf("UpsertImported (update): %v", err)
	}
	if inv2.ID != firstID {
		t.Fatalf("update must resolve to canonical id: got %s want %s", inv2.ID, firstID)
	}

	var rowCount int
	if err := superPool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM finance_invoices WHERE tenant_id = $1 AND source = 'bexio' AND external_id = $2",
		tenantID, extID,
	).Scan(&rowCount); err != nil {
		t.Fatalf("count imported rows: %v", err)
	}
	if rowCount != 1 {
		t.Fatalf("expected 1 imported row after re-pull, got %d", rowCount)
	}

	got2, err := repo.GetByID(ctx, tenantID, firstID)
	if err != nil {
		t.Fatalf("GetByID after update: %v", err)
	}
	if !got2.GrossTotal.Equal(decimal.NewFromFloat(150.0)) {
		t.Fatalf("gross_total not updated: got %s want 150", got2.GrossTotal)
	}
	if n := countLineRows(t, superPool, firstID); n != 3 {
		t.Fatalf("line rows after update: got %d want 3 (replaced)", n)
	}
}

// TestInvoiceLinesRelationalRoundtrip creates an invoice with 2 line items,
// reads it back, and asserts position ordering and exact decimal equality.
func TestInvoiceLinesRelationalRoundtrip(t *testing.T) {
	appPool, _ := pgtc.StartPostgres(t)
	tenantID := uuid.New()
	ctx := pgtc.TenantCtx(context.Background(), tenantID)

	// Seed minimal tenant row (required by FK on tenant_id if present).
	seedTenant(t, appPool, ctx, tenantID)

	repo := NewPostgresRepository(appPool)
	lines := twoLines()
	inv := makeInvoice(tenantID, lines)

	if err := repo.Create(ctx, inv); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(ctx, tenantID, inv.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	var gotLines []models.LineItem
	if err := json.Unmarshal(got.LineItems, &gotLines); err != nil {
		t.Fatalf("unmarshal line items: %v", err)
	}

	if len(gotLines) != 2 {
		t.Fatalf("line count: got %d, want 2", len(gotLines))
	}

	// Assert ordering by position
	if gotLines[0].Position != 1 {
		t.Errorf("line 0 position: got %d, want 1", gotLines[0].Position)
	}
	if gotLines[1].Position != 2 {
		t.Errorf("line 1 position: got %d, want 2", gotLines[1].Position)
	}

	// Exact decimal equality
	if !gotLines[0].Quantity.Equal(lines[0].Quantity) {
		t.Errorf("line 0 quantity: got %s, want %s", gotLines[0].Quantity, lines[0].Quantity)
	}
	if !gotLines[0].UnitPrice.Equal(lines[0].UnitPrice) {
		t.Errorf("line 0 unit_price: got %s, want %s", gotLines[0].UnitPrice, lines[0].UnitPrice)
	}
	if !gotLines[0].TaxRate.Equal(lines[0].TaxRate) {
		t.Errorf("line 0 tax_rate: got %s, want %s", gotLines[0].TaxRate, lines[0].TaxRate)
	}
	if !gotLines[1].UnitPrice.Equal(lines[1].UnitPrice) {
		t.Errorf("line 1 unit_price: got %s, want %s", gotLines[1].UnitPrice, lines[1].UnitPrice)
	}
}

// TestInvoiceUpdateReplacesLines creates an invoice with 2 lines, updates it
// with 3 different lines, and verifies exactly 3 rows in the DB (no orphan rows).
func TestInvoiceUpdateReplacesLines(t *testing.T) {
	appPool, superURL := pgtc.StartPostgres(t)
	superPool := pgtc.SuperPool(t, superURL)
	t.Cleanup(superPool.Close)

	tenantID := uuid.New()
	ctx := pgtc.TenantCtx(context.Background(), tenantID)
	seedTenant(t, appPool, ctx, tenantID)

	repo := NewPostgresRepository(appPool)

	inv := makeInvoice(tenantID, twoLines())
	if err := repo.Create(ctx, inv); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if n := countLineRows(t, superPool, inv.ID); n != 2 {
		t.Fatalf("after Create: got %d line rows, want 2", n)
	}

	// Update with 3 new lines.
	inv.LineItems = marshalLinesOrFatal(t, threeLines())
	inv.UpdatedAt = time.Now().UTC()
	if err := repo.Update(ctx, inv); err != nil {
		t.Fatalf("Update: %v", err)
	}

	// Assert via direct DB query — superPool bypasses RLS.
	if n := countLineRows(t, superPool, inv.ID); n != 3 {
		t.Fatalf("after Update: got %d line rows in DB, want exactly 3 (no orphan rows)", n)
	}

	// Assert via GetByID as well.
	got, err := repo.GetByID(ctx, tenantID, inv.ID)
	if err != nil {
		t.Fatalf("GetByID after update: %v", err)
	}
	var gotLines []models.LineItem
	if err := json.Unmarshal(got.LineItems, &gotLines); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(gotLines) != 3 {
		t.Fatalf("GetByID line count: got %d, want 3", len(gotLines))
	}
}

// TestInvoiceConstraintRejection verifies that lines violating DB CHECK constraints
// are rejected (quantity=0 violates chk_invoice_lines_quantity > 0;
// tax_rate=150 violates chk_invoice_lines_tax_rate <= 100).
func TestInvoiceConstraintRejection(t *testing.T) {
	appPool, _ := pgtc.StartPostgres(t)
	tenantID := uuid.New()
	ctx := pgtc.TenantCtx(context.Background(), tenantID)
	seedTenant(t, appPool, ctx, tenantID)
	repo := NewPostgresRepository(appPool)

	t.Run("quantity_zero", func(t *testing.T) {
		lines := []models.LineItem{{
			Position:    1,
			Description: "Bad line",
			Quantity:    decimal.NewFromFloat(0), // violates quantity > 0
			UnitPrice:   decimal.NewFromFloat(100.0),
			TaxRate:     decimal.NewFromFloat(19.0),
			LineTotal:   decimal.NewFromFloat(0),
		}}
		inv := makeInvoice(tenantID, lines)
		if err := repo.Create(ctx, inv); err == nil {
			t.Error("expected error for quantity=0, got nil")
		}
	})

	t.Run("tax_rate_over_100", func(t *testing.T) {
		lines := []models.LineItem{{
			Position:    1,
			Description: "Bad tax line",
			Quantity:    decimal.NewFromFloat(1.0),
			UnitPrice:   decimal.NewFromFloat(100.0),
			TaxRate:     decimal.NewFromFloat(150.0), // violates tax_rate <= 100
			LineTotal:   decimal.NewFromFloat(250.0),
		}}
		inv := makeInvoice(tenantID, lines)
		if err := repo.Create(ctx, inv); err == nil {
			t.Error("expected error for tax_rate=150, got nil")
		}
	})
}

// TestBackfillIdempotency verifies the relational line-item flow end-to-end:
// insert an invoice via the repo (which writes to finance_invoice_lines), then
// read it back via GetByID and assert the lines are present. This replaces the
// historical JSONB backfill test (Migration 000133) which is no longer relevant
// after Migration 000217 drops the line_items JSONB column.
func TestBackfillIdempotency(t *testing.T) {
	appPool, superURL := pgtc.StartPostgres(t)
	superPool := pgtc.SuperPool(t, superURL)
	t.Cleanup(superPool.Close)

	tenantID := uuid.New()
	ctx := pgtc.TenantCtx(context.Background(), tenantID)
	seedTenant(t, appPool, ctx, tenantID)

	repo := NewPostgresRepository(appPool)
	lines := twoLines()
	inv := makeInvoice(tenantID, lines)

	if err := repo.Create(ctx, inv); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Assert 2 rows in finance_invoice_lines (relational).
	if n := countLineRows(t, superPool, inv.ID); n != 2 {
		t.Fatalf("after Create: got %d line rows, want 2", n)
	}

	// Read back via GetByID — lines must be loaded from the relational table.
	got, err := repo.GetByID(ctx, tenantID, inv.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	var gotLines []models.LineItem
	if err := json.Unmarshal(got.LineItems, &gotLines); err != nil {
		t.Fatalf("unmarshal line items: %v", err)
	}
	if len(gotLines) != 2 {
		t.Fatalf("line count via GetByID: got %d, want 2", len(gotLines))
	}
}

// TestTenantIsolationInvoiceLines creates invoices+lines for two tenants and
// verifies that tenant A cannot see tenant B's data (RLS enforced on appPool
// which connects as kmuhub_app NOBYPASSRLS).
func TestTenantIsolationInvoiceLines(t *testing.T) {
	appPool, _ := pgtc.StartPostgres(t)
	tenantA := uuid.New()
	tenantB := uuid.New()
	ctxA := pgtc.TenantCtx(context.Background(), tenantA)
	ctxB := pgtc.TenantCtx(context.Background(), tenantB)
	seedTenant(t, appPool, ctxA, tenantA)
	seedTenant(t, appPool, ctxB, tenantB)

	repo := NewPostgresRepository(appPool)

	invA := makeInvoice(tenantA, twoLines())
	if err := repo.Create(ctxA, invA); err != nil {
		t.Fatalf("Create A: %v", err)
	}
	invB := makeInvoice(tenantB, threeLines())
	if err := repo.Create(ctxB, invB); err != nil {
		t.Fatalf("Create B: %v", err)
	}

	// Tenant A can see its own invoice.
	gotA, err := repo.GetByID(ctxA, tenantA, invA.ID)
	if err != nil {
		t.Fatalf("GetByID A with ctxA: %v", err)
	}
	var aLines []models.LineItem
	if err := json.Unmarshal(gotA.LineItems, &aLines); err != nil {
		t.Fatalf("unmarshal A lines: %v", err)
	}
	if len(aLines) != 2 {
		t.Errorf("tenant A line count: got %d, want 2", len(aLines))
	}

	// Tenant A cannot see Tenant B's invoice.
	_, err = repo.GetByID(ctxA, tenantB, invB.ID)
	if err == nil {
		t.Error("expected error when tenant A reads tenant B's invoice (RLS), got nil")
	}

	// List with tenant A context only returns A's invoices.
	list, total, err := repo.List(ctxA, tenantA, ListFilter{Limit: 20})
	if err != nil {
		t.Fatalf("List A: %v", err)
	}
	for _, inv := range list {
		if inv.TenantID != tenantA {
			t.Errorf("List leaked cross-tenant invoice: got tenant_id %s", inv.TenantID)
		}
	}
	if total < 1 {
		t.Error("List for tenant A returned 0 results")
	}
}

// TestInvoiceLockColumns creates an invoice, sets the GoBD lock, and verifies
// LockedAt and LockedBy are persisted and returned by GetByID.
func TestInvoiceLockColumns(t *testing.T) {
	appPool, _ := pgtc.StartPostgres(t)
	tenantID := uuid.New()
	ctx := pgtc.TenantCtx(context.Background(), tenantID)
	seedTenant(t, appPool, ctx, tenantID)

	repo := NewPostgresRepository(appPool)
	inv := makeInvoice(tenantID, twoLines())
	if err := repo.Create(ctx, inv); err != nil {
		t.Fatalf("Create: %v", err)
	}

	lockedAt := time.Now().UTC().Truncate(time.Second)
	lockedBy := uuid.New()
	if err := repo.SetLock(ctx, tenantID, inv.ID, lockedAt, lockedBy); err != nil {
		t.Fatalf("SetLock: %v", err)
	}

	got, err := repo.GetByID(ctx, tenantID, inv.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	if got.LockedAt == nil {
		t.Fatal("LockedAt is nil, expected non-nil")
	}
	if !got.LockedAt.Equal(lockedAt) {
		t.Errorf("LockedAt: got %v, want %v", got.LockedAt, lockedAt)
	}
	if got.LockedBy == nil {
		t.Fatal("LockedBy is nil, expected non-nil")
	}
	if *got.LockedBy != lockedBy {
		t.Errorf("LockedBy: got %s, want %s", *got.LockedBy, lockedBy)
	}
}

// TestGetOverdueReturnsOnlySentPastDue verifies GetOverdue returns only sent
// invoices whose due_date is in the past.
func TestGetOverdueReturnsOnlySentPastDue(t *testing.T) {
	appPool, _ := pgtc.StartPostgres(t)
	tenantID := uuid.New()
	ctx := pgtc.TenantCtx(context.Background(), tenantID)
	seedTenant(t, appPool, ctx, tenantID)

	repo := NewPostgresRepository(appPool)
	now := time.Now().UTC()

	// Overdue: sent, due yesterday.
	overdue := makeInvoice(tenantID, twoLines())
	overdue.Status = models.InvoiceStatusSent
	overdue.DueDate = now.Add(-48 * time.Hour)
	if err := repo.Create(ctx, overdue); err != nil {
		t.Fatalf("Create overdue: %v", err)
	}

	// Not overdue: sent, due tomorrow.
	notOverdue := makeInvoice(tenantID, twoLines())
	notOverdue.Status = models.InvoiceStatusSent
	notOverdue.DueDate = now.Add(48 * time.Hour)
	if err := repo.Create(ctx, notOverdue); err != nil {
		t.Fatalf("Create notOverdue: %v", err)
	}

	// Draft: should never appear in overdue list.
	draft := makeInvoice(tenantID, twoLines())
	if err := repo.Create(ctx, draft); err != nil {
		t.Fatalf("Create draft: %v", err)
	}

	results, err := repo.GetOverdue(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetOverdue: %v", err)
	}

	for _, inv := range results {
		if inv.ID == notOverdue.ID {
			t.Error("GetOverdue returned not-yet-due invoice")
		}
		if inv.ID == draft.ID {
			t.Error("GetOverdue returned draft invoice")
		}
		if inv.Status != models.InvoiceStatusSent {
			t.Errorf("GetOverdue returned invoice with status %q, want sent", inv.Status)
		}
	}

	found := false
	for _, inv := range results {
		if inv.ID == overdue.ID {
			found = true
		}
	}
	if !found {
		t.Error("GetOverdue did not return the actually-overdue invoice")
	}
}

// ============================================================================
// Internal helpers (test-only)
// ============================================================================

// marshalLinesOrFatal marshals line items to JSON, failing the test on error.
func marshalLinesOrFatal(t *testing.T, lines []models.LineItem) []byte {
	t.Helper()
	raw, err := json.Marshal(lines)
	if err != nil {
		t.Fatalf("marshal lines: %v", err)
	}
	return raw
}

// seedTenant inserts a minimal row into the tenants table so FK constraints are
// satisfied. Uses the app pool with the correct tenant context — the tenants
// table is either system-level or the INSERT will go through RLS.
// We use a direct superuser-style exec via the app pool with a system context
// workaround: since tenants has RLS, we insert through the app pool with the
// correct tenant_id set in context (which is the row we're inserting).
func seedTenant(t *testing.T, pool *pgxpool.Pool, ctx context.Context, tenantID uuid.UUID) {
	t.Helper()
	_, err := pool.Exec(ctx,
		`INSERT INTO tenants (id, name, created_at, updated_at)
		 VALUES ($1, $2, NOW(), NOW())
		 ON CONFLICT (id) DO NOTHING`,
		tenantID, fmt.Sprintf("Test Tenant %s", tenantID.String()[:8]),
	)
	if err != nil {
		t.Logf("seedTenant (non-fatal, may not be needed): %v", err)
	}
}
