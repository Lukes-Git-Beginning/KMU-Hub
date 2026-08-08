package contact

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"

	"github.com/kmuhub/kmuhub/internal/models"
)

// mockContactProvider implements ContactProvider for testing.
type mockContactProvider struct {
	contacts     map[string]*models.Contact // email -> contact
	created      []*models.Contact
	updated      []*models.Contact
	companies    map[string]uuid.UUID // normalized name -> id
	companyNames map[uuid.UUID]string // id -> name
	companyCalls int                  // FindOrCreateCompany invocations, to assert per-run caching
}

func newMockProvider() *mockContactProvider {
	return &mockContactProvider{
		contacts:     make(map[string]*models.Contact),
		companies:    make(map[string]uuid.UUID),
		companyNames: make(map[uuid.UUID]string),
	}
}

func (m *mockContactProvider) FindOrCreateCompany(_ context.Context, name string, _ uuid.UUID) (uuid.UUID, error) {
	m.companyCalls++
	key := strings.ToLower(strings.TrimSpace(name))
	if id, ok := m.companies[key]; ok {
		return id, nil
	}
	id := uuid.New()
	m.companies[key] = id
	m.companyNames[id] = strings.TrimSpace(name)
	return id, nil
}

func (m *mockContactProvider) GetCompanyNames(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	result := make(map[uuid.UUID]string)
	for _, id := range ids {
		if name, ok := m.companyNames[id]; ok {
			result[id] = name
		}
	}
	return result, nil
}

func (m *mockContactProvider) GetByEmail(_ context.Context, email string) (*models.Contact, error) {
	c, ok := m.contacts[strings.ToLower(email)]
	if !ok {
		return nil, ErrImportFailed
	}
	return c, nil
}

func (m *mockContactProvider) CreateForImport(_ context.Context, c *models.Contact) error {
	if c.Email != nil {
		m.contacts[strings.ToLower(*c.Email)] = c
	}
	m.created = append(m.created, c)
	return nil
}

func (m *mockContactProvider) UpdateForImport(_ context.Context, c *models.Contact) error {
	m.updated = append(m.updated, c)
	return nil
}

func (m *mockContactProvider) ListByIDs(_ context.Context, ids []uuid.UUID) ([]*models.Contact, error) {
	var result []*models.Contact
	for _, c := range m.contacts {
		for _, id := range ids {
			if c.ID == id {
				result = append(result, c)
			}
		}
	}
	return result, nil
}

func (m *mockContactProvider) ListVisible(_ context.Context, _ uuid.UUID, _ bool) ([]*models.Contact, error) {
	var result []*models.Contact
	for _, c := range m.contacts {
		result = append(result, c)
	}
	return result, nil
}

func TestPreviewCSV_CommaDelimiter(t *testing.T) {
	csv := "Vorname,Nachname,E-Mail,Telefon\nMax,Mustermann,max@example.com,+49123\nErika,Muster,erika@example.com,+49456\n"
	svc := NewImportService(newMockProvider(), nil)

	preview, err := svc.PreviewCSV(context.Background(), strings.NewReader(csv))
	if err != nil {
		t.Fatalf("PreviewCSV failed: %v", err)
	}

	if len(preview.Columns) != 4 {
		t.Errorf("expected 4 columns, got %d", len(preview.Columns))
	}

	if len(preview.SampleRows) != 2 {
		t.Errorf("expected 2 sample rows, got %d", len(preview.SampleRows))
	}

	// Check auto-detected mapping
	if preview.DetectedMapping["Vorname"] != "first_name" {
		t.Errorf("expected Vorname -> first_name, got %s", preview.DetectedMapping["Vorname"])
	}
	if preview.DetectedMapping["E-Mail"] != "email" {
		t.Errorf("expected E-Mail -> email, got %s", preview.DetectedMapping["E-Mail"])
	}
}

func TestPreviewCSV_SemicolonDelimiter(t *testing.T) {
	csv := "FirstName;LastName;Email\nJohn;Doe;john@example.com\n"
	svc := NewImportService(newMockProvider(), nil)

	preview, err := svc.PreviewCSV(context.Background(), strings.NewReader(csv))
	if err != nil {
		t.Fatalf("PreviewCSV failed: %v", err)
	}

	if len(preview.Columns) != 3 {
		t.Errorf("expected 3 columns, got %d", len(preview.Columns))
	}
	if preview.DetectedMapping["FirstName"] != "first_name" {
		t.Errorf("expected FirstName -> first_name, got %s", preview.DetectedMapping["FirstName"])
	}
}

