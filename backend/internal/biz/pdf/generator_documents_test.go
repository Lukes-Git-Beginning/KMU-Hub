package pdf

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

// hasText reports whether a rendered text leaf equals want exactly. The
// Contains-based helpers in invoice_ustg14_test.go would let "1" match "19".
func hasText(texts []string, want string) bool {
	for _, got := range texts {
		if got == want {
			return true
		}
	}
	return false
}

func requireExactText(t *testing.T, texts []string, want string) {
	t.Helper()
	if !hasText(texts, want) {
		t.Errorf("expected rendered text %q, got %v", want, texts)
	}
}

func requireNoExactText(t *testing.T, texts []string, unwanted string) {
	t.Helper()
	if hasText(texts, unwanted) {
		t.Errorf("expected rendered text to NOT contain %q, got %v", unwanted, texts)
	}
}

func structureTexts(t *testing.T, m core.Maroto) []string {
	t.Helper()
	return renderedTexts(t, m.GetStructure())
}

func requirePDF(t *testing.T, b []byte) {
	t.Helper()
	if !bytes.HasPrefix(b, []byte("%PDF-")) {
		t.Fatalf("expected PDF bytes, got prefix %q", string(b[:min(len(b), 8)]))
	}
	if !bytes.Contains(b[max(0, len(b)-32):], []byte("%%EOF")) {
		t.Errorf("PDF is not terminated by %%%%EOF")
	}
}

func twoLineItems(t *testing.T) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal([]models.LineItem{
		// Position 0 -> numbered by index; LineTotal is deliberately stale.
		{Description: "Montage", Quantity: decimal.NewFromInt(3), UnitPrice: decimal.RequireFromString("12.50"), LineTotal: decimal.NewFromInt(999)},
		// Explicit position wins over the index.
		{Position: 7, Description: "Anfahrt", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.RequireFromString("30"), LineTotal: decimal.NewFromInt(30)},
	})
	if err != nil {
		t.Fatalf("marshal line items: %v", err)
	}
	return raw
}

func baseTestQuote(t *testing.T) models.Quote {
	t.Helper()
	return models.Quote{
		ID:              uuid.New(),
		QuoteNumber:     "AN-2026-0007",
		CustomerName:    "Kunde AG",
		CustomerAddress: "Kundenstraße 5, 20095 Hamburg",
		CustomerUStIDNr: "DE987654321",
		TaxMode:         models.TaxModeStandard,
		LineItems:       twoLineItems(t),
		Subtotal:        decimal.RequireFromString("67.50"),
		TotalTax:        decimal.RequireFromString("12.83"),
		GrossTotal:      decimal.RequireFromString("80.33"),
		CreatedAt:       time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC),
	}
}

func TestParseColor(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		in      string
		r, g, b int
	}{
		{"uppercase hex", "#FF8000", 255, 128, 0},
		{"lowercase hex", "#1a2b3c", 0x1a, 0x2b, 0x3c},
		{"black", "#000000", 0, 0, 0},
		{"empty falls back to default", "", 31, 78, 121},
		{"missing hash falls back to default", "FF8000", 31, 78, 121},
		{"short value falls back to default", "#FFF", 31, 78, 121},
		{"too long falls back to default", "#FF80001", 31, 78, 121},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := parseColor(tc.in)
			if c.Red != tc.r || c.Green != tc.g || c.Blue != tc.b {
				t.Errorf("parseColor(%q) = (%d,%d,%d), want (%d,%d,%d)", tc.in, c.Red, c.Green, c.Blue, tc.r, tc.g, tc.b)
			}
		})
	}
}

func TestParseLineItems(t *testing.T) {
	t.Parallel()

	t.Run("empty input yields no items and no error", func(t *testing.T) {
		t.Parallel()
		items, err := parseLineItems(nil)
		if err != nil || items != nil {
			t.Errorf("got (%v, %v), want (nil, nil)", items, err)
		}
	})
	t.Run("malformed JSON is an error", func(t *testing.T) {
		t.Parallel()
		if _, err := parseLineItems(json.RawMessage(`{"not":"an array"}`)); err == nil {
			t.Error("expected an error for a JSON object where an array is required")
		}
	})
	t.Run("items round-trip", func(t *testing.T) {
		t.Parallel()
		items, err := parseLineItems(twoLineItems(t))
		if err != nil {
			t.Fatalf("parseLineItems: %v", err)
		}
		if len(items) != 2 || items[0].Description != "Montage" || items[1].Position != 7 {
			t.Errorf("unexpected items: %+v", items)
		}
	})
}

