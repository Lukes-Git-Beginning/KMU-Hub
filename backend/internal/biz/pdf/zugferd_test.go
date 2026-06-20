package pdf

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

// assertXMLContains fails the test if xml does not contain sub.
func assertXMLContains(t *testing.T, xml, sub, msg string) {
	t.Helper()
	if !strings.Contains(xml, sub) {
		t.Errorf("%s: expected XML to contain %q", msg, sub)
	}
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
		InvoiceDate:   time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		DueDate:       time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
		Currency:      "CHF",
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
		InvoiceDate:   time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		DueDate:       time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
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
		InvoiceDate:   time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		DueDate:       time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
		TaxMode:       models.TaxModeReverseCharge,
		LineItems:     liJSON,
		Subtotal:      decimal.NewFromInt(500),
		TotalTax:      decimal.Zero,
		GrossTotal:    decimal.NewFromInt(500),
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
