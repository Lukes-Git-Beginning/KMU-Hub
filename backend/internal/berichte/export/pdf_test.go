package export_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/berichte"
	"github.com/kmuhub/kmuhub/internal/berichte/export"
)

// sampleResult builds a minimal ReportResult with two columns and two data rows.
func sampleResult() *berichte.ReportResult {
	return &berichte.ReportResult{
		Columns: []berichte.Column{
			{Name: "name", Label: "Name", Kind: "string"},
			{Name: "revenue", Label: "Umsatz (EUR)", Kind: "currency"},
		},
		Rows: []map[string]any{
			{"name": "Kunde A", "revenue": 12345.67},
			{"name": "Kunde B", "revenue": 9876.00},
		},
		Meta: berichte.ReportMeta{
			GeneratedAt:  time.Now(),
			RowCount:     2,
			DefinitionID: uuid.New(),
		},
	}
}

// emptyResult has columns but no data rows.
func emptyResult() *berichte.ReportResult {
	return &berichte.ReportResult{
		Columns: []berichte.Column{
			{Name: "name", Label: "Name", Kind: "string"},
		},
		Rows: []map[string]any{},
		Meta: berichte.ReportMeta{
			GeneratedAt:  time.Now(),
			DefinitionID: uuid.New(),
		},
	}
}

func TestPDFExporter_prefix(t *testing.T) {
	t.Parallel()

	e, err := export.NewExporter("pdf")
	if err != nil {
		t.Fatalf("NewExporter: %v", err)
	}

	var buf bytes.Buffer
	if err := e.Export(sampleResult(), &buf); err != nil {
		t.Fatalf("Export: %v", err)
	}

	b := buf.Bytes()
	if len(b) < 4 {
		t.Fatalf("output too short: %d bytes", len(b))
	}
	if string(b[:4]) != "%PDF" {
		t.Errorf("expected %%PDF prefix, got %q", string(b[:4]))
	}
}

func TestPDFExporter_zeroRows(t *testing.T) {
	t.Parallel()

	e, _ := export.NewExporter("pdf")
	var buf bytes.Buffer
	if err := e.Export(emptyResult(), &buf); err != nil {
		t.Fatalf("Export with zero rows: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("expected non-empty PDF even for zero-row result")
	}
	if string(buf.Bytes()[:4]) != "%PDF" {
		t.Error("expected %PDF prefix on zero-row result")
	}
}

func TestPDFExporter_contentType(t *testing.T) {
	t.Parallel()
	e := &export.PDFExporter{}
	if e.ContentType() != "application/pdf" {
		t.Errorf("unexpected content type: %s", e.ContentType())
	}
	if e.FileExtension() != ".pdf" {
		t.Errorf("unexpected extension: %s", e.FileExtension())
	}
}

func TestPDFExporter_noColumns(t *testing.T) {
	t.Parallel()

	result := &berichte.ReportResult{
		Columns: []berichte.Column{},
		Rows:    []map[string]any{},
		Meta:    berichte.ReportMeta{GeneratedAt: time.Now(), DefinitionID: uuid.New()},
	}

	e, _ := export.NewExporter("pdf")
	var buf bytes.Buffer
	if err := e.Export(result, &buf); err != nil {
		t.Fatalf("Export no-columns: %v", err)
	}
	if !strings.HasPrefix(buf.String(), "%PDF") {
		t.Error("expected %PDF prefix on no-column result")
	}
}
