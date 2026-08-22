package einvoice

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/shopspring/decimal"
)

// violationTerms returns the business terms reported by err, so a test can state
// which requirements it expects without depending on the wording.
func violationTerms(t *testing.T, err error) []string {
	t.Helper()

	var ve *ValidationError
	require.ErrorAs(t, err, &ve, "expected a *ValidationError, got %v", err)
	require.ErrorIs(t, err, ErrValidationFailed, "a validation error must match ErrValidationFailed")

	terms := make([]string, 0, len(ve.Violations))
	for _, v := range ve.Violations {
		terms = append(terms, v.Term)
	}
	return terms
}

// ============================================================================
// A complete invoice passes
// ============================================================================

func TestValidate_CompleteInvoicePassesBothProfiles(t *testing.T) {
	t.Parallel()

	inv := testInvoice(t)
	assert.NoError(t, Validate(inv, testSettings(), "", ProfileEN16931))
	assert.NoError(t, Validate(inv, testSettings(), "991-12345-67", ProfileXRechnung))
}

// TestValidate_XRechnungNeedsRoutingID pins the one requirement that separates
// the two profiles for an otherwise complete invoice: without the Leitweg-ID the
// document is fine for a private buyer and rejected by a public-sector one.
func TestValidate_XRechnungNeedsRoutingID(t *testing.T) {
	t.Parallel()

	inv := testInvoice(t)
	require.NoError(t, Validate(inv, testSettings(), "", ProfileEN16931))

	err := Validate(inv, testSettings(), "", ProfileXRechnung)
	assert.Equal(t, []string{"BT-10"}, violationTerms(t, err))
}

// ============================================================================
// Every violation is reported at once
// ============================================================================

// TestValidate_ReportsEveryViolationAtOnce is the point of the whole file. A
// receiver rejects the invoice weeks later, through the customer's accounting
// department, so stopping at the first missing field would make the user fix one
// thing per round trip.
func TestValidate_ReportsEveryViolationAtOnce(t *testing.T) {
	t.Parallel()

	items := []models.LineItem{
		{Position: 1, Description: "", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.RequireFromString("-10.00"), TaxRate: decimal.NewFromInt(19), LineTotal: decimal.RequireFromString("100.00")},
	}
	raw, err := json.Marshal(items)
	require.NoError(t, err)

	inv := models.Invoice{
		InvoiceNumber: "RE-2026-0099",
		// No customer name, no due date, no payment terms.
		TaxMode:     models.TaxModeStandard,
		LineItems:   raw,
		InvoiceDate: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
	}
	// No company name and no tax identification of any kind.
	settings := models.CompanySettings{Country: "Deutschland"}

	verr := Validate(inv, settings, "", ProfileEN16931)
	terms := violationTerms(t, verr)

	assert.ElementsMatch(t, []string{
		"BT-27",  // seller name
		"BT-44",  // buyer name
		"BT-31",  // seller tax identification
		"BT-153", // line item name
		"BT-146", // negative unit price
		"BT-9",   // neither due date nor payment terms
	}, terms, "every unmet requirement has to be reported in one pass")

	// The rule identifiers travel with the violations so a user can match them
	// against the receiver's validation report.
	assert.Contains(t, verr.Error(), "BR-S-02")
	assert.Contains(t, verr.Error(), "BR-CO-25")
}

// TestValidate_XRechnungReportsAddressPartsAtOnce covers the CIUS-only block:
// a free-text customer address that splitPostalAddress cannot take apart leaves
// three terms empty, and all three have to surface together.
func TestValidate_XRechnungReportsAddressPartsAtOnce(t *testing.T) {
	t.Parallel()

	inv := testInvoice(t)
	inv.CustomerAddress = "Postfach 12 34 56"

	settings := testSettings()
	settings.IBAN = ""

	err := Validate(inv, settings, "", ProfileXRechnung)
	assert.ElementsMatch(t, []string{
		"BT-10", // routing id
		"BG-16", // payment instructions
		"BT-52", // buyer city
		"BT-53", // buyer post code
	}, violationTerms(t, err))
}

// ============================================================================
// Individual requirements
// ============================================================================

func TestValidate_ReverseChargeNeedsBuyerVATID(t *testing.T) {
	t.Parallel()

	inv := testInvoice(t)
	inv.TaxMode = models.TaxModeReverseCharge
	inv.TotalTax = decimal.Zero
	inv.GrossTotal = inv.Subtotal
	inv.CustomerUStIDNr = ""

	err := Validate(inv, testSettings(), "", ProfileEN16931)
	assert.Equal(t, []string{"BT-48"}, violationTerms(t, err))
	// BR-AE-02 covers reverse charge on an invoice line (BG-25) and requires both
	// seller and buyer identification — this is its buyer half. BR-AE-03 is a
	// different rule (document-level allowance, BG-20) this product never emits.
	assert.Contains(t, err.Error(), "BR-AE-02")

	inv.CustomerUStIDNr = "ATU12345678"
	assert.NoError(t, Validate(inv, testSettings(), "", ProfileEN16931))
}

