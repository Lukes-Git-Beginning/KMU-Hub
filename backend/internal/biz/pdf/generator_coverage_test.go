package pdf

import (
	"strings"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

// TestValidateCompanySettingsForPDF_MissingIndividualFields belegt jeden
// einzelnen Fehlerpfad von ValidateCompanySettingsForPDF. Der bestehende Test
// in zugferd_test.go prüft nur "alles fehlt" und "nur city fehlt" -- name,
// street und plz waren als eigenständige Fehlerpfade ungeprüft.
func TestValidateCompanySettingsForPDF_MissingIndividualFields(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		mutate      func(s *models.CompanySettings)
		wantMissing string
	}{
		{"missing name", func(s *models.CompanySettings) { s.Name = "" }, "name"},
		{"missing street", func(s *models.CompanySettings) { s.Street = "" }, "street"},
		{"missing plz", func(s *models.CompanySettings) { s.PLZ = "" }, "plz"},
		{"whitespace-only name", func(s *models.CompanySettings) { s.Name = "   " }, "name"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := completeTestSettings()
			tc.mutate(&s)
			err := ValidateCompanySettingsForPDF(s)
			if err == nil {
				t.Fatalf("expected an error for %s, got nil", tc.name)
			}
			if got := err.Error(); !strings.Contains(got, tc.wantMissing) {
				t.Errorf("expected error to name missing field %q, got %q", tc.wantMissing, got)
			}
		})
	}
}

// TestValidateCompanySettingsForPDF_AllMissingFieldsAreListed belegt, dass die
// Fehlermeldung bei mehreren fehlenden Feldern jedes davon nennt -- nicht nur
// das erste, das dem Aufrufer nur einen Teil des Problems zeigen würde.
func TestValidateCompanySettingsForPDF_AllMissingFieldsAreListed(t *testing.T) {
	t.Parallel()

	err := ValidateCompanySettingsForPDF(models.CompanySettings{})
	if err == nil {
		t.Fatal("expected an error for empty settings")
	}
	got := err.Error()
	for _, want := range []string{"name", "street", "plz", "city", "steuernummer or ust_id_nr"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected error to list missing field %q, got %q", want, got)
		}
	}
}

// TestInvoicePDF_UmlautsSurviveRenderPath belegt, dass Umlaute in Kundenname,
// -anschrift und Firmenname unverändert bis zur extrahierten Textebene
// durchlaufen -- kein Ersatzzeichen, keine Transliteration. Der Renderpfad
// selbst wandelt keine Zeichenkodierung um (kein charmap/iconv im Paket,
// geprüft per Grep); dieser Test friert diese Garantie ein.
func TestInvoicePDF_UmlautsSurviveRenderPath(t *testing.T) {
	t.Parallel()

	settings := fullTestSettings()
	settings.Name = "Müller & Söhne GmbH"
	settings.Street = "Grüner Weg 3"
	settings.City = "Köln"

	g := NewGenerator(settings)
	inv := baseTestInvoice(t)
	inv.CustomerName = "Weiß & Groß KG"
	inv.CustomerAddress = "Straße der Einheit 7, 04109 Leipzig"

	m, err := g.buildInvoiceDoc(inv)
	if err != nil {
		t.Fatalf("buildInvoiceDoc: %v", err)
	}
	texts := renderedTexts(t, m.GetStructure())

	assertContainsOnce(t, texts, "Müller & Söhne GmbH")
	assertContainsOnce(t, texts, "Grüner Weg 3")
	assertContainsOnce(t, texts, "Köln")
	assertContainsOnce(t, texts, "Weiß & Groß KG")
	assertContainsOnce(t, texts, "Straße der Einheit 7, 04109 Leipzig")
}

// TestInvoicePDF_ReverseChargeHintPresent belegt Nr. 8 (Hinweispflicht) für
// den Reverse-Charge-Fall: kein Steuerausweis, aber der vorgeschriebene
// Hinweis auf § 13b UStG. tax.RequiresReverseChargeNote(ModeReverseCharge)
// verlangt genau diesen Hinweis; die Modell-Konstante models.TaxModeReverseCharge
// und tax.ModeReverseCharge tragen denselben String-Wert ("reverse_charge").
func TestInvoicePDF_ReverseChargeHintPresent(t *testing.T) {
	t.Parallel()

	g := NewGenerator(fullTestSettings())
	inv := baseTestInvoice(t)
	inv.TaxMode = models.TaxModeReverseCharge
	inv.TotalTax = decimal.Zero
	inv.GrossTotal = inv.Subtotal
	inv.TaxBreakdownRaw = nil

	m, err := g.buildInvoiceDoc(inv)
	if err != nil {
		t.Fatalf("buildInvoiceDoc: %v", err)
	}
	texts := renderedTexts(t, m.GetStructure())

	assertContainsOnce(t, texts, "Steuerschuldnerschaft")
	assertContainsOnce(t, texts, "13b UStG")
}

// TestInvoicePDF_StandardModeShowsNoExemptionHint ist die Gegenprobe zu den
// beiden Hinweistests: eine reguläre Rechnung darf weder den Reverse-Charge-
// noch den Kleinunternehmer-Hinweis zeigen.
func TestInvoicePDF_StandardModeShowsNoExemptionHint(t *testing.T) {
	t.Parallel()

	g := NewGenerator(fullTestSettings())
	inv := baseTestInvoice(t)

	m, err := g.buildInvoiceDoc(inv)
	if err != nil {
		t.Fatalf("buildInvoiceDoc: %v", err)
	}
	texts := renderedTexts(t, m.GetStructure())

	assertNotContains(t, texts, "Steuerschuldnerschaft")
	assertNotContains(t, texts, "Kleinunternehmer")
}
