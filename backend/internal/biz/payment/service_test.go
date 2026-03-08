package payment

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ============================================================================
// Mock Implementations
// ============================================================================

// MockRepository implements Repository for testing.
type MockRepository struct {
	payments  map[uuid.UUID]*models.Payment
	createErr error
	listErr   error
	deleteErr error
	sumErr    error
	getErr    error
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		payments: make(map[uuid.UUID]*models.Payment),
	}
}

func (m *MockRepository) Create(ctx context.Context, payment *models.Payment) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.payments[payment.ID] = payment
	return nil
}

func (m *MockRepository) List(ctx context.Context, tenantID, invoiceID uuid.UUID) ([]*models.Payment, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var result []*models.Payment
	for _, p := range m.payments {
		if p.TenantID == tenantID && p.InvoiceID == invoiceID {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *MockRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.payments, id)
	return nil
}

func (m *MockRepository) SumByInvoiceID(ctx context.Context, tenantID, invoiceID uuid.UUID) (decimal.Decimal, error) {
	if m.sumErr != nil {
		return decimal.Zero, m.sumErr
	}
	total := decimal.Zero
	for _, p := range m.payments {
		if p.TenantID == tenantID && p.InvoiceID == invoiceID {
			total = total.Add(p.Amount)
		}
	}
	return total, nil
}

func (m *MockRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Payment, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	p, ok := m.payments[id]
	if !ok {
		return nil, errors.New("payment not found")
	}
	return p, nil
}

// MockInvoiceReader implements InvoiceReader for testing.
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
	return inv, nil
}

// MockInvoiceStatusUpdater implements InvoiceStatusUpdater for testing.
type MockInvoiceStatusUpdater struct {
	updates   map[uuid.UUID]string // invoiceID -> last status
	updateErr error
}

func NewMockInvoiceStatusUpdater() *MockInvoiceStatusUpdater {
	return &MockInvoiceStatusUpdater{
		updates: make(map[uuid.UUID]string),
	}
}

func (m *MockInvoiceStatusUpdater) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status string) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.updates[id] = status
	return nil
}

// ============================================================================
// Helpers
// ============================================================================

func newTestService() (*Service, *MockRepository, *MockInvoiceReader, *MockInvoiceStatusUpdater) {
	repo := NewMockRepository()
	reader := NewMockInvoiceReader()
	updater := NewMockInvoiceStatusUpdater()
	svc := NewService(repo, reader, updater)
	return svc, repo, reader, updater
}

func newSentInvoice(tenantID uuid.UUID, grossTotal decimal.Decimal) *models.Invoice {
	return &models.Invoice{
		ID:         uuid.New(),
		TenantID:   tenantID,
		Status:     models.InvoiceStatusSent,
		GrossTotal: grossTotal,
		DueDate:    time.Now().Add(30 * 24 * time.Hour),
	}
}

func newRecordInput(tenantID, invoiceID uuid.UUID, amount decimal.Decimal) RecordInput {
	return RecordInput{
		TenantID:  tenantID,
		InvoiceID: invoiceID,
		Amount:    amount,
		Date:      time.Now(),
		Method:    models.PaymentMethodBankTransfer,
		Reference: "REF-001",
		Notes:     "Test payment",
		UserID:    uuid.New(),
	}
}

// ============================================================================
// Record Tests
// ============================================================================

func TestRecord_Success(t *testing.T) {
	svc, _, reader, _ := newTestService()

	tenantID := uuid.New()
	inv := newSentInvoice(tenantID, decimal.NewFromInt(1000))
	reader.invoices[inv.ID] = inv

	input := newRecordInput(tenantID, inv.ID, decimal.NewFromInt(500))
	payment, err := svc.Record(context.Background(), input)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, payment.ID)
	assert.Equal(t, tenantID, payment.TenantID)
	assert.Equal(t, inv.ID, payment.InvoiceID)
	assert.True(t, payment.Amount.Equal(decimal.NewFromInt(500)))
	assert.Equal(t, models.PaymentMethodBankTransfer, payment.Method)
	assert.Equal(t, "REF-001", payment.Reference)
	assert.Equal(t, "Test payment", payment.Notes)
	assert.NotZero(t, payment.CreatedAt)
}

