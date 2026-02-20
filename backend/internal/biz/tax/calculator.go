// Package tax provides pure-function tax calculation for German VAT (Umsatzsteuer).
// All monetary arithmetic uses shopspring/decimal for cent-accurate computation.
// No float64 is used anywhere -- all values flow through decimal.Decimal to prevent
// rounding errors that are unacceptable in financial calculations.
package tax

import (
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
// Tax is calculated per line item and rounded to 2 decimal places at the line level
// (not at the total level). This prevents cent discrepancies that arise from rounding
// a sum vs. summing rounded values.
//
// For ReverseCharge and Kleinunternehmer modes, all tax is zero regardless of the
// tax rates specified on line items.
func Calculate(items []LineItem, mode TaxMode) Breakdown {
	subtotal := decimal.Zero
	totalTax := decimal.Zero
	taxByRate := make(map[string]decimal.Decimal)

	taxExempt := mode == ModeReverseCharge || mode == ModeKleinunternehmer

	for _, item := range items {
		// Line total = quantity * unit_price
		lineTotal := item.Quantity.Mul(item.UnitPrice)
		subtotal = subtotal.Add(lineTotal)

		if !taxExempt {
			// Per-line tax: round to 2 decimal places at line level
			lineTax := lineTotal.Mul(item.TaxRate).Div(hundred).Round(2)
			totalTax = totalTax.Add(lineTax)

			// Aggregate by rate key (strip trailing zeros for clean keys: "19.00" -> "19")
			rateKey := item.TaxRate.Truncate(0).StringFixed(0)
			existing, ok := taxByRate[rateKey]
			if !ok {
				existing = decimal.Zero
			}
			taxByRate[rateKey] = existing.Add(lineTax)
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
