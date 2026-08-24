package einvoice

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/shopspring/decimal"
)

// Fixtures testInvoice/testSettings live in generator_ubl_test.go — both writers
// render the same invoice on purpose, which is what TestOutboundFormatsAgree
// checks.

// ============================================================================
// GenerateCII — document shape
// ============================================================================

func TestGenerateCII_EmitsFacturXDocument(t *testing.T) {
	out, err := GenerateCII(testInvoice(t), testSettings(), "04011000-12345-67")
	require.NoError(t, err)

	xmlStr := string(out)
	assert.True(t, strings.HasPrefix(xmlStr, `<?xml version="1.0" encoding="UTF-8"?>`), "XML declaration missing")

	// Namespaces and the EN 16931 guideline marker — a receiver picks its
	// validation profile from the guideline ID and rejects the document without it.
	assert.Contains(t, xmlStr, `xmlns:rsm="`+ciiNamespaceRSM+`"`)
	assert.Contains(t, xmlStr, `xmlns:ram="`+ciiNamespaceRAM+`"`)
	assert.Contains(t, xmlStr, `xmlns:udt="`+ciiNamespaceUDT+`"`)
	assert.Contains(t, xmlStr, "<ram:ID>"+facturXEN16931GuidelineID+"</ram:ID>")

	// Mandatory header fields. CII dates are CCYYMMDD with the format code.
	assert.Contains(t, xmlStr, "<ram:ID>RE-2026-0042</ram:ID>")
	assert.Contains(t, xmlStr, "<ram:TypeCode>380</ram:TypeCode>")
	assert.Contains(t, xmlStr, `<udt:DateTimeString format="102">20260801</udt:DateTimeString>`)
	assert.Contains(t, xmlStr, `<udt:DateTimeString format="102">20260815</udt:DateTimeString>`)
	assert.Contains(t, xmlStr, `<udt:DateTimeString format="102">20260730</udt:DateTimeString>`)
	assert.Contains(t, xmlStr, "<ram:InvoiceCurrencyCode>EUR</ram:InvoiceCurrencyCode>")
	assert.Contains(t, xmlStr, "<ram:BuyerReference>04011000-12345-67</ram:BuyerReference>")

	// Seller VAT registration and SEPA payment details.
	assert.Contains(t, xmlStr, `<ram:ID schemeID="VA">DE123456789</ram:ID>`)
	assert.Contains(t, xmlStr, "<ram:IBANID>DE02120300000000202051</ram:IBANID>")
	assert.Contains(t, xmlStr, "<ram:BICID>BYLADEM1001</ram:BICID>")
	assert.Contains(t, xmlStr, "<ram:TypeCode>58</ram:TypeCode>")

	// Amounts always carry two decimals, never a float rendering.
	assert.Contains(t, xmlStr, "<ram:LineTotalAmount>350.00</ram:LineTotalAmount>")
	assert.Contains(t, xmlStr, "<ram:TaxBasisTotalAmount>350.00</ram:TaxBasisTotalAmount>")
	assert.Contains(t, xmlStr, `<ram:TaxTotalAmount currencyID="EUR">48.50</ram:TaxTotalAmount>`)
	assert.Contains(t, xmlStr, "<ram:GrandTotalAmount>398.50</ram:GrandTotalAmount>")
	assert.Contains(t, xmlStr, "<ram:DuePayableAmount>398.50</ram:DuePayableAmount>")

	// Markup in free-text fields is escaped, not injected. The previous writer did
	// this by hand; encoding/xml now guarantees it.
	assert.Contains(t, xmlStr, "Handbuch &lt;gedruckt&gt;")
	assert.NotContains(t, xmlStr, "<gedruckt>")

	// One header ApplicableTradeTax per rate. CalculatedAmount appears only in the
	// header groups, so counting it counts the BG-23 breakdown without picking up
	// the per-line ApplicableTradeTax blocks.
	assert.Equal(t, 2, strings.Count(xmlStr, "<ram:CalculatedAmount>"), "one BG-23 group per rate")
	// Category code on both levels: two header groups plus two lines.
	assert.Equal(t, 4, strings.Count(xmlStr, "<ram:CategoryCode>S</ram:CategoryCode>"))
	assert.Contains(t, xmlStr, "<ram:BasisAmount>200.00</ram:BasisAmount>")
	assert.Contains(t, xmlStr, "<ram:CalculatedAmount>38.00</ram:CalculatedAmount>")
	assert.Contains(t, xmlStr, "<ram:BasisAmount>150.00</ram:BasisAmount>")
	assert.Contains(t, xmlStr, "<ram:CalculatedAmount>10.50</ram:CalculatedAmount>")
}