func TestImportCSV_BasicImport(t *testing.T) {
	csvData := "Vorname,Nachname,E-Mail\nMax,Mustermann,max@example.com\nErika,Muster,erika@example.com\n"
	provider := newMockProvider()
	svc := NewImportService(provider, nil)

	mapping := map[string]string{
		"Vorname":  "first_name",
		"Nachname": "last_name",
		"E-Mail":   "email",
	}

	ownerID := uuid.New()
	result, err := svc.ImportCSV(context.Background(), strings.NewReader(csvData), mapping, VisibilityShared, ownerID, false)
	if err != nil {
		t.Fatalf("ImportCSV failed: %v", err)
	}

	if result.ImportedCount != 2 {
		t.Errorf("expected 2 imported, got %d", result.ImportedCount)
	}
	if result.MergedCount != 0 {
		t.Errorf("expected 0 merged, got %d", result.MergedCount)
	}
	if len(provider.created) != 2 {
		t.Errorf("expected 2 contacts created, got %d", len(provider.created))
	}
}

func TestImportCSV_MergeByEmail(t *testing.T) {
	provider := newMockProvider()
	existingEmail := "max@example.com"
	provider.contacts[existingEmail] = &models.Contact{
		ID:        uuid.New(),
		FirstName: "Max",
		LastName:  "Mustermann",
		Email:     &existingEmail,
	}

	csvData := "Vorname,Nachname,E-Mail,Telefon\nMax,Mustermann,max@example.com,+49123\n"
	svc := NewImportService(provider, nil)

	mapping := map[string]string{
		"Vorname":  "first_name",
		"Nachname": "last_name",
		"E-Mail":   "email",
		"Telefon":  "phone",
	}

	result, err := svc.ImportCSV(context.Background(), strings.NewReader(csvData), mapping, VisibilityShared, uuid.New(), true)
	if err != nil {
		t.Fatalf("ImportCSV failed: %v", err)
	}

	if result.MergedCount != 1 {
		t.Errorf("expected 1 merged, got %d", result.MergedCount)
	}
	if result.ImportedCount != 0 {
		t.Errorf("expected 0 imported, got %d", result.ImportedCount)
	}
	if len(provider.updated) != 1 {
		t.Errorf("expected 1 update, got %d", len(provider.updated))
	}
	if provider.updated[0].Phone == nil || *provider.updated[0].Phone != "+49123" {
		t.Error("expected phone to be merged")
	}
}

func TestImportCSV_SkipInvalidEmail(t *testing.T) {
	csvData := "Vorname,Nachname,E-Mail\nBad,Row,not-an-email\n"
	provider := newMockProvider()
	svc := NewImportService(provider, nil)

	mapping := map[string]string{
		"Vorname":  "first_name",
		"Nachname": "last_name",
		"E-Mail":   "email",
	}

	result, err := svc.ImportCSV(context.Background(), strings.NewReader(csvData), mapping, VisibilityShared, uuid.New(), false)
	if err != nil {
		t.Fatalf("ImportCSV failed: %v", err)
	}

	if result.SkippedCount != 1 {
		t.Errorf("expected 1 skipped, got %d", result.SkippedCount)
	}
	if len(result.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(result.Errors))
	}
}

func TestImportCSV_TabDelimiter(t *testing.T) {
	csvData := "Vorname\tNachname\tE-Mail\nMax\tMustermann\tmax@example.com\n"
	provider := newMockProvider()
	svc := NewImportService(provider, nil)

	mapping := map[string]string{
		"Vorname":  "first_name",
		"Nachname": "last_name",
		"E-Mail":   "email",
	}

	result, err := svc.ImportCSV(context.Background(), strings.NewReader(csvData), mapping, VisibilityShared, uuid.New(), false)
	if err != nil {
		t.Fatalf("ImportCSV failed: %v", err)
	}

	if result.ImportedCount != 1 {
		t.Errorf("expected 1 imported, got %d", result.ImportedCount)
	}
}

