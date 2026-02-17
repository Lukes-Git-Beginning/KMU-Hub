// Package tax provides pure-function tax calculation for German VAT (Umsatzsteuer).
// All monetary arithmetic uses shopspring/decimal for cent-accurate computation.
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
func Calculate(items []LineItem, mode TaxMode) Breakdown {
	// TODO: implement
	return Breakdown{}
}

// RequiresReverseChargeNote returns true if the tax mode requires a Reverse Charge note
// on the document (per EU VAT directive).
func RequiresReverseChargeNote(mode TaxMode) bool {
	return false
}

// RequiresKleinunternehmerNote returns true if the tax mode requires a Kleinunternehmer
// exemption note (per §19 UStG).
func RequiresKleinunternehmerNote(mode TaxMode) bool {
	return false
}

// StandardRate returns the German standard VAT rate (19%).
func StandardRate() decimal.Decimal {
	return decimal.Zero
}

// ReducedRate returns the German reduced VAT rate (7%).
func ReducedRate() decimal.Decimal {
	return decimal.Zero
}
