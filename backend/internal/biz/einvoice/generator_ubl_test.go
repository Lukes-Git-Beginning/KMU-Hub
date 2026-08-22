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

// ============================================================================
// Fixtures
// ============================================================================

func testSettings() models.CompanySettings {
	return models.CompanySettings{
		Name:     "Zentria UG",
		Street:   "Musterstrasse 1",
		PLZ:      "80331",
		City:     "München",
		Country:  "Deutschland",
		UStIDNr:  "DE123456789",
		BankName: "Beispielbank",
		IBAN:     "DE02120300000000202051",
		BIC:      "BYLADEM1001",
	}
}

// testInvoice builds a two-rate invoice: 2 × 100.00 at 19 % and 3 × 50.00 at 7 %.
// Net 350.00, tax 38.00 + 10.50 = 48.50, gross 398.50.
func testInvoice(t *testing.T) models.Invoice {
	t.Helper()

	items := []models.LineItem{
		{
			Position: 1, Description: "Beratung & Analyse",
			Quantity:  decimal.NewFromInt(2),
			UnitPrice: decimal.RequireFromString("100.00"),
			TaxRate:   decimal.NewFromInt(19),
			LineTotal: decimal.RequireFromString("200.00"),
		},
		{
			Position: 2, Description: "Handbuch <gedruckt>",
			Quantity:  decimal.NewFromInt(3),
			UnitPrice: decimal.RequireFromString("50.00"),
			TaxRate:   decimal.NewFromInt(7),
			LineTotal: decimal.RequireFromString("150.00"),
		},
	}
	raw, err := json.Marshal(items)
	require.NoError(t, err)

	delivery := time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC)
	return models.Invoice{
		InvoiceNumber:   "RE-2026-0042",
		CustomerName:    "Stadtwerke Musterstadt AöR",
		CustomerAddress: "Rathausplatz 3\n12345 Musterstadt",
		CustomerUStIDNr: "DE987654321",
		TaxMode:         models.TaxModeStandard,
		LineItems:       raw,
		Subtotal:        decimal.RequireFromString("350.00"),
		TotalTax:        decimal.RequireFromString("48.50"),
		GrossTotal:      decimal.RequireFromString("398.50"),
		Currency:        "EUR",
		InvoiceDate:     time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		DeliveryDate:    &delivery,
		DueDate:         time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC),
		PaymentTerms:    "Zahlbar innerhalb von 14 Tagen ohne Abzug",
	}
}

// ============================================================================
// GenerateUBL — document shape
// ============================================================================

func TestGenerateUBL_EmitsXRechnungDocument(t *testing.T) {
	out, err := GenerateUBL(testInvoice(t), testSettings(), "04011000-12345-67")
	require.NoError(t, err)

	xmlStr := string(out)
	assert.True(t, strings.HasPrefix(xmlStr, `<?xml version="1.0" encoding="UTF-8"?>`), "XML declaration missing")

	// Namespaces and the XRechnung CIUS marker — without these a receiver rejects
	// the document even though it parses.
	assert.Contains(t, xmlStr, `xmlns="`+ublNamespaceInvoice+`"`)
	assert.Contains(t, xmlStr, `xmlns:cac="`+ublNamespaceCAC+`"`)
	assert.Contains(t, xmlStr, `xmlns:cbc="`+ublNamespaceCBC+`"`)
	assert.Contains(t, xmlStr, "<cbc:CustomizationID>"+xrechnungCustomizationID+"</cbc:CustomizationID>")

	// Mandatory header fields
	assert.Contains(t, xmlStr, "<cbc:ID>RE-2026-0042</cbc:ID>")
	assert.Contains(t, xmlStr, "<cbc:IssueDate>2026-08-01</cbc:IssueDate>")
	assert.Contains(t, xmlStr, "<cbc:DueDate>2026-08-15</cbc:DueDate>")
	assert.Contains(t, xmlStr, "<cbc:InvoiceTypeCode>380</cbc:InvoiceTypeCode>")
	assert.Contains(t, xmlStr, "<cbc:DocumentCurrencyCode>EUR</cbc:DocumentCurrencyCode>")
	assert.Contains(t, xmlStr, "<cbc:BuyerReference>04011000-12345-67</cbc:BuyerReference>")
	assert.Contains(t, xmlStr, "<cbc:ActualDeliveryDate>2026-07-30</cbc:ActualDeliveryDate>")

	// Seller VAT ID and payment details
	assert.Contains(t, xmlStr, "<cbc:CompanyID>DE123456789</cbc:CompanyID>")
	assert.Contains(t, xmlStr, "<cbc:ID>DE02120300000000202051</cbc:ID>")
	assert.Contains(t, xmlStr, "<cbc:ID>BYLADEM1001</cbc:ID>")
	assert.Contains(t, xmlStr, "<cbc:PaymentMeansCode>58</cbc:PaymentMeansCode>")

	// Amounts carry the currency attribute and two decimals, never a float rendering.
	assert.Contains(t, xmlStr, `<cbc:PayableAmount currencyID="EUR">398.50</cbc:PayableAmount>`)
	assert.Contains(t, xmlStr, `<cbc:TaxExclusiveAmount currencyID="EUR">350.00</cbc:TaxExclusiveAmount>`)

	// Markup in free-text fields is escaped, not injected.
	assert.Contains(t, xmlStr, "Handbuch &lt;gedruckt&gt;")
	assert.NotContains(t, xmlStr, "<gedruckt>")

	// One TaxSubtotal per rate, in first-seen order.
	assert.Equal(t, 2, strings.Count(xmlStr, "<cac:TaxSubtotal>"))
	assert.Less(t, strings.Index(xmlStr, "<cbc:Percent>19.00</cbc:Percent>"),
		strings.Index(xmlStr, "<cbc:Percent>7.00</cbc:Percent>"))
}

