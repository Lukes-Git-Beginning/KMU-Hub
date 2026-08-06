package export_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"

	"github.com/kmuhub/kmuhub/internal/berichte"
	"github.com/kmuhub/kmuhub/internal/berichte/export"
)

func TestXLSXExporter_parseable(t *testing.T) {
	t.Parallel()

	e, _ := export.NewExporter("xlsx")
	var buf bytes.Buffer
	if err := e.Export(sampleResult(), &buf); err != nil {
		t.Fatalf("Export: %v", err)
	}

	f, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("excelize.OpenReader: %v", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		t.Fatal("expected at least one sheet")
	}
	found := false
	for _, s := range sheets {
		if s == "Bericht" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("sheet 'Bericht' not found; sheets: %v", sheets)
	}
}

func TestXLSXExporter_headerRow(t *testing.T) {
	t.Parallel()

	e, _ := export.NewExporter("xlsx")
	var buf bytes.Buffer
	if err := e.Export(sampleResult(), &buf); err != nil {
		t.Fatalf("Export: %v", err)
	}

	f, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("excelize.OpenReader: %v", err)
	}
	defer f.Close()

	// A1 should be the first header label.
	cell, err := f.GetCellValue("Bericht", "A1")
	if err != nil {
		t.Fatalf("GetCellValue A1: %v", err)
	}
	if cell != "Name" {
		t.Errorf("A1 = %q, want %q", cell, "Name")
	}

	// B1 should be the second header label.
	cell, err = f.GetCellValue("Bericht", "B1")
	if err != nil {
		t.Fatalf("GetCellValue B1: %v", err)
	}
	if cell != "Umsatz (EUR)" {
		t.Errorf("B1 = %q, want %q", cell, "Umsatz (EUR)")
	}
}

func TestXLSXExporter_dataRows(t *testing.T) {
	t.Parallel()

	e, _ := export.NewExporter("xlsx")
	var buf bytes.Buffer
	if err := e.Export(sampleResult(), &buf); err != nil {
		t.Fatalf("Export: %v", err)
	}

	f, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("excelize.OpenReader: %v", err)
	}
	defer f.Close()

	// Data starts at row 2.
	cell, err := f.GetCellValue("Bericht", "A2")
	if err != nil {
		t.Fatalf("GetCellValue A2: %v", err)
	}
	if cell != "Kunde A" {
		t.Errorf("A2 = %q, want %q", cell, "Kunde A")
	}

	cell, err = f.GetCellValue("Bericht", "A3")
	if err != nil {
		t.Fatalf("GetCellValue A3: %v", err)
	}
	if cell != "Kunde B" {
		t.Errorf("A3 = %q, want %q", cell, "Kunde B")
	}
}

func TestXLSXExporter_emptyRows(t *testing.T) {
	t.Parallel()

	e, _ := export.NewExporter("xlsx")
	var buf bytes.Buffer
	if err := e.Export(emptyResult(), &buf); err != nil {
		t.Fatalf("Export empty rows: %v", err)
	}

	f, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("excelize.OpenReader on empty result: %v", err)
	}
	defer f.Close()

	// Header must still be present.
	cell, err := f.GetCellValue("Bericht", "A1")
	if err != nil {
		t.Fatalf("GetCellValue A1 on empty: %v", err)
	}
	if cell != "Name" {
		t.Errorf("A1 = %q, want %q", cell, "Name")
	}

	// A2 should be empty (no data rows).
	cell, err = f.GetCellValue("Bericht", "A2")
	if err != nil {
		t.Fatalf("GetCellValue A2 on empty: %v", err)
	}
	if cell != "" {
		t.Errorf("A2 should be empty for zero-row result, got %q", cell)
	}
}

