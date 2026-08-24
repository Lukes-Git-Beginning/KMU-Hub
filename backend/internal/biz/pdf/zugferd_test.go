package pdf

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/biz/einvoice"
	"github.com/kmuhub/kmuhub/internal/models"
)

// assertXMLContains fails the test if xml does not contain sub.
func assertXMLContains(t *testing.T, xml, sub, msg string) {
	t.Helper()
	if !strings.Contains(xml, sub) {
		t.Errorf("%s: expected XML to contain %q", msg, sub)
	}
}

// singleLineItemJSON is one 100.00 net line at 19 % VAT — subtotal 100.00,
// tax 19.00, gross 119.00. EN 16931 BG-25 requires at least one line, so every
// invoice that is expected to render needs one.
func singleLineItemJSON(t *testing.T) []byte {
	t.Helper()
	raw, err := json.Marshal([]models.LineItem{{
		ID: "1", Position: 1, Description: "Leistung",
		Quantity:  decimal.NewFromInt(1),
		UnitPrice: decimal.NewFromInt(100),
		TaxRate:   decimal.NewFromInt(19),
		LineTotal: decimal.NewFromInt(100),
	}})
	if err != nil {
		t.Fatalf("marshal line items: %v", err)
	}
	return raw
}

func completeTestSettings() models.CompanySettings {
	return models.CompanySettings{
		Name:         "Muster GmbH",
		Street:       "Musterstraße 1",
		PLZ:          "10115",
		City:         "Berlin",
		Country:      "DE",
		Steuernummer: "12/345/67890",
	}
}

func TestValidateCompanySettingsForPDF(t *testing.T) {
	t.Parallel()

	if err := ValidateCompanySettingsForPDF(completeTestSettings()); err != nil {
		t.Fatalf("complete settings should be valid, got: %v", err)
	}

	if err := ValidateCompanySettingsForPDF(models.CompanySettings{}); err == nil {
		t.Fatal("empty settings should be rejected")
	}

	// A UStIDNr alone satisfies the tax-id requirement (Steuernummer OR UStIDNr).
	s := completeTestSettings()
	s.Steuernummer = ""
	s.UStIDNr = "DE123456789"
	if err := ValidateCompanySettingsForPDF(s); err != nil {
		t.Fatalf("ust_id_nr alone should satisfy the tax-id requirement, got: %v", err)
	}

	// Missing city must be rejected.
	s = completeTestSettings()
	s.City = ""
	if err := ValidateCompanySettingsForPDF(s); err == nil {
		t.Fatal("missing city should be rejected")
	}
}

func TestGenerateZUGFeRDXML_ZeroDueDate(t *testing.T) {
	t.Parallel()

	inv := models.Invoice{
		InvoiceNumber: "RE-2026-0001",
		InvoiceDate:   time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		// DueDate intentionally left as the zero value.
	}
	if _, err := GenerateZUGFeRDXML(inv, completeTestSettings()); err == nil {
		t.Fatal("expected an error for a zero due date, got nil")
	}
}

func TestGenerateZUGFeRDXML_ZeroInvoiceDate(t *testing.T) {
	t.Parallel()

	inv := models.Invoice{
		InvoiceNumber: "RE-2026-0001",
		DueDate:       time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
		// InvoiceDate intentionally left as the zero value.
	}
	if _, err := GenerateZUGFeRDXML(inv, completeTestSettings()); err == nil {
		t.Fatal("expected an error for a zero issue date, got nil")
	}
}

func TestGenerateZUGFeRDXML_CurrencyFromInvoice(t *testing.T) {
	t.Parallel()

	inv := models.Invoice{
		InvoiceNumber: "RE-2026-0001",
		CustomerName:  "Beispiel GmbH",
		InvoiceDate:   time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		DueDate:       time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
		Currency:      "CHF",
		LineItems:     singleLineItemJSON(t),
	}
	out, err := GenerateZUGFeRDXML(inv, completeTestSettings())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	xml := string(out)
	if !strings.Contains(xml, "<ram:InvoiceCurrencyCode>CHF</ram:InvoiceCurrencyCode>") {
		t.Error("expected InvoiceCurrencyCode CHF in XML")
	}
	if !strings.Contains(xml, `currencyID="CHF"`) {
		t.Error("expected TaxTotalAmount currencyID CHF in XML")
	}
}

