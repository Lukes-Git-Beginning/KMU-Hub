package tax

import (
	"testing"

	"github.com/shopspring/decimal"
)

// TestLineTotal_RoundsFractionalQuantityToCents pins the contract every write path
// relies on: a fractional quantity times a fractional unit price never leaves more
// than two decimals behind.
func TestLineTotal_RoundsFractionalQuantityToCents(t *testing.T) {
	cases := []struct {
		name     string
		qty      string
		price    string
		expected string
	}{
		{"hours from time tracking", "2.25", "83.33", "187.49"},
		{"fractional quantity rounds up", "1.5", "33.33", "50.00"},
		{"whole quantity is untouched", "3", "10.00", "30.00"},
		{"third of a unit", "0.333", "99.99", "33.30"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			qty := decimal.RequireFromString(tc.qty)
			price := decimal.RequireFromString(tc.price)
			got := LineTotal(qty, price)
			if got.StringFixed(2) != tc.expected {
				t.Fatalf("LineTotal(%s, %s) = %s, want %s", tc.qty, tc.price, got, tc.expected)
			}
			if got.Exponent() < -2 {
				t.Fatalf("LineTotal(%s, %s) kept more than 2 decimals: %s", tc.qty, tc.price, got)
			}
		})
	}
}

// TestCalculate_SubtotalIsSumOfLineTotals proves Calculate does not sum anything
// other than what LineTotal produces — the property the stored line items are
// expected to satisfy after the write path applies LineTotal itself.
func TestCalculate_SubtotalIsSumOfLineTotals(t *testing.T) {
	items := []LineItem{
		{Quantity: decimal.RequireFromString("1.5"), UnitPrice: decimal.RequireFromString("33.33"), TaxRate: decimal.NewFromInt(19)},
		{Quantity: decimal.RequireFromString("1.5"), UnitPrice: decimal.RequireFromString("33.33"), TaxRate: decimal.NewFromInt(19)},
		{Quantity: decimal.RequireFromString("1.5"), UnitPrice: decimal.RequireFromString("33.33"), TaxRate: decimal.NewFromInt(19)},
		{Quantity: decimal.RequireFromString("2.25"), UnitPrice: decimal.RequireFromString("83.33"), TaxRate: decimal.NewFromInt(7)},
	}

	breakdown := Calculate(items, ModeStandard)

	sum := decimal.Zero
	for _, it := range items {
		sum = sum.Add(LineTotal(it.Quantity, it.UnitPrice))
	}
	if !breakdown.Subtotal.Equal(sum) {
		t.Fatalf("subtotal %s != sum of rounded line totals %s", breakdown.Subtotal, sum)
	}
	// 50.00 * 3 + 187.49
	if breakdown.Subtotal.StringFixed(2) != "337.49" {
		t.Fatalf("subtotal = %s, want 337.49", breakdown.Subtotal.StringFixed(2))
	}
}
