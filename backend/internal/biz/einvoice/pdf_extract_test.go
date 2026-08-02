// This file is an external test package on purpose. It exercises the extraction
// side against a PDF built by internal/biz/pdf, and that package now depends on
// einvoice for the CII document itself — an in-package test importing pdf would
// close an import cycle.
package einvoice_test

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/biz/einvoice"
	"github.com/kmuhub/kmuhub/internal/biz/pdf"
	"github.com/kmuhub/kmuhub/internal/models"
)

// ============================================================================
// ExtractXMLFromPDF
// ============================================================================

// TestExtractXMLFromPDF_HappyPath creates a real PDF with an embedded ZUGFeRD
// XML attachment using EmbedZUGFeRDXML, then verifies ExtractXMLFromPDF returns
// the correct XML bytes.
func TestExtractXMLFromPDF_HappyPath(t *testing.T) {
	inv, settings := testInvoiceAndSettings()

	xmlBytes, err := pdf.GenerateZUGFeRDXML(inv, settings)
	require.NoError(t, err)

	// A rendered invoice, not a hand-written stub. The stub this test used before
	// was unreadable for pdfcpu, so the embedding silently degraded and the test
	// skipped itself — the extraction path was effectively untested.
	basePDF := renderInvoicePDF(t, inv, settings)

	pdfWithXML, err := pdf.EmbedZUGFeRDXML(basePDF, xmlBytes, inv.InvoiceNumber)
	require.NoError(t, err)

	extracted, err := einvoice.ExtractXMLFromPDF(pdfWithXML)
	require.NoError(t, err)
	assert.Equal(t, xmlBytes, extracted)
}

// TestExtractXMLFromPDF_NoAttachment verifies that ErrNoEmbeddedXML is returned
// for a PDF without any embedded XML attachment.
func TestExtractXMLFromPDF_NoAttachment(t *testing.T) {
	inv, settings := testInvoiceAndSettings()
	_, err := einvoice.ExtractXMLFromPDF(renderInvoicePDF(t, inv, settings))
	assert.ErrorIs(t, err, einvoice.ErrNoEmbeddedXML)
}

// TestExtractXMLFromPDF_NotPDF verifies that random bytes return ErrNoEmbeddedXML.
func TestExtractXMLFromPDF_NotPDF(t *testing.T) {
	_, err := einvoice.ExtractXMLFromPDF([]byte("this is not a PDF file"))
	assert.ErrorIs(t, err, einvoice.ErrNoEmbeddedXML)
}

// ============================================================================
// Helpers
// ============================================================================

// testInvoiceAndSettings returns an invoice that renders both as a PDF and as a
// conforming EN 16931 document.
func testInvoiceAndSettings() (models.Invoice, models.CompanySettings) {
	inv := models.Invoice{
		InvoiceNumber: "RE-PDF-001",
		CustomerName:  "PDF Testkunde GmbH",
		InvoiceDate:   time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC),
		DueDate:       time.Date(2024, 4, 30, 0, 0, 0, 0, time.UTC),
		LineItems: []byte(`[{
			"id": "li-1",
			"position": 1,
			"description": "PDF Test-Leistung",
			"quantity": "1.00",
			"unit_price": "100.00",
			"tax_rate": "19.00",
			"line_total": "100.00"
		}]`),
		Subtotal:   decimal.NewFromFloat(100.00),
		TotalTax:   decimal.NewFromFloat(19.00),
		GrossTotal: decimal.NewFromFloat(119.00),
	}
	settings := models.CompanySettings{
		Name:    "PDF Verkäufer GmbH",
		Street:  "Testweg 5",
		PLZ:     "20099",
		City:    "Hamburg",
		Country: "DE",
		UStIDNr: "DE999888777",
	}
	return inv, settings
}

// renderInvoicePDF renders the invoice through the real PDF generator. Reading
// the attachment back requires a PDF pdfcpu accepts, which a hand-written stub
// is not.
func renderInvoicePDF(t *testing.T, inv models.Invoice, settings models.CompanySettings) []byte {
	t.Helper()
	pdfBytes, err := pdf.NewGenerator(settings).GenerateInvoicePDF(inv)
	require.NoError(t, err)
	return pdfBytes
}
