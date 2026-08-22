package datev

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

func makeLineItems(items []models.LineItem) json.RawMessage {
	data, _ := json.Marshal(items)
	return data
}

func TestExport_SingleInvoice19Percent(t *testing.T) {
	exporter := NewExporter()

	invoices := []*models.Invoice{
		{
			ID:            uuid.New(),
			TenantID:      uuid.New(),
			InvoiceNumber: "RE-2026-0001",
			Status:        models.InvoiceStatusSent,
			CustomerName:  "Test GmbH",
			TaxMode:       models.TaxModeStandard,
			LineItems: makeLineItems([]models.LineItem{
				{
					ID:          "1",
					Position:    1,
					Description: "Beratung",
					Quantity:    decimal.NewFromInt(2),
					UnitPrice:   decimal.NewFromFloat(100.00),
					TaxRate:     decimal.NewFromInt(19),
					LineTotal:   decimal.NewFromFloat(200.00),
				},
			}),
			InvoiceDate: time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
		},
	}

	csv, err := exporter.Export(invoices, nil, "", "", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Now())
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	content := string(csv)

	// Verify UTF-8 BOM
	if !strings.HasPrefix(content, "\xEF\xBB\xBF") {
		t.Error("missing UTF-8 BOM")
	}

	// Verify EXTF header
	if !strings.Contains(content, "EXTF") {
		t.Error("missing EXTF header")
	}

	// Verify booking line contains correct data
	if !strings.Contains(content, "238,00") { // 200 * 1.19 = 238.00
		t.Error("expected gross amount 238,00")
	}
	if !strings.Contains(content, ";S;") {
		t.Error("expected Soll (S) for invoice")
	}
	if !strings.Contains(content, ";EUR;") {
		t.Error("expected EUR currency")
	}
	if !strings.Contains(content, ";8400;") {
		t.Error("expected SKR03 account 8400 for 19%")
	}
	if !strings.Contains(content, ";3;") {
		t.Error("expected BU-Schluessel 3 for 19%")
	}
	if !strings.Contains(content, "1503") { // Belegdatum 15.03 -> 1503
		t.Error("expected Belegdatum 1503")
	}
	if !strings.Contains(content, "RE-2026-0001") {
		t.Error("expected invoice number in Belegfeld 1")
	}
	if !strings.Contains(content, "Beratung") {
		t.Error("expected description in Buchungstext")
	}
}