func TestGenerateUBL_DetectedAsUBL(t *testing.T) {
	out, err := GenerateUBL(testInvoice(t), testSettings(), "")
	require.NoError(t, err)
	assert.Equal(t, "xrechnung_ubl", DetectFormat(out))
}

func TestGenerateUBL_OmitsBuyerReferenceWhenEmpty(t *testing.T) {
	out, err := GenerateUBL(testInvoice(t), testSettings(), "  ")
	require.NoError(t, err)
	assert.NotContains(t, string(out), "<cbc:BuyerReference>")
}

// ============================================================================
// Round trip — the generated document must survive the inbound parser
// ============================================================================

func TestGenerateUBL_RoundTripThroughParser(t *testing.T) {
	invoice := testInvoice(t)
	settings := testSettings()

	out, err := GenerateUBL(invoice, settings, "04011000-12345-67")
	require.NoError(t, err)

	parsed, err := ParseUBL(out)
	require.NoError(t, err)

	assert.Equal(t, "RE-2026-0042", parsed.InvoiceNumber)
	assert.Equal(t, "EUR", parsed.Currency)
	assert.Equal(t, "xrechnung_ubl", parsed.SourceFormat)
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
// Tax modes
// ============================================================================

func TestGenerateUBL_TaxModes(t *testing.T) {
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

			out, err := GenerateUBL(invoice, testSettings(), "")
			require.NoError(t, err)

			xmlStr := string(out)
			assert.Contains(t, xmlStr, "<cbc:ID>"+tt.wantCategory+"</cbc:ID>")
			assert.Contains(t, xmlStr, "<cbc:TaxExemptionReason>"+tt.wantExemptReason+"</cbc:TaxExemptionReason>")
			// BR-AE-05 / BR-E-05: the rate on such a line must be zero.
			assert.NotContains(t, xmlStr, "<cbc:Percent>19.00</cbc:Percent>")
			assert.Contains(t, xmlStr, "<cbc:Percent>0.00</cbc:Percent>")
			assert.Contains(t, xmlStr, `<cbc:TaxAmount currencyID="EUR">0.00</cbc:TaxAmount>`)

			// Both rates collapse into a single exempt category.
			assert.Equal(t, 1, strings.Count(xmlStr, "<cac:TaxSubtotal>"))

			parsed, err := ParseUBL(out)
			require.NoError(t, err)
			assert.True(t, parsed.GrossTotal.Equal(invoice.Subtotal))
			assert.True(t, parsed.TotalTax.IsZero())
		})
	}
}

func TestGenerateUBL_ZeroRatedLineUsesCategoryZ(t *testing.T) {
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

	out, err := GenerateUBL(invoice, testSettings(), "")
	require.NoError(t, err)
	assert.Contains(t, string(out), "<cbc:ID>"+taxCategoryZeroRated+"</cbc:ID>")
}

