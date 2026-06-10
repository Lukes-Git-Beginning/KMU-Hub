package einvoice

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/biz/pdf"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/shopspring/decimal"
)

// ============================================================================
// DetectFormat
// ============================================================================

func TestDetectFormat_CII(t *testing.T) {
	xmlData, err := os.ReadFile("testdata/cii_minimal.xml")
	require.NoError(t, err)
	assert.Equal(t, "zugferd_cii", DetectFormat(xmlData))
}

func TestDetectFormat_UBL(t *testing.T) {
	xmlData, err := os.ReadFile("testdata/ubl_minimal.xml")
	require.NoError(t, err)
	assert.Equal(t, "xrechnung_ubl", DetectFormat(xmlData))
}

func TestDetectFormat_Unknown(t *testing.T) {
	xml := []byte(`<?xml version="1.0"?><SomeOtherDocument><ID>1</ID></SomeOtherDocument>`)
	assert.Equal(t, "xml_only", DetectFormat(xml))
}

// ============================================================================
// ParseCII
// ============================================================================

func TestParseCII_FromFixture(t *testing.T) {
	xmlData, err := os.ReadFile("testdata/cii_minimal.xml")
	require.NoError(t, err)

	parsed, err := ParseCII(xmlData)
	require.NoError(t, err)

	assert.Equal(t, "Lieferant GmbH", parsed.SupplierName)
	assert.Equal(t, "DE123456789", parsed.SupplierVATID)
	assert.Equal(t, "RE-TEST-2024-001", parsed.InvoiceNumber)
	assert.Equal(t, "EUR", parsed.Currency)

	expectedDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, expectedDate, parsed.InvoiceDate)

	require.NotNil(t, parsed.DueDate)
	expectedDue := time.Date(2024, 2, 14, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, expectedDue, *parsed.DueDate)

	assert.Equal(t, "300.00", parsed.Subtotal.StringFixed(2))
	assert.Equal(t, "57.00", parsed.TotalTax.StringFixed(2))
	assert.Equal(t, "357.00", parsed.GrossTotal.StringFixed(2))

	require.Len(t, parsed.LineItems, 1)
	li := parsed.LineItems[0]
	assert.Equal(t, "Beratungsleistung", li.Description)
	assert.Equal(t, "150.00", li.UnitPrice.StringFixed(2))
	assert.Equal(t, "2.00", li.Quantity.StringFixed(2))
	assert.Equal(t, "300.00", li.LineTotal.StringFixed(2))
	assert.Equal(t, "19.00", li.TaxRate.StringFixed(2))

	assert.Equal(t, "zugferd_cii", parsed.SourceFormat)
}

// TestParseCII_Roundtrip uses GenerateZUGFeRDXML to produce CII XML,
// then parses it back — testing end-to-end roundtrip compatibility.
func TestParseCII_Roundtrip(t *testing.T) {
	inv := buildTestInvoice()
	settings := buildTestSettings()

	xmlBytes, err := pdf.GenerateZUGFeRDXML(inv, settings)
	require.NoError(t, err)
	require.NotEmpty(t, xmlBytes)

	parsed, err := ParseCII(xmlBytes)
	require.NoError(t, err)

	assert.Equal(t, settings.Name, parsed.SupplierName)
	assert.Equal(t, settings.UStIDNr, parsed.SupplierVATID)
	assert.Equal(t, inv.InvoiceNumber, parsed.InvoiceNumber)
	assert.Equal(t, "EUR", parsed.Currency)
	assert.Equal(t, inv.Subtotal.StringFixed(2), parsed.Subtotal.StringFixed(2))
	assert.Equal(t, inv.TotalTax.StringFixed(2), parsed.TotalTax.StringFixed(2))
	assert.Equal(t, inv.GrossTotal.StringFixed(2), parsed.GrossTotal.StringFixed(2))
	assert.NotEmpty(t, parsed.LineItems)
}

