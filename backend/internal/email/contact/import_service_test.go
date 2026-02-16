package contact

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// mockContactProvider implements ContactProvider for testing.
type mockContactProvider struct {
	contacts map[string]*models.Contact // email -> contact
	created  []*models.Contact
	updated  []*models.Contact
}

func newMockProvider() *mockContactProvider {
	return &mockContactProvider{
		contacts: make(map[string]*models.Contact),
	}
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

func testLogger() *testLogHandler {
	return &testLogHandler{}
}

type testLogHandler struct{}

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
