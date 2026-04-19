package export_test

import (
	"bytes"
	"testing"

	"github.com/xuri/excelize/v2"

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