func TestGenerateZUGFeRDXML_CurrencyDefaultsToEUR(t *testing.T) {
	t.Parallel()

	inv := models.Invoice{
		InvoiceNumber: "RE-2026-0001",
		CustomerName:  "Beispiel GmbH",
		InvoiceDate:   time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		DueDate:       time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
		LineItems:     singleLineItemJSON(t),
		// Currency intentionally left empty — must fall back to EUR.
	}
	out, err := GenerateZUGFeRDXML(inv, completeTestSettings())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(out), "<ram:InvoiceCurrencyCode>EUR</ram:InvoiceCurrencyCode>") {
		t.Error("expected currency to default to EUR when invoice.Currency is empty")
	}
}

// TestGenerateZUGFeRDXML_NoLineItems pins the stricter behaviour the shared
// renderer brought to this path: an invoice without a single line violates
// EN 16931 BG-25 and is rejected instead of producing a line-less document that
// the receiver silently discards.
func TestGenerateZUGFeRDXML_NoLineItems(t *testing.T) {
	t.Parallel()

	inv := models.Invoice{
		InvoiceNumber: "RE-2026-0001",
		InvoiceDate:   time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		DueDate:       time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
	}
	if _, err := GenerateZUGFeRDXML(inv, completeTestSettings()); err == nil {
		t.Fatal("expected an error for an invoice without line items, got nil")
	}
}

func TestGenerateZUGFeRDXML_IncompleteSettings(t *testing.T) {
	t.Parallel()

	inv := models.Invoice{
		InvoiceNumber: "RE-2026-0001",
		InvoiceDate:   time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		DueDate:       time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
	}
	if _, err := GenerateZUGFeRDXML(inv, models.CompanySettings{}); err == nil {
		t.Fatal("expected an error for incomplete company settings, got nil")
	}
}

// TestGenerateZUGFeRDXML_EN16931HeaderTaxAndDelivery verifies the three EN16931
// elements that were previously missing: the BT-72 delivery date, the BT-106
// TaxBasisTotalAmount and the BG-23 per-rate VAT breakdown in the header.
func TestGenerateZUGFeRDXML_EN16931HeaderTaxAndDelivery(t *testing.T) {
	t.Parallel()

	items := []models.LineItem{
		{ID: "1", Position: 1, Description: "Beratung", Quantity: decimal.NewFromInt(2), UnitPrice: decimal.NewFromInt(100), TaxRate: decimal.NewFromInt(19), LineTotal: decimal.NewFromInt(200)},
		{ID: "2", Position: 2, Description: "Buch", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(50), TaxRate: decimal.NewFromInt(7), LineTotal: decimal.NewFromInt(50)},
	}
	liJSON, _ := json.Marshal(items)
	delivery := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	inv := models.Invoice{
		InvoiceNumber: "RE-2026-0042",
		CustomerName:  "Beispiel GmbH",
		InvoiceDate:   time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		DueDate:       time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
		DeliveryDate:  &delivery,
		TaxMode:       models.TaxModeStandard,
		LineItems:     liJSON,
		Subtotal:      decimal.NewFromInt(250),
		TotalTax:      decimal.RequireFromString("41.50"),
		GrossTotal:    decimal.RequireFromString("291.50"),
	}

	out, err := GenerateZUGFeRDXML(inv, completeTestSettings())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	xml := string(out)

	assertXMLContains(t, xml, "<ram:ActualDeliverySupplyChainEvent>", "BT-72 delivery event")
	assertXMLContains(t, xml, "20260110", "explicit delivery date")
	assertXMLContains(t, xml, "<ram:TaxBasisTotalAmount>250.00</ram:TaxBasisTotalAmount>", "BT-106 tax basis total")
	assertXMLContains(t, xml, "<ram:CategoryCode>S</ram:CategoryCode>", "standard VAT category")
	assertXMLContains(t, xml, "<ram:RateApplicablePercent>19.00</ram:RateApplicablePercent>", "19% rate block")
	assertXMLContains(t, xml, "<ram:RateApplicablePercent>7.00</ram:RateApplicablePercent>", "7% rate block")
	assertXMLContains(t, xml, "<ram:BasisAmount>200.00</ram:BasisAmount>", "19% basis")
	assertXMLContains(t, xml, "<ram:BasisAmount>50.00</ram:BasisAmount>", "7% basis")
	assertXMLContains(t, xml, "<ram:CalculatedAmount>38.00</ram:CalculatedAmount>", "19% calculated tax")
	assertXMLContains(t, xml, "<ram:CalculatedAmount>3.50</ram:CalculatedAmount>", "7% calculated tax")
}