func TestExport_MixedTaxRates(t *testing.T) {
	exporter := NewExporter()

	invoices := []*models.Invoice{
		{
			ID:            uuid.New(),
			TenantID:      uuid.New(),
			InvoiceNumber: "RE-2026-0002",
			Status:        models.InvoiceStatusPaid,
			CustomerName:  "Mixed GmbH",
			TaxMode:       models.TaxModeStandard,
			LineItems: makeLineItems([]models.LineItem{
				{
					ID:          "1",
					Position:    1,
					Description: "Software-Entwicklung",
					Quantity:    decimal.NewFromInt(1),
					UnitPrice:   decimal.NewFromFloat(1000.00),
					TaxRate:     decimal.NewFromInt(19),
					LineTotal:   decimal.NewFromFloat(1000.00),
				},
				{
					ID:          "2",
					Position:    2,
					Description: "Buch: Go Programmierung",
					Quantity:    decimal.NewFromInt(1),
					UnitPrice:   decimal.NewFromFloat(50.00),
					TaxRate:     decimal.NewFromInt(7),
					LineTotal:   decimal.NewFromFloat(50.00),
				},
			}),
			InvoiceDate: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	csv, err := exporter.Export(invoices, nil, "", "", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Now())
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	content := string(csv)

	// Should have 2 booking lines (one per line item)
	lines := strings.Split(content, "\n")
	bookingLines := 0
	for _, line := range lines {
		if strings.Contains(line, ";S;") {
			bookingLines++
		}
	}
	if bookingLines != 2 {
		t.Errorf("expected 2 booking lines, got %d", bookingLines)
	}

	// Verify 19% line: 1000 * 1.19 = 1190.00
	if !strings.Contains(content, "1190,00") {
		t.Error("expected gross amount 1190,00 for 19% item")
	}
	if !strings.Contains(content, ";8400;") {
		t.Error("expected account 8400 for 19%")
	}

	// Verify 7% line: 50 * 1.07 = 53.50
	if !strings.Contains(content, "53,50") {
		t.Error("expected gross amount 53,50 for 7% item")
	}
	if !strings.Contains(content, ";8300;") {
		t.Error("expected account 8300 for 7%")
	}
}

func TestExport_CreditNote(t *testing.T) {
	exporter := NewExporter()

	creditNotes := []*models.CreditNote{
		{
			ID:               uuid.New(),
			TenantID:         uuid.New(),
			CreditNoteNumber: "GS-2026-0001",
			Status:           models.CreditNoteStatusSent,
			CustomerName:     "Credit Customer",
			TaxMode:          models.TaxModeStandard,
			LineItems: makeLineItems([]models.LineItem{
				{
					ID:          "1",
					Position:    1,
					Description: "Gutschrift Beratung",
					Quantity:    decimal.NewFromInt(1),
					UnitPrice:   decimal.NewFromFloat(500.00),
					TaxRate:     decimal.NewFromInt(19),
					LineTotal:   decimal.NewFromFloat(500.00),
				},
			}),
			CreatedAt: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC),
		},
	}

	csv, err := exporter.Export(nil, creditNotes, "", "", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Now())
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	content := string(csv)

	// Verify "H" (Haben = credit) for credit note
	if !strings.Contains(content, ";H;") {
		t.Error("expected Haben (H) for credit note")
	}

	// 500 * 1.19 = 595.00
	if !strings.Contains(content, "595,00") {
		t.Error("expected gross amount 595,00")
	}

	if !strings.Contains(content, "GS-2026-0001") {
		t.Error("expected credit note number in Belegfeld 1")
	}
}

func TestExport_EmptyInput(t *testing.T) {
	exporter := NewExporter()

	csv, err := exporter.Export(nil, nil, "", "", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Now())
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	content := string(csv)

	// Should still have header and column headers, but no booking lines
	if !strings.Contains(content, "EXTF") {
		t.Error("missing EXTF header")
	}
	if !strings.Contains(content, "Umsatz") {
		t.Error("missing column headers")
	}

	// Should not have any S or H booking lines
	lines := strings.Split(content, "\n")
	bookingLines := 0
	for _, line := range lines {
		if strings.Contains(line, ";S;") || strings.Contains(line, ";H;") {
			bookingLines++
		}
	}
	if bookingLines != 0 {
		t.Errorf("expected 0 booking lines for empty input, got %d", bookingLines)
	}
}

// TestStreamWriter_PagedMatchesSingleShot proves the streaming path (paged
// WriteInvoices/WriteCreditNotes) produces byte-identical output to the in-memory
// Export, i.e. the customerIndex/debitor state persists correctly across pages.
// Gamma GmbH (first seen on page 2) is the discriminator: if the index reset per
// page, it would be assigned a different debitor account and the bytes would differ.
func TestStreamWriter_PagedMatchesSingleShot(t *testing.T) {
	exporter := NewExporter()
	tenant := uuid.New()
	fy := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	gen := time.Date(2026, 3, 15, 10, 30, 0, 0, time.UTC)

	mkInv := func(num, cust string, day int) *models.Invoice {
		return &models.Invoice{
			ID:            uuid.New(),
			TenantID:      tenant,
			InvoiceNumber: num,
			Status:        models.InvoiceStatusSent,
			CustomerName:  cust,
			TaxMode:       models.TaxModeStandard,
			LineItems: makeLineItems([]models.LineItem{{
				ID: "1", Position: 1, Description: "Beratung",
				Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromFloat(100.00),
				TaxRate: decimal.NewFromInt(19), LineTotal: decimal.NewFromFloat(100.00),
			}}),
			InvoiceDate: time.Date(2026, 3, day, 0, 0, 0, 0, time.UTC),
		}
	}
	invoices := []*models.Invoice{
		mkInv("RE-2026-0001", "Alpha GmbH", 1),
		mkInv("RE-2026-0002", "Beta GmbH", 2),
		mkInv("RE-2026-0003", "Alpha GmbH", 3), // same customer as #1 -> same debitor
		mkInv("RE-2026-0004", "Gamma GmbH", 4),
	}
	cn := &models.CreditNote{
		ID:               uuid.New(),
		TenantID:         tenant,
		CreditNoteNumber: "GS-2026-0001",
		Status:           models.CreditNoteStatusSent,
		CustomerName:     "Beta GmbH",
		TaxMode:          models.TaxModeStandard,
		LineItems: makeLineItems([]models.LineItem{{
			ID: "1", Position: 1, Description: "Gutschrift",
			Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromFloat(50.00),
			TaxRate: decimal.NewFromInt(19), LineTotal: decimal.NewFromFloat(50.00),
		}}),
		CreatedAt: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC),
	}

	golden, err := exporter.Export(invoices, []*models.CreditNote{cn}, "", "", fy, gen)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Same data, but written in pages via the streaming API.
	var buf bytes.Buffer
	sw, err := exporter.NewStreamWriter(&buf, "", "", fy, gen)
	if err != nil {
		t.Fatalf("NewStreamWriter failed: %v", err)
	}
	if err := sw.WriteInvoices(invoices[:2]); err != nil {
		t.Fatalf("WriteInvoices page 1: %v", err)
	}
	if err := sw.WriteInvoices(invoices[2:]); err != nil {
		t.Fatalf("WriteInvoices page 2: %v", err)
	}
	if err := sw.WriteCreditNotes([]*models.CreditNote{cn}); err != nil {
		t.Fatalf("WriteCreditNotes: %v", err)
	}
	if err := sw.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if !bytes.Equal(golden, buf.Bytes()) {
		t.Errorf("paged stream bytes differ from single-shot Export\n--- golden ---\n%q\n--- paged ---\n%q", golden, buf.Bytes())
	}

	// 4 invoice booking lines + 1 credit note booking line.
	if got := sw.LineCount(); got != 5 {
		t.Errorf("expected LineCount 5, got %d", got)
	}
}

