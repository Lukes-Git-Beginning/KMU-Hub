// Package tax provides pure-function tax calculation for German VAT (Umsatzsteuer).
// All monetary arithmetic uses shopspring/decimal for cent-accurate computation.
// No float64 is used anywhere -- all values flow through decimal.Decimal to prevent
// rounding errors that are unacceptable in financial calculations.
package tax

import (
	"strings"

	"github.com/shopspring/decimal"
)

// TaxMode determines how tax is calculated on a financial document.
type TaxMode string

const (
	// ModeStandard applies regular German VAT rates (19% / 7%).
	ModeStandard TaxMode = "standard"
	// ModeReverseCharge sets all tax to zero (buyer handles VAT, EU B2B).
	ModeReverseCharge TaxMode = "reverse_charge"
	// ModeKleinunternehmer sets all tax to zero (small business exemption per §19 UStG).
	ModeKleinunternehmer TaxMode = "kleinunternehmer"
)

// hundred is used to convert percentage rates (e.g., 19.00) to multipliers (0.19).
var hundred = decimal.NewFromInt(100)

// LineItem represents a single line for tax calculation.
type LineItem struct {
	Quantity  decimal.Decimal
	UnitPrice decimal.Decimal
	TaxRate   decimal.Decimal // e.g., 19.00 or 7.00
}

// Breakdown holds the computed tax summary.
type Breakdown struct {
	Subtotal   decimal.Decimal
	TaxByRate  map[string]decimal.Decimal // Rate key (e.g., "19") -> aggregated tax
	TotalTax   decimal.Decimal
	GrossTotal decimal.Decimal
}

// Calculate computes the tax breakdown for a set of line items under the given tax mode.
// This is a pure function: no database access, no side effects, fully deterministic.
//
// Both the line net and the line tax are rounded to 2 decimal places at the line
// level (not at the total level) before they are summed. This prevents cent
// discrepancies that arise from rounding a sum vs. summing rounded values, and it
// matches the order the PDF uses: it prints a net and a tax column per line, so the
// printed lines have to add up to the printed total.
//
// Remaining rounding-order difference from the e-invoice XML: this breakdown sums
// per-line tax (round each line's tax, then add), while EN 16931's BR-CO-17 derives
// the group tax (BT-117) from the already-rounded group net (BT-116) in one step.
// Both are right for their medium; over many fractional lines with mixed rates they
// can still drift apart by a cent or two. einvoice.buildLinesAndTaxGroups carries
// the counterpart note, and einvoice.totalsTolerance is sized for exactly this gap.
//
// For ReverseCharge and Kleinunternehmer modes, all tax is zero regardless of the
// tax rates specified on line items.
func Calculate(items []LineItem, mode TaxMode) Breakdown {
	subtotal := decimal.Zero
	totalTax := decimal.Zero
	taxByRate := make(map[string]decimal.Decimal)

	taxExempt := mode == ModeReverseCharge || mode == ModeKleinunternehmer

	for _, item := range items {
		// Line total = quantity * unit_price, rounded to 2 decimal places before
		// summing (same rounding order the e-invoice generator uses for BT-106,
		// see fix-tax-calculator-line-total-unrounded in the loop backlog) so the
		// stored subtotal and the exported document net agree line by line.
		lineTotal := item.Quantity.Mul(item.UnitPrice).Round(2)
		subtotal = subtotal.Add(lineTotal)

		if !taxExempt {
			// Per-line tax: round to 2 decimal places at line level
			lineTax := lineTotal.Mul(item.TaxRate).Div(hundred).Round(2)
			totalTax = totalTax.Add(lineTax)

			// A 0% line in standard mode (e.g. an out-of-scope item on an otherwise
			// taxable invoice) must not create a rate group: TaxByRate has no category
			// of its own, so a "0" key renders and exports as VAT category S at 0%,
			// which every EN 16931 validator rejects (S requires a positive rate).
			if !item.TaxRate.IsZero() {
				rateKey := rateGroupKey(item.TaxRate)
				existing, ok := taxByRate[rateKey]
				if !ok {
					existing = decimal.Zero
				}
				taxByRate[rateKey] = existing.Add(lineTax)
			}
		}
	}

	grossTotal := subtotal.Add(totalTax)

	// For tax-exempt modes, return empty TaxByRate map
	if taxExempt {
		taxByRate = make(map[string]decimal.Decimal)
	}

	return Breakdown{
		Subtotal:   subtotal,
		TaxByRate:  taxByRate,
		TotalTax:   totalTax,
		GrossTotal: grossTotal,
	}
}

// rateGroupKey formats a tax rate as a TaxByRate map key: whole rates stay bare
// ("19", "7"), fractional rates keep the decimal ("7.5"). Truncating to an integer
// (the previous behaviour) collided 7.5% into the same key as 7% — two different
// legal rates aggregated into one, silently picking whichever's revenue account
// happened to be looked up last.
func rateGroupKey(rate decimal.Decimal) string {
	s := rate.StringFixed(2)
	s = strings.TrimRight(s, "0")
	return strings.TrimRight(s, ".")
}

// RequiresReverseChargeNote returns true if the tax mode requires a Reverse Charge note
// on the document (per EU VAT directive).
func RequiresReverseChargeNote(mode TaxMode) bool {
	return mode == ModeReverseCharge
}

// RequiresKleinunternehmerNote returns true if the tax mode requires a Kleinunternehmer
// exemption note (per §19 UStG).
func RequiresKleinunternehmerNote(mode TaxMode) bool {
	return mode == ModeKleinunternehmer
}

// StandardRate returns the German standard VAT rate (19%).
func StandardRate() decimal.Decimal {
	return decimal.NewFromInt(19)
}

// ReducedRate returns the German reduced VAT rate (7%).
func ReducedRate() decimal.Decimal {
	return decimal.NewFromInt(7)
}