func TestImportVCard_Basic(t *testing.T) {
	vcardData := "BEGIN:VCARD\r\nVERSION:4.0\r\nFN:Max Mustermann\r\nN:Mustermann;Max;;;\r\nEMAIL:max@example.com\r\nTEL:+49123456\r\nTITLE:Manager\r\nEND:VCARD\r\n"
	provider := newMockProvider()
	svc := NewImportService(provider, nil)

	result, err := svc.ImportVCard(context.Background(), strings.NewReader(vcardData), VisibilityPersonal, uuid.New(), false)
	if err != nil {
		t.Fatalf("ImportVCard failed: %v", err)
	}

	if result.ImportedCount != 1 {
		t.Errorf("expected 1 imported, got %d", result.ImportedCount)
	}

	if len(provider.created) != 1 {
		t.Fatalf("expected 1 contact created, got %d", len(provider.created))
	}

	c := provider.created[0]
	if c.FirstName != "Max" {
		t.Errorf("expected first_name Max, got %s", c.FirstName)
	}
	if c.LastName != "Mustermann" {
		t.Errorf("expected last_name Mustermann, got %s", c.LastName)
	}
	if c.Email == nil || *c.Email != "max@example.com" {
		t.Error("expected email max@example.com")
	}
	if c.Phone == nil || *c.Phone != "+49123456" {
		t.Error("expected phone +49123456")
	}
	if c.Position == nil || *c.Position != "Manager" {
		t.Error("expected position Manager")
	}
	if c.Visibility != VisibilityPersonal {
		t.Errorf("expected personal visibility, got %s", c.Visibility)
	}
}

func TestImportVCard_MergeByEmail(t *testing.T) {
	existingEmail := "max@example.com"
	provider := newMockProvider()
	provider.contacts[existingEmail] = &models.Contact{
		ID:        uuid.New(),
		FirstName: "Max",
		LastName:  "Mustermann",
		Email:     &existingEmail,
	}

	vcardData := "BEGIN:VCARD\r\nVERSION:4.0\r\nFN:Max Mustermann\r\nEMAIL:max@example.com\r\nTEL:+49123456\r\nEND:VCARD\r\n"
	svc := NewImportService(provider, nil)

	result, err := svc.ImportVCard(context.Background(), strings.NewReader(vcardData), VisibilityShared, uuid.New(), true)
	if err != nil {
		t.Fatalf("ImportVCard failed: %v", err)
	}

	if result.MergedCount != 1 {
		t.Errorf("expected 1 merged, got %d", result.MergedCount)
	}
}

func TestImportCSV_EmptyNames(t *testing.T) {
	csvData := "Vorname,Nachname,E-Mail\n,,test@example.com\n"
	provider := newMockProvider()
	svc := NewImportService(provider, nil)

	mapping := map[string]string{
		"Vorname":  "first_name",
		"Nachname": "last_name",
		"E-Mail":   "email",
	}

	result, err := svc.ImportCSV(context.Background(), strings.NewReader(csvData), mapping, VisibilityShared, uuid.New(), false)
	if err != nil {
		t.Fatalf("ImportCSV failed: %v", err)
	}

	if result.SkippedCount != 1 {
		t.Errorf("expected 1 skipped (empty names), got %d", result.SkippedCount)
	}
}

func TestImportCSV_CompanyResolvedOnCreate(t *testing.T) {
	csvData := "Vorname,Nachname,E-Mail,Firma\nMax,Mustermann,max@example.com,Acme GmbH\n"
	provider := newMockProvider()
	svc := NewImportService(provider, nil)

	mapping := map[string]string{
		"Vorname":  "first_name",
		"Nachname": "last_name",
		"E-Mail":   "email",
		"Firma":    "company",
	}

	result, err := svc.ImportCSV(context.Background(), strings.NewReader(csvData), mapping, VisibilityShared, uuid.New(), false)
	if err != nil {
		t.Fatalf("ImportCSV failed: %v", err)
	}
	if result.ImportedCount != 1 {
		t.Fatalf("expected 1 imported, got %d", result.ImportedCount)
	}

	c := provider.created[0]
	if c.CompanyID == nil {
		t.Fatal("expected contact to have a company_id")
	}
	if provider.companyNames[*c.CompanyID] != "Acme GmbH" {
		t.Errorf("expected company name Acme GmbH, got %q", provider.companyNames[*c.CompanyID])
	}
}