// TestValidate_ZeroRatedNeedsSellerIdentification covers the one VAT category
// TestValidate_SellerTaxRuleFollowsTheCategory cannot reach by flipping TaxMode:
// zero-rated is a per-line outcome (TaxMode stays "standard", a line's own rate
// is 0), not a document-wide setting like Kleinunternehmer or reverse charge.
func TestValidate_ZeroRatedNeedsSellerIdentification(t *testing.T) {
	t.Parallel()

	items := []models.LineItem{
		{Position: 1, Description: "Ausfuhrlieferung", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.RequireFromString("500.00"), TaxRate: decimal.Zero, LineTotal: decimal.RequireFromString("500.00")},
	}
	raw, err := json.Marshal(items)
	require.NoError(t, err)

	inv := models.Invoice{
		InvoiceNumber:   "RE-2026-0100",
		CustomerName:    "Beispielkunde AG",
		CustomerAddress: "Musterweg 1\n8001 Zürich",
		TaxMode:         models.TaxModeStandard,
		LineItems:       raw,
		Subtotal:        decimal.RequireFromString("500.00"),
		TotalTax:        decimal.Zero,
		GrossTotal:      decimal.RequireFromString("500.00"),
		Currency:        "EUR",
		InvoiceDate:     time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		DueDate:         time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC),
	}

	settings := testSettings()
	settings.UStIDNr = ""
	settings.Steuernummer = ""

	verr := Validate(inv, settings, "", ProfileEN16931)
	require.Error(t, verr)
	assert.Contains(t, verr.Error(), "BR-Z-02")
	assert.Equal(t, []string{"BT-31"}, violationTerms(t, verr))

	settings.Steuernummer = "143/815/08154"
	assert.NoError(t, Validate(inv, settings, "", ProfileEN16931))
}

// TestValidate_PaymentTermsSatisfyTheDueDateRequirement guards against rejecting
// invoices that state the payment deadline in words instead of as a date —
// BR-CO-25 asks for either, not for both.
func TestValidate_PaymentTermsSatisfyTheDueDateRequirement(t *testing.T) {
	t.Parallel()

	inv := testInvoice(t)
	inv.DueDate = time.Time{}
	assert.NoError(t, Validate(inv, testSettings(), "", ProfileEN16931),
		"payment terms alone satisfy BR-CO-25")

	inv.PaymentTerms = ""
	err := Validate(inv, testSettings(), "", ProfileEN16931)
	assert.Equal(t, []string{"BT-9"}, violationTerms(t, err))
}

// TestValidate_SellerTaxRuleFollowsTheCategory checks that the reported rule
// identifier is the one the receiver's validator will quote back.
func TestValidate_SellerTaxRuleFollowsTheCategory(t *testing.T) {
	t.Parallel()

	settings := testSettings()
	settings.UStIDNr = ""
	settings.Steuernummer = ""

	for _, tc := range []struct {
		name     string
		taxMode  string
		wantRule string
	}{
		{"standard rated", models.TaxModeStandard, "BR-S-02"},
		{"kleinunternehmer", models.TaxModeKleinunternehmer, "BR-E-02"},
		{"reverse charge", models.TaxModeReverseCharge, "BR-AE-02"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			inv := testInvoice(t)
			inv.TaxMode = tc.taxMode
			if tc.taxMode != models.TaxModeStandard {
				inv.TotalTax = decimal.Zero
				inv.GrossTotal = inv.Subtotal
			}

			err := Validate(inv, settings, "", ProfileEN16931)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantRule)
		})
	}
}

// ============================================================================
// Code lists (BR-CL-04, BR-CL-14)
// ============================================================================

func TestValidate_RejectsUnsupportedCurrency(t *testing.T) {
	t.Parallel()

	inv := testInvoice(t)
	inv.Currency = "EURO"

	err := Validate(inv, testSettings(), "", ProfileEN16931)
	assert.Equal(t, []string{"BT-5"}, violationTerms(t, err))
	assert.Contains(t, err.Error(), "BR-CL-04")

	for _, cur := range []string{"EUR", "CHF", "USD"} {
		inv.Currency = cur
		assert.NoError(t, Validate(inv, testSettings(), "", ProfileEN16931), "currency %s must be accepted", cur)
	}
}

// TestValidate_RejectsUnsupportedCountry uses a two-letter, real ISO 3166-1
// code outside the DACH whitelist ("FR") rather than a nonsense name: any
// unrecognised name of a different length already falls back to "DE" inside
// isoCountryCode (generator_doc.go) and would never reach this check — only a
// bare two-letter passthrough does. Both BT-40 and BT-55 fire, because
// buildBuyerParty currently derives the buyer country from the seller's own
// (see its lean comment) — a receiver's validator still reports them as two
// distinct terms.
func TestValidate_RejectsUnsupportedCountry(t *testing.T) {
	t.Parallel()

	settings := testSettings()
	settings.Country = "FR"

	err := Validate(testInvoice(t), settings, "", ProfileEN16931)
	assert.ElementsMatch(t, []string{"BT-40", "BT-55"}, violationTerms(t, err))
	assert.Contains(t, err.Error(), "BR-CL-14")

	for _, country := range []string{"Deutschland", "Österreich", "Schweiz", "AT", "CH"} {
		settings.Country = country
		assert.NoError(t, Validate(testInvoice(t), settings, "", ProfileEN16931), "country %s must be accepted", country)
	}
}

