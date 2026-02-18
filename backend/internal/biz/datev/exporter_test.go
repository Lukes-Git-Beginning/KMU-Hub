package datev

import (
	"encoding/json"
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

	csv, err := exporter.Export(invoices, nil, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Now())
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

	csv, err := exporter.Export(invoices, nil, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Now())
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

	csv, err := exporter.Export(nil, creditNotes, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Now())
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

	csv, err := exporter.Export(nil, nil, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Now())
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

func TestExport_EXTFHeaderFormat(t *testing.T) {
	exporter := NewExporter()

	fiscalYear := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	generated := time.Date(2026, 3, 15, 10, 30, 0, 0, time.UTC)

	csv, err := exporter.Export(nil, nil, fiscalYear, generated)
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