func TestImportCSV_CompanyDedupedWithinRun(t *testing.T) {
	csvData := "Vorname,Nachname,E-Mail,Firma\n" +
		"Max,Mustermann,max@example.com,Acme GmbH\n" +
		"Erika,Muster,erika@example.com, acme gmbh \n"
	provider := newMockProvider()
	svc := NewImportService(provider, nil)

	mapping := map[string]string{
		"Vorname":  "first_name",
		"Nachname": "last_name",
		"E-Mail":   "email",
		"Firma":    "company",
	}

	result, err := svc.ImportCSV(context.Background(), strings.NewReader(csvData), mapping, VisibilityShared, uuid.New(), false)
	if err != nil {
		t.Fatalf("ImportCSV failed: %v", err)
	}
	if result.ImportedCount != 2 {
		t.Fatalf("expected 2 imported, got %d", result.ImportedCount)
	}

	if provider.companyCalls != 1 {
		t.Errorf("expected exactly 1 FindOrCreateCompany call for 2 rows of the same company (case/whitespace-insensitive), got %d", provider.companyCalls)
	}
	if len(provider.created) != 2 || provider.created[0].CompanyID == nil || provider.created[1].CompanyID == nil {
		t.Fatal("expected both contacts to carry a company_id")
	}
	if *provider.created[0].CompanyID != *provider.created[1].CompanyID {
		t.Error("expected both contacts to resolve to the same company")
	}
}

func TestImportCSV_MergeByEmail_FillsMissingCompany(t *testing.T) {
	provider := newMockProvider()
	existingEmail := "max@example.com"
	provider.contacts[existingEmail] = &models.Contact{
		ID:        uuid.New(),
		FirstName: "Max",
		LastName:  "Mustermann",
		Email:     &existingEmail,
	}

	csvData := "Vorname,Nachname,E-Mail,Firma\nMax,Mustermann,max@example.com,Acme GmbH\n"
	svc := NewImportService(provider, nil)

	mapping := map[string]string{
		"Vorname":  "first_name",
		"Nachname": "last_name",
		"E-Mail":   "email",
		"Firma":    "company",
	}

	result, err := svc.ImportCSV(context.Background(), strings.NewReader(csvData), mapping, VisibilityShared, uuid.New(), true)
	if err != nil {
		t.Fatalf("ImportCSV failed: %v", err)
	}
	if result.MergedCount != 1 {
		t.Fatalf("expected 1 merged, got %d", result.MergedCount)
	}
	if len(provider.updated) != 1 || provider.updated[0].CompanyID == nil {
		t.Fatal("expected the merged contact to have picked up the company from the import row")
	}
}

func TestImportCSV_EmptyCompanyLeavesContactUnassigned(t *testing.T) {
	csvData := "Vorname,Nachname,E-Mail,Firma\nMax,Mustermann,max@example.com,\n"
	provider := newMockProvider()
	svc := NewImportService(provider, nil)

	mapping := map[string]string{
		"Vorname":  "first_name",
		"Nachname": "last_name",
		"E-Mail":   "email",
		"Firma":    "company",
	}

	result, err := svc.ImportCSV(context.Background(), strings.NewReader(csvData), mapping, VisibilityShared, uuid.New(), false)
	if err != nil {
		t.Fatalf("ImportCSV failed: %v", err)
	}
	if result.ImportedCount != 1 {
		t.Fatalf("expected 1 imported, got %d", result.ImportedCount)
	}
	if provider.created[0].CompanyID != nil {
		t.Error("expected no company_id when the company column is blank")
	}
	if provider.companyCalls != 0 {
		t.Errorf("expected FindOrCreateCompany not to be called for a blank company, got %d calls", provider.companyCalls)
	}
}