// ============================================================================
// Rejections
// ============================================================================

func TestGenerateUBL_Rejections(t *testing.T) {
	t.Run("missing invoice number", func(t *testing.T) {
		invoice := testInvoice(t)
		invoice.InvoiceNumber = "  "
		_, err := GenerateUBL(invoice, testSettings(), "")
		require.ErrorIs(t, err, ErrGenerateFailed)
	})

	t.Run("missing issue date", func(t *testing.T) {
		invoice := testInvoice(t)
		invoice.InvoiceDate = time.Time{}
		_, err := GenerateUBL(invoice, testSettings(), "")
		require.ErrorIs(t, err, ErrGenerateFailed)
	})

	t.Run("no line items", func(t *testing.T) {
		invoice := testInvoice(t)
		invoice.LineItems = json.RawMessage(`[]`)
		_, err := GenerateUBL(invoice, testSettings(), "")
		require.ErrorIs(t, err, ErrGenerateFailed)
	})

	t.Run("unparseable line items", func(t *testing.T) {
		invoice := testInvoice(t)
		invoice.LineItems = json.RawMessage(`{"broken":`)
		_, err := GenerateUBL(invoice, testSettings(), "")
		require.ErrorIs(t, err, ErrGenerateFailed)
	})

	// A stale stored total must never ship silently — the customer would receive a
	// different amount than the PDF shows.
	t.Run("stored totals disagree with the lines", func(t *testing.T) {
		invoice := testInvoice(t)
		invoice.GrossTotal = decimal.RequireFromString("500.00")
		_, err := GenerateUBL(invoice, testSettings(), "")
		require.ErrorIs(t, err, ErrTotalsMismatch)
	})

	// One cent of per-line rounding drift stays acceptable.
	t.Run("rounding drift within one cent passes", func(t *testing.T) {
		invoice := testInvoice(t)
		invoice.TotalTax = decimal.RequireFromString("48.51")
		invoice.GrossTotal = decimal.RequireFromString("398.51")
		_, err := GenerateUBL(invoice, testSettings(), "")
		require.NoError(t, err)
	})

	// The write path (invoice.Service, tax.Calculate) sums UNROUNDED line nets while
	// this generator rounds each net to cents first (BR-CO-10). Four lines that each
	// drift half a cent put the two orders two cents apart — past the flat 0.01 this
	// guard used to carry, which would have refused to export an ordinary invoice.
	t.Run("write-path rounding order stays acceptable on many lines", func(t *testing.T) {
		invoice := halfCentDriftInvoice(t)
		// What tax.Calculate stores today: the unrounded nets sum to 199.98 exactly,
		// against the document's sum of rounded nets, 200.00.
		invoice.Subtotal = decimal.RequireFromString("199.98")
		invoice.TotalTax = decimal.RequireFromString("38.00")
		invoice.GrossTotal = decimal.RequireFromString("237.98")
		_, err := GenerateUBL(invoice, testSettings(), "")
		require.NoError(t, err)
	})

	// Scaling the tolerance with the line count must not turn the guard off: a stale
	// figure on the same invoice still has to be caught.
	t.Run("stale total is still caught on many lines", func(t *testing.T) {
		invoice := halfCentDriftInvoice(t)
		invoice.GrossTotal = decimal.RequireFromString("238.50")
		_, err := GenerateUBL(invoice, testSettings(), "")
		require.ErrorIs(t, err, ErrTotalsMismatch)
	})
}

// ============================================================================
// Neutral document model
// ============================================================================

func TestBuildInvoiceDoc_DerivesNetFromQuantityWhenLineTotalMissing(t *testing.T) {
	invoice := testInvoice(t)
	items := []models.LineItem{{
		Position: 1, Description: "Ohne gespeicherte Zeilensumme",
		Quantity:  decimal.NewFromInt(4),
		UnitPrice: decimal.RequireFromString("25.00"),
		TaxRate:   decimal.NewFromInt(19),
		// LineTotal deliberately left at zero.
	}}
	raw, err := json.Marshal(items)
	require.NoError(t, err)
	invoice.LineItems = raw
	invoice.Subtotal = decimal.RequireFromString("100.00")
	invoice.TotalTax = decimal.RequireFromString("19.00")
	invoice.GrossTotal = decimal.RequireFromString("119.00")

	doc, err := buildInvoiceDoc(invoice, testSettings(), "")
	require.NoError(t, err)
	require.Len(t, doc.Lines, 1)
	assert.True(t, doc.Lines[0].Net.Equal(decimal.RequireFromString("100.00")))
	assert.True(t, doc.GrossTotal.Equal(decimal.RequireFromString("119.00")))
}