func TestParseTaxBreakdown(t *testing.T) {
	t.Parallel()

	t.Run("empty input yields nil so the caller falls back", func(t *testing.T) {
		t.Parallel()
		tb, err := parseTaxBreakdown(nil)
		if err != nil || tb != nil {
			t.Errorf("got (%v, %v), want (nil, nil)", tb, err)
		}
	})
	t.Run("malformed JSON is an error", func(t *testing.T) {
		t.Parallel()
		if _, err := parseTaxBreakdown(json.RawMessage(`[1,2]`)); err == nil {
			t.Error("expected an error for a JSON array where an object is required")
		}
	})
	t.Run("breakdown round-trips", func(t *testing.T) {
		t.Parallel()
		raw := testTaxBreakdownRaw(t, decimal.NewFromInt(100), decimal.NewFromInt(19), decimal.NewFromInt(119), map[string]decimal.Decimal{"19": decimal.NewFromInt(19)})
		tb, err := parseTaxBreakdown(raw)
		if err != nil || tb == nil {
			t.Fatalf("parseTaxBreakdown: (%v, %v)", tb, err)
		}
		if !tb.GrossTotal.Equal(decimal.NewFromInt(119)) || !tb.TaxByRate["19"].Equal(decimal.NewFromInt(19)) {
			t.Errorf("unexpected breakdown: %+v", tb)
		}
	})
}

func TestQuotePDF_RendersContent(t *testing.T) {
	t.Parallel()

	g := NewGenerator(fullTestSettings())
	q := baseTestQuote(t)
	valid := time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC)
	q.ValidUntil = &valid
	q.Notes = "Preise freibleibend."

	m, err := g.buildQuoteDoc(q)
	if err != nil {
		t.Fatalf("buildQuoteDoc: %v", err)
	}
	texts := structureTexts(t, m)

	requireExactText(t, texts, "Kunde AG")
	assertContainsOnce(t, texts, "USt-IdNr.: DE987654321")
	assertContainsOnce(t, texts, "AN-2026-0007")
	assertContainsOnce(t, texts, "03.09.2026")
	assertContainsOnce(t, texts, "Angebot")
	assertContainsOnce(t, texts, "gueltig bis 03.10.2026")
	requireExactText(t, texts, "Preise freibleibend.")

	// Position: index when unset, explicit value otherwise.
	requireExactText(t, texts, "1")
	requireExactText(t, texts, "7")
	requireNoExactText(t, texts, "2")
	// Line total is recomputed from quantity x unit price, never read from the stale stored value.
	requireExactText(t, texts, "37.50 EUR")
	requireNoExactText(t, texts, "999.00 EUR")
}

func TestQuotePDF_FallbackTotalsAndOptionalSections(t *testing.T) {
	t.Parallel()

	g := NewGenerator(fullTestSettings())
	q := baseTestQuote(t) // no TaxBreakdownRaw, no ValidUntil, no Notes

	m, err := g.buildQuoteDoc(q)
	if err != nil {
		t.Fatalf("buildQuoteDoc: %v", err)
	}
	texts := structureTexts(t, m)

	// Without a stored breakdown the totals come from the quote columns.
	requireExactText(t, texts, "67.50 EUR")
	requireExactText(t, texts, "80.33 EUR")
	assertNotContains(t, texts, "gueltig bis")
}

func TestQuotePDF_ParseErrors(t *testing.T) {
	t.Parallel()

	g := NewGenerator(fullTestSettings())

	badItems := baseTestQuote(t)
	badItems.LineItems = json.RawMessage(`{"x":1}`)
	if _, err := g.GenerateQuotePDF(badItems); err == nil || !strings.Contains(err.Error(), "parse quote line items") {
		t.Errorf("expected a line-item parse error, got %v", err)
	}

	badBreakdown := baseTestQuote(t)
	badBreakdown.TaxBreakdownRaw = json.RawMessage(`[1]`)
	if _, err := g.GenerateQuotePDF(badBreakdown); err == nil || !strings.Contains(err.Error(), "parse quote tax breakdown") {
		t.Errorf("expected a tax-breakdown parse error, got %v", err)
	}
}