// TestGenerateZUGFeRDXML_ReverseChargeExemptCategory verifies the tax-free path
// emits a single exempt category (AE) with a zero rate and an exemption reason.
func TestGenerateZUGFeRDXML_ReverseChargeExemptCategory(t *testing.T) {
	t.Parallel()

	items := []models.LineItem{
		{ID: "1", Position: 1, Description: "Export", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(500), TaxRate: decimal.Zero, LineTotal: decimal.NewFromInt(500)},
	}
	liJSON, _ := json.Marshal(items)
	inv := models.Invoice{
		InvoiceNumber: "RE-2026-0043",
		CustomerName:  "Beispiel GmbH",
		// BR-AE-02: reverse charge shifts the liability, so the buyer has to carry
		// a VAT identifier of their own.
		CustomerUStIDNr: "ATU12345678",
		InvoiceDate:     time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		DueDate:         time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
		TaxMode:         models.TaxModeReverseCharge,
		LineItems:       liJSON,
		Subtotal:        decimal.NewFromInt(500),
		TotalTax:        decimal.Zero,
		GrossTotal:      decimal.NewFromInt(500),
	}

	out, err := GenerateZUGFeRDXML(inv, completeTestSettings())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	xml := string(out)

	assertXMLContains(t, xml, "<ram:CategoryCode>AE</ram:CategoryCode>", "reverse-charge category")
	assertXMLContains(t, xml, "<ram:BasisAmount>500.00</ram:BasisAmount>", "reverse-charge basis")
	assertXMLContains(t, xml, "Reverse charge", "exemption reason")
}

// ============================================================================
// EmbedZUGFeRDXML / GenerateZUGFeRDInvoicePDF
// ============================================================================

// completeTestInvoice is an invoice that renders both as a PDF and as a
// conforming EN 16931 document.
func completeTestInvoice(t *testing.T) models.Invoice {
	t.Helper()
	return models.Invoice{
		InvoiceNumber: "RE-2026-0100",
		CustomerName:  "Beispiel GmbH",
		InvoiceDate:   time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		DueDate:       time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
		TaxMode:       models.TaxModeStandard,
		LineItems:     singleLineItemJSON(t),
		Subtotal:      decimal.NewFromInt(100),
		TotalTax:      decimal.NewFromInt(19),
		GrossTotal:    decimal.NewFromInt(119),
	}
}