// TestGenerateCII_LineItemsPrecedeHeaderGroups pins the CII XSD sequence.
// IncludedSupplyChainTradeLineItem comes before the three header groups; the
// previous string-template writer appended the lines last, which parses here but
// fails schema validation at the receiver.
func TestGenerateCII_LineItemsPrecedeHeaderGroups(t *testing.T) {
	out, err := GenerateCII(testInvoice(t), testSettings(), "")
	require.NoError(t, err)

	xmlStr := string(out)
	firstLine := strings.Index(xmlStr, "<ram:IncludedSupplyChainTradeLineItem>")
	agreement := strings.Index(xmlStr, "<ram:ApplicableHeaderTradeAgreement>")
	delivery := strings.Index(xmlStr, "<ram:ApplicableHeaderTradeDelivery>")
	settlement := strings.Index(xmlStr, "<ram:ApplicableHeaderTradeSettlement>")

	require.NotEqual(t, -1, firstLine)
	assert.Less(t, firstLine, agreement, "line items must precede ApplicableHeaderTradeAgreement")
	assert.Less(t, agreement, delivery)
	assert.Less(t, delivery, settlement)
}

func TestGenerateCII_DetectedAsCII(t *testing.T) {
	out, err := GenerateCII(testInvoice(t), testSettings(), "")
	require.NoError(t, err)
	assert.Equal(t, "zugferd_cii", DetectFormat(out))
}

func TestGenerateCII_OmitsBuyerReferenceWhenEmpty(t *testing.T) {
	out, err := GenerateCII(testInvoice(t), testSettings(), "  ")
	require.NoError(t, err)
	assert.NotContains(t, string(out), "<ram:BuyerReference>")
}

// ============================================================================
// Round trip — the generated document must survive the inbound parser
// ============================================================================

