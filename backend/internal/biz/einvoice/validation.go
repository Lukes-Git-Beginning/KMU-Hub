package einvoice

import (
	"fmt"
	"strings"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ============================================================================
// EN 16931 conformance checking
//
// buildInvoiceDoc already rejects invoices that cannot be rendered at all
// (no number, no issue date, no line items, totals drift). This file covers the
// other failure mode: an invoice that renders into a well-formed document the
// receiver then refuses, because a value their validator requires is absent.
//
// That rejection reaches the customer weeks later, through their accounting
// department, so every violation is reported at once rather than one per attempt.
// ============================================================================

// Profile selects the rule set an outbound document is checked against.
type Profile string

const (
	// ProfileEN16931 is the European core invoice — the floor below which no
	// document is deliverable, and what ZUGFeRD/Factur-X is measured against.
	ProfileEN16931 Profile = "en16931"

	// ProfileXRechnung adds the German CIUS rules on top of EN 16931. German
	// public-sector buyers enforce them; private-sector receivers do not, which is
	// why the generators do not apply this profile on their own — only the caller
	// knows who the invoice goes to.
	ProfileXRechnung Profile = "xrechnung"
)

// allowedCurrencies are the ISO 4217 codes BR-CL-04 lets an invoice currency
// (BT-5) carry.
//
// lean: DACH-only whitelist, not the full ISO 4217 list (180+ entries this
// product never emits). USD/CHF are already handled downstream (see the WKZ
// comment in datev/exporter.go); widen here the day a customer bills in
// another currency, not before.
var allowedCurrencies = map[string]bool{
	"EUR": true,
	"CHF": true,
	"USD": true,
}

// allowedCountries are the ISO 3166-1 alpha-2 codes BR-CL-14 lets a seller
// (BT-40) or buyer (BT-55) country carry.
//
// lean: DE/AT/CH whitelist, per the DACH-KMU target market (CLAUDE.md
// Zielgruppe). isoCountryCode (generator_doc.go) already normalises common
// spellings of these three to their code; anything else it passes through
// unchecked as a bare two-letter guess, which is exactly the gap this closes.
// Widen once a customer invoices outside DE/AT/CH.
var allowedCountries = map[string]bool{
	"DE": true,
	"AT": true,
	"CH": true,
}

// ValidationViolation is a single unmet requirement.
type ValidationViolation struct {
	// Rule is the business rule identifier (e.g. "BR-S-02"), empty where the CIUS
	// states the requirement without naming one.
	Rule string
	// Term is the business term the requirement is about (e.g. "BT-31").
	Term string
	// Message says what is missing, in the vocabulary of the invoice form rather
	// than of the standard.
	Message string
}

func (v ValidationViolation) String() string {
	if v.Rule == "" {
		return fmt.Sprintf("%s (%s)", v.Message, v.Term)
	}
	return fmt.Sprintf("%s (%s, %s)", v.Message, v.Rule, v.Term)
}

// ValidationError carries every requirement an invoice fails.
type ValidationError struct {
	InvoiceNumber string
	Profile       Profile
	Violations    []ValidationViolation
}

func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Violations))
	for _, v := range e.Violations {
		parts = append(parts, v.String())
	}
	return fmt.Sprintf("%s: invoice %s fails %d %s requirement(s): %s",
		ErrValidationFailed, e.InvoiceNumber, len(e.Violations), e.Profile, strings.Join(parts, "; "))
}

// Unwrap lets callers match the error with errors.Is(err, ErrValidationFailed)
// while still reading the individual violations through errors.As.
func (e *ValidationError) Unwrap() error { return ErrValidationFailed }

// Validate reports every requirement the invoice fails for the given profile,
// without rendering a document. GenerateUBL and GenerateCII already apply
// ProfileEN16931 themselves; call this with ProfileXRechnung before sending an
// invoice to a public-sector buyer, where the German CIUS applies on top.
//
// buyerReference is BT-10 (Leitweg-ID), as passed to the generators.
func Validate(invoice models.Invoice, settings models.CompanySettings, buyerReference string, profile Profile) error {
	doc, err := buildInvoiceDoc(invoice, settings, buyerReference)
	if err != nil {
		return err
	}
	return validateInvoiceDoc(doc, profile)
}