// TestGenerateZUGFeRDInvoicePDF_DeclaresFacturXAttachment covers the whole
// outbound path — maroto renders the PDF, einvoice renders the CII document,
// and the attachment is declared so a receiver's software finds it. Until now
// this path had no test at all and the embedding step silently degraded to a
// PDF without XML, so a regression here was invisible.
func TestGenerateZUGFeRDInvoicePDF_DeclaresFacturXAttachment(t *testing.T) {
	t.Parallel()

	settings := completeTestSettings()
	inv := completeTestInvoice(t)

	pdfBytes, err := NewGenerator(settings).GenerateZUGFeRDInvoicePDF(inv)
	if err != nil {
		t.Fatalf("generate ZUGFeRD invoice PDF: %v", err)
	}

	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed

	// The attachment carries the fixed Factur-X name — receiving software
	// matches on it literally.
	attachments, err := api.Attachments(bytes.NewReader(pdfBytes), conf)
	if err != nil {
		t.Fatalf("list attachments: %v", err)
	}
	if len(attachments) != 1 {
		t.Fatalf("expected exactly one attachment, got %d", len(attachments))
	}
	if attachments[0].FileName != FacturXAttachmentName {
		t.Errorf("expected attachment named %q, got %q", FacturXAttachmentName, attachments[0].FileName)
	}

	// The bytes that come back out are the document the generator produced.
	extracted, err := einvoice.ExtractXMLFromPDF(pdfBytes)
	if err != nil {
		t.Fatalf("extract embedded XML: %v", err)
	}
	expected, err := GenerateZUGFeRDXML(inv, settings)
	if err != nil {
		t.Fatalf("generate reference XML: %v", err)
	}
	if !bytes.Equal(extracted, expected) {
		t.Errorf("embedded XML differs from the generated document (%d vs %d bytes)", len(extracted), len(expected))
	}

	// The declaration is what makes the attachment an e-invoice rather than an
	// enclosure: /AF on the catalog, /AFRelationship /Alternative on the file
	// specification, text/xml on the stream.
	ctx, _, _, _, err := api.ReadValidateAndOptimize(bytes.NewReader(pdfBytes), conf, time.Now())
	if err != nil {
		t.Fatalf("read generated PDF: %v", err)
	}
	root, err := ctx.Catalog()
	if err != nil {
		t.Fatalf("catalog: %v", err)
	}
	afObj, found := root.Find("AF")
	if !found {
		t.Fatal("catalog has no /AF entry — no receiver would look for the invoice data")
	}
	af, err := ctx.DereferenceArray(afObj)
	if err != nil {
		t.Fatalf("dereference /AF: %v", err)
	}
	if len(af) != 1 {
		t.Fatalf("expected one associated file, got %d", len(af))
	}
	fileSpec, err := ctx.DereferenceDict(af[0])
	if err != nil {
		t.Fatalf("dereference file specification: %v", err)
	}
	if rel := fileSpec.NameEntry("AFRelationship"); rel == nil || *rel != "Alternative" {
		t.Errorf("expected /AFRelationship /Alternative, got %v", rel)
	}
	efDict := fileSpec.DictEntry("EF")
	if efDict == nil {
		t.Fatal("file specification has no /EF entry")
	}
	sd, _, err := ctx.DereferenceStreamDict(efDict["F"])
	if err != nil {
		t.Fatalf("dereference embedded file stream: %v", err)
	}
	if sub := sd.NameEntry("Subtype"); sub == nil || *sub != "text/xml" {
		t.Errorf("expected embedded file subtype text/xml, got %v", sub)
	}
}

// TestEmbedZUGFeRDXML_ReportsUnreadablePDF pins the behaviour that replaced the
// silent degradation: a PDF pdfcpu cannot read is an error, not a reason to
// hand back a document that looks like an e-invoice and carries no data.
func TestEmbedZUGFeRDXML_ReportsUnreadablePDF(t *testing.T) {
	t.Parallel()

	if _, err := EmbedZUGFeRDXML([]byte("not a PDF"), []byte("<xml/>"), "RE-2026-0100"); err == nil {
		t.Fatal("expected an error for an unreadable PDF, got nil")
	}
}

// TestGenerateZUGFeRDInvoicePDF_RejectsIncompleteInvoice verifies the EN 16931
// check reaches this path: an invoice without a customer name violates BR-07
// and must not be delivered as an e-invoice.
func TestGenerateZUGFeRDInvoicePDF_RejectsIncompleteInvoice(t *testing.T) {
	t.Parallel()

	inv := completeTestInvoice(t)
	inv.CustomerName = ""

	if _, err := NewGenerator(completeTestSettings()).GenerateZUGFeRDInvoicePDF(inv); err == nil {
		t.Fatal("expected an error for an invoice without a customer name, got nil")
	}
}