func TestParseCII_MissingID(t *testing.T) {
	xml := []byte(`<?xml version="1.0"?>
<rsm:CrossIndustryInvoice
	xmlns:rsm="urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100"
	xmlns:ram="urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100"
	xmlns:udt="urn:un:unece:uncefact:data:standard:UnqualifiedDataType:100">
	<rsm:ExchangedDocument>
		<ram:ID></ram:ID>
		<ram:IssueDateTime><udt:DateTimeString format="102">20240101</udt:DateTimeString></ram:IssueDateTime>
	</rsm:ExchangedDocument>
	<rsm:SupplyChainTradeTransaction>
		<ram:ApplicableHeaderTradeAgreement>
			<ram:SellerTradeParty><ram:Name>X</ram:Name></ram:SellerTradeParty>
			<ram:BuyerTradeParty><ram:Name>Y</ram:Name></ram:BuyerTradeParty>
		</ram:ApplicableHeaderTradeAgreement>
		<ram:ApplicableHeaderTradeSettlement>
			<ram:InvoiceCurrencyCode>EUR</ram:InvoiceCurrencyCode>
			<ram:SpecifiedTradeSettlementHeaderMonetarySummation>
				<ram:LineTotalAmount>0</ram:LineTotalAmount>
				<ram:TaxTotalAmount>0</ram:TaxTotalAmount>
				<ram:GrandTotalAmount>0</ram:GrandTotalAmount>
				<ram:DuePayableAmount>0</ram:DuePayableAmount>
			</ram:SpecifiedTradeSettlementHeaderMonetarySummation>
		</ram:ApplicableHeaderTradeSettlement>
	</rsm:SupplyChainTradeTransaction>
</rsm:CrossIndustryInvoice>`)
	_, err := ParseCII(xml)
	assert.ErrorIs(t, err, ErrParseFailed)
}

func TestParseCII_InvalidXML(t *testing.T) {
	_, err := ParseCII([]byte("not xml at all <<>>"))
	assert.ErrorIs(t, err, ErrParseFailed)
}

// ============================================================================
// ParseUBL
// ============================================================================

func TestParseUBL_FromFixture(t *testing.T) {
	xmlData, err := os.ReadFile("testdata/ubl_minimal.xml")
	require.NoError(t, err)

	parsed, err := ParseUBL(xmlData)
	require.NoError(t, err)

	assert.Equal(t, "XRechnung Lieferant GmbH", parsed.SupplierName)
	assert.Equal(t, "DE987654321", parsed.SupplierVATID)
	assert.Equal(t, "XR-TEST-2024-001", parsed.InvoiceNumber)
	assert.Equal(t, "EUR", parsed.Currency)

	expectedDate := time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, expectedDate, parsed.InvoiceDate)

	require.NotNil(t, parsed.DueDate)
	expectedDue := time.Date(2024, 2, 19, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, expectedDue, *parsed.DueDate)

	assert.Equal(t, "500.00", parsed.Subtotal.StringFixed(2))
	assert.Equal(t, "95.00", parsed.TotalTax.StringFixed(2))
	assert.Equal(t, "595.00", parsed.GrossTotal.StringFixed(2))

	require.Len(t, parsed.LineItems, 1)
	li := parsed.LineItems[0]
	assert.Equal(t, "Softwareentwicklung", li.Description)
	assert.Equal(t, "100.00", li.UnitPrice.StringFixed(2))
	assert.Equal(t, "500.00", li.LineTotal.StringFixed(2))

	assert.Equal(t, "xrechnung_ubl", parsed.SourceFormat)
}

func TestParseUBL_InvalidXML(t *testing.T) {
	_, err := ParseUBL([]byte("<<broken>>"))
	assert.ErrorIs(t, err, ErrParseFailed)
}

// ============================================================================
// Helpers — build test models for roundtrip
// ============================================================================

func buildTestInvoice() models.Invoice {
	lineItemsJSON := []byte(`[{
		"id": "li-1",
		"position": 1,
		"description": "Test-Leistung",
		"quantity": "2.00",
		"unit_price": "100.00",
		"tax_rate": "19.00",
		"line_total": "200.00"
	}]`)

	invoiceDate := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	dueDate := time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC)

	return models.Invoice{
		InvoiceNumber: fmt.Sprintf("RE-ROUNDTRIP-001"),
		CustomerName:  "Testkunde GmbH",
		InvoiceDate:   invoiceDate,
		DueDate:       dueDate,
		LineItems:     lineItemsJSON,
		Subtotal:      decimal.NewFromFloat(200.00),
		TotalTax:      decimal.NewFromFloat(38.00),
		GrossTotal:    decimal.NewFromFloat(238.00),
	}
}

func buildTestSettings() models.CompanySettings {
	return models.CompanySettings{
		Name:    "Test Verkäufer GmbH",
		Street:  "Hauptstraße 1",
		PLZ:     "10117",
		City:    "Berlin",
		Country: "DE",
		UStIDNr: "DE111222333",
	}
}
