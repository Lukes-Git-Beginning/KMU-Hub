package audit

import (
	"bytes"
	"encoding/csv"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

func TestExportCSV_NeutralizesFormulaInjection(t *testing.T) {
	entries := []*models.AuditEntry{
		{
			ID:          uuid.New(),
			SequenceNum: 1,
			Timestamp:   time.Now(),
			Action:      "contact.update",
			Target:      "=HYPERLINK(\"http://evil.example/\",\"x\")",
			TargetType:  "contact",
			Details:     "+cmd|'/C calc'!A0",
			IPAddress:   "10.0.0.1",
			UserAgent:   "-Custom Agent",
			Result:      "success",
			EntryHash:   "deadbeef",
		},
	}

	data, err := ExportCSV(entries)
	if err != nil {
		t.Fatalf("ExportCSV failed: %v", err)
	}

	r := csv.NewReader(bytes.NewReader(data[3:])) // skip BOM
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("CSV parse error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected header + 1 data row, got %d", len(records))
	}
	row := records[1]

	if got := row[5]; got != "'"+entries[0].Target {
		t.Errorf("expected Target neutralized to %q, got %q", "'"+entries[0].Target, got)
	}
	if got := row[7]; got != "'"+entries[0].Details {
		t.Errorf("expected Details neutralized to %q, got %q", "'"+entries[0].Details, got)
	}
	if got := row[9]; got != "'"+entries[0].UserAgent {
		t.Errorf("expected UserAgent neutralized to %q, got %q", "'"+entries[0].UserAgent, got)
	}
}

func TestExportCSV_NormalValuesUnchanged(t *testing.T) {
	entries := []*models.AuditEntry{
		{
			ID:          uuid.New(),
			SequenceNum: 1,
			Timestamp:   time.Now(),
			Action:      "contact.update",
			Target:      "contact-123",
			TargetType:  "contact",
			Details:     "normal details text",
			IPAddress:   "10.0.0.1",
			UserAgent:   "Mozilla/5.0",
			Result:      "success",
			EntryHash:   "deadbeef",
		},
	}

	data, err := ExportCSV(entries)
	if err != nil {
		t.Fatalf("ExportCSV failed: %v", err)
	}

	r := csv.NewReader(bytes.NewReader(data[3:]))
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("CSV parse error: %v", err)
	}
	row := records[1]

	if row[5] != entries[0].Target {
		t.Errorf("expected Target untouched, got %q", row[5])
	}
	if row[7] != entries[0].Details {
		t.Errorf("expected Details untouched, got %q", row[7])
	}
	if row[9] != entries[0].UserAgent {
		t.Errorf("expected UserAgent untouched, got %q", row[9])
	}
}