func TestGenerateCII_RoundTripThroughParser(t *testing.T) {
	invoice := testInvoice(t)
	settings := testSettings()

	out, err := GenerateCII(invoice, settings, "04011000-12345-67")
	require.NoError(t, err)

	parsed, err := ParseCII(out)
	require.NoError(t, err)

	assert.Equal(t, "RE-2026-0042", parsed.InvoiceNumber)
	assert.Equal(t, "EUR", parsed.Currency)
	assert.Equal(t, "zugferd_cii", parsed.SourceFormat)
	assert.Equal(t, settings.Name, parsed.SupplierName)
	assert.Equal(t, settings.UStIDNr, parsed.SupplierVATID)
	assert.Equal(t, invoice.InvoiceDate, parsed.InvoiceDate)
	require.NotNil(t, parsed.DueDate)
	assert.Equal(t, invoice.DueDate, *parsed.DueDate)

	// Totals survive unchanged.
	assert.True(t, parsed.Subtotal.Equal(invoice.Subtotal), "subtotal: got %s", parsed.Subtotal)
	assert.True(t, parsed.TotalTax.Equal(invoice.TotalTax), "total tax: got %s", parsed.TotalTax)
	assert.True(t, parsed.GrossTotal.Equal(invoice.GrossTotal), "gross total: got %s", parsed.GrossTotal)

	// Line items survive with quantity, unit price, rate and net.
	require.Len(t, parsed.LineItems, 2)
	assert.Equal(t, "Beratung & Analyse", parsed.LineItems[0].Description)
	assert.True(t, parsed.LineItems[0].Quantity.Equal(decimal.NewFromInt(2)))
	assert.True(t, parsed.LineItems[0].UnitPrice.Equal(decimal.RequireFromString("100.00")))
	assert.True(t, parsed.LineItems[0].TaxRate.Equal(decimal.NewFromInt(19)))
	assert.True(t, parsed.LineItems[0].LineTotal.Equal(decimal.RequireFromString("200.00")))
	assert.Equal(t, "Handbuch <gedruckt>", parsed.LineItems[1].Description)
	assert.True(t, parsed.LineItems[1].LineTotal.Equal(decimal.RequireFromString("150.00")))

	// Tax breakdown survives per rate.
	require.Len(t, parsed.TaxBreakdown, 2)
	assert.True(t, parsed.TaxBreakdown[0].TaxRate.Equal(decimal.NewFromInt(19)))
	assert.True(t, parsed.TaxBreakdown[0].TaxableNet.Equal(decimal.RequireFromString("200.00")))
	assert.True(t, parsed.TaxBreakdown[0].TaxAmount.Equal(decimal.RequireFromString("38.00")))
	assert.True(t, parsed.TaxBreakdown[1].TaxRate.Equal(decimal.NewFromInt(7)))
	assert.True(t, parsed.TaxBreakdown[1].TaxableNet.Equal(decimal.RequireFromString("150.00")))
	assert.True(t, parsed.TaxBreakdown[1].TaxAmount.Equal(decimal.RequireFromString("10.50")))
}

// ============================================================================
// The two defects the merge fixed
// ============================================================================

// TestGenerateCII_TaxBasisMatchesLineAmounts covers BR-S-08: the taxable amount of
// a VAT category must equal the sum of the line net amounts written into the same
// document. The previous writer took quantity × unit price as the basis but wrote
// the stored line total as the line amount, so any invoice whose line total was
// rounded or discounted shipped a self-contradicting document.
func TestGenerateCII_TaxBasisMatchesLineAmounts(t *testing.T) {
	invoice := testInvoice(t)
	items := []models.LineItem{{
		Position: 1, Description: "Rabattierte Position",
		Quantity:  decimal.NewFromInt(3),
		UnitPrice: decimal.RequireFromString("100.00"),
		TaxRate:   decimal.NewFromInt(19),
		// 300.00 before the discount — quantity × unit price disagrees on purpose.
		LineTotal: decimal.RequireFromString("270.00"),
	}}
	raw, err := json.Marshal(items)
	require.NoError(t, err)
	invoice.LineItems = raw
	invoice.Subtotal = decimal.RequireFromString("270.00")
	invoice.TotalTax = decimal.RequireFromString("51.30")
	invoice.GrossTotal = decimal.RequireFromString("321.30")

	out, err := GenerateCII(invoice, testSettings(), "")
	require.NoError(t, err)

	xmlStr := string(out)
	assert.Contains(t, xmlStr, "<ram:BasisAmount>270.00</ram:BasisAmount>", "tax basis must be the line net")
	assert.NotContains(t, xmlStr, "<ram:BasisAmount>300.00</ram:BasisAmount>", "quantity × unit price must not be the basis")

	parsed, err := ParseCII(out)
	require.NoError(t, err)
	require.Len(t, parsed.TaxBreakdown, 1)
	require.Len(t, parsed.LineItems, 1)
	assert.True(t, parsed.TaxBreakdown[0].TaxableNet.Equal(parsed.LineItems[0].LineTotal),
		"BR-S-08: category basis %s must equal the line net %s",
		parsed.TaxBreakdown[0].TaxableNet, parsed.LineItems[0].LineTotal)
}

