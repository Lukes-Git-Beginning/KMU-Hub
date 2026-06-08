//go:build integration

package creditnote

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

// insertRawInvoice inserts a minimal finance_invoices row via the app pool
// (using the tenant context) so that CreditNote.OriginalInvoiceID FK is satisfied.
func insertRawInvoice(t *testing.T, pool *pgxpool.Pool, ctx context.Context, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	invID := uuid.New()
	now := time.Now().UTC()
	_, err := pool.Exec(ctx,
		`INSERT INTO finance_invoices (
			id, tenant_id, invoice_number, status,
			customer_name, customer_address, customer_email, customer_ust_id_nr,
			tax_mode, line_items,
			subtotal, total_tax, gross_total,
			invoice_date, due_date, payment_terms,
			created_by, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'[]'::jsonb,$10,$11,$12,$13,$14,$15,$16,$17,$18)`,
		invID, tenantID,
		fmt.Sprintf("RE-CN-%s", invID.String()[:8]), "sent",
		"CN Kunde GmbH", "Addr", "cn@example.com", "",
		"standard",
		"100", "19", "119",
		now, now.Add(30*24*time.Hour), "30 days",
		uuid.New(), now, now,
	)
	if err != nil {
		t.Fatalf("insertRawInvoice: %v", err)
	}
	return invID
}

