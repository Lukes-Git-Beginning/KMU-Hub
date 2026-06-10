package einvoice

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ============================================================================
// Mock Repository
// ============================================================================

type mockRepo struct {
	invoices      map[uuid.UUID]*models.IncomingInvoice
	createErr     error
	getErr        error
	updateErr     error
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		invoices: make(map[uuid.UUID]*models.IncomingInvoice),
	}
}

func (m *mockRepo) Create(_ context.Context, inv *models.IncomingInvoice) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.invoices[inv.ID] = inv
	return nil
}

func (m *mockRepo) GetByID(_ context.Context, tenantID, id uuid.UUID) (*models.IncomingInvoice, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	inv, ok := m.invoices[id]
	if !ok || inv.TenantID != tenantID {
		return nil, ErrNotFound
	}
	return inv, nil
}

func (m *mockRepo) List(_ context.Context, tenantID uuid.UUID, _ ListFilter) ([]*models.IncomingInvoice, int, error) {
	var result []*models.IncomingInvoice
	for _, inv := range m.invoices {
		if inv.TenantID == tenantID {
			result = append(result, inv)
		}
	}
	return result, len(result), nil
}

func (m *mockRepo) UpdateStatus(_ context.Context, tenantID, id uuid.UUID, status string, updatedAt time.Time) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	inv, ok := m.invoices[id]
	if !ok || inv.TenantID != tenantID {
		return ErrNotFound
	}
	inv.Status = status
	inv.UpdatedAt = updatedAt
	return nil
}

// ============================================================================
// Import Tests
// ============================================================================

func TestService_Import_XMLHappyPath(t *testing.T) {
	xmlData, err := os.ReadFile("testdata/cii_minimal.xml")
	require.NoError(t, err)

	svc := NewService(newMockRepo())
	tenantID := uuid.New()
	createdBy := uuid.New()

	inv, err := svc.Import(context.Background(), ImportInput{
		TenantID:  tenantID,
		CreatedBy: createdBy,
		Filename:  "test.xml",
		Content:   xmlData,
		MimeType:  "application/xml",
	})

	require.NoError(t, err)
	require.NotNil(t, inv)
	assert.Equal(t, tenantID, inv.TenantID)
	assert.Equal(t, "Lieferant GmbH", inv.SupplierName)
	assert.Equal(t, "RE-TEST-2024-001", inv.InvoiceNumber)
	assert.Equal(t, models.IncomingInvoiceStatusReceived, inv.Status)
	assert.Equal(t, models.IncomingInvoiceFormatZUGFeRDCII, inv.SourceFormat)
	assert.NotEmpty(t, inv.OriginalXML)
	assert.Equal(t, "test.xml", inv.OriginalFilename)

	// line_items must be valid JSON
	var items []json.RawMessage
	require.NoError(t, json.Unmarshal(inv.LineItems, &items))
	assert.Len(t, items, 1)
}

func TestService_Import_UBLXMLHappyPath(t *testing.T) {
	xmlData, err := os.ReadFile("testdata/ubl_minimal.xml")
	require.NoError(t, err)

	svc := NewService(newMockRepo())
	inv, err := svc.Import(context.Background(), ImportInput{
		TenantID:  uuid.New(),
		CreatedBy: uuid.New(),
		Filename:  "xrechnung.xml",
		Content:   xmlData,
		MimeType:  "text/xml",
	})

	require.NoError(t, err)
	assert.Equal(t, "XRechnung Lieferant GmbH", inv.SupplierName)
	assert.Equal(t, "XR-TEST-2024-001", inv.InvoiceNumber)
	assert.Equal(t, models.IncomingInvoiceFormatXRechnungUBL, inv.SourceFormat)
}

func TestService_Import_PDFWithoutXML(t *testing.T) {
	svc := NewService(newMockRepo())
	_, err := svc.Import(context.Background(), ImportInput{
		TenantID:  uuid.New(),
		CreatedBy: uuid.New(),
		Filename:  "plain.pdf",
		Content:   []byte("%PDF-1.4 minimal stub"),
		MimeType:  "application/pdf",
	})
	assert.ErrorIs(t, err, ErrNoEmbeddedXML)
}

