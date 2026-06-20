package creditnote

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// noopTxBeginner returns no-op transactions so Send's orchestration can run with
// mock repos in unit tests; the mock repo/sequence methods ignore the tx and the
// real DB rollback is covered by the integration test.
type noopTxBeginner struct{}

func (noopTxBeginner) Begin(context.Context) (pgx.Tx, error) { return noopTx{}, nil }

// noopTx satisfies pgx.Tx with no-op Commit/Rollback; all other methods are never
// called by the mock-backed Send path (embedded nil pgx.Tx).
type noopTx struct{ pgx.Tx }

func (noopTx) Commit(context.Context) error   { return nil }
func (noopTx) Rollback(context.Context) error { return nil }

// ============================================================================
// Mock Repository
// ============================================================================

type MockRepository struct {
	creditNotes map[uuid.UUID]*models.CreditNote
	createErr   error
	getErr      error
	updateErr   error
	listErr     error
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		creditNotes: make(map[uuid.UUID]*models.CreditNote),
	}
}

func (m *MockRepository) Create(ctx context.Context, cn *models.CreditNote) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.creditNotes[cn.ID] = cn
	return nil
}

func (m *MockRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.CreditNote, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	cn, ok := m.creditNotes[id]
	if !ok {
		return nil, ErrCreditNoteNotFound
	}
	if cn.TenantID != tenantID {
		return nil, ErrCreditNoteNotFound
	}
	return cn, nil
}

func (m *MockRepository) Update(ctx context.Context, cn *models.CreditNote) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.creditNotes[cn.ID] = cn
	return nil
}

func (m *MockRepository) UpdateInTx(_ context.Context, _ pgx.Tx, cn *models.CreditNote) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.creditNotes[cn.ID] = cn
	return nil
}

func (m *MockRepository) List(ctx context.Context, tenantID uuid.UUID, filter ListFilter) ([]*models.CreditNote, int, error) {
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	var result []*models.CreditNote
	for _, cn := range m.creditNotes {
		if cn.TenantID != tenantID {
			continue
		}
		if filter.Status != "" && cn.Status != filter.Status {
			continue
		}
		if filter.OriginalInvID != nil && cn.OriginalInvoiceID != *filter.OriginalInvID {
			continue
		}
		result = append(result, cn)
	}
	return result, len(result), nil
}

func (m *MockRepository) GetByInvoiceID(ctx context.Context, tenantID, invoiceID uuid.UUID) ([]*models.CreditNote, error) {
	var result []*models.CreditNote
	for _, cn := range m.creditNotes {
		if cn.TenantID == tenantID && cn.OriginalInvoiceID == invoiceID {
			result = append(result, cn)
		}
	}
	return result, nil
}

// ListForDATEVExport returns sent credit notes in the date range (by created_at).
// The mock ignores keyset paging (afterDate/afterID/limit) and returns all matches.
func (m *MockRepository) ListForDATEVExport(_ context.Context, tenantID uuid.UUID, fromDate, toDate time.Time, _ *time.Time, _ *uuid.UUID, _ int) ([]*models.CreditNote, error) {
	var result []*models.CreditNote
	for _, cn := range m.creditNotes {
		if cn.TenantID == tenantID && cn.Status == models.CreditNoteStatusSent &&
			!cn.CreatedAt.Before(fromDate) && cn.CreatedAt.Before(toDate.AddDate(0, 0, 1)) {
			result = append(result, cn)
		}
	}
	return result, nil
}

// ============================================================================
// Mock Invoice Reader
// ============================================================================

type MockInvoiceReader struct {
	invoices map[uuid.UUID]*models.Invoice
	getErr   error
}

func NewMockInvoiceReader() *MockInvoiceReader {
	return &MockInvoiceReader{
		invoices: make(map[uuid.UUID]*models.Invoice),
	}
}

func (m *MockInvoiceReader) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Invoice, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	inv, ok := m.invoices[id]
	if !ok {
		return nil, errors.New("invoice not found")
	}
	if inv.TenantID != tenantID {
		return nil, errors.New("invoice not found")
	}
	return inv, nil
}

func (m *MockInvoiceReader) AddInvoice(inv *models.Invoice) {
	m.invoices[inv.ID] = inv
}

// ============================================================================
// Mock Number Sequence Repo
// ============================================================================

type MockNumberSequenceRepo struct {
	nextNumber string
	nextErr    error
}