func TestRecord_Success_OverdueInvoice(t *testing.T) {
	svc, _, reader, _ := newTestService()

	tenantID := uuid.New()
	inv := &models.Invoice{
		ID:         uuid.New(),
		TenantID:   tenantID,
		Status:     models.InvoiceStatusOverdue,
		GrossTotal: decimal.NewFromInt(1000),
		DueDate:    time.Now().Add(-7 * 24 * time.Hour),
	}
	reader.invoices[inv.ID] = inv

	input := newRecordInput(tenantID, inv.ID, decimal.NewFromInt(500))
	payment, err := svc.Record(context.Background(), input)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, payment.ID)
	assert.Equal(t, inv.ID, payment.InvoiceID)
}

func TestRecord_AllowsPaymentOnPaidInvoice(t *testing.T) {
	svc, _, reader, _ := newTestService()

	tenantID := uuid.New()
	inv := &models.Invoice{
		ID:         uuid.New(),
		TenantID:   tenantID,
		Status:     models.InvoiceStatusPaid,
		GrossTotal: decimal.NewFromInt(1000),
		DueDate:    time.Now().Add(30 * 24 * time.Hour),
	}
	reader.invoices[inv.ID] = inv

	input := newRecordInput(tenantID, inv.ID, decimal.NewFromInt(100))
	payment, err := svc.Record(context.Background(), input)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, payment.ID)
}

func TestRecord_RejectsDraftInvoice(t *testing.T) {
	svc, _, reader, _ := newTestService()

	tenantID := uuid.New()
	inv := &models.Invoice{
		ID:         uuid.New(),
		TenantID:   tenantID,
		Status:     models.InvoiceStatusDraft,
		GrossTotal: decimal.NewFromInt(1000),
		DueDate:    time.Now().Add(30 * 24 * time.Hour),
	}
	reader.invoices[inv.ID] = inv

	input := newRecordInput(tenantID, inv.ID, decimal.NewFromInt(500))
	_, err := svc.Record(context.Background(), input)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot record payment on invoice with status draft")
}

func TestRecord_RejectsCancelledInvoice(t *testing.T) {
	svc, _, reader, _ := newTestService()

	tenantID := uuid.New()
	inv := &models.Invoice{
		ID:         uuid.New(),
		TenantID:   tenantID,
		Status:     models.InvoiceStatusCancelled,
		GrossTotal: decimal.NewFromInt(1000),
		DueDate:    time.Now().Add(30 * 24 * time.Hour),
	}
	reader.invoices[inv.ID] = inv

	input := newRecordInput(tenantID, inv.ID, decimal.NewFromInt(500))
	_, err := svc.Record(context.Background(), input)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot record payment on invoice with status cancelled")
}

func TestRecord_RejectsZeroAmount(t *testing.T) {
	svc, _, reader, _ := newTestService()

	tenantID := uuid.New()
	inv := newSentInvoice(tenantID, decimal.NewFromInt(1000))
	reader.invoices[inv.ID] = inv

	input := newRecordInput(tenantID, inv.ID, decimal.Zero)
	_, err := svc.Record(context.Background(), input)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "payment amount must be positive")
}

func TestRecord_RejectsNegativeAmount(t *testing.T) {
	svc, _, reader, _ := newTestService()

	tenantID := uuid.New()
	inv := newSentInvoice(tenantID, decimal.NewFromInt(1000))
	reader.invoices[inv.ID] = inv

	input := newRecordInput(tenantID, inv.ID, decimal.NewFromInt(-100))
	_, err := svc.Record(context.Background(), input)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "payment amount must be positive")
}

func TestRecord_InvoiceNotFound(t *testing.T) {
	svc, _, _, _ := newTestService()

	tenantID := uuid.New()
	input := newRecordInput(tenantID, uuid.New(), decimal.NewFromInt(500))
	_, err := svc.Record(context.Background(), input)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get invoice")
}

func TestRecord_AutoTransitionToPaid(t *testing.T) {
	svc, _, reader, updater := newTestService()

	tenantID := uuid.New()
	inv := newSentInvoice(tenantID, decimal.NewFromInt(1000))
	reader.invoices[inv.ID] = inv

	// Pay the full amount
	input := newRecordInput(tenantID, inv.ID, decimal.NewFromInt(1000))
	payment, err := svc.Record(context.Background(), input)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, payment.ID)
	assert.Equal(t, models.InvoiceStatusPaid, updater.updates[inv.ID])
}

func TestRecord_NoTransitionWhenPartialPayment(t *testing.T) {
	svc, _, reader, updater := newTestService()

	tenantID := uuid.New()
	inv := newSentInvoice(tenantID, decimal.NewFromInt(1000))
	reader.invoices[inv.ID] = inv

	// Pay only half
	input := newRecordInput(tenantID, inv.ID, decimal.NewFromInt(500))
	payment, err := svc.Record(context.Background(), input)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, payment.ID)
	// No status update should have been recorded
	_, statusUpdated := updater.updates[inv.ID]
	assert.False(t, statusUpdated)
}