// TestGenerateCII_ZeroRatedLineUsesCategoryZ covers the second defect: the
// previous writer emitted category S even for a 0 % line, which is formally
// invalid — a standard-rated line cannot carry a zero rate.
func TestGenerateCII_ZeroRatedLineUsesCategoryZ(t *testing.T) {
	invoice := testInvoice(t)
	items := []models.LineItem{{
		Position: 1, Description: "Innergemeinschaftliche Lieferung",
		Quantity:  decimal.NewFromInt(1),
		UnitPrice: decimal.RequireFromString("100.00"),
		TaxRate:   decimal.Zero,
		LineTotal: decimal.RequireFromString("100.00"),
	}}
	raw, err := json.Marshal(items)
	require.NoError(t, err)
	invoice.LineItems = raw
	invoice.Subtotal = decimal.RequireFromString("100.00")
	invoice.TotalTax = decimal.Zero
	invoice.GrossTotal = decimal.RequireFromString("100.00")

	out, err := GenerateCII(invoice, testSettings(), "")
	require.NoError(t, err)

	xmlStr := string(out)
	assert.Contains(t, xmlStr, "<ram:CategoryCode>"+taxCategoryZeroRated+"</ram:CategoryCode>")
	assert.NotContains(t, xmlStr, "<ram:CategoryCode>S</ram:CategoryCode>")
}

// TestOutboundFormatsAgree is the regression guard for the whole merge: both
// writers must report the same amounts for the same invoice. Two renderers with
// their own amount logic is how they drifted apart in the first place.
func TestOutboundFormatsAgree(t *testing.T) {
	invoice := testInvoice(t)
	settings := testSettings()

	ublBytes, err := GenerateUBL(invoice, settings, "04011000-12345-67")
	require.NoError(t, err)
	ciiBytes, err := GenerateCII(invoice, settings, "04011000-12345-67")
	require.NoError(t, err)

	fromUBL, err := ParseUBL(ublBytes)
	require.NoError(t, err)
	fromCII, err := ParseCII(ciiBytes)
	require.NoError(t, err)

	assert.True(t, fromUBL.Subtotal.Equal(fromCII.Subtotal), "subtotal: UBL %s vs CII %s", fromUBL.Subtotal, fromCII.Subtotal)
	assert.True(t, fromUBL.TotalTax.Equal(fromCII.TotalTax), "total tax: UBL %s vs CII %s", fromUBL.TotalTax, fromCII.TotalTax)
	assert.True(t, fromUBL.GrossTotal.Equal(fromCII.GrossTotal), "gross total: UBL %s vs CII %s", fromUBL.GrossTotal, fromCII.GrossTotal)

	require.Equal(t, len(fromUBL.LineItems), len(fromCII.LineItems))
	for i := range fromUBL.LineItems {
		assert.True(t, fromUBL.LineItems[i].LineTotal.Equal(fromCII.LineItems[i].LineTotal), "line %d net", i+1)
		assert.True(t, fromUBL.LineItems[i].TaxRate.Equal(fromCII.LineItems[i].TaxRate), "line %d rate", i+1)
	}

	require.Equal(t, len(fromUBL.TaxBreakdown), len(fromCII.TaxBreakdown))
	for i := range fromUBL.TaxBreakdown {
		assert.True(t, fromUBL.TaxBreakdown[i].TaxableNet.Equal(fromCII.TaxBreakdown[i].TaxableNet), "tax group %d basis", i+1)
		assert.True(t, fromUBL.TaxBreakdown[i].TaxAmount.Equal(fromCII.TaxBreakdown[i].TaxAmount), "tax group %d amount", i+1)
	}
}

// ============================================================================
// Tax modes and rejections
// ============================================================================

