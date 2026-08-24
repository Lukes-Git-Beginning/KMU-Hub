package pdf

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/johnfercher/go-tree/node"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

// renderedTexts walks a maroto document structure and collects every leaf
// text value that was rendered, without generating actual PDF bytes. This is
// the "template data structure / extracted text content" level the loop's
// backlog explicitly asked for instead of a binary PDF diff.
func renderedTexts(t *testing.T, n *node.Node[core.Structure]) []string {
	t.Helper()
	var out []string
	var walk func(n *node.Node[core.Structure])
	walk = func(n *node.Node[core.Structure]) {
		data := n.GetData()
		if data.Type == "text" {
			if s, ok := data.Value.(string); ok {
				out = append(out, s)
			}
		}
		for _, next := range n.GetNexts() {
			walk(next)
		}
	}
	walk(n)
	return out
}

func assertContainsOnce(t *testing.T, texts []string, want string) {
	t.Helper()
	for _, got := range texts {
		if strings.Contains(got, want) {
			return
		}
	}
	t.Errorf("expected rendered invoice to contain %q, got %v", want, texts)
}

func assertNotContains(t *testing.T, texts []string, unwanted string) {
	t.Helper()
	for _, got := range texts {
		if strings.Contains(got, unwanted) {
			t.Errorf("expected rendered invoice to NOT contain %q, got %v", unwanted, texts)
			return
		}
	}
}

// fullTestSettings carries every §14 Abs. 4 Nr. 2 issuer field (Steuernummer
// AND UStIDNr, since the header/footer print both when present).
func fullTestSettings() models.CompanySettings {
	return models.CompanySettings{
		Name:            "Muster GmbH",
		Street:          "Musterstraße 1",
		PLZ:             "10115",
		City:            "Berlin",
		Country:         "DE",
		Steuernummer:    "12/345/67890",
		UStIDNr:         "DE123456789",
		Handelsregister: "HRB 12345 B",
		BankName:        "Muster Bank",
		IBAN:            "DE89370400440532013000",
		BIC:             "MUSTDE1XXX",
	}
}

func testInvoiceLineItems(t *testing.T) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal([]models.LineItem{{
		ID: "1", Position: 1, Description: "Beratungsleistung",
		Quantity:  decimal.NewFromInt(2),
		UnitPrice: decimal.NewFromInt(100),
		TaxRate:   decimal.NewFromInt(19),
		LineTotal: decimal.NewFromInt(200),
	}})
	if err != nil {
		t.Fatalf("marshal line items: %v", err)
	}
	return raw
}

func testTaxBreakdownRaw(t *testing.T, subtotal, totalTax, grossTotal decimal.Decimal, taxByRate map[string]decimal.Decimal) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(models.TaxBreakdown{
		Subtotal:   subtotal,
		TaxByRate:  taxByRate,
		TotalTax:   totalTax,
		GrossTotal: grossTotal,
	})
	if err != nil {
		t.Fatalf("marshal tax breakdown: %v", err)
	}
	return raw
}

func baseTestInvoice(t *testing.T) models.Invoice {
	t.Helper()
	invoiceDate := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	dueDate := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
	subtotal := decimal.NewFromInt(200)
	totalTax := decimal.NewFromFloat(38)
	grossTotal := decimal.NewFromFloat(238)
	return models.Invoice{
		ID:              uuid.New(),
		InvoiceNumber:   "RE-2026-0042",
		CustomerName:    "Kunde AG",
		CustomerAddress: "Kundenstraße 5, 20095 Hamburg",
		TaxMode:         models.TaxModeStandard,
		LineItems:       testInvoiceLineItems(t),
		TaxBreakdownRaw: testTaxBreakdownRaw(t, subtotal, totalTax, grossTotal, map[string]decimal.Decimal{"19": totalTax}),
		Subtotal:        subtotal,
		TotalTax:        totalTax,
		GrossTotal:      grossTotal,
		InvoiceDate:     invoiceDate,
		DueDate:         dueDate,
	}
}