// buildXLSX writes rows (row 0 = header) to sheet "Sheet1" of a new in-memory
// workbook and returns its bytes, ready to feed into ImportXLSX/excelize.OpenReader.
func buildXLSX(t *testing.T, rows [][]string) []byte {
	t.Helper()
	f := excelize.NewFile()
	defer f.Close() //nolint:errcheck // close on cleanup only

	for rowIdx, row := range rows {
		for colIdx, val := range row {
			cell, err := excelize.CoordinatesToCellName(colIdx+1, rowIdx+1)
			if err != nil {
				t.Fatalf("CoordinatesToCellName: %v", err)
			}
			if err := f.SetCellValue("Sheet1", cell, val); err != nil {
				t.Fatalf("SetCellValue: %v", err)
			}
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatalf("write xlsx: %v", err)
	}
	return buf.Bytes()
}

func TestImportXLSX_BasicImport(t *testing.T) {
	data := buildXLSX(t, [][]string{
		{"Vorname", "Nachname", "E-Mail"},
		{"Max", "Mustermann", "max@example.com"},
		{"Erika", "Muster", "erika@example.com"},
	})
	provider := newMockProvider()
	svc := NewImportService(provider, nil)

	mapping := map[string]string{
		"Vorname":  "first_name",
		"Nachname": "last_name",
		"E-Mail":   "email",
	}

	result, err := svc.ImportXLSX(context.Background(), bytes.NewReader(data), mapping, VisibilityShared, uuid.New(), false)
	if err != nil {
		t.Fatalf("ImportXLSX failed: %v", err)
	}
	if result.ImportedCount != 2 {
		t.Errorf("expected 2 imported, got %d", result.ImportedCount)
	}
	if len(provider.created) != 2 {
		t.Errorf("expected 2 contacts created, got %d", len(provider.created))
	}
}

func TestImportXLSX_MergeByEmail(t *testing.T) {
	provider := newMockProvider()
	existingEmail := "max@example.com"
	provider.contacts[existingEmail] = &models.Contact{
		ID:        uuid.New(),
		FirstName: "Max",
		LastName:  "Mustermann",
		Email:     &existingEmail,
	}

	data := buildXLSX(t, [][]string{
		{"Vorname", "Nachname", "E-Mail", "Telefon"},
		{"Max", "Mustermann", "max@example.com", "+49123"},
	})
	svc := NewImportService(provider, nil)

	mapping := map[string]string{
		"Vorname":  "first_name",
		"Nachname": "last_name",
		"E-Mail":   "email",
		"Telefon":  "phone",
	}

	result, err := svc.ImportXLSX(context.Background(), bytes.NewReader(data), mapping, VisibilityShared, uuid.New(), true)
	if err != nil {
		t.Fatalf("ImportXLSX failed: %v", err)
	}
	if result.MergedCount != 1 {
		t.Errorf("expected 1 merged, got %d", result.MergedCount)
	}
	if provider.updated[0].Phone == nil || *provider.updated[0].Phone != "+49123" {
		t.Error("expected phone to be merged")
	}
}

func TestImportXLSX_SkipInvalidEmail(t *testing.T) {
	data := buildXLSX(t, [][]string{
		{"Vorname", "Nachname", "E-Mail"},
		{"Bad", "Row", "not-an-email"},
	})
	provider := newMockProvider()
	svc := NewImportService(provider, nil)

	mapping := map[string]string{
		"Vorname":  "first_name",
		"Nachname": "last_name",
		"E-Mail":   "email",
	}

	result, err := svc.ImportXLSX(context.Background(), bytes.NewReader(data), mapping, VisibilityShared, uuid.New(), false)
	if err != nil {
		t.Fatalf("ImportXLSX failed: %v", err)
	}
	if result.SkippedCount != 1 {
		t.Errorf("expected 1 skipped, got %d", result.SkippedCount)
	}
	if len(result.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(result.Errors))
	}
	if result.Errors[0].Row != 2 {
		t.Errorf("expected the failing row to be reported as workbook row 2, got %d", result.Errors[0].Row)
	}
}

func TestImportXLSX_CorruptFileReturnsErrInvalidXLSX(t *testing.T) {
	provider := newMockProvider()
	svc := NewImportService(provider, nil)

	_, err := svc.ImportXLSX(context.Background(), strings.NewReader("this is not a zip/xlsx file"), nil, VisibilityShared, uuid.New(), false)
	if err == nil {
		t.Fatal("expected an error for a corrupt file, got nil")
	}
	if !errors.Is(err, ErrInvalidXLSX) {
		t.Errorf("expected ErrInvalidXLSX, got %v", err)
	}
}

func TestImportXLSX_RowCapTruncatesImport(t *testing.T) {
	rows := [][]string{{"Vorname", "Nachname", "E-Mail"}}
	for i := 0; i < maxXLSXImportRows+50; i++ {
		rows = append(rows, []string{"Person", fmt.Sprintf("Nr%d", i), fmt.Sprintf("person%d@example.com", i)})
	}
	data := buildXLSX(t, rows)
	provider := newMockProvider()
	svc := NewImportService(provider, nil)

	mapping := map[string]string{
		"Vorname":  "first_name",
		"Nachname": "last_name",
		"E-Mail":   "email",
	}

	result, err := svc.ImportXLSX(context.Background(), bytes.NewReader(data), mapping, VisibilityShared, uuid.New(), false)
	if err != nil {
		t.Fatalf("ImportXLSX failed: %v", err)
	}
	if result.ImportedCount != maxXLSXImportRows {
		t.Errorf("expected exactly the cap of %d imported, got %d", maxXLSXImportRows, result.ImportedCount)
	}
	if len(provider.created) != maxXLSXImportRows {
		t.Errorf("expected exactly %d contacts created, got %d", maxXLSXImportRows, len(provider.created))
	}
}
