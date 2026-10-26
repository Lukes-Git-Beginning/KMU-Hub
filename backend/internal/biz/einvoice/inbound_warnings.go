package einvoice

import (
	"fmt"
	"strings"
)

// Warning is a finding about an accepted inbound invoice. The invoice is stored
// either way: a supplier's document is a fact, and rejecting it would only push
// the booking back into e-mail and paper. The warning tells the reviewer where
// to look before the invoice moves on from "received".
type Warning struct {
	// Rule is the EN 16931 rule identifier (e.g. "BR-CL-04").
	Rule string
	// Term is the business term the finding is about (e.g. "BT-5").
	Term string
	// Message says what is unusual, in the vocabulary of the invoice form.
	Message string
}

// allowedTaxCategories are the VAT category codes (BT-118) the outbound
// generators emit; see the taxCategory* constants in generator_doc.go.
var allowedTaxCategories = map[string]bool{
	taxCategoryStandard:      true,
	taxCategoryZeroRated:     true,
	taxCategoryExempt:        true,
	taxCategoryReverseCharge: true,
}

// CheckCodelists compares an inbound invoice against the same currency and VAT
// category whitelists the outbound validation applies (allowedCurrencies,
// allowedTaxCategories) and reports every deviation. It never fails the import.
//
// lean: only currency and VAT category. Supplier country and VAT-ID prefix are
// not compared; add them once a reviewer asks for it.
func CheckCodelists(p *ParsedInvoice) []Warning {
	var warnings []Warning

	if currency := strings.ToUpper(strings.TrimSpace(p.Currency)); !allowedCurrencies[currency] {
		warnings = append(warnings, Warning{
			Rule:    "BR-CL-04",
			Term:    "BT-5",
			Message: fmt.Sprintf("currency %q is not one of the supported ISO 4217 codes", p.Currency),
		})
	}
	for _, category := range p.TaxCategories {
		if !allowedTaxCategories[category] {
			warnings = append(warnings, Warning{
				Rule:    "BR-CL-17",
				Term:    "BT-118",
				Message: fmt.Sprintf("VAT category %q is not one of the supported codes (S, Z, E, AE)", category),
			})
		}
	}
	return warnings
}

// distinctCategories upper-cases and trims VAT category codes, dropping empty and
// repeated ones while keeping the document order.
func distinctCategories(codes []string) []string {
	seen := make(map[string]bool, len(codes))
	out := make([]string, 0, len(codes))
	for _, c := range codes {
		c = strings.ToUpper(strings.TrimSpace(c))
		if c == "" || seen[c] {
			continue
		}
		seen[c] = true
		out = append(out, c)
	}
	return out
}