func makeCreditNote(tenantID, originalInvoiceID uuid.UUID, lineItems []models.LineItem) *models.CreditNote {
	raw, _ := json.Marshal(lineItems)
	now := time.Now().UTC().Truncate(time.Second)
	return &models.CreditNote{
		ID:                uuid.New(),
		TenantID:          tenantID,
		CreditNoteNumber:  fmt.Sprintf("GS-TEST-%s", uuid.New().String()[:8]),
		Status:            models.CreditNoteStatusDraft,
		OriginalInvoiceID: originalInvoiceID,
		CustomerName:      "Test GmbH",
		CustomerAddress:   "Musterstraße 1, 10115 Berlin",
		CustomerEmail:     "cn@example.com",
		TaxMode:           models.TaxModeStandard,
		LineItems:         raw,
		Subtotal:          decimal.NewFromFloat(100.00),
		TotalTax:          decimal.NewFromFloat(19.00),
		GrossTotal:        decimal.NewFromFloat(119.00),
		Reason:            "Stornierung Teilleistung",
		CreatedBy:         uuid.New(),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

func twoCNLines() []models.LineItem {
	return []models.LineItem{
		{
			ID:          "cn-1",
			Position:    1,
			Description: "Gutschrift Position 1",
			Quantity:    decimal.NewFromFloat(1.0),
			UnitPrice:   decimal.NewFromFloat(75.00),
			TaxRate:     decimal.NewFromFloat(19.0),
			LineTotal:   decimal.NewFromFloat(75.00),
		},
		{
			ID:          "cn-2",
			Position:    2,
			Description: "Gutschrift Position 2",
			Quantity:    decimal.NewFromFloat(1.0),
			UnitPrice:   decimal.NewFromFloat(25.00),
			TaxRate:     decimal.NewFromFloat(19.0),
			LineTotal:   decimal.NewFromFloat(25.00),
		},
	}
}

func countCNLineRows(t *testing.T, superPool *pgxpool.Pool, cnID uuid.UUID) int {
	t.Helper()
	var n int
	if err := superPool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM finance_credit_note_lines WHERE credit_note_id = $1",
		cnID,
	).Scan(&n); err != nil {
		t.Fatalf("count cn line rows: %v", err)
	}
	return n
}

func seedTenantCN(t *testing.T, pool *pgxpool.Pool, ctx context.Context, tenantID uuid.UUID) {
	t.Helper()
	_, err := pool.Exec(ctx,
		`INSERT INTO tenants (id, name, created_at, updated_at)
		 VALUES ($1, $2, NOW(), NOW())
		 ON CONFLICT (id) DO NOTHING`,
		tenantID, fmt.Sprintf("Test Tenant %s", tenantID.String()[:8]),
	)
	if err != nil {
		t.Logf("seedTenantCN (non-fatal): %v", err)
	}
}

// ============================================================================
// Tests
// ============================================================================

// TestCreditNoteLinesRelationalRoundtrip creates a credit note with 2 line items,
// reads it back via GetByID, and asserts position ordering and decimal equality.
func TestCreditNoteLinesRelationalRoundtrip(t *testing.T) {
	appPool, _ := pgtc.StartPostgres(t)
	tenantID := uuid.New()
	ctx := pgtc.TenantCtx(context.Background(), tenantID)
	seedTenantCN(t, appPool, ctx, tenantID)

	// CreditNote requires a valid original_invoice_id FK.
	origInvID := insertRawInvoice(t, appPool, ctx, tenantID)

	repo := NewPostgresRepository(appPool)
	lines := twoCNLines()
	cn := makeCreditNote(tenantID, origInvID, lines)

	if err := repo.Create(ctx, cn); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(ctx, tenantID, cn.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	var gotLines []models.LineItem
	if err := json.Unmarshal(got.LineItems, &gotLines); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(gotLines) != 2 {
		t.Fatalf("line count: got %d, want 2", len(gotLines))
	}
	if gotLines[0].Position != 1 {
		t.Errorf("line 0 position: got %d, want 1", gotLines[0].Position)
	}
	if gotLines[1].Position != 2 {
		t.Errorf("line 1 position: got %d, want 2", gotLines[1].Position)
	}
	if !gotLines[0].UnitPrice.Equal(lines[0].UnitPrice) {
		t.Errorf("line 0 unit_price: got %s, want %s", gotLines[0].UnitPrice, lines[0].UnitPrice)
	}
	if !gotLines[1].UnitPrice.Equal(lines[1].UnitPrice) {
		t.Errorf("line 1 unit_price: got %s, want %s", gotLines[1].UnitPrice, lines[1].UnitPrice)
	}

	// Check original_invoice_id is preserved.
	if got.OriginalInvoiceID != origInvID {
		t.Errorf("OriginalInvoiceID: got %s, want %s", got.OriginalInvoiceID, origInvID)
	}
}

// TestCreditNoteUpdateReplacesLines creates a credit note with 2 lines, updates
// it with 1 line, and verifies the old lines are gone (no orphan rows).
func TestCreditNoteUpdateReplacesLines(t *testing.T) {
	appPool, superURL := pgtc.StartPostgres(t)
	superPool := pgtc.SuperPool(t, superURL)
	t.Cleanup(superPool.Close)

	tenantID := uuid.New()
	ctx := pgtc.TenantCtx(context.Background(), tenantID)
	seedTenantCN(t, appPool, ctx, tenantID)

	origInvID := insertRawInvoice(t, appPool, ctx, tenantID)
	repo := NewPostgresRepository(appPool)

	cn := makeCreditNote(tenantID, origInvID, twoCNLines())
	if err := repo.Create(ctx, cn); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if n := countCNLineRows(t, superPool, cn.ID); n != 2 {
		t.Fatalf("after Create: got %d line rows, want 2", n)
	}

	// Update with 1 new line.
	newLines := []models.LineItem{{
		Position:    1,
		Description: "Vollstorno",
		Quantity:    decimal.NewFromFloat(1.0),
		UnitPrice:   decimal.NewFromFloat(100.00),
		TaxRate:     decimal.NewFromFloat(19.0),
		LineTotal:   decimal.NewFromFloat(100.00),
	}}
	raw, _ := json.Marshal(newLines)
	cn.LineItems = raw
	cn.UpdatedAt = time.Now().UTC()
	if err := repo.Update(ctx, cn); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if n := countCNLineRows(t, superPool, cn.ID); n != 1 {
		t.Fatalf("after Update: got %d line rows, want 1 (old rows deleted)", n)
	}
}

// TestCreditNoteTenantIsolation verifies RLS prevents cross-tenant reads.
func TestCreditNoteTenantIsolation(t *testing.T) {
	appPool, _ := pgtc.StartPostgres(t)
	tenantA := uuid.New()
	tenantB := uuid.New()
	ctxA := pgtc.TenantCtx(context.Background(), tenantA)
	ctxB := pgtc.TenantCtx(context.Background(), tenantB)
	seedTenantCN(t, appPool, ctxA, tenantA)
	seedTenantCN(t, appPool, ctxB, tenantB)

	origA := insertRawInvoice(t, appPool, ctxA, tenantA)
	origB := insertRawInvoice(t, appPool, ctxB, tenantB)

	repo := NewPostgresRepository(appPool)

	cnA := makeCreditNote(tenantA, origA, twoCNLines())
	if err := repo.Create(ctxA, cnA); err != nil {
		t.Fatalf("Create A: %v", err)
	}
	cnB := makeCreditNote(tenantB, origB, twoCNLines())
	if err := repo.Create(ctxB, cnB); err != nil {
		t.Fatalf("Create B: %v", err)
	}

	// Tenant A reads own credit note — must succeed with lines.
	gotA, err := repo.GetByID(ctxA, tenantA, cnA.ID)
	if err != nil {
		t.Fatalf("GetByID A: %v", err)
	}
	var aLines []models.LineItem
	if err := json.Unmarshal(gotA.LineItems, &aLines); err != nil {
		t.Fatalf("unmarshal A: %v", err)
	}
	if len(aLines) != 2 {
		t.Errorf("tenant A line count: got %d, want 2", len(aLines))
	}

	// Tenant A cannot read Tenant B's credit note.
	_, err = repo.GetByID(ctxA, tenantB, cnB.ID)
	if err == nil {
		t.Error("expected RLS error when tenant A reads tenant B's credit note, got nil")
	}
}
