package server

import (
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/kmuhub/kmuhub/internal/models"
)

// dec is a test helper for decimal literals.
func dec(t *testing.T, s string) decimal.Decimal {
	t.Helper()
	d, err := decimal.NewFromString(s)
	require.NoError(t, err)
	return d
}

// TestProtoTaxBreakdown_FallsBackToColumns guards the finance-list "0,00 €"
// regression: rows that carry the denormalised subtotal/total_tax/gross_total
// columns but no tax_breakdown JSONB must still serialise a breakdown. Seed
// rows and Bexio mirrors (field_mapper fills the columns, never the JSONB) are
// exactly that shape, so a nil breakdown blanks every amount in the list.
func TestProtoTaxBreakdown_FallsBackToColumns(t *testing.T) {
	tb := protoTaxBreakdown(nil, dec(t, "100.00"), dec(t, "19.00"), dec(t, "119.00"))

	require.NotNil(t, tb, "breakdown must never be nil when the columns carry values")
	assert.Equal(t, "100", tb.GetSubtotal())
	assert.Equal(t, "19", tb.GetTotalTax())
	assert.Equal(t, "119", tb.GetGrossTotal())
	assert.Empty(t, tb.GetTaxByRate(), "tax_by_rate is not a column and stays unset in the fallback")
}

// TestProtoTaxBreakdown_PrefersJSONB pins the precedence: when the JSONB is
// present it wins, including tax_by_rate, which the column fallback cannot
// reconstruct.
func TestProtoTaxBreakdown_PrefersJSONB(t *testing.T) {
	raw, err := json.Marshal(models.TaxBreakdown{
		Subtotal:   dec(t, "200.00"),
		TaxByRate:  map[string]decimal.Decimal{"19": dec(t, "38.00")},
		TotalTax:   dec(t, "38.00"),
		GrossTotal: dec(t, "238.00"),
	})
	require.NoError(t, err)

	// Columns deliberately disagree — the JSONB must win.
	tb := protoTaxBreakdown(raw, dec(t, "1"), dec(t, "1"), dec(t, "1"))

	require.NotNil(t, tb)
	assert.Equal(t, "238", tb.GetGrossTotal())
	assert.Equal(t, map[string]string{"19": "38"}, tb.GetTaxByRate())
}

// TestToProtoInvoice_AmountsAndCurrency covers the list path end to end for an
// invoice: the amount comes from the columns when the JSONB is absent, and the
// currency travels with it. Without the currency the frontend renders every
// amount with the EUR symbol — a CHF Bexio mirror would read as euros.
func TestToProtoInvoice_AmountsAndCurrency(t *testing.T) {
	externalID := "bx-4711"
	externalNumber := "RE-CH-0007"
	inv := &models.Invoice{
		InvoiceNumber:  "RE-2026-0042",
		Status:         models.InvoiceStatusSent,
		Subtotal:       dec(t, "1000.00"),
		TotalTax:       dec(t, "81.00"),
		GrossTotal:     dec(t, "1081.00"),
		Currency:       "CHF",
		Source:         models.InvoiceSourceBexio,
		ExternalID:     &externalID,
		ExternalNumber: &externalNumber,
	}

	pi := toProtoInvoice(inv)

	require.NotNil(t, pi)
	assert.Equal(t, "1081", pi.GetTaxBreakdown().GetGrossTotal())
	assert.Equal(t, "CHF", pi.GetCurrency())
	assert.Equal(t, models.InvoiceSourceBexio, pi.GetSource(),
		"provenance gates editing in the detail view; a read-only mirror must not read as cosmi")
	assert.Equal(t, externalNumber, pi.GetExternalNumber())
	assert.Equal(t, externalID, pi.GetExternalId())
}

// TestToProtoInvoice_LegacyCurrencyDefaultsToEUR pins the wire contract for
// rows written before the currency column existed: the wire stays
// self-describing instead of relying on a frontend default.
func TestToProtoInvoice_LegacyCurrencyDefaultsToEUR(t *testing.T) {
	pi := toProtoInvoice(&models.Invoice{GrossTotal: dec(t, "42.00")})

	require.NotNil(t, pi)
	assert.Equal(t, models.DefaultCurrency, pi.GetCurrency())
	assert.Empty(t, pi.GetExternalNumber(), "a cosmi invoice carries no external number")
}

// TestDocumentCurrency pins the helper directly, not just through the three
// toProto* converters: GenerateGoBDExport's two BuildGoBDRows call sites route
// through it too (fix-gobd-export-currency-empty-for-legacy-documents), and a
// converter-only test would not have caught a raw inv.Currency/cn.Currency
// passed around it there.
func TestDocumentCurrency(t *testing.T) {
	assert.Equal(t, models.DefaultCurrency, documentCurrency(""), "legacy documents with no stored currency must default to EUR")
	assert.Equal(t, "CHF", documentCurrency("CHF"), "a stored currency must pass through unchanged")
}

// TestToProtoQuoteAndCreditNote_AmountsAndCurrency mirrors the invoice
// assertions for the other two finance documents — all three lists render from
// the same converter shape.
func TestToProtoQuoteAndCreditNote_AmountsAndCurrency(t *testing.T) {
	pq := toProtoQuote(&models.Quote{
		QuoteNumber: "AN-2026-0001",
		Subtotal:    dec(t, "500.00"),
		TotalTax:    dec(t, "95.00"),
		GrossTotal:  dec(t, "595.00"),
		Currency:    "CHF",
	})
	require.NotNil(t, pq)
	assert.Equal(t, "595", pq.GetTaxBreakdown().GetGrossTotal())
	assert.Equal(t, "CHF", pq.GetCurrency())

	pcn := toProtoCreditNote(&models.CreditNote{
		CreditNoteNumber: "GS-2026-0001",
		Subtotal:         dec(t, "100.00"),
		TotalTax:         dec(t, "19.00"),
		GrossTotal:       dec(t, "119.00"),
	})
	require.NotNil(t, pcn)
	assert.Equal(t, "119", pcn.GetTaxBreakdown().GetGrossTotal())
	assert.Equal(t, models.DefaultCurrency, pcn.GetCurrency())
}

// TestInvoiceWireShape checks the JSON the gateway actually emits, with the
// same protojson options the invoice routes use: snake_case keys, so the
// frontend's tax_breakdown.gross_total / currency / source reads hit.
func TestInvoiceWireShape(t *testing.T) {
	pi := toProtoInvoice(&models.Invoice{
		GrossTotal: dec(t, "1081.00"),
		Currency:   "CHF",
		Source:     models.InvoiceSourceCosmi,
	})

	raw, err := protojson.MarshalOptions{UseProtoNames: true, UseEnumNumbers: true}.Marshal(pi)
	require.NoError(t, err)

	var wire struct {
		Currency     string `json:"currency"`
		Source       string `json:"source"`
		TaxBreakdown struct {
			GrossTotal string `json:"gross_total"`
		} `json:"tax_breakdown"`
	}
	require.NoError(t, json.Unmarshal(raw, &wire))

	assert.Equal(t, "1081", wire.TaxBreakdown.GrossTotal)
	assert.Equal(t, "CHF", wire.Currency)
	assert.Equal(t, models.InvoiceSourceCosmi, wire.Source)
}