func TestGenerateCII_TaxModes(t *testing.T) {
	tests := []struct {
		name             string
		taxMode          string
		wantCategory     string
		wantExemptReason string
	}{
		{"reverse charge", models.TaxModeReverseCharge, taxCategoryReverseCharge, exemptionReasonReverseCharge},
		{"kleinunternehmer", models.TaxModeKleinunternehmer, taxCategoryExempt, exemptionReasonKleinunternehmer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invoice := testInvoice(t)
			invoice.TaxMode = tt.taxMode
			// A tax-free invoice carries no VAT: the stored totals must say so too,
			// otherwise the totals guard rejects it.
			invoice.TotalTax = decimal.Zero
			invoice.GrossTotal = invoice.Subtotal

			out, err := GenerateCII(invoice, testSettings(), "")
			require.NoError(t, err)

			xmlStr := string(out)
			assert.Contains(t, xmlStr, "<ram:CategoryCode>"+tt.wantCategory+"</ram:CategoryCode>")
			assert.Contains(t, xmlStr, "<ram:ExemptionReason>"+tt.wantExemptReason+"</ram:ExemptionReason>")
			// BR-AE-05 / BR-E-05: the rate on such a line must be zero.
			assert.NotContains(t, xmlStr, "<ram:RateApplicablePercent>19.00</ram:RateApplicablePercent>")
			assert.Contains(t, xmlStr, "<ram:RateApplicablePercent>0.00</ram:RateApplicablePercent>")

			// Both rates collapse into a single exempt category in the header.
			assert.Equal(t, 1, strings.Count(xmlStr, "<ram:ExemptionReason>"))

			parsed, err := ParseCII(out)
			require.NoError(t, err)
			assert.True(t, parsed.GrossTotal.Equal(invoice.Subtotal))
			assert.True(t, parsed.TotalTax.IsZero())
		})
	}
}

func TestGenerateCII_Rejections(t *testing.T) {
	t.Run("missing invoice number", func(t *testing.T) {
		invoice := testInvoice(t)
		invoice.InvoiceNumber = "  "
		_, err := GenerateCII(invoice, testSettings(), "")
		require.ErrorIs(t, err, ErrGenerateFailed)
		assert.Contains(t, err.Error(), "BR-02", "the rejection must be searchable against a receiver's validation report")
	})

	t.Run("missing issue date", func(t *testing.T) {
		invoice := testInvoice(t)
		invoice.InvoiceDate = time.Time{}
		_, err := GenerateCII(invoice, testSettings(), "")
		require.ErrorIs(t, err, ErrGenerateFailed)
		assert.Contains(t, err.Error(), "BR-03")
	})

	t.Run("no line items", func(t *testing.T) {
		invoice := testInvoice(t)
		invoice.LineItems = json.RawMessage(`[]`)
		_, err := GenerateCII(invoice, testSettings(), "")
		require.ErrorIs(t, err, ErrGenerateFailed)
		assert.Contains(t, err.Error(), "BR-16")
	})

	// A stale stored total must never ship silently — the customer would receive a
	// different amount than the PDF shows.
	t.Run("stored totals disagree with the lines", func(t *testing.T) {
		invoice := testInvoice(t)
		invoice.GrossTotal = decimal.RequireFromString("500.00")
		_, err := GenerateCII(invoice, testSettings(), "")
		require.ErrorIs(t, err, ErrTotalsMismatch)
	})
}

// TestGenerateCII_OmitsPaymentMeansWithoutIBAN keeps an empty SEPA block out of
// the document — an ApplicableHeaderTradeSettlement carrying an empty IBANID is
// rejected by validators.
func TestGenerateCII_OmitsPaymentMeansWithoutIBAN(t *testing.T) {
	settings := testSettings()
	settings.IBAN = ""
	settings.BIC = ""

	out, err := GenerateCII(testInvoice(t), settings, "")
	require.NoError(t, err)
	assert.NotContains(t, string(out), "<ram:SpecifiedTradeSettlementPaymentMeans>")
}
