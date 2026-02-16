package contact

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

func TestExportCSV_BasicExport(t *testing.T) {
	provider := newMockProvider()

	email1 := "max@example.com"
	phone1 := "+49123"
	pos1 := "Manager"
	c1 := &models.Contact{
		ID:        uuid.New(),
		FirstName: "Max",
		LastName:  "Mustermann",
		Email:     &email1,
		Phone:     &phone1,
		Position:  &pos1,
	}
	provider.contacts[email1] = c1

	svc := NewExportService(provider, nil)

	data, err := svc.ExportCSV(context.Background(), []uuid.UUID{c1.ID}, []string{"first_name", "last_name", "email", "phone", "position"})
	if err != nil {
		t.Fatalf("ExportCSV failed: %v", err)
	}

	csvStr := string(data)

	// Check BOM
	if !strings.HasPrefix(csvStr, "\xef\xbb\xbf") {
		t.Error("expected UTF-8 BOM at start of CSV")
	}

	// Check German headers
	if !strings.Contains(csvStr, "Vorname") {
		t.Error("expected Vorname in CSV header")
	}
	if !strings.Contains(csvStr, "Nachname") {
		t.Error("expected Nachname in CSV header")
	}
	if !strings.Contains(csvStr, "E-Mail") {
		t.Error("expected E-Mail in CSV header")
	}

	// Check data
	if !strings.Contains(csvStr, "Max") {
		t.Error("expected Max in CSV data")
	}
	if !strings.Contains(csvStr, "max@example.com") {
		t.Error("expected max@example.com in CSV data")
	}
}

func TestExportCSV_FieldSelection(t *testing.T) {
	provider := newMockProvider()

	email := "test@example.com"
	notes := "Some notes"
	c := &models.Contact{
		ID:        uuid.New(),
		FirstName: "Erika",
		LastName:  "Muster",
		Email:     &email,
		Notes:     &notes,
	}
	provider.contacts[email] = c

	svc := NewExportService(provider, nil)

	// Only export first_name and email
	data, err := svc.ExportCSV(context.Background(), []uuid.UUID{c.ID}, []string{"first_name", "email"})
	if err != nil {
		t.Fatalf("ExportCSV failed: %v", err)
	}

	csvStr := string(data)

	if !strings.Contains(csvStr, "Vorname") {
		t.Error("expected Vorname in header")
	}
	if !strings.Contains(csvStr, "E-Mail") {
		t.Error("expected E-Mail in header")
	}
	// Should NOT contain Nachname, Notizen
	if strings.Contains(csvStr, "Nachname") {
		t.Error("expected Nachname NOT in header for selected fields")
	}
	if strings.Contains(csvStr, "Notizen") {
		t.Error("expected Notizen NOT in header for selected fields")
	}
}

func TestExportVCard_Basic(t *testing.T) {
	provider := newMockProvider()

	email := "max@example.com"
	phone := "+49123"
	position := "CEO"
	notes := "Important contact"
	c := &models.Contact{
		ID:        uuid.New(),
		FirstName: "Max",
		LastName:  "Mustermann",
		Email:     &email,
		Phone:     &phone,
		Position:  &position,
		Notes:     &notes,
	}
	provider.contacts[email] = c

	svc := NewExportService(provider, nil)

	data, err := svc.ExportVCard(context.Background(), []uuid.UUID{c.ID})
	if err != nil {
		t.Fatalf("ExportVCard failed: %v", err)
	}

	vcardStr := string(data)

	if !strings.Contains(vcardStr, "BEGIN:VCARD") {
		t.Error("expected BEGIN:VCARD")
	}
	if !strings.Contains(vcardStr, "END:VCARD") {
		t.Error("expected END:VCARD")
	}
	if !strings.Contains(vcardStr, "FN:Max Mustermann") {
		t.Error("expected FN:Max Mustermann")
	}
	if !strings.Contains(vcardStr, "max@example.com") {
		t.Error("expected email in vCard")
	}
	if !strings.Contains(vcardStr, "+49123") {
		t.Error("expected phone in vCard")
	}
}

func TestExportVCard_RoundTrip(t *testing.T) {
	provider := newMockProvider()

	email := "roundtrip@example.com"
	phone := "+49999"
	position := "Developer"
	c := &models.Contact{
		ID:        uuid.New(),
		FirstName: "Test",
		LastName:  "Roundtrip",
		Email:     &email,
		Phone:     &phone,
		Position:  &position,
	}
	provider.contacts[email] = c

	exportSvc := NewExportService(provider, nil)
	importSvc := NewImportService(newMockProvider(), nil)

	// Export
	data, err := exportSvc.ExportVCard(context.Background(), []uuid.UUID{c.ID})
	if err != nil {
		t.Fatalf("ExportVCard failed: %v", err)
	}

	// Import
	importProvider := newMockProvider()
	importSvc = NewImportService(importProvider, nil)

	result, err := importSvc.ImportVCard(context.Background(), strings.NewReader(string(data)), VisibilityShared, uuid.New(), false)
	if err != nil {
		t.Fatalf("ImportVCard round-trip failed: %v", err)
	}

	if result.ImportedCount != 1 {
		t.Errorf("expected 1 imported in round-trip, got %d", result.ImportedCount)
	}

	if len(importProvider.created) != 1 {
		t.Fatalf("expected 1 created in round-trip, got %d", len(importProvider.created))
	}

	imported := importProvider.created[0]
	if imported.FirstName != "Test" {
		t.Errorf("expected FirstName Test, got %s", imported.FirstName)
	}
	if imported.LastName != "Roundtrip" {
		t.Errorf("expected LastName Roundtrip, got %s", imported.LastName)
	}
	if imported.Email == nil || *imported.Email != "roundtrip@example.com" {
		t.Error("expected email roundtrip@example.com after round-trip")
	}
}

func TestExportCSV_BOMPresence(t *testing.T) {
	provider := newMockProvider()
	email := "test@test.com"
	provider.contacts[email] = &models.Contact{
		ID:        uuid.New(),
		FirstName: "T",
		LastName:  "T",
		Email:     &email,
	}

	svc := NewExportService(provider, nil)
	data, err := svc.ExportCSV(context.Background(), []uuid.UUID{provider.contacts[email].ID}, nil)
	if err != nil {
		t.Fatalf("ExportCSV failed: %v", err)
	}

	if len(data) < 3 {
		t.Fatal("CSV too short")
	}
	if data[0] != 0xEF || data[1] != 0xBB || data[2] != 0xBF {
		t.Error("expected UTF-8 BOM (EF BB BF) at start of CSV")
	}
}
