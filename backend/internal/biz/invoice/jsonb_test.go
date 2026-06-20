package invoice

// jsonb_test.go — ADR-0007 unit tests for LineItem and TaxBreakdown serialization.
//
// These tests verify JSON marshal/unmarshal behavior of the LineItem and
// TaxBreakdown model types. The Sprint 4 relational migration (000132/000133)
// moved line storage from JSONB to finance_invoice_lines, and Migration 000217
// dropped the JSONB column. LineItems on the model struct is now an in-memory
// transport field populated from the relational table.
//
// See: docs/adr/0007-finance-line-items-normalization.md

import (
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

// TestLineItemsJSONBRoundtrip verifies that a []models.LineItem survives a
// Marshal → JSON → Unmarshal cycle with exact decimal equality.
// LineItems is now an in-memory transport field (not stored as JSONB); this test
// validates the serialization contract used between the repo and service layers.
func TestLineItemsJSONBRoundtrip(t *testing.T) {
	original := []models.LineItem{
		{
			ID:          "line-1",
			Position:    1,
			Description: "Beratungsleistung 1h",
			Quantity:    decimal.NewFromFloat(1.0),
			UnitPrice:   decimal.NewFromFloat(120.00),
			TaxRate:     decimal.NewFromFloat(19.0),
			LineTotal:   decimal.NewFromFloat(120.00),
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

	marshaled, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal line items: %v", err)
	}

	var restored []models.LineItem
	if err := json.Unmarshal(marshaled, &restored); err != nil {
		t.Fatalf("unmarshal line items: %v", err)
	}

	if len(restored) != len(original) {
		t.Fatalf("length mismatch: got %d, want %d", len(restored), len(original))
	}

	for i, orig := range original {
		got := restored[i]
		if !orig.Quantity.Equal(got.Quantity) {
			t.Errorf("line %d: quantity mismatch: got %s, want %s", i, got.Quantity, orig.Quantity)
		}
		if !orig.UnitPrice.Equal(got.UnitPrice) {
			t.Errorf("line %d: unit_price mismatch: got %s, want %s", i, got.UnitPrice, orig.UnitPrice)
		}
		if !orig.TaxRate.Equal(got.TaxRate) {
			t.Errorf("line %d: tax_rate mismatch: got %s, want %s", i, got.TaxRate, orig.TaxRate)
		}
		if !orig.LineTotal.Equal(got.LineTotal) {
			t.Errorf("line %d: line_total mismatch: got %s, want %s", i, got.LineTotal, orig.LineTotal)
		}
	}
}

// TestTaxBreakdownRoundtrip verifies that models.TaxBreakdown with a
// map[string]decimal.Decimal (tax_by_rate) survives serialization consistently.
//
// Decimal keys like "19" vs "19.00" can cause silent inequality after roundtrip.
// tax_breakdown remains stored as JSONB on the invoice header.
func TestTaxBreakdownRoundtrip(t *testing.T) {
	original := models.TaxBreakdown{
		Subtotal: decimal.NewFromFloat(165.50),
		TaxByRate: map[string]decimal.Decimal{
			"19": decimal.NewFromFloat(22.80),
			"7":  decimal.NewFromFloat(3.185),
		},
		TotalTax:   decimal.NewFromFloat(25.985),
		GrossTotal: decimal.NewFromFloat(191.485),
	}

	marshaled, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal tax breakdown: %v", err)
	}

	var restored models.TaxBreakdown
	if err := json.Unmarshal(marshaled, &restored); err != nil {
		t.Fatalf("unmarshal tax breakdown: %v", err)
	}

	if !original.Subtotal.Equal(restored.Subtotal) {
		t.Errorf("subtotal mismatch: got %s, want %s", restored.Subtotal, original.Subtotal)
	}
	if !original.TotalTax.Equal(restored.TotalTax) {
		t.Errorf("total_tax mismatch: got %s, want %s", restored.TotalTax, original.TotalTax)
	}
	if !original.GrossTotal.Equal(restored.GrossTotal) {
		t.Errorf("gross_total mismatch: got %s, want %s", restored.GrossTotal, original.GrossTotal)
	}

	for rate, origAmt := range original.TaxByRate {
		gotAmt, ok := restored.TaxByRate[rate]
		if !ok {
			t.Errorf("tax_by_rate key %q missing after roundtrip", rate)
			continue
		}
		if !origAmt.Equal(gotAmt) {
			t.Errorf("tax_by_rate[%q] mismatch: got %s, want %s", rate, gotAmt, origAmt)
		}
	}
}

// TestCorruptLineItemsHandling verifies that attempting to unmarshal a corrupt
// (non-array) JSON value into []models.LineItem does not panic and returns a
// clear error. This guards the in-memory unmarshal path used in the service layer.
func TestCorruptLineItemsHandling(t *testing.T) {
	corruptInputs := []struct {
		name  string
		input []byte
	}{
		{"null bytes", []byte(`null`)},
		{"object instead of array", []byte(`{"position": 1}`)},
		{"truncated", []byte(`[{"position": 1, "desc`)},
		{"empty string", []byte(`""`)},
		{"number", []byte(`42`)},
	}

	for _, tc := range corruptInputs {
		t.Run(tc.name, func(t *testing.T) {
			// Must not panic — use recover as last resort assertion.
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("unmarshal panicked on corrupt input %q: %v", tc.name, r)
				}
			}()

			var items []models.LineItem
			err := json.Unmarshal(tc.input, &items)
			if tc.name != "null bytes" && err == nil && len(items) == 0 {
				// null unmarshals to nil slice — that is acceptable Go behavior.
				// All other corrupt inputs must return an error.
				t.Errorf("expected unmarshal error for %q, got nil", tc.name)
			}
		})
	}
}
