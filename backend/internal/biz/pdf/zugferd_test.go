package pdf

import (
	"testing"
	"time"

	"github.com/kmuhub/kmuhub/internal/models"
)

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