func TestRecord_RepoCreateError(t *testing.T) {
	svc, repo, reader, _ := newTestService()

	tenantID := uuid.New()
	inv := newSentInvoice(tenantID, decimal.NewFromInt(1000))
	reader.invoices[inv.ID] = inv

	repo.createErr = errors.New("database connection lost")

	input := newRecordInput(tenantID, inv.ID, decimal.NewFromInt(500))
	_, err := svc.Record(context.Background(), input)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database connection lost")
}

func TestRecord_DefaultsDateWhenZero(t *testing.T) {
	svc, _, reader, _ := newTestService()

	tenantID := uuid.New()
	inv := newSentInvoice(tenantID, decimal.NewFromInt(1000))
	reader.invoices[inv.ID] = inv

	input := newRecordInput(tenantID, inv.ID, decimal.NewFromInt(500))
	input.Date = time.Time{} // Zero time

	payment, err := svc.Record(context.Background(), input)

	require.NoError(t, err)
	assert.False(t, payment.PaymentDate.IsZero())
}

// ============================================================================
// List Tests
// ============================================================================

func TestList_Success(t *testing.T) {
	svc, repo, _, _ := newTestService()

	tenantID := uuid.New()
	invoiceID := uuid.New()

	for i := 0; i < 3; i++ {
		id := uuid.New()
		repo.payments[id] = &models.Payment{
			ID:        id,
			TenantID:  tenantID,
			InvoiceID: invoiceID,
			Amount:    decimal.NewFromInt(100),
			CreatedAt: time.Now(),
		}
	}

	payments, err := svc.List(context.Background(), tenantID, invoiceID)

	require.NoError(t, err)
	assert.Len(t, payments, 3)
}

func TestList_Empty(t *testing.T) {
	svc, _, _, _ := newTestService()

	payments, err := svc.List(context.Background(), uuid.New(), uuid.New())

	require.NoError(t, err)
	assert.Empty(t, payments)
}

func TestList_RepoError(t *testing.T) {
	svc, repo, _, _ := newTestService()

	repo.listErr = errors.New("database error")

	_, err := svc.List(context.Background(), uuid.New(), uuid.New())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
}

// ============================================================================
// Delete Tests
// ============================================================================

func TestDelete_Success(t *testing.T) {
	svc, repo, reader, _ := newTestService()

	tenantID := uuid.New()
	invoiceID := uuid.New()
	paymentID := uuid.New()

	reader.invoices[invoiceID] = &models.Invoice{
		ID:         invoiceID,
		TenantID:   tenantID,
		Status:     models.InvoiceStatusSent,
		GrossTotal: decimal.NewFromInt(1000),
		DueDate:    time.Now().Add(30 * 24 * time.Hour),
	}

	repo.payments[paymentID] = &models.Payment{
		ID:        paymentID,
		TenantID:  tenantID,
		InvoiceID: invoiceID,
		Amount:    decimal.NewFromInt(500),
		CreatedAt: time.Now(),
	}

	err := svc.Delete(context.Background(), tenantID, paymentID)

	require.NoError(t, err)
	_, exists := repo.payments[paymentID]
	assert.False(t, exists)
}

func TestDelete_PaymentNotFound(t *testing.T) {
	svc, _, _, _ := newTestService()

	err := svc.Delete(context.Background(), uuid.New(), uuid.New())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "payment not found")
}