func TestBuildInvoiceDoc_FallsBackToDefaultsForCurrencyAndDelivery(t *testing.T) {
	invoice := testInvoice(t)
	invoice.Currency = ""
	invoice.DeliveryDate = nil

	doc, err := buildInvoiceDoc(invoice, testSettings(), "")
	require.NoError(t, err)
	assert.Equal(t, models.DefaultCurrency, doc.Currency)
	assert.Equal(t, invoice.InvoiceDate, doc.DeliveryDate, "delivery date falls back to the issue date (BT-72)")
}

func TestSplitPostalAddress(t *testing.T) {
	tests := []struct {
		name                     string
		raw                      string
		street, postalZone, city string
	}{
		{"street and postal line", "Rathausplatz 3\n12345 Musterstadt", "Rathausplatz 3", "12345", "Musterstadt"},
		{"four digit postal code", "Hauptgasse 9\n8001 Zürich", "Hauptgasse 9", "8001", "Zürich"},
		{"multi word city", "Am Markt 1\n80331 Bad Reichenhall", "Am Markt 1", "80331", "Bad Reichenhall"},
		{"extra line kept in street", "c/o Zentrale\nRathausplatz 3\n12345 Musterstadt", "c/o Zentrale, Rathausplatz 3", "12345", "Musterstadt"},
		{"single line stays intact", "Rathausplatz 3, 12345 Musterstadt", "Rathausplatz 3, 12345 Musterstadt", "", ""},
		{"no postal code stays intact", "Rathausplatz 3\nMusterstadt", "Rathausplatz 3, Musterstadt", "", ""},
		{"empty", "   ", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			street, postalZone, city := splitPostalAddress(tt.raw)
			assert.Equal(t, tt.street, street)
			assert.Equal(t, tt.postalZone, postalZone)
			assert.Equal(t, tt.city, city)
		})
	}
}

func TestGenerateUBL_BuyerAddressIsStructured(t *testing.T) {
	out, err := GenerateUBL(testInvoice(t), testSettings(), "")
	require.NoError(t, err)

	xmlStr := string(out)
	assert.Contains(t, xmlStr, "<cbc:StreetName>Rathausplatz 3</cbc:StreetName>")
	assert.Contains(t, xmlStr, "<cbc:CityName>Musterstadt</cbc:CityName>")
	assert.Contains(t, xmlStr, "<cbc:PostalZone>12345</cbc:PostalZone>")
}

func TestIsoCountryCode(t *testing.T) {
	assert.Equal(t, "DE", isoCountryCode("Deutschland"))
	assert.Equal(t, "DE", isoCountryCode(""))
	assert.Equal(t, "AT", isoCountryCode("Österreich"))
	assert.Equal(t, "CH", isoCountryCode("Schweiz"))
	assert.Equal(t, "FR", isoCountryCode("fr"))
	assert.Equal(t, "DE", isoCountryCode("Irgendwo"))
}

// halfCentDriftInvoice is four identical lines of 1.5 x 33.33 = 49.995 at 19 %.
// Every net sits on a half cent and drifts the same way, so the two rounding orders
// separate by the full two cents: 4 x 50.00 = 200.00 against round(199.98) = 199.98.
// That is the smallest invoice on which the old flat 0.01 tolerance was wrong.
func halfCentDriftInvoice(t *testing.T) models.Invoice {
	t.Helper()

	dec := decimal.RequireFromString
	items := make([]models.LineItem, 0, 4)
	for i := 1; i <= 4; i++ {
		items = append(items, models.LineItem{
			Position:    i,
			Description: "Projektstunden",
			Quantity:    dec("1.5"),
			UnitPrice:   dec("33.33"),
			TaxRate:     decimal.NewFromInt(19),
		})
	}
	raw, err := json.Marshal(items)
	require.NoError(t, err)

	invoice := testInvoice(t)
	invoice.LineItems = raw
	invoice.Subtotal = dec("200.00")
	invoice.TotalTax = dec("38.00")
	invoice.GrossTotal = dec("238.00")
	return invoice
}
