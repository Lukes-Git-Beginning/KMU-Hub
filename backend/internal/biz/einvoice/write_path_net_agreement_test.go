package einvoice

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/biz/tax"
	"github.com/kmuhub/kmuhub/internal/models"
)

// TestWritePathNetMatchesDocumentNetExactly closes the loop for
// fix-write-path-line-total-unrounded-everywhere: it prices line items exactly the
// way invoice/quote/creditnote/recurring now do (tax.LineTotal per line, then
// tax.Calculate) and asserts the document net BT-106 equals the stored subtotal
// EXACTLY — zero difference, not "inside totalsTolerance".
//
// Before the fix the write path stored 4 x 1.5 x 33.33 = 199.98 while
// buildLinesAndTaxGroups, which always rounds per line for BR-CO-10, emitted
// 200.00. The scaled tolerance kept the export alive; it did not make the two
// cents agree.
func TestWritePathNetMatchesDocumentNetExactly(t *testing.T) {
	cases := []struct {
		name  string
		qty   string
		price string
		rate  int64
		lines int
	}{
		{"quarter hours at a fractional rate", "2.25", "83.33", 19, 5},
		{"halves at a repeating price", "1.5", "33.33", 19, 4},
		{"thirds at 7 percent", "0.333", "99.99", 7, 6},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			qty := decimal.RequireFromString(tc.qty)
			price := decimal.RequireFromString(tc.price)

			items := make([]models.LineItem, tc.lines)
			taxItems := make([]tax.LineItem, tc.lines)
			for i := range items {
				items[i] = models.LineItem{
					Position:    i + 1,
					Description: "Leistung",
					Quantity:    qty,
					UnitPrice:   price,
					TaxRate:     decimal.NewFromInt(tc.rate),
					// Exactly what the write paths store since the fix.
					LineTotal: tax.LineTotal(qty, price),
				}
				taxItems[i] = tax.LineItem{Quantity: qty, UnitPrice: price, TaxRate: decimal.NewFromInt(tc.rate)}
			}
			breakdown := tax.Calculate(taxItems, tax.ModeStandard)

			raw, err := json.Marshal(items)
			require.NoError(t, err)

			invoice := models.Invoice{
				InvoiceNumber:   "RE-2026-0099",
				CustomerName:    "Stadtwerke Musterstadt AöR",
				CustomerAddress: "Rathausplatz 3\n12345 Musterstadt",
				TaxMode:         models.TaxModeStandard,
				LineItems:       raw,
				Subtotal:        breakdown.Subtotal,
				TotalTax:        breakdown.TotalTax,
				GrossTotal:      breakdown.GrossTotal,
				Currency:        "EUR",
				InvoiceDate:     time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
				DueDate:         time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC),
			}

			doc, err := buildInvoiceDoc(invoice, testSettings(), "991-12345-67")
			require.NoError(t, err)

			require.Truef(t, doc.LineTotal.Equal(invoice.Subtotal),
				"BT-106 %s != stored subtotal %s (difference %s)",
				doc.LineTotal, invoice.Subtotal, doc.LineTotal.Sub(invoice.Subtotal))

			// BR-CO-10: BT-106 is the sum of the line amounts as written, exactly.
			sum := decimal.Zero
			for _, l := range doc.Lines {
				sum = sum.Add(l.Net)
			}
			require.Truef(t, sum.Equal(doc.LineTotal),
				"sum of BT-131 line amounts %s != BT-106 %s", sum, doc.LineTotal)
		})
	}
}