func TestDelete_InvoiceCancelled(t *testing.T) {
	svc, repo, reader, _ := newTestService()

	tenantID := uuid.New()
	invoiceID := uuid.New()
	paymentID := uuid.New()

	reader.invoices[invoiceID] = &models.Invoice{
		ID:         invoiceID,
		TenantID:   tenantID,
		Status:     models.InvoiceStatusCancelled,
		GrossTotal: decimal.NewFromInt(1000),
		DueDate:    time.Now().Add(30 * 24 * time.Hour),
	}

	repo.payments[paymentID] = &models.Payment{
		ID:        paymentID,
		TenantID:  tenantID,
		InvoiceID: invoiceID,
		Amount:    decimal.NewFromInt(500),
		CreatedAt: time.Now(),
	}

	err := svc.Delete(context.Background(), tenantID, paymentID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot delete payment on cancelled invoice")
	// Payment should NOT have been deleted
	_, exists := repo.payments[paymentID]
	assert.True(t, exists)
}

func TestDelete_RevertsFromPaid(t *testing.T) {
	svc, repo, reader, updater := newTestService()

	tenantID := uuid.New()
	invoiceID := uuid.New()
	paymentID := uuid.New()

	reader.invoices[invoiceID] = &models.Invoice{
		ID:         invoiceID,
		TenantID:   tenantID,
		Status:     models.InvoiceStatusPaid,
		GrossTotal: decimal.NewFromInt(1000),
		DueDate:    time.Now().Add(30 * 24 * time.Hour), // Future due date -> reverts to sent
	}

	repo.payments[paymentID] = &models.Payment{
		ID:        paymentID,
		TenantID:  tenantID,
		InvoiceID: invoiceID,
		Amount:    decimal.NewFromInt(1000),
		CreatedAt: time.Now(),
	}

	err := svc.Delete(context.Background(), tenantID, paymentID)

	require.NoError(t, err)
	// After deleting the only payment, sum = 0 < 1000, so status reverts to sent
	assert.Equal(t, models.InvoiceStatusSent, updater.updates[invoiceID])
}

func TestDelete_RevertsToOverdueWhenPastDue(t *testing.T) {
	svc, repo, reader, updater := newTestService()

	tenantID := uuid.New()
	invoiceID := uuid.New()
	paymentID := uuid.New()

	reader.invoices[invoiceID] = &models.Invoice{
		ID:         invoiceID,
		TenantID:   tenantID,
		Status:     models.InvoiceStatusPaid,
		GrossTotal: decimal.NewFromInt(1000),
		DueDate:    time.Now().Add(-7 * 24 * time.Hour), // Past due date -> reverts to overdue
	}

	repo.payments[paymentID] = &models.Payment{
		ID:        paymentID,
		TenantID:  tenantID,
		InvoiceID: invoiceID,
		Amount:    decimal.NewFromInt(1000),
		CreatedAt: time.Now(),
	}

	err := svc.Delete(context.Background(), tenantID, paymentID)

	require.NoError(t, err)
	assert.Equal(t, models.InvoiceStatusOverdue, updater.updates[invoiceID])
}

func TestDelete_NoRevertWhenStillFullyPaid(t *testing.T) {
	svc, repo, reader, updater := newTestService()

	tenantID := uuid.New()
	invoiceID := uuid.New()
	paymentID1 := uuid.New()
	paymentID2 := uuid.New()

	reader.invoices[invoiceID] = &models.Invoice{
		ID:         invoiceID,
		TenantID:   tenantID,
		Status:     models.InvoiceStatusPaid,
		GrossTotal: decimal.NewFromInt(1000),
		DueDate:    time.Now().Add(30 * 24 * time.Hour),
	}

	// Two payments totaling 1500, deleting one still leaves 1000 >= gross_total
	repo.payments[paymentID1] = &models.Payment{
		ID:        paymentID1,
		TenantID:  tenantID,
		InvoiceID: invoiceID,
		Amount:    decimal.NewFromInt(500),
		CreatedAt: time.Now(),
	}
	repo.payments[paymentID2] = &models.Payment{
		ID:        paymentID2,
		TenantID:  tenantID,
		InvoiceID: invoiceID,
		Amount:    decimal.NewFromInt(1000),
		CreatedAt: time.Now(),
	}

	// Delete the 500 payment; remaining 1000 still covers gross_total
	err := svc.Delete(context.Background(), tenantID, paymentID1)

	require.NoError(t, err)
	// No status update should have been recorded since 1000 >= 1000
	_, statusUpdated := updater.updates[invoiceID]
	assert.False(t, statusUpdated)
}

func TestDelete_RepoDeleteError(t *testing.T) {
	svc, repo, reader, _ := newTestService()

	tenantID := uuid.New()
	invoiceID := uuid.New()
	paymentID := uuid.New()

	reader.invoices[invoiceID] = &models.Invoice{
		ID:         invoiceID,
		TenantID:   tenantID,
		Status:     models.InvoiceStatusSent,
		GrossTotal: decimal.NewFromInt(1000),
		DueDate:    time.Now().Add(30 * 24 * time.Hour),
	}

	repo.payments[paymentID] = &models.Payment{
		ID:        paymentID,
		TenantID:  tenantID,
		InvoiceID: invoiceID,
		Amount:    decimal.NewFromInt(500),
		CreatedAt: time.Now(),
	}

	repo.deleteErr = errors.New("database write failed")

	err := svc.Delete(context.Background(), tenantID, paymentID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database write failed")
}