func TestExport_EXTFHeaderFormat(t *testing.T) {
	exporter := NewExporter()

	fiscalYear := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	generated := time.Date(2026, 3, 15, 10, 30, 0, 0, time.UTC)

	csv, err := exporter.Export(nil, nil, "", "", fiscalYear, generated)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	content := string(csv)
	// Remove BOM for easier parsing
	content = strings.TrimPrefix(content, "\xEF\xBB\xBF")

	lines := strings.Split(content, "\n")
	if len(lines) < 2 {
		t.Fatal("expected at least 2 lines (header + column headers)")
	}

	headerLine := lines[0]

	// Verify EXTF header fields
	if !strings.HasPrefix(headerLine, "EXTF;700;21;Buchungsstapel;13;") {
		t.Errorf("EXTF header format incorrect: %s", headerLine)
	}

	// Verify generated timestamp format (YYYYMMDDHHMMSSMMM)
	if !strings.Contains(headerLine, "20260315103000000") {
		t.Errorf("expected generated timestamp 20260315103000000 in header, got: %s", headerLine)
	}

	// Verify source application
	if !strings.Contains(headerLine, "KMU Hub") {
		t.Error("expected 'KMU Hub' source application in header")
	}

	// Verify fiscal year start
	if !strings.Contains(headerLine, "20260101") {
		t.Error("expected fiscal year start 20260101 in header")
	}

	// Verify account length (SKR03 = 4 digits)
	if !strings.Contains(headerLine, ";4;") {
		t.Error("expected account length 4 in header")
	}
}

// TestExport_HeaderConsultantClientAndForeignCurrency verifies the EXTF header
// carries Beraternummer/Mandantennummer and that booking lines use the document
// currency instead of a hardcoded EUR (R3-7c-4).
func TestExport_HeaderConsultantClientAndForeignCurrency(t *testing.T) {
	exporter := NewExporter()

	invoices := []*models.Invoice{
		{
			ID:            uuid.New(),
			TenantID:      uuid.New(),
			InvoiceNumber: "RE-2026-0099",
			Status:        models.InvoiceStatusSent,
			CustomerName:  "Helvetia AG",
			TaxMode:       models.TaxModeStandard,
			Currency:      "CHF",
			LineItems: makeLineItems([]models.LineItem{
				{ID: "1", Position: 1, Description: "Beratung", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromFloat(100.00), TaxRate: decimal.NewFromInt(19), LineTotal: decimal.NewFromFloat(100.00)},
			}),
			InvoiceDate: time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
		},
	}

	csv, err := exporter.Export(invoices, nil, "1234", "56789", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Now())
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}
	content := string(csv)

	if !strings.Contains(content, ";1234;56789;") {
		t.Error("expected Beraternummer 1234 and Mandantennummer 56789 in the EXTF header")
	}
	if !strings.Contains(content, ";CHF;") {
		t.Error("expected WKZ Umsatz to be the document currency CHF, not EUR")
	}
}