// invoiceLikeResult mirrors the shape executor.invoicesOpen produces: a string
// column, a currency (number) column and a date column pre-formatted as
// "2006-01-02" — the real report path never hands the exporter a time.Time.
func invoiceLikeResult() *berichte.ReportResult {
	return &berichte.ReportResult{
		Columns: []berichte.Column{
			{Name: "number", Label: "Rechnungsnr.", Kind: "string"},
			{Name: "amount", Label: "Betrag", Kind: "currency"},
			{Name: "due_date", Label: "Faellig", Kind: "date"},
		},
		Rows: []map[string]any{
			{"number": "RE-2026-001", "amount": 1234.56, "due_date": "2026-08-06"},
		},
		Meta: berichte.ReportMeta{
			GeneratedAt:  time.Now(),
			RowCount:     1,
			DefinitionID: uuid.New(),
		},
	}
}

// TestXLSXExporter_dateAndCurrencyFields is the excelize-bump regression test:
// it writes a workbook through the real exporter and reads the raw bytes back
// with the same library, so a version bump that silently changes cell/style
// encoding fails here even though the code still compiles and "just builds".
func TestXLSXExporter_dateAndCurrencyFields(t *testing.T) {
	t.Parallel()

	e, _ := export.NewExporter("xlsx")
	var buf bytes.Buffer
	if err := e.Export(invoiceLikeResult(), &buf); err != nil {
		t.Fatalf("Export: %v", err)
	}

	f, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("excelize.OpenReader: %v", err)
	}
	defer f.Close()

	// Header row for all three columns, including the date column.
	wantHeader := map[string]string{"A1": "Rechnungsnr.", "B1": "Betrag", "C1": "Faellig"}
	for cell, want := range wantHeader {
		got, err := f.GetCellValue("Bericht", cell)
		if err != nil {
			t.Fatalf("GetCellValue %s: %v", cell, err)
		}
		if got != want {
			t.Errorf("%s = %q, want %q", cell, got, want)
		}
	}

	// Currency field: written as text, must round-trip exactly.
	amount, err := f.GetCellValue("Bericht", "B2")
	if err != nil {
		t.Fatalf("GetCellValue B2: %v", err)
	}
	if amount != "1234.56" {
		t.Errorf("B2 (amount) = %q, want %q", amount, "1234.56")
	}

	// Date field: written as the pre-formatted "2006-01-02" string, must not
	// be reinterpreted as an Excel date serial by the writer or reader.
	dueDate, err := f.GetCellValue("Bericht", "C2")
	if err != nil {
		t.Fatalf("GetCellValue C2: %v", err)
	}
	if dueDate != "2026-08-06" {
		t.Errorf("C2 (due_date) = %q, want %q", dueDate, "2026-08-06")
	}
}

// TestXLSXExporter_headerBoldStyleRoundTrips guards the one real cell format
// the exporter applies (bold header). A style-ID/encoding regression from an
// excelize bump would leave the header cell on the default (unstyled) style
// while the build and every value-only test still pass.
func TestXLSXExporter_headerBoldStyleRoundTrips(t *testing.T) {
	t.Parallel()

	e, _ := export.NewExporter("xlsx")
	var buf bytes.Buffer
	if err := e.Export(sampleResult(), &buf); err != nil {
		t.Fatalf("Export: %v", err)
	}

	f, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("excelize.OpenReader: %v", err)
	}
	defer f.Close()

	styleID, err := f.GetCellStyle("Bericht", "A1")
	if err != nil {
		t.Fatalf("GetCellStyle A1: %v", err)
	}
	if styleID == 0 {
		t.Fatal("A1 has the default style; expected the bold header style to survive the round trip")
	}
	style, err := f.GetStyle(styleID)
	if err != nil {
		t.Fatalf("GetStyle %d: %v", styleID, err)
	}
	if style.Font == nil || !style.Font.Bold {
		t.Errorf("header style = %+v, want Font.Bold = true", style)
	}
}

func TestXLSXExporter_contentType(t *testing.T) {
	t.Parallel()
	e := &export.XLSXExporter{}
	if e.ContentType() != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Errorf("unexpected content type: %s", e.ContentType())
	}
	if e.FileExtension() != ".xlsx" {
		t.Errorf("unexpected extension: %s", e.FileExtension())
	}
}