func NewMockNumberSequenceRepo() *MockNumberSequenceRepo {
	return &MockNumberSequenceRepo{
		nextNumber: "GS-2026-0001",
	}
}

func (m *MockNumberSequenceRepo) NextNumber(ctx context.Context, tenantID uuid.UUID, documentType string, fiscalYear int, prefix string) (string, error) {
	if m.nextErr != nil {
		return "", m.nextErr
	}
	return m.nextNumber, nil
}

func (m *MockNumberSequenceRepo) NextNumberInTx(ctx context.Context, _ pgx.Tx, tenantID uuid.UUID, documentType string, fiscalYear int, prefix string) (string, error) {
	return m.NextNumber(ctx, tenantID, documentType, fiscalYear, prefix)
}

// ============================================================================
// Helpers
// ============================================================================

func newTestInvoice(tenantID uuid.UUID, status string) *models.Invoice {
	return &models.Invoice{
		ID:              uuid.New(),
		TenantID:        tenantID,
		Status:          status,
		CustomerName:    "Test GmbH",
		CustomerAddress: "Teststrasse 1, 80331 Muenchen",
		CustomerEmail:   "test@example.com",
		CustomerUStIDNr: "DE123456789",
	}
}

func newTestLineItems() []models.LineItem {
	return []models.LineItem{{
		ID:          "1",
		Position:    1,
		Description: "Credit Item",
		Quantity:    decimal.NewFromInt(1),
		UnitPrice:   decimal.NewFromInt(100),
		TaxRate:     decimal.NewFromInt(19),
	}}
}

func newTestService() (*Service, *MockRepository, *MockInvoiceReader, *MockNumberSequenceRepo) {
	repo := NewMockRepository()
	invReader := NewMockInvoiceReader()
	numSeq := NewMockNumberSequenceRepo()
	svc := NewService(repo, invReader, numSeq, noopTxBeginner{})
	return svc, repo, invReader, numSeq
}

// ============================================================================
// Create Tests
// ============================================================================

func TestService_Create_Success_SentInvoice(t *testing.T) {
	svc, _, invReader, _ := newTestService()

	tenantID := uuid.New()
	inv := newTestInvoice(tenantID, models.InvoiceStatusSent)
	invReader.AddInvoice(inv)

	input := CreateInput{
		TenantID:          tenantID,
		OriginalInvoiceID: inv.ID,
		LineItems:         newTestLineItems(),
		TaxMode:           models.TaxModeStandard,
		Reason:            "Partial refund",
		UserID:            uuid.New(),
	}

	cn, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, cn.ID)
	assert.Equal(t, tenantID, cn.TenantID)
	assert.Equal(t, models.CreditNoteStatusDraft, cn.Status)
	assert.Equal(t, inv.ID, cn.OriginalInvoiceID)
	assert.Equal(t, "", cn.CreditNoteNumber)
	assert.Equal(t, "Test GmbH", cn.CustomerName)
	assert.Equal(t, "Teststrasse 1, 80331 Muenchen", cn.CustomerAddress)
	assert.Equal(t, "test@example.com", cn.CustomerEmail)
	assert.Equal(t, "DE123456789", cn.CustomerUStIDNr)
	assert.Equal(t, "Partial refund", cn.Reason)
	assert.NotZero(t, cn.CreatedAt)
	assert.NotZero(t, cn.UpdatedAt)
}

func TestService_Create_Success_PaidInvoice(t *testing.T) {
	svc, _, invReader, _ := newTestService()

	tenantID := uuid.New()
	inv := newTestInvoice(tenantID, models.InvoiceStatusPaid)
	invReader.AddInvoice(inv)

	input := CreateInput{
		TenantID:          tenantID,
		OriginalInvoiceID: inv.ID,
		LineItems:         newTestLineItems(),
		TaxMode:           models.TaxModeStandard,
		Reason:            "Full refund",
		UserID:            uuid.New(),
	}

	cn, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, models.CreditNoteStatusDraft, cn.Status)
	assert.Equal(t, inv.ID, cn.OriginalInvoiceID)
}

func TestService_Create_Success_OverdueInvoice(t *testing.T) {
	svc, _, invReader, _ := newTestService()

	tenantID := uuid.New()
	inv := newTestInvoice(tenantID, models.InvoiceStatusOverdue)
	invReader.AddInvoice(inv)

	input := CreateInput{
		TenantID:          tenantID,
		OriginalInvoiceID: inv.ID,
		LineItems:         newTestLineItems(),
		TaxMode:           models.TaxModeStandard,
		Reason:            "Disputed amount",
		UserID:            uuid.New(),
	}

	cn, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, models.CreditNoteStatusDraft, cn.Status)
}