func TestGenerateQuotePDF_ProducesPDF(t *testing.T) {
	t.Parallel()

	b, err := NewGenerator(fullTestSettings()).GenerateQuotePDF(baseTestQuote(t))
	if err != nil {
		t.Fatalf("GenerateQuotePDF: %v", err)
	}
	requirePDF(t, b)
}

func TestGenerateInvoicePDF_ProducesPDFAndPropagatesErrors(t *testing.T) {
	t.Parallel()

	g := NewGenerator(fullTestSettings())

	b, err := g.GenerateInvoicePDF(baseTestInvoice(t))
	if err != nil {
		t.Fatalf("GenerateInvoicePDF: %v", err)
	}
	requirePDF(t, b)

	inv := baseTestInvoice(t)
	inv.LineItems = json.RawMessage(`"oops"`)
	if _, err := g.GenerateInvoicePDF(inv); err == nil || !strings.Contains(err.Error(), "parse invoice line items") {
		t.Errorf("expected a line-item parse error, got %v", err)
	}

	inv = baseTestInvoice(t)
	inv.TaxBreakdownRaw = json.RawMessage(`"oops"`)
	if _, err := g.GenerateInvoicePDF(inv); err == nil || !strings.Contains(err.Error(), "parse invoice tax breakdown") {
		t.Errorf("expected a tax-breakdown parse error, got %v", err)
	}

	// An incomplete issuer must not produce bytes at all.
	if _, err := NewGenerator(models.CompanySettings{}).GenerateInvoicePDF(baseTestInvoice(t)); err == nil {
		t.Error("expected GenerateInvoicePDF to reject incomplete company settings")
	}
}

func TestInvoicePDF_FallbackTotalsAndNotes(t *testing.T) {
	t.Parallel()

	g := NewGenerator(fullTestSettings())
	inv := baseTestInvoice(t)
	inv.TaxBreakdownRaw = nil
	inv.PaymentTerms = "14 Tage netto"
	inv.Notes = "Danke für Ihren Auftrag."

	m, err := g.buildInvoiceDoc(inv)
	if err != nil {
		t.Fatalf("buildInvoiceDoc: %v", err)
	}
	texts := structureTexts(t, m)

	requireExactText(t, texts, "200.00 EUR")
	requireExactText(t, texts, "238.00 EUR")
	requireExactText(t, texts, "Zahlungsbedingungen: 14 Tage netto")
	requireExactText(t, texts, "Danke für Ihren Auftrag.")

	// Neither section is rendered when empty.
	inv.PaymentTerms, inv.Notes = "", ""
	m, err = g.buildInvoiceDoc(inv)
	if err != nil {
		t.Fatalf("buildInvoiceDoc: %v", err)
	}
	assertNotContains(t, structureTexts(t, m), "Zahlungsbedingungen")
}

func TestTotalsSection_ListsEveryTaxRate(t *testing.T) {
	t.Parallel()

	g := NewGenerator(fullTestSettings())
	inv := baseTestInvoice(t)
	inv.TaxBreakdownRaw = testTaxBreakdownRaw(t,
		decimal.NewFromInt(300), decimal.RequireFromString("40.50"), decimal.RequireFromString("340.50"),
		map[string]decimal.Decimal{"19": decimal.RequireFromString("38"), "7": decimal.RequireFromString("2.50")})

	m, err := g.buildInvoiceDoc(inv)
	if err != nil {
		t.Fatalf("buildInvoiceDoc: %v", err)
	}
	texts := structureTexts(t, m)

	requireExactText(t, texts, "MwSt 19%")
	requireExactText(t, texts, "38.00 EUR")
	requireExactText(t, texts, "MwSt 7%")
	requireExactText(t, texts, "2.50 EUR")
	requireExactText(t, texts, "Zwischensumme")
	requireExactText(t, texts, "Gesamtbetrag")
	requireExactText(t, texts, "340.50 EUR")
}

func baseTestCreditNote(t *testing.T) models.CreditNote {
	t.Helper()
	return models.CreditNote{
		ID:                uuid.New(),
		CreditNoteNumber:  "GS-2026-0003",
		OriginalInvoiceID: uuid.MustParse("11111111-2222-3333-4444-555555555555"),
		CustomerName:      "Kunde AG",
		CustomerAddress:   "Kundenstraße 5, 20095 Hamburg",
		TaxMode:           models.TaxModeStandard,
		LineItems:         testInvoiceLineItems(t),
		Subtotal:          decimal.NewFromInt(200),
		TotalTax:          decimal.NewFromInt(38),
		GrossTotal:        decimal.NewFromInt(238),
		CreatedAt:         time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC),
	}
}