// TestExport_GoldenBytesWithUmlauts is a byte-for-byte comparison against a
// fixed expected output, per c-cov-biz-datev's done_when. The backlog notes
// asked for the charset to be "belegt" as Windows-1252 — that premise does
// not hold against this code: NewStreamWriter writes a UTF-8 BOM (EF BB BF)
// and encoding/csv itself only ever emits UTF-8. There is no cp1252
// transcoding step anywhere in this package (confirmed by reading
// exporter.go top to bottom and grepping the whole datev package for a
// charmap/golang.org/x/text/encoding import — there is none). So this test
// documents and locks in what actually ships: UTF-8, with umlauts and the
// eszett as their real multi-byte UTF-8 sequences (c3bc for "ü", c39f for
// "ß" — verified byte-for-byte below, not just via string containment). If
// DATEV import ever turns out to require cp1252 in practice, that is a
// product decision (a real encoding step, a new dependency) — out of scope
// for a coverage-only unit, and not something to silently invent here.
//
// RESEARCH UPDATE (2026-08-22, verify-datev-extf-encoding-requirement,
// iteration 39): could not find a directly quotable, version-numbered
// primary source that settles this. developer.datev.de's own booking-batch
// and header field specs for format 700 / category 21 (fetched directly,
// not from a search snippet) say nothing about character encoding at all.
// A dedicated "Zeichensatz" page exists on that portal but 404s without a
// login. Search-engine synthesis of that page's indexed content (not
// independently verified by reading the page myself) claims: default
// import charset is ISO-8859-1/CP1252, but Unicode (UTF-8/-16/-32) is also
// accepted if a BOM is present — restricted to manual import in DATEV
// Rechnungswesen or the "accounting:extf-files" Online API, i.e. exactly
// what this exporter produces. Separately, the independent open-source
// "ledermann/datev" Ruby gem (unrelated to DATEV or this project)
// hardcodes CP1252 for its EXTF booking-batch export, which is real-world
// signal that automated/non-API import paths may still expect CP1252.
// Net: evidence is mixed and doesn't clear the bar this unit set for
// itself. Do NOT reinterpret this as either a confirmation or a refutation
// — it's an open question that needs either paid developer.datev.de portal
// access to the primary spec, or an empirical test against a real DATEV
// import. Unit is `blocked` for that reason; encoding behavior here is
// unchanged.
func TestExport_GoldenBytesWithUmlauts(t *testing.T) {
	exporter := NewExporter()
	items := []models.LineItem{{
		ID: "1", Position: 1, Description: "Prüfung für Straßenbau",
		Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromFloat(100.00),
		TaxRate: decimal.NewFromInt(19), LineTotal: decimal.NewFromFloat(100.00),
	}}
	lineItemsRaw, err := json.Marshal(items)
	if err != nil {
		t.Fatalf("marshal line items: %v", err)
	}
	invoices := []*models.Invoice{
		{
			ID:            uuid.New(),
			TenantID:      uuid.New(),
			InvoiceNumber: "RE-2026-0100",
			Status:        models.InvoiceStatusSent,
			CustomerName:  "Müller GmbH",
			TaxMode:       models.TaxModeStandard,
			Currency:      "EUR",
			LineItems:     lineItemsRaw,
			InvoiceDate:   time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC),
		},
	}

	got, err := exporter.Export(invoices, nil, "1001", "2002",
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 15, 10, 30, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	want := "\xef\xbb\xbf" +
		"EXTF;700;21;Buchungsstapel;13;20260315103000000;;KMU Hub;;;1001;2002;20260101;4;;;;;;\n" +
		"Umsatz (ohne Soll/Haben-Kz);Soll/Haben-Kennzeichen;WKZ Umsatz;Kurs;Basis-Umsatz;WKZ Basis-Umsatz;Konto;Gegenkonto (ohne BU-Schluessel);BU-Schluessel;Belegdatum;Belegfeld 1;Belegfeld 2;Skonto;Buchungstext\n" +
		"119,00;S;EUR;;;;10001;8400;3;2003;RE-2026-0100;;;Prüfung für Straßenbau\n"

	if !bytes.Equal(got, []byte(want)) {
		t.Fatalf("golden byte mismatch\n--- got ---\n%q\n--- want ---\n%q", got, want)
	}

	// Belt-and-braces: prove the umlaut/eszett bytes are the real UTF-8
	// sequences, not an accidental ASCII-substitution or mojibake that a
	// looser string comparison could miss.
	if !bytes.Contains(got, []byte{0xC3, 0xBC}) { // "ü"
		t.Error("expected the UTF-8 byte sequence for 'ü' (0xC3 0xBC) in the output")
	}
	if !bytes.Contains(got, []byte{0xC3, 0x9F}) { // "ß"
		t.Error("expected the UTF-8 byte sequence for 'ß' (0xC3 0x9F) in the output")
	}
	if !bytes.HasPrefix(got, []byte{0xEF, 0xBB, 0xBF}) {
		t.Error("expected the file to start with a UTF-8 BOM (0xEF 0xBB 0xBF)")
	}
}