func TestService_Create_RejectsDraftInvoice(t *testing.T) {
	svc, _, invReader, _ := newTestService()

	tenantID := uuid.New()
	inv := newTestInvoice(tenantID, models.InvoiceStatusDraft)
	invReader.AddInvoice(inv)

	input := CreateInput{
		TenantID:          tenantID,
		OriginalInvoiceID: inv.ID,
		LineItems:         newTestLineItems(),
		TaxMode:           models.TaxModeStandard,
		Reason:            "Should fail",
		UserID:            uuid.New(),
	}

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrInvalidInvoiceForCredit)
}

func TestService_Create_RejectsCancelledInvoice(t *testing.T) {
	svc, _, invReader, _ := newTestService()

	tenantID := uuid.New()
	inv := newTestInvoice(tenantID, models.InvoiceStatusCancelled)
	invReader.AddInvoice(inv)

	input := CreateInput{
		TenantID:          tenantID,
		OriginalInvoiceID: inv.ID,
		LineItems:         newTestLineItems(),
		TaxMode:           models.TaxModeStandard,
		Reason:            "Should fail",
		UserID:            uuid.New(),
	}

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrInvalidInvoiceForCredit)
}

func TestService_Create_NoLineItems(t *testing.T) {
	svc, _, _, _ := newTestService()

	input := CreateInput{
		TenantID:          uuid.New(),
		OriginalInvoiceID: uuid.New(),
		LineItems:         []models.LineItem{},
		TaxMode:           models.TaxModeStandard,
		Reason:            "No items",
		UserID:            uuid.New(),
	}

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrNoLineItems)
}

func TestService_Create_InvoiceNotFound(t *testing.T) {
	svc, _, _, _ := newTestService()

	input := CreateInput{
		TenantID:          uuid.New(),
		OriginalInvoiceID: uuid.New(),
		LineItems:         newTestLineItems(),
		TaxMode:           models.TaxModeStandard,
		Reason:            "Invoice missing",
		UserID:            uuid.New(),
	}

	_, err := svc.Create(context.Background(), input)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invoice")
}

func TestService_Create_TaxCalculation(t *testing.T) {
	svc, _, invReader, _ := newTestService()

	tenantID := uuid.New()
	inv := newTestInvoice(tenantID, models.InvoiceStatusSent)
	invReader.AddInvoice(inv)

	items := []models.LineItem{
		{
			ID:          "1",
			Position:    1,
			Description: "Item A",
			Quantity:    decimal.NewFromInt(2),
			UnitPrice:   decimal.NewFromInt(100),
			TaxRate:     decimal.NewFromInt(19),
		},
		{
			ID:          "2",
			Position:    2,
			Description: "Item B",
			Quantity:    decimal.NewFromInt(1),
			UnitPrice:   decimal.NewFromInt(50),
			TaxRate:     decimal.NewFromInt(7),
		},
	}

	input := CreateInput{
		TenantID:          tenantID,
		OriginalInvoiceID: inv.ID,
		LineItems:         items,
		TaxMode:           models.TaxModeStandard,
		Reason:            "Tax test",
		UserID:            uuid.New(),
	}

	cn, err := svc.Create(context.Background(), input)

	require.NoError(t, err)

	// Subtotal: 2*100 + 1*50 = 250
	expectedSubtotal := decimal.NewFromInt(250)
	assert.True(t, expectedSubtotal.Equal(cn.Subtotal),
		"expected subtotal %s, got %s", expectedSubtotal, cn.Subtotal)

	// Tax: 200*0.19 + 50*0.07 = 38 + 3.50 = 41.50
	expectedTax := decimal.NewFromFloat(41.50)
	assert.True(t, expectedTax.Equal(cn.TotalTax),
		"expected total tax %s, got %s", expectedTax, cn.TotalTax)

	// Gross: 250 + 41.50 = 291.50
	expectedGross := decimal.NewFromFloat(291.50)
	assert.True(t, expectedGross.Equal(cn.GrossTotal),
		"expected gross total %s, got %s", expectedGross, cn.GrossTotal)
}