func TestCreditNotePDF_RendersReferenceReasonAndFallbackTotals(t *testing.T) {
	t.Parallel()

	g := NewGenerator(fullTestSettings())
	cn := baseTestCreditNote(t)
	cn.Reason = "Falschlieferung"

	m, err := g.buildCreditNoteDoc(cn)
	if err != nil {
		t.Fatalf("buildCreditNoteDoc: %v", err)
	}
	texts := structureTexts(t, m)

	requireExactText(t, texts, "Zu Rechnung: 11111111-2222-3333-4444-555555555555")
	requireExactText(t, texts, "Grund: Falschlieferung")
	assertContainsOnce(t, texts, "GS-2026-0003")
	assertContainsOnce(t, texts, "10.09.2026")
	assertContainsOnce(t, texts, "Gutschrift")
	// No stored breakdown: totals come from the credit-note columns.
	requireExactText(t, texts, "238.00 EUR")

	cn.Reason = ""
	m, err = g.buildCreditNoteDoc(cn)
	if err != nil {
		t.Fatalf("buildCreditNoteDoc: %v", err)
	}
	assertNotContains(t, structureTexts(t, m), "Grund:")
}

func TestGenerateCreditNotePDF_ProducesPDFAndPropagatesErrors(t *testing.T) {
	t.Parallel()

	g := NewGenerator(fullTestSettings())

	b, err := g.GenerateCreditNotePDF(baseTestCreditNote(t))
	if err != nil {
		t.Fatalf("GenerateCreditNotePDF: %v", err)
	}
	requirePDF(t, b)

	cn := baseTestCreditNote(t)
	cn.LineItems = json.RawMessage(`7`)
	if _, err := g.GenerateCreditNotePDF(cn); err == nil || !strings.Contains(err.Error(), "parse credit note line items") {
		t.Errorf("expected a line-item parse error, got %v", err)
	}

	cn = baseTestCreditNote(t)
	cn.TaxBreakdownRaw = json.RawMessage(`7`)
	if _, err := g.GenerateCreditNotePDF(cn); err == nil || !strings.Contains(err.Error(), "parse credit note tax breakdown") {
		t.Errorf("expected a tax-breakdown parse error, got %v", err)
	}

	if _, err := NewGenerator(models.CompanySettings{}).GenerateCreditNotePDF(baseTestCreditNote(t)); err == nil {
		t.Error("expected GenerateCreditNotePDF to reject incomplete company settings")
	}
}

func dunningFixture(t *testing.T) (models.DunningRecord, models.Invoice) {
	t.Helper()
	inv := baseTestInvoice(t) // gross 238.00, RE-2026-0042
	d := models.DunningRecord{
		ID:        uuid.New(),
		InvoiceID: inv.ID,
		Fee:       decimal.RequireFromString("5"),
		Interest:  decimal.RequireFromString("1.75"),
		CreatedAt: time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC),
	}
	return d, inv
}

func TestDunningPDF_LevelsCarryTheirTone(t *testing.T) {
	t.Parallel()

	cases := []struct {
		level     int
		title     string
		bodyHas   string
		bodyLacks string
	}{
		{1, "Zahlungserinnerung", "hoeflich daran erinnern", "Inkasso"},
		{2, "1. Mahnung", "trotz unserer Erinnerung", "Inkasso"},
		{3, "2. Mahnung / Letzte Mahnung", "Inkassounternehmen", "hoeflich"},
	}
	for _, tc := range cases {
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()
			d, inv := dunningFixture(t)
			texts := structureTexts(t, NewGenerator(fullTestSettings()).buildDunningDoc(d, inv, tc.level))

			requireExactText(t, texts, tc.title)
			assertContainsOnce(t, texts, tc.bodyHas)
			assertNotContains(t, texts, tc.bodyLacks)
			// Every level names the invoice it refers to.
			assertContainsOnce(t, texts, "RE-2026-0042")
			requireExactText(t, texts, "Datum: 20.09.2026")
		})
	}
}

func TestDunningPDF_Level1NamesTheOpenAmount(t *testing.T) {
	t.Parallel()

	d, inv := dunningFixture(t)
	g := NewGenerator(fullTestSettings())

	level1 := structureTexts(t, g.buildDunningDoc(d, inv, 1))
	assertContainsOnce(t, level1, "ueber 238.00 EUR noch nicht beglichen")
	assertNotContains(t, level1, "EUR EUR")
}

