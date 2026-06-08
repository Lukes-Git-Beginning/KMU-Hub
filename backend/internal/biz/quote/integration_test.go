//go:build integration

package quote

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

func makeQuote(tenantID uuid.UUID, lineItems []models.LineItem) *models.Quote {
	raw, _ := json.Marshal(lineItems)
	now := time.Now().UTC().Truncate(time.Second)
	return &models.Quote{
		ID:             uuid.New(),
		TenantID:       tenantID,
		QuoteNumber:    fmt.Sprintf("AN-TEST-%s", uuid.New().String()[:8]),
		Status:         models.QuoteStatusDraft,
		CustomerName:   "Test AG",
		CustomerAddress: "Hauptstraße 5, 80331 München",
		CustomerEmail:  "q@example.com",
		TaxMode:        models.TaxModeStandard,
		LineItems:      raw,
		Subtotal:       decimal.NewFromFloat(200.00),
		TotalTax:       decimal.NewFromFloat(38.00),
		GrossTotal:     decimal.NewFromFloat(238.00),
		CreatedBy:      uuid.New(),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func twoQuoteLines() []models.LineItem {
	return []models.LineItem{
		{
			ID:          "ql-1",
			Position:    1,
			Description: "Analyse-Paket",
			Quantity:    decimal.NewFromFloat(1.0),
			UnitPrice:   decimal.NewFromFloat(150.00),
			TaxRate:     decimal.NewFromFloat(19.0),
			LineTotal:   decimal.NewFromFloat(150.00),
		},
		{
			ID:          "ql-2",
			Position:    2,
			Description: "Reisezeit",
			Quantity:    decimal.NewFromFloat(2.5),
			UnitPrice:   decimal.NewFromFloat(20.00),
			TaxRate:     decimal.NewFromFloat(7.0),
			LineTotal:   decimal.NewFromFloat(50.00),
		},
	}
}

func countQuoteLineRows(t *testing.T, superPool *pgxpool.Pool, quoteID uuid.UUID) int {
	t.Helper()
	var n int
	if err := superPool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM finance_quote_lines WHERE quote_id = $1",
		quoteID,
	).Scan(&n); err != nil {
		t.Fatalf("count quote line rows: %v", err)
	}
	return n
}

func seedTenantQ(t *testing.T, pool *pgxpool.Pool, ctx context.Context, tenantID uuid.UUID) {
	t.Helper()
	_, err := pool.Exec(ctx,
		`INSERT INTO tenants (id, name, created_at, updated_at)
		 VALUES ($1, $2, NOW(), NOW())
		 ON CONFLICT (id) DO NOTHING`,
		tenantID, fmt.Sprintf("Test Tenant %s", tenantID.String()[:8]),
	)
	if err != nil {
		t.Logf("seedTenantQ (non-fatal): %v", err)
	}
}

// ============================================================================
// Tests
// ============================================================================

// TestQuoteLinesRelationalRoundtrip creates a quote with 2 line items and
// verifies exact decimal equality and position ordering after GetByID.
func TestQuoteLinesRelationalRoundtrip(t *testing.T) {
	appPool, _ := pgtc.StartPostgres(t)
	tenantID := uuid.New()
	ctx := pgtc.TenantCtx(context.Background(), tenantID)
	seedTenantQ(t, appPool, ctx, tenantID)

	repo := NewPostgresRepository(appPool)
	lines := twoQuoteLines()
	q := makeQuote(tenantID, lines)

	if err := repo.Create(ctx, q); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(ctx, tenantID, q.ID)
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
	if !gotLines[0].Quantity.Equal(lines[0].Quantity) {
		t.Errorf("line 0 quantity: got %s, want %s", gotLines[0].Quantity, lines[0].Quantity)
	}
	if !gotLines[0].UnitPrice.Equal(lines[0].UnitPrice) {
		t.Errorf("line 0 unit_price: got %s, want %s", gotLines[0].UnitPrice, lines[0].UnitPrice)
	}
	if !gotLines[1].UnitPrice.Equal(lines[1].UnitPrice) {
		t.Errorf("line 1 unit_price: got %s, want %s", gotLines[1].UnitPrice, lines[1].UnitPrice)
	}
}

// TestQuoteUpdateReplacesLines verifies that Update with new lines replaces old
// lines with no orphan rows left in finance_quote_lines.
func TestQuoteUpdateReplacesLines(t *testing.T) {
	appPool, superURL := pgtc.StartPostgres(t)
	superPool := pgtc.SuperPool(t, superURL)
	t.Cleanup(superPool.Close)

	tenantID := uuid.New()
	ctx := pgtc.TenantCtx(context.Background(), tenantID)
	seedTenantQ(t, appPool, ctx, tenantID)

	repo := NewPostgresRepository(appPool)
	q := makeQuote(tenantID, twoQuoteLines())
	if err := repo.Create(ctx, q); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if n := countQuoteLineRows(t, superPool, q.ID); n != 2 {
		t.Fatalf("after Create: got %d line rows, want 2", n)
	}

	// Replace with 1 new line.
	newLines := []models.LineItem{{
		Position:    1,
		Description: "Einzel-Posten",
		Quantity:    decimal.NewFromFloat(1.0),
		UnitPrice:   decimal.NewFromFloat(500.00),
		TaxRate:     decimal.NewFromFloat(19.0),
		LineTotal:   decimal.NewFromFloat(500.00),
	}}
	raw, _ := json.Marshal(newLines)
	q.LineItems = raw
	q.UpdatedAt = time.Now().UTC()
	if err := repo.Update(ctx, q); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if n := countQuoteLineRows(t, superPool, q.ID); n != 1 {
		t.Fatalf("after Update: got %d line rows, want 1 (old rows deleted)", n)
	}
}

// TestQuoteTenantIsolation verifies RLS prevents cross-tenant reads for both
// the quote rows and their relational lines.
func TestQuoteTenantIsolation(t *testing.T) {
	appPool, _ := pgtc.StartPostgres(t)
	tenantA := uuid.New()
	tenantB := uuid.New()
	ctxA := pgtc.TenantCtx(context.Background(), tenantA)
	ctxB := pgtc.TenantCtx(context.Background(), tenantB)
	seedTenantQ(t, appPool, ctxA, tenantA)
	seedTenantQ(t, appPool, ctxB, tenantB)

	repo := NewPostgresRepository(appPool)

	qA := makeQuote(tenantA, twoQuoteLines())
	if err := repo.Create(ctxA, qA); err != nil {
		t.Fatalf("Create A: %v", err)
	}
	qB := makeQuote(tenantB, twoQuoteLines())
	if err := repo.Create(ctxB, qB); err != nil {
		t.Fatalf("Create B: %v", err)
	}

	// Tenant A reads own quote — should succeed with lines.
	gotA, err := repo.GetByID(ctxA, tenantA, qA.ID)
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

	// Tenant A cannot read Tenant B's quote.
	_, err = repo.GetByID(ctxA, tenantB, qB.ID)
	if err == nil {
		t.Error("expected RLS error when tenant A reads tenant B's quote, got nil")
	}
}

// TestQuoteListAssemblesLines verifies List bulk-loads relational lines for
// every quote in the page (the loadQuoteLines ANY($1) path, no N+1).
func TestQuoteListAssemblesLines(t *testing.T) {
	appPool, _ := pgtc.StartPostgres(t)
	tenantID := uuid.New()
	ctx := pgtc.TenantCtx(context.Background(), tenantID)
	seedTenantQ(t, appPool, ctx, tenantID)

	repo := NewPostgresRepository(appPool)
	for i := 0; i < 2; i++ {
		if err := repo.Create(ctx, makeQuote(tenantID, twoQuoteLines())); err != nil {
			t.Fatalf("Create %d: %v", i, err)
		}
	}

	quotes, total, err := repo.List(ctx, tenantID, ListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total < 2 || len(quotes) < 2 {
		t.Fatalf("List: got total=%d len=%d, want >=2", total, len(quotes))
	}
	for _, q := range quotes {
		var lines []models.LineItem
		if err := json.Unmarshal(q.LineItems, &lines); err != nil {
			t.Fatalf("unmarshal lines for %s: %v", q.ID, err)
		}
		if len(lines) != 2 {
			t.Errorf("quote %s: got %d lines, want 2 (List must assemble relational lines)", q.ID, len(lines))
		}
	}
}

// TestQuoteGetByDealIDEmpty exercises the GetByDealID read path for a deal with
// no quotes — must return an empty slice without error (covers the query +
// empty bulk-load branch).
func TestQuoteGetByDealIDEmpty(t *testing.T) {
	appPool, _ := pgtc.StartPostgres(t)
	tenantID := uuid.New()
	ctx := pgtc.TenantCtx(context.Background(), tenantID)
	seedTenantQ(t, appPool, ctx, tenantID)

	repo := NewPostgresRepository(appPool)
	got, err := repo.GetByDealID(ctx, tenantID, uuid.New())
	if err != nil {
		t.Fatalf("GetByDealID (no rows) should not error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("GetByDealID for unknown deal: got %d quotes, want 0", len(got))
	}
}