// ============================================================================
// BT-32 fallback
// ============================================================================

// TestGenerate_SellerTaxNumberIsTheFallbackIdentification covers the case that
// made the tax-identification rule actionable in the first place: a
// Kleinunternehmer has no VAT identifier by definition. Without writing the tax
// number as BT-32 their invoices would violate BR-E-02 with nothing the user
// could enter to fix it.
func TestGenerate_SellerTaxNumberIsTheFallbackIdentification(t *testing.T) {
	t.Parallel()

	settings := testSettings()
	settings.UStIDNr = ""
	settings.Steuernummer = "143/815/08154"

	inv := testInvoice(t)
	inv.TaxMode = models.TaxModeKleinunternehmer
	inv.TotalTax = decimal.Zero
	inv.GrossTotal = inv.Subtotal

	require.NoError(t, Validate(inv, settings, "", ProfileEN16931))

	ubl, err := GenerateUBL(inv, settings, "")
	require.NoError(t, err)
	assert.Contains(t, string(ubl), "<cbc:CompanyID>143/815/08154</cbc:CompanyID>")
	assert.Contains(t, string(ubl), "<cbc:ID>FC</cbc:ID>", "BT-32 uses tax scheme FC, not VAT")

	cii, err := GenerateCII(inv, settings, "")
	require.NoError(t, err)
	assert.Contains(t, string(cii), `<ram:ID schemeID="FC">143/815/08154</ram:ID>`)
}

// TestGenerate_VATIDWinsOverTaxNumber pins that only one tax registration is
// written: the inbound parser reads a single element and would otherwise take
// the tax number for the VAT identifier.
func TestGenerate_VATIDWinsOverTaxNumber(t *testing.T) {
	t.Parallel()

	settings := testSettings() // carries UStIDNr
	settings.Steuernummer = "143/815/08154"

	out, err := GenerateCII(testInvoice(t), settings, "")
	require.NoError(t, err)

	xml := string(out)
	assert.Contains(t, xml, `<ram:ID schemeID="VA">DE123456789</ram:ID>`)
	assert.NotContains(t, xml, "143/815/08154")
	assert.Equal(t, 0, strings.Count(xml, `schemeID="FC"`),
		"no second registration next to the VAT id, otherwise the parser picks up the wrong one")

	parsed, err := ParseCII(out)
	require.NoError(t, err)
	assert.Equal(t, "DE123456789", parsed.SupplierVATID)
}

// ============================================================================
// The generators refuse rather than ship a rejected document
// ============================================================================

func TestGenerators_RejectAnInvoiceThatWouldBeRefusedOnArrival(t *testing.T) {
	t.Parallel()

	inv := testInvoice(t)
	inv.CustomerName = ""

	for _, tc := range []struct {
		name string
		gen  func() ([]byte, error)
	}{
		{"UBL", func() ([]byte, error) { return GenerateUBL(inv, testSettings(), "991-12345-67") }},
		{"CII", func() ([]byte, error) { return GenerateCII(inv, testSettings(), "") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			out, err := tc.gen()
			assert.Nil(t, out, "no half-valid document may leave the generator")
			assert.ErrorIs(t, err, ErrValidationFailed)
			assert.Equal(t, []string{"BT-44"}, violationTerms(t, err))
		})
	}
}

// TestGenerators_ApplyTheCoreProfileOnly documents the deliberate split: the
// generators cannot tell whether the receiver is a public authority, so they
// enforce EN 16931 and leave the CIUS to the caller. A missing Leitweg-ID must
// not stop a private-sector invoice.
func TestGenerators_ApplyTheCoreProfileOnly(t *testing.T) {
	t.Parallel()

	inv := testInvoice(t)

	ubl, err := GenerateUBL(inv, testSettings(), "")
	require.NoError(t, err)
	assert.NotContains(t, string(ubl), "<cbc:BuyerReference>")

	err = Validate(inv, testSettings(), "", ProfileXRechnung)
	assert.ErrorIs(t, err, ErrValidationFailed, "the stricter profile still flags it")
}

// TestValidate_PropagatesBuildRejections makes sure the structural checks in
// buildInvoiceDoc are not shadowed: they come back as ErrGenerateFailed, not as
// an empty violation list.
func TestValidate_PropagatesBuildRejections(t *testing.T) {
	t.Parallel()

	inv := testInvoice(t)
	inv.LineItems = nil

	err := Validate(inv, testSettings(), "", ProfileEN16931)
	assert.ErrorIs(t, err, ErrGenerateFailed)
	assert.NotErrorIs(t, err, ErrValidationFailed)

	var ve *ValidationError
	assert.False(t, errors.As(err, &ve))
}