// TestWriteInvoices_SkipsNonExportableStatuses proves draft and cancelled
// invoices never reach the CSV — only sent/paid/overdue do (line 103-107 of
// exporter.go). A skipped invoice must not even count towards DocumentCount.
func TestWriteInvoices_SkipsNonExportableStatuses(t *testing.T) {
	exporter := NewExporter()
	cancelled := sentInvoice("RE-CANCELLED")
	cancelled.Status = "cancelled"
	invoices := []*models.Invoice{draftInvoice("RE-DRAFT"), cancelled}

	var buf bytes.Buffer
	sw, err := exporter.NewStreamWriter(&buf, "", "", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("NewStreamWriter failed: %v", err)
	}
	if err := sw.WriteInvoices(invoices); err != nil {
		t.Fatalf("WriteInvoices failed: %v", err)
	}
	if sw.LineCount() != 0 {
		t.Errorf("expected 0 booking lines for draft/cancelled invoices, got %d", sw.LineCount())
	}
	if sw.DocumentCount() != 0 {
		t.Errorf("expected DocumentCount 0, got %d", sw.DocumentCount())
	}
	content := buf.String()
	if strings.Contains(content, ";S;") {
		t.Error("draft/cancelled invoice must not produce a booking line")
	}
}

// TestWriteCreditNotes_SkipsNonSentStatus mirrors the invoice skip test for
// credit notes: only "sent" credit notes are exported (line 137 of exporter.go).
func TestWriteCreditNotes_SkipsNonSentStatus(t *testing.T) {
	exporter := NewExporter()
	creditNotes := []*models.CreditNote{
		{
			ID: uuid.New(), CreditNoteNumber: "GS-DRAFT", Status: models.CreditNoteStatusDraft,
			CustomerName: "Test GmbH", TaxMode: models.TaxModeStandard,
			LineItems: makeLineItems([]models.LineItem{{
				ID: "1", Position: 1, Description: "Draft", Quantity: decimal.NewFromInt(1),
				UnitPrice: decimal.NewFromFloat(10), TaxRate: decimal.NewFromInt(19), LineTotal: decimal.NewFromFloat(10),
			}}),
			CreatedAt: time.Now(),
		},
	}

	var buf bytes.Buffer
	sw, err := exporter.NewStreamWriter(&buf, "", "", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("NewStreamWriter failed: %v", err)
	}
	if err := sw.WriteCreditNotes(creditNotes); err != nil {
		t.Fatalf("WriteCreditNotes failed: %v", err)
	}
	if sw.LineCount() != 0 {
		t.Errorf("expected 0 booking lines for a draft credit note, got %d", sw.LineCount())
	}
	if strings.Contains(buf.String(), ";H;") {
		t.Error("draft credit note must not produce a booking line")
	}
}

// TestWriteInvoices_InvalidLineItemsJSON_ReturnsWrappedError proves a
// malformed line_items JSONB blob is surfaced as an error naming the invoice,
// not silently skipped or panicked on.
func TestWriteInvoices_InvalidLineItemsJSON_ReturnsWrappedError(t *testing.T) {
	exporter := NewExporter()
	inv := sentInvoice("RE-BROKEN")
	inv.LineItems = json.RawMessage(`{not valid json`)

	var buf bytes.Buffer
	sw, err := exporter.NewStreamWriter(&buf, "", "", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("NewStreamWriter failed: %v", err)
	}
	err = sw.WriteInvoices([]*models.Invoice{inv})
	if err == nil {
		t.Fatal("expected an error for malformed line_items JSON")
	}
	if !strings.Contains(err.Error(), "RE-BROKEN") && !strings.Contains(err.Error(), inv.ID.String()) {
		t.Errorf("expected the error to identify the offending invoice, got: %v", err)
	}
}