func TestService_Import_DuplicateReturnsError(t *testing.T) {
	repo := newMockRepo()
	repo.createErr = ErrDuplicateInvoice
	svc := NewService(repo)

	xmlData, err := os.ReadFile("testdata/cii_minimal.xml")
	require.NoError(t, err)

	_, err = svc.Import(context.Background(), ImportInput{
		TenantID:  uuid.New(),
		CreatedBy: uuid.New(),
		Filename:  "dup.xml",
		Content:   xmlData,
		MimeType:  "application/xml",
	})
	assert.ErrorIs(t, err, ErrDuplicateInvoice)
}

// ============================================================================
// Get / List Tests
// ============================================================================

func TestService_Get_NotFound(t *testing.T) {
	svc := NewService(newMockRepo())
	_, err := svc.Get(context.Background(), uuid.New(), uuid.New())
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestService_List_ReturnsResults(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)
	tenantID := uuid.New()

	// Seed two invoices
	xmlData, err := os.ReadFile("testdata/cii_minimal.xml")
	require.NoError(t, err)
	for i := 0; i < 2; i++ {
		_, err := svc.Import(context.Background(), ImportInput{
			TenantID:  tenantID,
			CreatedBy: uuid.New(),
			Filename:  "inv.xml",
			Content:   xmlData,
			MimeType:  "application/xml",
		})
		// Second import will hit duplicate due to same invoice_number in XML
		// This is expected behavior — only first should succeed
		_ = err
	}

	result, total, err := svc.List(context.Background(), tenantID, ListInput{})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, 1)
	assert.NotEmpty(t, result)
}

// ============================================================================
// Status Transition Tests
// ============================================================================

func TestService_UpdateStatus_ValidTransition(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)
	tenantID := uuid.New()

	xmlData, err := os.ReadFile("testdata/cii_minimal.xml")
	require.NoError(t, err)

	inv, err := svc.Import(context.Background(), ImportInput{
		TenantID:  tenantID,
		CreatedBy: uuid.New(),
		Content:   xmlData,
		MimeType:  "application/xml",
	})
	require.NoError(t, err)
	assert.Equal(t, models.IncomingInvoiceStatusReceived, inv.Status)

	// received → reviewed
	updated, err := svc.UpdateStatus(context.Background(), tenantID, inv.ID, models.IncomingInvoiceStatusReviewed)
	require.NoError(t, err)
	assert.Equal(t, models.IncomingInvoiceStatusReviewed, updated.Status)

	// reviewed → booked
	booked, err := svc.UpdateStatus(context.Background(), tenantID, inv.ID, models.IncomingInvoiceStatusBooked)
	require.NoError(t, err)
	assert.Equal(t, models.IncomingInvoiceStatusBooked, booked.Status)
}

func TestService_UpdateStatus_InvalidTransition(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)
	tenantID := uuid.New()

	xmlData, err := os.ReadFile("testdata/cii_minimal.xml")
	require.NoError(t, err)

	inv, err := svc.Import(context.Background(), ImportInput{
		TenantID:  tenantID,
		CreatedBy: uuid.New(),
		Content:   xmlData,
		MimeType:  "application/xml",
	})
	require.NoError(t, err)

	// received → booked is not a valid direct transition
	_, err = svc.UpdateStatus(context.Background(), tenantID, inv.ID, models.IncomingInvoiceStatusBooked)
	assert.True(t, errors.Is(err, ErrInvalidStatusTransition), "expected ErrInvalidStatusTransition, got %v", err)
}

func TestService_UpdateStatus_NotFound(t *testing.T) {
	svc := NewService(newMockRepo())
	_, err := svc.UpdateStatus(context.Background(), uuid.New(), uuid.New(), models.IncomingInvoiceStatusReviewed)
	assert.ErrorIs(t, err, ErrNotFound)
}
