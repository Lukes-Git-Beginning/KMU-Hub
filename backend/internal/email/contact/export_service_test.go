package contact

import (
	"context"
	"encoding/csv"
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

	// Export
	data, err := exportSvc.ExportVCard(context.Background(), []uuid.UUID{c.ID})
	if err != nil {
		t.Fatalf("ExportVCard failed: %v", err)
	}

	// Import
	importProvider := newMockProvider()
	importSvc := NewImportService(importProvider, nil)

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

func TestExportImportRoundTrip_CompanyCSV(t *testing.T) {
	provider := newMockProvider()

	email := "roundtrip-company@example.com"
	companyID, err := provider.FindOrCreateCompany(context.Background(), "Acme GmbH", uuid.New())
	if err != nil {
		t.Fatalf("seed company failed: %v", err)
	}
	provider.contacts[email] = &models.Contact{
		ID:        uuid.New(),
		FirstName: "Max",
		LastName:  "Mustermann",
		Email:     &email,
		CompanyID: &companyID,
	}

	exportSvc := NewExportService(provider, nil)
	data, err := exportSvc.ExportCSV(context.Background(), []uuid.UUID{provider.contacts[email].ID}, nil)
	if err != nil {
		t.Fatalf("ExportCSV failed: %v", err)
	}
	if !strings.Contains(string(data), "Acme GmbH") {
		t.Fatal("expected company name Acme GmbH in exported CSV")
	}

	// Re-import into a fresh provider (no shared company table) to prove the company
	// relation survives the export -> import round trip, not just the in-process cache.
	// The BOM the exporter writes for Excel compatibility isn't stripped by the CSV
	// reader, so drop it here the same way a re-upload through the field-mapping wizard
	// would (the wizard reads the header row for column detection before this point).
	csvWithoutBOM := trimBOM(data)

	mapping := map[string]string{}
	for field, header := range crmFieldToGermanHeader {
		mapping[header] = field
	}

	importProvider := newMockProvider()
	importSvc := NewImportService(importProvider, nil)
	result, err := importSvc.ImportCSV(context.Background(), strings.NewReader(csvWithoutBOM), mapping, VisibilityShared, uuid.New(), false)
	if err != nil {
		t.Fatalf("ImportCSV round-trip failed: %v", err)
	}
	if result.ImportedCount != 1 {
		t.Fatalf("expected 1 imported in round-trip, got %d", result.ImportedCount)
	}

	imported := importProvider.created[0]
	if imported.CompanyID == nil {
		t.Fatal("expected the round-tripped contact to have a company_id")
	}
	if importProvider.companyNames[*imported.CompanyID] != "Acme GmbH" {
		t.Errorf("expected company Acme GmbH to survive the round trip, got %q", importProvider.companyNames[*imported.CompanyID])
	}
}

func TestExportCSV_FormulaInjectionNeutralized(t *testing.T) {
	provider := newMockProvider()
	email := "formula@example.com"
	lastName := "=HYPERLINK(\"http://evil.example/\",\"x\")"
	notes := "+cmd|'/C calc'!A0"
	c := &models.Contact{
		ID:        uuid.New(),
		FirstName: "-2+3",
		LastName:  lastName,
		Email:     &email,
		Notes:     &notes,
	}
	provider.contacts[email] = c

	svc := NewExportService(provider, nil)
	data, err := svc.ExportCSV(context.Background(), []uuid.UUID{c.ID}, []string{"first_name", "last_name", "email", "notes"})
	if err != nil {
		t.Fatalf("ExportCSV failed: %v", err)
	}

	rows := parseCSVRows(t, data)
	dataRow := rows[1] // rows[0] is the header

	if dataRow[0] != "'-2+3" {
		t.Errorf("expected first_name neutralized to %q, got %q", "'-2+3", dataRow[0])
	}
	if dataRow[1] != "'"+lastName {
		t.Errorf("expected last_name neutralized to %q, got %q", "'"+lastName, dataRow[1])
	}
	if dataRow[2] != email {
		t.Errorf("expected email untouched (no formula prefix), got %q", dataRow[2])
	}
	if dataRow[3] != "'"+notes {
		t.Errorf("expected notes neutralized to %q, got %q", "'"+notes, dataRow[3])
	}
}

func TestExportCSV_NormalValuesUnchanged(t *testing.T) {
	provider := newMockProvider()
	email := "normal@example.com"
	notes := "Regular note about the customer"
	c := &models.Contact{
		ID:        uuid.New(),
		FirstName: "Max",
		LastName:  "Mustermann",
		Email:     &email,
		Notes:     &notes,
	}
	provider.contacts[email] = c

	svc := NewExportService(provider, nil)
	data, err := svc.ExportCSV(context.Background(), []uuid.UUID{c.ID}, []string{"first_name", "last_name", "email", "notes"})
	if err != nil {
		t.Fatalf("ExportCSV failed: %v", err)
	}

	rows := parseCSVRows(t, data)
	dataRow := rows[1]

	if dataRow[0] != "Max" {
		t.Errorf("expected first_name unchanged, got %q", dataRow[0])
	}
	if dataRow[1] != "Mustermann" {
		t.Errorf("expected last_name unchanged, got %q", dataRow[1])
	}
	if dataRow[3] != notes {
		t.Errorf("expected notes unchanged, got %q", dataRow[3])
	}
}

// parseCSVRows strips the BOM and parses the CSV body into rows for cell-level assertions.
func parseCSVRows(t *testing.T, data []byte) [][]string {
	t.Helper()
	r := csv.NewReader(strings.NewReader(trimBOM(data)))
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatalf("failed to parse exported CSV: %v", err)
	}
	return rows
}

func trimBOM(data []byte) string {
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return string(data[3:])
	}
	return string(data)
}

func TestExportVCard_CompanyRoundTrip(t *testing.T) {
	provider := newMockProvider()

	email := "vcard-company@example.com"
	companyID, err := provider.FindOrCreateCompany(context.Background(), "Beta AG", uuid.New())
	if err != nil {
		t.Fatalf("seed company failed: %v", err)
	}
	provider.contacts[email] = &models.Contact{
		ID:        uuid.New(),
		FirstName: "Erika",
		LastName:  "Muster",
		Email:     &email,
		CompanyID: &companyID,
	}

	exportSvc := NewExportService(provider, nil)
	data, err := exportSvc.ExportVCard(context.Background(), []uuid.UUID{provider.contacts[email].ID})
	if err != nil {
		t.Fatalf("ExportVCard failed: %v", err)
	}
	if !strings.Contains(string(data), "Beta AG") {
		t.Fatal("expected ORG:Beta AG in exported vCard")
	}

	importProvider := newMockProvider()
	importSvc := NewImportService(importProvider, nil)
	result, err := importSvc.ImportVCard(context.Background(), strings.NewReader(string(data)), VisibilityShared, uuid.New(), false)
	if err != nil {
		t.Fatalf("ImportVCard round-trip failed: %v", err)
	}
	if result.ImportedCount != 1 {
		t.Fatalf("expected 1 imported in round-trip, got %d", result.ImportedCount)
	}

	imported := importProvider.created[0]
	if imported.CompanyID == nil {
		t.Fatal("expected the round-tripped contact to have a company_id")
	}
	if importProvider.companyNames[*imported.CompanyID] != "Beta AG" {
		t.Errorf("expected company Beta AG to survive the round trip, got %q", importProvider.companyNames[*imported.CompanyID])
	}
}
