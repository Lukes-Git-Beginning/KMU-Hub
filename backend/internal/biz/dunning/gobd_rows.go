// Package dunning — GoBD posting-row construction.
// Lives here (not in the gRPC handler) because GoBDExportRow is defined here and
// the grouping is business logic: it picks the SKR03 revenue account and the
// DATEV BU-Schluessel that end up in the customer's bookkeeping.
package dunning

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/biz/datev"
	"github.com/kmuhub/kmuhub/internal/models"
)

// BuildGoBDRows splits one invoice or credit note into GoBD posting rows — one
// per VAT rate for standard mode, or a single exempt row for reverse-charge /
// Kleinunternehmer. Each row carries the SKR03 revenue account, DATEV BU key,
// net and VAT amounts. sign is +1 for invoices and -1 for credit notes so the
// journal nets out.
//
// Rates are grouped on their exact value (datev.RateKey), never on the integer
// part: 7.5% and 7% are different postings with different accounts and BU keys.
// Line nets are rounded to cents before they are summed, so the group net never
// carries sub-cent precision that StringFixed(2) would silently cut off.
func BuildGoBDRows(
	number string,
	docDate time.Time,
	customer, docStatus, taxMode string,
	lineItemsJSON json.RawMessage,
	subtotal decimal.Decimal,
	sign int,
) []GoBDExportRow {
	dateStr := docDate.Format("2006-01-02")
	bookingText := fmt.Sprintf("Rechnung %s %s", number, customer)
	signed := func(d decimal.Decimal) decimal.Decimal {
		if sign < 0 {
			return d.Neg()
		}
		return d
	}

	if taxMode == models.TaxModeReverseCharge || taxMode == models.TaxModeKleinunternehmer {
		zero := decimal.Zero
		return []GoBDExportRow{{
			InvoiceNumber: number,
			InvoiceDate:   dateStr,
			CustomerName:  customer,
			Account:       datev.RevenueAccountForRateAndMode(zero, taxMode),
			TaxKey:        datev.BUSchluesselForRate(zero),
			TaxRate:       datev.RateKey(zero),
			NetAmount:     signed(subtotal).StringFixed(2),
			TaxAmount:     "0.00",
			GrossTotal:    signed(subtotal).StringFixed(2),
			Status:        docStatus,
			TaxMode:       taxMode,
			BookingText:   bookingText,
		}}
	}

	var items []models.LineItem
	if len(lineItemsJSON) > 0 {
		if err := json.Unmarshal(lineItemsJSON, &items); err != nil {
			// Do not swallow: an unparseable line_items column drops every posting
			// of this document from the GoBD journal, which is exactly the gap an
			// auditor looks for.
			slog.Error("GoBD export: line items unparseable, document has no postings",
				"document_number", number, "error", err)
		}
	}

	hundred := decimal.NewFromInt(100)
	type rateGroup struct {
		rate decimal.Decimal
		net  decimal.Decimal
		tax  decimal.Decimal
	}
	order := make([]string, 0)
	groups := make(map[string]*rateGroup)
	for _, item := range items {
		lineTotal := item.Quantity.Mul(item.UnitPrice).Round(2)
		lineTax := lineTotal.Mul(item.TaxRate).Div(hundred).Round(2)
		key := datev.RateKey(item.TaxRate)
		g, ok := groups[key]
		if !ok {
			g = &rateGroup{rate: item.TaxRate}
			groups[key] = g
			order = append(order, key)
		}
		g.net = g.net.Add(lineTotal)
		g.tax = g.tax.Add(lineTax)
	}

	rows := make([]GoBDExportRow, 0, len(order))
	for _, key := range order {
		g := groups[key]
		rows = append(rows, GoBDExportRow{
			InvoiceNumber: number,
			InvoiceDate:   dateStr,
			CustomerName:  customer,
			Account:       datev.RevenueAccountForRateAndMode(g.rate, taxMode),
			TaxKey:        datev.BUSchluesselForRate(g.rate),
			TaxRate:       key,
			NetAmount:     signed(g.net).StringFixed(2),
			TaxAmount:     signed(g.tax).StringFixed(2),
			GrossTotal:    signed(g.net.Add(g.tax)).StringFixed(2),
			Status:        docStatus,
			TaxMode:       taxMode,
			BookingText:   bookingText,
		})
	}
	return rows
}