// TestWriteCreditNotes_InvalidLineItemsJSON_ReturnsWrappedError mirrors the
// invoice case for credit notes.
func TestWriteCreditNotes_InvalidLineItemsJSON_ReturnsWrappedError(t *testing.T) {
	exporter := NewExporter()
	cn := &models.CreditNote{
		ID: uuid.New(), CreditNoteNumber: "GS-BROKEN", Status: models.CreditNoteStatusSent,
		CustomerName: "Test GmbH", TaxMode: models.TaxModeStandard,
		LineItems: json.RawMessage(`{not valid json`),
		CreatedAt: time.Now(),
	}

	var buf bytes.Buffer
	sw, err := exporter.NewStreamWriter(&buf, "", "", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("NewStreamWriter failed: %v", err)
	}
	err = sw.WriteCreditNotes([]*models.CreditNote{cn})
	if err == nil {
		t.Fatal("expected an error for malformed line_items JSON")
	}
	if !strings.Contains(err.Error(), "GS-BROKEN") && !strings.Contains(err.Error(), cn.ID.String()) {
		t.Errorf("expected the error to identify the offending credit note, got: %v", err)
	}
}

// TestExport_WriteInvoicesErrorPropagates proves Export() forwards a
// WriteInvoices failure instead of returning a partial/truncated file.
func TestExport_WriteInvoicesErrorPropagates(t *testing.T) {
	exporter := NewExporter()
	inv := sentInvoice("RE-BROKEN")
	inv.LineItems = json.RawMessage(`{not valid json`)

	_, err := exporter.Export([]*models.Invoice{inv}, nil, "", "", time.Now(), time.Now())
	if err == nil {
		t.Fatal("expected Export to propagate the WriteInvoices error")
	}
}

// TestExport_WriteCreditNotesErrorPropagates mirrors the above for credit notes.
func TestExport_WriteCreditNotesErrorPropagates(t *testing.T) {
	exporter := NewExporter()
	cn := &models.CreditNote{
		ID: uuid.New(), CreditNoteNumber: "GS-BROKEN", Status: models.CreditNoteStatusSent,
		CustomerName: "Test GmbH", TaxMode: models.TaxModeStandard,
		LineItems: json.RawMessage(`{not valid json`),
		CreatedAt: time.Now(),
	}

	_, err := exporter.Export(nil, []*models.CreditNote{cn}, "", "", time.Now(), time.Now())
	if err == nil {
		t.Fatal("expected Export to propagate the WriteCreditNotes error")
	}
}

// alwaysFailWriter fails on every Write call, simulating a downstream sink
// (disk, pipe, HTTP body) that stops accepting bytes mid-export.
type alwaysFailWriter struct{}

func (alwaysFailWriter) Write(p []byte) (int, error) {
	return 0, errors.New("simulated write failure")
}

// TestNewStreamWriter_BOMWriteErrorIsWrapped proves a failing sink is caught
// at the very first write (the UTF-8 BOM), not later.
func TestNewStreamWriter_BOMWriteErrorIsWrapped(t *testing.T) {
	exporter := NewExporter()
	_, err := exporter.NewStreamWriter(alwaysFailWriter{}, "", "", time.Now(), time.Now())
	if err == nil {
		t.Fatal("expected an error when the underlying writer rejects the BOM")
	}
	if !strings.Contains(err.Error(), "write BOM") {
		t.Errorf("expected the error to name the BOM write step, got: %v", err)
	}
}

// TestStreamWriter_CloseReturnsWrappedFlushError proves a downstream write
// failure surfaces through Close() as a wrapped "csv flush" error instead of
// being swallowed. Constructed directly (same-package test) so the failure
// is deterministic instead of depending on bufio's internal buffer size.
func TestStreamWriter_CloseReturnsWrappedFlushError(t *testing.T) {
	sw := &StreamWriter{
		w:             csv.NewWriter(alwaysFailWriter{}),
		customerIndex: make(map[string]int),
		nextIndex:     1,
	}
	if err := sw.w.Write([]string{"buffered", "row"}); err != nil {
		t.Fatalf("buffering a row into csv.Writer should not itself fail: %v", err)
	}

	err := sw.Close()
	if err == nil {
		t.Fatal("expected Close to surface the flush error")
	}
	if !strings.Contains(err.Error(), "csv flush") {
		t.Errorf("expected a wrapped 'csv flush' error, got: %v", err)
	}
}
