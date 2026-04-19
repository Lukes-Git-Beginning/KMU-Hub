package export_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/kmuhub/kmuhub/internal/berichte/export"
)

func TestCSVExporter_BOM(t *testing.T) {
	t.Parallel()

	e, _ := export.NewExporter("csv")
	var buf bytes.Buffer
	if err := e.Export(sampleResult(), &buf); err != nil {
		t.Fatalf("Export: %v", err)
	}

	b := buf.Bytes()
	if len(b) < 3 {
		t.Fatalf("output too short")
	}
	// UTF-8 BOM
	if b[0] != 0xEF || b[1] != 0xBB || b[2] != 0xBF {
		t.Errorf("missing UTF-8 BOM: got %02X %02X %02X", b[0], b[1], b[2])
	}
}

func TestCSVExporter_semicolonSeparator(t *testing.T) {
	t.Parallel()

	e, _ := export.NewExporter("csv")
	var buf bytes.Buffer
	if err := e.Export(sampleResult(), &buf); err != nil {
		t.Fatalf("Export: %v", err)
	}

	// Strip BOM, then inspect header line.
	content := string(buf.Bytes()[3:])
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	if len(lines) == 0 {
		t.Fatal("no lines in CSV output")
	}

	header := lines[0]
	if !strings.Contains(header, ";") {
		t.Errorf("header line has no semicolon separator: %q", header)
	}
	// Must not use comma as primary separator.
	if strings.Contains(header, ",") {
		t.Errorf("header line must not use comma as separator: %q", header)
	}
}

func TestCSVExporter_headerLabels(t *testing.T) {
	t.Parallel()

	e, _ := export.NewExporter("csv")
	var buf bytes.Buffer
	if err := e.Export(sampleResult(), &buf); err != nil {
		t.Fatalf("Export: %v", err)
	}

	content := string(buf.Bytes()[3:])
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")

	header := lines[0]
	if !strings.Contains(header, "Name") {
		t.Errorf("header missing 'Name' column: %q", header)
	}
	if !strings.Contains(header, "Umsatz (EUR)") {
		t.Errorf("header missing 'Umsatz (EUR)' column: %q", header)
	}
}

func TestCSVExporter_dataRows(t *testing.T) {
	t.Parallel()

	e, _ := export.NewExporter("csv")
	var buf bytes.Buffer
	if err := e.Export(sampleResult(), &buf); err != nil {
		t.Fatalf("Export: %v", err)
	}

	content := string(buf.Bytes()[3:])
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")

	// 1 header + 2 data rows.
	if len(lines) != 3 {
		t.Errorf("expected 3 lines (header + 2 data), got %d", len(lines))
	}

	if !strings.Contains(lines[1], "Kunde A") {
		t.Errorf("data row 1 missing 'Kunde A': %q", lines[1])
	}
	if !strings.Contains(lines[2], "Kunde B") {
		t.Errorf("data row 2 missing 'Kunde B': %q", lines[2])
	}
}

func TestCSVExporter_emptyRows(t *testing.T) {
	t.Parallel()

	e, _ := export.NewExporter("csv")
	var buf bytes.Buffer
	if err := e.Export(emptyResult(), &buf); err != nil {
		t.Fatalf("Export empty: %v", err)
	}

	content := string(buf.Bytes()[3:])
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")

	// Should still have a header.
	if len(lines) != 1 || lines[0] == "" {
		t.Errorf("expected exactly 1 non-empty header line for empty result, got %d lines", len(lines))
	}
}

func TestCSVExporter_escapeSpecialChars(t *testing.T) {
	t.Parallel()

	result := sampleResult()
	// Inject a value with a semicolon — must be quoted.
	result.Rows[0]["name"] = `Mustermann; GmbH`

	e, _ := export.NewExporter("csv")
	var buf bytes.Buffer
	if err := e.Export(result, &buf); err != nil {
		t.Fatalf("Export: %v", err)
	}

	content := string(buf.Bytes()[3:])
	if !strings.Contains(content, `"Mustermann; GmbH"`) {
		t.Errorf("expected quoted semicolon field in output:\n%s", content)
	}
}

func TestCSVExporter_contentType(t *testing.T) {
	t.Parallel()
	e := &export.CSVExporter{}
	if e.ContentType() != "text/csv; charset=utf-8" {
		t.Errorf("unexpected content type: %s", e.ContentType())
	}
	if e.FileExtension() != ".csv" {
		t.Errorf("unexpected extension: %s", e.FileExtension())
	}
}