func validateInvoiceDoc(doc *invoiceDoc, profile Profile) error {
	var violations []ValidationViolation
	add := func(rule, term, format string, args ...any) {
		violations = append(violations, ValidationViolation{Rule: rule, Term: term, Message: fmt.Sprintf(format, args...)})
	}

	// BG-4 seller, BG-7 buyer. Name and country are the identifying minimum; the
	// address parts below are only mandatory under the German CIUS.
	//
	// BR-08 (seller postal address, BG-5) and BR-10 (buyer postal address, BG-8)
	// need no check of their own: country (BT-40/BT-55) is the only element EN
	// 16931 actually requires inside either group, so the BR-09/BR-11 checks below
	// already prove the group is present.
	if strings.TrimSpace(doc.Seller.Name) == "" {
		add("BR-06", "BT-27", "the company settings carry no company name")
	}
	if strings.TrimSpace(doc.Seller.Country) == "" {
		add("BR-09", "BT-40", "the company settings carry no country")
	}
	if strings.TrimSpace(doc.Buyer.Name) == "" {
		add("BR-07", "BT-44", "the invoice has no customer name")
	}
	if strings.TrimSpace(doc.Buyer.Country) == "" {
		add("BR-11", "BT-55", "the invoice has no customer country")
	}

	// BR-CL-04: the invoice currency has to be a real ISO 4217 code, not
	// whatever a caller happened to pass in — buildInvoiceDoc only defaults an
	// empty one, it never validates it.
	if !allowedCurrencies[doc.Currency] {
		add("BR-CL-04", "BT-5", "currency %q is not a supported ISO 4217 code", doc.Currency)
	}

	// BR-CL-14: seller and buyer country both have to be real ISO 3166-1
	// alpha-2 codes. isoCountryCode maps known spellings to DE/AT/CH but
	// passes anything else through as a bare two-letter guess (e.g. settings
	// carrying "XX" comes out as "XX") — that guess is what this catches.
	if doc.Seller.Country != "" && !allowedCountries[doc.Seller.Country] {
		add("BR-CL-14", "BT-40", "seller country %q is not a supported ISO 3166-1 code", doc.Seller.Country)
	}
	if doc.Buyer.Country != "" && !allowedCountries[doc.Buyer.Country] {
		add("BR-CL-14", "BT-55", "customer country %q is not a supported ISO 3166-1 code", doc.Buyer.Country)
	}

	// Whichever VAT category an invoice uses, the seller has to be identified for
	// tax purposes — a VAT identifier (BT-31) or, failing that, a tax number (BT-32).
	if strings.TrimSpace(doc.Seller.VATID) == "" && strings.TrimSpace(doc.Seller.TaxRegID) == "" {
		add(sellerTaxRuleFor(doc.TaxGroups), "BT-31",
			"the company settings carry neither a VAT identifier nor a tax number")
	}

	// Reverse charge moves the tax liability to the buyer, so the buyer has to be
	// identifiable for tax as well. This is the buyer half of BR-AE-02 (the seller
	// half is the sellerTaxRuleFor check above, for the same category) — BR-AE-03 is
	// a different rule entirely: it covers a document-level allowance (BG-20) with
	// its own VAT category, which this product does not emit.
	if usesTaxCategory(doc.TaxGroups, taxCategoryReverseCharge) && strings.TrimSpace(doc.Buyer.VATID) == "" {
		add("BR-AE-02", "BT-48", "reverse charge requires the customer VAT identifier")
	}

	for _, l := range doc.Lines {
		if strings.TrimSpace(l.Description) == "" {
			add("BR-25", "BT-153", "line %d has no item name", l.Position)
		}
		if l.UnitPrice.IsNegative() {
			add("BR-27", "BT-146", "line %d has a negative unit price", l.Position)
		}
	}

	// BR-CO-25: something is payable, so the receiver has to be told when.
	if doc.GrossTotal.IsPositive() && doc.DueDate.IsZero() && strings.TrimSpace(doc.PaymentTerms) == "" {
		add("BR-CO-25", "BT-9", "the invoice states neither a due date nor payment terms")
	}

	if profile == ProfileXRechnung {
		violations = append(violations, xrechnungViolations(doc)...)
	}

	if len(violations) == 0 {
		return nil
	}
	return &ValidationError{InvoiceNumber: doc.Number, Profile: profile, Violations: violations}
}

// xrechnungViolations covers the German CIUS requirements that go beyond the
// EN 16931 core.
//
// lean: BR-DE-2 and BR-DE-5..7 (seller contact point, telephone, e-mail) are not
// checked, because the outbound document carries no contact group to check — the
// company settings have no such fields. Add both the fields and these rules once a
// customer invoices a public-sector buyer that enforces them.
func xrechnungViolations(doc *invoiceDoc) []ValidationViolation {
	var violations []ValidationViolation
	add := func(rule, term, format string, args ...any) {
		violations = append(violations, ValidationViolation{Rule: rule, Term: term, Message: fmt.Sprintf(format, args...)})
	}

	if doc.BuyerReference == "" {
		add("BR-DE-15", "BT-10", "a public-sector buyer requires the routing id (Leitweg-ID)")
	}
	if strings.TrimSpace(doc.Bank.IBAN) == "" {
		add("BR-DE-1", "BG-16", "the company settings carry no IBAN, so the invoice states no way to pay it")
	}

	// BT-35/37/38 and BT-50/52/53: the CIUS wants the postal addresses split into
	// street, post code and city. splitPostalAddress leaves the buyer parts empty
	// when the stored free-text address does not follow "street\npost code city",
	// and an address squeezed into one line is what the receiver rejects.
	for _, f := range []struct{ term, value, what string }{
		{"BT-35", doc.Seller.Street, "the company street"},
		{"BT-37", doc.Seller.City, "the company city"},
		{"BT-38", doc.Seller.PostalZone, "the company post code"},
		{"BT-50", doc.Buyer.Street, "the customer street"},
		{"BT-52", doc.Buyer.City, "the customer city"},
		{"BT-53", doc.Buyer.PostalZone, "the customer post code"},
	} {
		if strings.TrimSpace(f.value) == "" {
			add("", f.term, "a structured postal address is required; %s is missing", f.what)
		}
	}
	return violations
}

// sellerTaxRuleFor names the rule that makes seller tax identification mandatory.
// Every VAT category states the same requirement under its own identifier, and
// quoting the one that actually applies is what makes the message searchable
// against the receiver's validation report.
func sellerTaxRuleFor(groups []docTaxGroup) string {
	if len(groups) == 0 {
		return "BR-S-02"
	}
	switch groups[0].CategoryCode {
	case taxCategoryReverseCharge:
		return "BR-AE-02"
	case taxCategoryExempt:
		return "BR-E-02"
	case taxCategoryZeroRated:
		return "BR-Z-02"
	default:
		return "BR-S-02"
	}
}

func usesTaxCategory(groups []docTaxGroup, category string) bool {
	for _, g := range groups {
		if g.CategoryCode == category {
			return true
		}
	}
	return false
}
