package recurring

import (
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

// TestPriceLineItems_RoundsLineTotals: priceLineItems is the single write path for
// every schedule and for every invoice generated from it. Four lines of
// 1.5 x 33.33 EUR sum to 199.98 unrounded and 200.00 rounded per line — a recurring
// schedule that stored the unrounded value would emit that cent difference on every
// single run.
func TestPriceLineItems_RoundsLineTotals(t *testing.T) {
	items := make([]models.LineItem, 4)
	for i := range items {
		items[i] = models.LineItem{
			Description: "Wartungspauschale",
			Quantity:    decimal.RequireFromString("1.5"),
			UnitPrice:   decimal.RequireFromString("33.33"),
			TaxRate:     decimal.NewFromInt(19),
		}
	}

	itemsJSON, breakdownJSON, err := priceLineItems(items, models.TaxModeStandard)
	if err != nil {
		t.Fatalf("priceLineItems: %v", err)
	}

	var stored []models.LineItem
	if err := json.Unmarshal(itemsJSON, &stored); err != nil {
		t.Fatalf("unmarshal line items: %v", err)
	}
	var breakdown models.TaxBreakdown
	if err := json.Unmarshal(breakdownJSON, &breakdown); err != nil {
		t.Fatalf("unmarshal breakdown: %v", err)
	}

	sum := decimal.Zero
	for _, it := range stored {
		if it.LineTotal.Exponent() < -2 {
			t.Fatalf("stored line total %s has more than 2 decimals", it.LineTotal)
		}
		sum = sum.Add(it.LineTotal)
	}
	if !sum.Equal(breakdown.Subtotal) {
		t.Fatalf("sum of stored line totals %s != stored subtotal %s", sum, breakdown.Subtotal)
	}
	if breakdown.Subtotal.StringFixed(2) != "200.00" {
		t.Fatalf("subtotal = %s, want 200.00", breakdown.Subtotal.StringFixed(2))
	}
}