func TestService_Create_RepoError(t *testing.T) {
	svc, repo, invReader, _ := newTestService()

	tenantID := uuid.New()
	inv := newTestInvoice(tenantID, models.InvoiceStatusSent)
	invReader.AddInvoice(inv)

	repo.createErr = errors.New("database connection lost")

	input := CreateInput{
		TenantID:          tenantID,
		OriginalInvoiceID: inv.ID,
		LineItems:         newTestLineItems(),
		TaxMode:           models.TaxModeStandard,
		Reason:            "Repo error test",
		UserID:            uuid.New(),
	}

	_, err := svc.Create(context.Background(), input)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "database connection lost")
}

// ============================================================================
// GetByID Tests
// ============================================================================

func TestService_GetByID_Success(t *testing.T) {
	svc, repo, _, _ := newTestService()

	tenantID := uuid.New()
	cnID := uuid.New()
	repo.creditNotes[cnID] = &models.CreditNote{
		ID:       cnID,
		TenantID: tenantID,
		Status:   models.CreditNoteStatusDraft,
		Reason:   "Test credit note",
	}

	cn, err := svc.GetByID(context.Background(), tenantID, cnID)

	require.NoError(t, err)
	assert.Equal(t, cnID, cn.ID)
	assert.Equal(t, "Test credit note", cn.Reason)
}

func TestService_GetByID_NotFound(t *testing.T) {
	svc, _, _, _ := newTestService()

	_, err := svc.GetByID(context.Background(), uuid.New(), uuid.New())

	assert.ErrorIs(t, err, ErrCreditNoteNotFound)
}

// ============================================================================
// List Tests
// ============================================================================

func TestService_List_Success(t *testing.T) {
	svc, repo, _, _ := newTestService()

	tenantID := uuid.New()
	for i := 0; i < 3; i++ {
		id := uuid.New()
		repo.creditNotes[id] = &models.CreditNote{
			ID:       id,
			TenantID: tenantID,
			Status:   models.CreditNoteStatusDraft,
		}
	}

	// Add one from a different tenant to verify filtering
	otherID := uuid.New()
	repo.creditNotes[otherID] = &models.CreditNote{
		ID:       otherID,
		TenantID: uuid.New(),
		Status:   models.CreditNoteStatusDraft,
	}

	result, total, err := svc.List(context.Background(), tenantID, ListFilter{})

	require.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, 3, total)
}

// ============================================================================
// Send Tests
// ============================================================================

func TestService_Send_Success(t *testing.T) {
	svc, repo, _, numSeq := newTestService()

	tenantID := uuid.New()
	cnID := uuid.New()
	repo.creditNotes[cnID] = &models.CreditNote{
		ID:       cnID,
		TenantID: tenantID,
		Status:   models.CreditNoteStatusDraft,
	}

	numSeq.nextNumber = "GS-2026-0001"

	err := svc.Send(context.Background(), tenantID, cnID, uuid.New())

	require.NoError(t, err)

	cn := repo.creditNotes[cnID]
	assert.Equal(t, models.CreditNoteStatusSent, cn.Status)
	assert.Equal(t, "GS-2026-0001", cn.CreditNoteNumber)
	assert.NotZero(t, cn.UpdatedAt)
}

func TestService_Send_RejectsNonDraft(t *testing.T) {
	svc, repo, _, _ := newTestService()

	tenantID := uuid.New()
	cnID := uuid.New()
	repo.creditNotes[cnID] = &models.CreditNote{
		ID:       cnID,
		TenantID: tenantID,
		Status:   models.CreditNoteStatusSent,
	}

	err := svc.Send(context.Background(), tenantID, cnID, uuid.New())

	assert.ErrorIs(t, err, ErrCreditNoteNotDraft)
}

func TestService_Send_NumberSeqError(t *testing.T) {
	svc, repo, _, numSeq := newTestService()

	tenantID := uuid.New()
	cnID := uuid.New()
	repo.creditNotes[cnID] = &models.CreditNote{
		ID:       cnID,
		TenantID: tenantID,
		Status:   models.CreditNoteStatusDraft,
	}

	numSeq.nextErr = errors.New("sequence exhausted")

	err := svc.Send(context.Background(), tenantID, cnID, uuid.New())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "sequence exhausted")

	// Status should remain draft since send failed
	assert.Equal(t, models.CreditNoteStatusDraft, repo.creditNotes[cnID].Status)
}