// TestInvoicePDF_UStG14MandatoryFields belegt jede vom Template gerenderte
// Pflichtangabe aus §14 Abs. 4 UStG gegen eine vollständige Rechnung.
func TestInvoicePDF_UStG14MandatoryFields(t *testing.T) {
	t.Parallel()

	g := NewGenerator(fullTestSettings())
	inv := baseTestInvoice(t)

	m, err := g.buildInvoiceDoc(inv)
	if err != nil {
		t.Fatalf("buildInvoiceDoc: %v", err)
	}
	texts := renderedTexts(t, m.GetStructure())

	// Nr. 1: vollständiger Name und Anschrift beider Parteien
	assertContainsOnce(t, texts, "Muster GmbH")
	assertContainsOnce(t, texts, "Musterstraße 1")
	assertContainsOnce(t, texts, "10115 Berlin")
	assertContainsOnce(t, texts, "Kunde AG")
	assertContainsOnce(t, texts, "Kundenstraße 5, 20095 Hamburg")

	// Nr. 2: Steuernummer oder USt-IdNr. des Leistenden
	assertContainsOnce(t, texts, "12/345/67890")
	assertContainsOnce(t, texts, "DE123456789")

	// Nr. 3: Ausstellungsdatum
	assertContainsOnce(t, texts, "01.08.2026")

	// Nr. 4: fortlaufende Rechnungsnummer
	assertContainsOnce(t, texts, "RE-2026-0042")

	// Nr. 5: Menge und Art der Leistung
	assertContainsOnce(t, texts, "Beratungsleistung")
	assertContainsOnce(t, texts, "2.00")

	// Nr. 6: Zeitpunkt der Leistung (Lieferdatum) -- falls back to invoice date
	assertContainsOnce(t, texts, "Lieferdatum: 01.08.2026")

	// Nr. 7/8: Entgelt aufgeschlüsselt nach Steuersätzen, Steuersatz und Steuerbetrag
	assertContainsOnce(t, texts, "MwSt 19%")
	assertContainsOnce(t, texts, "38.00 EUR")
	assertContainsOnce(t, texts, "238.00 EUR")
}

// TestInvoicePDF_DeliveryDateFallsBackToInvoiceDate is the mutation-relevant
// regression test for the fix in this iteration: before the fix, an invoice
// with DeliveryDate == nil silently omitted the Leistungsdatum line entirely
// (a §14 Abs. 4 Nr. 6 UStG violation) instead of falling back to InvoiceDate.
func TestInvoicePDF_DeliveryDateFallsBackToInvoiceDate(t *testing.T) {
	t.Parallel()

	g := NewGenerator(fullTestSettings())
	inv := baseTestInvoice(t)
	inv.DeliveryDate = nil

	m, err := g.buildInvoiceDoc(inv)
	if err != nil {
		t.Fatalf("buildInvoiceDoc: %v", err)
	}
	texts := renderedTexts(t, m.GetStructure())

	assertContainsOnce(t, texts, "Lieferdatum: 01.08.2026")
}

// TestInvoicePDF_ExplicitDeliveryDateIsUsed proves the fallback does not mask
// an explicitly set delivery date that differs from the invoice date.
func TestInvoicePDF_ExplicitDeliveryDateIsUsed(t *testing.T) {
	t.Parallel()

	g := NewGenerator(fullTestSettings())
	inv := baseTestInvoice(t)
	delivery := time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC)
	inv.DeliveryDate = &delivery

	m, err := g.buildInvoiceDoc(inv)
	if err != nil {
		t.Fatalf("buildInvoiceDoc: %v", err)
	}
	texts := renderedTexts(t, m.GetStructure())

	assertContainsOnce(t, texts, "Lieferdatum: 28.07.2026")
	assertNotContains(t, texts, "Lieferdatum: 01.08.2026")
}

// TestInvoicePDF_KleinunternehmerExemptionHint belegt Nr. 8 für den
// Befreiungsfall: kein Steuerausweis, aber ein Hinweis auf die Befreiung.
func TestInvoicePDF_KleinunternehmerExemptionHint(t *testing.T) {
	t.Parallel()

	g := NewGenerator(fullTestSettings())
	inv := baseTestInvoice(t)
	inv.TaxMode = models.TaxModeKleinunternehmer
	inv.TotalTax = decimal.Zero
	inv.GrossTotal = inv.Subtotal
	inv.TaxBreakdownRaw = nil

	m, err := g.buildInvoiceDoc(inv)
	if err != nil {
		t.Fatalf("buildInvoiceDoc: %v", err)
	}
	texts := renderedTexts(t, m.GetStructure())

	assertContainsOnce(t, texts, "Kleinunternehmer")
	assertContainsOnce(t, texts, "Abschnitt 19 UStG")
}

// TestInvoicePDF_IncompleteCompanySettingsRejected belegt, dass ein
// unvollständiger Leistender das Dokument gar nicht erst rendert (statt eine
// Rechnung ohne Nr. 1/2 Pflichtangaben stillschweigend auszugeben).
func TestInvoicePDF_IncompleteCompanySettingsRejected(t *testing.T) {
	t.Parallel()

	g := NewGenerator(models.CompanySettings{Name: "Muster GmbH"})
	inv := baseTestInvoice(t)

	if _, err := g.buildInvoiceDoc(inv); err == nil {
		t.Fatal("expected buildInvoiceDoc to reject incomplete company settings")
	}
}