func TestDunningPDF_FeeInterestAndTotalDemand(t *testing.T) {
	t.Parallel()

	g := NewGenerator(fullTestSettings())
	d, inv := dunningFixture(t)

	texts := structureTexts(t, g.buildDunningDoc(d, inv, 2))
	requireExactText(t, texts, "Rechnungsbetrag")
	requireExactText(t, texts, "Mahngebuehr")
	requireExactText(t, texts, "5.00 EUR")
	requireExactText(t, texts, "Verzugszinsen")
	requireExactText(t, texts, "1.75 EUR")
	requireExactText(t, texts, "Gesamtforderung")
	// 238.00 + 5.00 + 1.75
	requireExactText(t, texts, "244.75 EUR")

	// Zero fee / zero interest print no line, and the total is the invoice amount.
	d.Fee, d.Interest = decimal.Zero, decimal.Zero
	texts = structureTexts(t, g.buildDunningDoc(d, inv, 2))
	requireNoExactText(t, texts, "Mahngebuehr")
	requireNoExactText(t, texts, "Verzugszinsen")
	requireExactText(t, texts, "Gesamtforderung")

	// Fee only.
	d.Fee = decimal.RequireFromString("5")
	texts = structureTexts(t, g.buildDunningDoc(d, inv, 2))
	requireExactText(t, texts, "Mahngebuehr")
	requireNoExactText(t, texts, "Verzugszinsen")
	requireExactText(t, texts, "243.00 EUR")

	// Interest only.
	d.Fee, d.Interest = decimal.Zero, decimal.RequireFromString("1.75")
	texts = structureTexts(t, g.buildDunningDoc(d, inv, 2))
	requireNoExactText(t, texts, "Mahngebuehr")
	requireExactText(t, texts, "Verzugszinsen")
	requireExactText(t, texts, "239.75 EUR")
}

func TestGenerateDunningPDF_ProducesPDF(t *testing.T) {
	t.Parallel()

	d, inv := dunningFixture(t)
	b, err := NewGenerator(fullTestSettings()).GenerateDunningPDF(d, inv, 3)
	if err != nil {
		t.Fatalf("GenerateDunningPDF: %v", err)
	}
	requirePDF(t, b)
}

func TestGenerateZUGFeRDInvoicePDF_RejectsIncompleteSettings(t *testing.T) {
	t.Parallel()

	_, err := NewGenerator(models.CompanySettings{}).GenerateZUGFeRDInvoicePDF(completeTestInvoice(t))
	if err == nil {
		t.Fatal("expected GenerateZUGFeRDInvoicePDF to reject incomplete company settings")
	}
}

// TestEmbedZUGFeRDXML_AppendsToExistingAssociatedFiles belegt den Append-Pfad in
// declareFacturXAssociatedFile: trägt das PDF schon eine /AF-Zuordnung, bleibt
// sie erhalten und die neue Datei kommt dazu, statt das Array zu ersetzen.
func TestEmbedZUGFeRDXML_AppendsToExistingAssociatedFiles(t *testing.T) {
	t.Parallel()

	plain, err := NewGenerator(completeTestSettings()).GenerateInvoicePDF(completeTestInvoice(t))
	if err != nil {
		t.Fatalf("GenerateInvoicePDF: %v", err)
	}
	first, err := EmbedZUGFeRDXML(plain, []byte("<a/>"), "RE-1")
	if err != nil {
		t.Fatalf("first embed: %v", err)
	}
	second, err := EmbedZUGFeRDXML(first, []byte("<b/>"), "RE-1")
	if err != nil {
		t.Fatalf("second embed: %v", err)
	}

	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed
	ctx, _, _, _, err := api.ReadValidateAndOptimize(bytes.NewReader(second), conf, time.Now())
	if err != nil {
		t.Fatalf("read embedded PDF: %v", err)
	}
	root, err := ctx.Catalog()
	if err != nil {
		t.Fatalf("catalog: %v", err)
	}
	afObj, found := root.Find("AF")
	if !found {
		t.Fatal("catalog has no /AF entry")
	}
	af, err := ctx.DereferenceArray(afObj)
	if err != nil {
		t.Fatalf("dereference /AF: %v", err)
	}
	if len(af) != 2 {
		t.Errorf("expected the earlier association plus the new one (2 entries), got %d", len(af))
	}
}
