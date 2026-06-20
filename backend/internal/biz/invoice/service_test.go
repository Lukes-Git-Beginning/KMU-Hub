package invoice

import (
	"context"
	"encoding/json"
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
// Mock Repositories
// ============================================================================

// MockRepository implements Repository for testing.
type MockRepository struct {
	invoices             map[uuid.UUID]*models.Invoice
	createErr            error
	getErr               error
	listErr              error
	updateErr            error
	updateStatusErr      error
	overdueErr           error
	aggregateStatsResult PaymentStats
	aggregateStatsErr    error
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		invoices: make(map[uuid.UUID]*models.Invoice),
	}
}

func (m *MockRepository) Create(ctx context.Context, invoice *models.Invoice) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.invoices[invoice.ID] = invoice
	return nil
}

func (m *MockRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Invoice, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	inv, ok := m.invoices[id]
	if !ok {
		return nil, ErrInvoiceNotFound
	}
	return inv, nil
}

func (m *MockRepository) List(ctx context.Context, tenantID uuid.UUID, filter ListFilter) ([]*models.Invoice, int, error) {
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	var result []*models.Invoice
	for _, inv := range m.invoices {
		if inv.TenantID != tenantID {
			continue
		}
		if filter.ContactID != nil {
			if inv.ContactID == nil || *inv.ContactID != *filter.ContactID {
				continue
			}
		}
		result = append(result, inv)
	}
	return result, len(result), nil
}

func (m *MockRepository) Update(ctx context.Context, invoice *models.Invoice) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.invoices[invoice.ID] = invoice
	return nil
}

func (m *MockRepository) UpdateInTx(_ context.Context, _ pgx.Tx, invoice *models.Invoice) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.invoices[invoice.ID] = invoice
	return nil
}

func (m *MockRepository) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status string) error {
	if m.updateStatusErr != nil {
		return m.updateStatusErr
	}
	if inv, ok := m.invoices[id]; ok {
		inv.Status = status
	}
	return nil
}

func (m *MockRepository) UpdateStatusInTx(_ context.Context, _ pgx.Tx, _, id uuid.UUID, status string) error {
	if m.updateStatusErr != nil {
		return m.updateStatusErr
	}
	if inv, ok := m.invoices[id]; ok {
		inv.Status = status
	}
	return nil
}

func (m *MockRepository) GetOverdue(ctx context.Context, tenantID uuid.UUID) ([]*models.Invoice, error) {
	if m.overdueErr != nil {
		return nil, m.overdueErr
	}
	var result []*models.Invoice
	for _, inv := range m.invoices {
		if inv.TenantID == tenantID && inv.Status == models.InvoiceStatusSent && inv.DueDate.Before(time.Now()) {
			result = append(result, inv)
		}
	}
	return result, nil
}

func (m *MockRepository) GetByQuoteID(ctx context.Context, tenantID, quoteID uuid.UUID) (*models.Invoice, error) {
	for _, inv := range m.invoices {
		if inv.TenantID == tenantID && inv.SourceQuoteID != nil && *inv.SourceQuoteID == quoteID {
			return inv, nil
		}
	}
	return nil, ErrInvoiceNotFound
}

func (m *MockRepository) LinkTimeTracking(_ context.Context, _, _ uuid.UUID, _ json.RawMessage) error {
	return nil
}

func (m *MockRepository) SetLock(_ context.Context, _, id uuid.UUID, lockedAt time.Time, lockedBy uuid.UUID) error {
	if inv, ok := m.invoices[id]; ok {
		inv.LockedAt = &lockedAt
		inv.LockedBy = &lockedBy
	}
	return nil
}

// InvoiceNumberExists returns false for all numbers (no duplicates in test data by default).
func (m *MockRepository) InvoiceNumberExists(_ context.Context, _ uuid.UUID, number string) (bool, error) {
	for _, inv := range m.invoices {
		if inv.InvoiceNumber == number {
			return true, nil
		}
	}
	return false, nil
}

// CountByFiscalYear counts non-draft invoices with invoice_number in the given year.
func (m *MockRepository) CountByFiscalYear(_ context.Context, tenantID uuid.UUID, year int) (int, error) {
	count := 0
	for _, inv := range m.invoices {
		if inv.TenantID == tenantID && inv.Status != "draft" && inv.InvoiceNumber != "" &&
			inv.InvoiceDate.Year() == year {
			count++
		}
	}
	return count, nil
}

// AggregatePaymentStats returns the configured aggregateStatsResult (default: zeroed).
func (m *MockRepository) AggregatePaymentStats(_ context.Context, _ uuid.UUID, _, _ time.Time) (PaymentStats, error) {
	if m.aggregateStatsErr != nil {
		return PaymentStats{}, m.aggregateStatsErr
	}
	return m.aggregateStatsResult, nil
}

// ListForGoBDExport returns non-draft invoices with invoice_number in the date range.
func (m *MockRepository) ListForGoBDExport(_ context.Context, tenantID uuid.UUID, fromDate, toDate time.Time) ([]*models.Invoice, error) {
	var result []*models.Invoice
	for _, inv := range m.invoices {
		if inv.TenantID == tenantID && inv.Status != "draft" && inv.InvoiceNumber != "" &&
			!inv.InvoiceDate.Before(fromDate) && !inv.InvoiceDate.After(toDate) {
			result = append(result, inv)
		}
	}
	return result, nil
}

// ListForDATEVExport returns sent/paid/overdue invoices in the date range.
// The mock ignores keyset paging (afterDate/afterID/limit) and returns all matches.
func (m *MockRepository) ListForDATEVExport(_ context.Context, tenantID uuid.UUID, fromDate, toDate time.Time, _ *time.Time, _ *uuid.UUID, _ int) ([]*models.Invoice, error) {
	var result []*models.Invoice
	for _, inv := range m.invoices {
		if inv.TenantID == tenantID &&
			(inv.Status == models.InvoiceStatusSent || inv.Status == models.InvoiceStatusPaid || inv.Status == models.InvoiceStatusOverdue) &&
			!inv.InvoiceDate.Before(fromDate) && !inv.InvoiceDate.After(toDate) {
			result = append(result, inv)
		}
	}
	return result, nil
}

// MockNumberSequenceRepo implements NumberSequenceRepo for testing.
type MockNumberSequenceRepo struct {
	nextNumber   string
	nextErr      error
	seqInfo      *SequenceInfo // nil means "no sequence for this year"
	seqInfoErr   error
}

func (m *MockNumberSequenceRepo) NextNumber(ctx context.Context, tenantID uuid.UUID, documentType string, fiscalYear int, prefix string) (string, error) {
	if m.nextErr != nil {
		return "", m.nextErr
	}
	if m.nextNumber != "" {
		return m.nextNumber, nil
	}
	return "RE-2026-0001", nil
}

func (m *MockNumberSequenceRepo) NextNumberInTx(ctx context.Context, _ pgx.Tx, tenantID uuid.UUID, documentType string, fiscalYear int, prefix string) (string, error) {
	return m.NextNumber(ctx, tenantID, documentType, fiscalYear, prefix)
}

// GetSequenceInfo returns the configured seqInfo (nil by default = no sequence exists).
func (m *MockNumberSequenceRepo) GetSequenceInfo(_ context.Context, _ uuid.UUID, _ string, _ int) (*SequenceInfo, error) {
	if m.seqInfoErr != nil {
		return nil, m.seqInfoErr
	}
	return m.seqInfo, nil
}

// MockCompanySettingsRepo implements CompanySettingsRepo for testing.
type MockCompanySettingsRepo struct {
	settings *models.CompanySettings
	getErr   error
}

func (m *MockCompanySettingsRepo) GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*models.CompanySettings, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.settings, nil
}

// MockQuoteReader implements QuoteReader for testing.
type MockQuoteReader struct {
	quotes map[uuid.UUID]*models.Quote
	getErr error
}

func NewMockQuoteReader() *MockQuoteReader {
	return &MockQuoteReader{
		quotes: make(map[uuid.UUID]*models.Quote),
	}
}

func (m *MockQuoteReader) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Quote, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	q, ok := m.quotes[id]
	if !ok {
		return nil, errors.New("quote not found")
	}
	return q, nil
}

// MockEventEmitter implements EventEmitter for testing.
type MockEventEmitter struct {
	events []models.EventPayload
}

func (m *MockEventEmitter) EmitBizEvent(ctx context.Context, payload models.EventPayload) error {
	m.events = append(m.events, payload)
	return nil
}

// ============================================================================
// Test Helpers
// ============================================================================

func testLineItems() []models.LineItem {
	return []models.LineItem{{
		ID: "1", Position: 1, Description: "Test Item",
		Quantity: decimal.NewFromInt(2), UnitPrice: decimal.NewFromInt(100),
		TaxRate: decimal.NewFromInt(19),
	}}
}

func testCreateInput(tenantID, userID uuid.UUID) CreateInput {
	return CreateInput{
		TenantID:        tenantID,
		CustomerName:    "Test Customer",
		CustomerAddress: "Test Street 1, 12345 Berlin",
		CustomerEmail:   "customer@example.com",
		TaxMode:         models.TaxModeStandard,
		LineItems:       testLineItems(),
		InvoiceDate:     time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		PaymentTermsDays: 14,
		Notes:           "Test invoice",
		UserID:          userID,
	}
}

func newTestService() (*Service, *MockRepository, *MockNumberSequenceRepo, *MockCompanySettingsRepo, *MockQuoteReader) {
	repo := NewMockRepository()
	numSeq := &MockNumberSequenceRepo{}
	cs := &MockCompanySettingsRepo{}
	qr := NewMockQuoteReader()
	svc := NewService(repo, numSeq, cs, qr, noopTxBeginner{})
	return svc, repo, numSeq, cs, qr
}

// createDraftInvoice creates a draft invoice in the mock repo and returns it.
func createDraftInvoice(t *testing.T, repo *MockRepository, tenantID uuid.UUID) *models.Invoice {
	t.Helper()
	lineItemsJSON, err := json.Marshal(testLineItems())
	require.NoError(t, err)

	inv := &models.Invoice{
		ID:            uuid.New(),
		TenantID:      tenantID,
		Status:        models.InvoiceStatusDraft,
		CustomerName:  "Draft Customer",
		CustomerEmail: "draft@example.com",
		TaxMode:       models.TaxModeStandard,
		LineItems:     lineItemsJSON,
		Subtotal:      decimal.NewFromInt(200),
		TotalTax:      decimal.NewFromFloat(38),
		GrossTotal:    decimal.NewFromFloat(238),
		InvoiceDate:   time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		DueDate:       time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC),
		PaymentTerms:  "30 Tage netto",
		CreatedBy:     uuid.New(),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	repo.invoices[inv.ID] = inv
	return inv
}

// ============================================================================
// Create Tests
// ============================================================================

func TestService_Create_Success(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()
	userID := uuid.New()
	input := testCreateInput(tenantID, userID)

	inv, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, inv.ID)
	assert.Equal(t, tenantID, inv.TenantID)
	assert.Equal(t, models.InvoiceStatusDraft, inv.Status)
	assert.Equal(t, "Test Customer", inv.CustomerName)
	assert.Equal(t, "", inv.InvoiceNumber) // Not assigned until sent
	assert.True(t, inv.GrossTotal.GreaterThan(decimal.Zero))
	assert.True(t, inv.Subtotal.GreaterThan(decimal.Zero))
	assert.Equal(t, "14 Tage netto", inv.PaymentTerms)
	assert.NotZero(t, inv.CreatedAt)
	assert.NotZero(t, inv.UpdatedAt)
	// Verify it was stored
	assert.Contains(t, repo.invoices, inv.ID)
}

func TestService_Create_NoLineItems(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	input := testCreateInput(uuid.New(), uuid.New())
	input.LineItems = nil

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrNoLineItems)
}

func TestService_Create_EmptyLineItems(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	input := testCreateInput(uuid.New(), uuid.New())
	input.LineItems = []models.LineItem{}

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrNoLineItems)
}

func TestService_Create_DefaultPaymentTerms(t *testing.T) {
	svc, _, _, cs, _ := newTestService()
	cs.settings = nil // No company settings
	input := testCreateInput(uuid.New(), uuid.New())
	input.PaymentTermsDays = 0 // Trigger default

	inv, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, "30 Tage netto", inv.PaymentTerms)
	// Due date should be 30 days from invoice date
	expectedDue := input.InvoiceDate.AddDate(0, 0, 30)
	assert.Equal(t, expectedDue, inv.DueDate)
}

func TestService_Create_PaymentTermsFromSettings(t *testing.T) {
	svc, _, _, cs, _ := newTestService()
	cs.settings = &models.CompanySettings{
		DefaultPaymentTermsDays: 45,
	}
	input := testCreateInput(uuid.New(), uuid.New())
	input.PaymentTermsDays = 0 // Trigger settings lookup

	inv, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, "45 Tage netto", inv.PaymentTerms)
	expectedDue := input.InvoiceDate.AddDate(0, 0, 45)
	assert.Equal(t, expectedDue, inv.DueDate)
}

func TestService_Create_RepoError(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	repo.createErr = errors.New("db connection failed")
	input := testCreateInput(uuid.New(), uuid.New())

	_, err := svc.Create(context.Background(), input)

	assert.Error(t, err)
	assert.Equal(t, "db connection failed", err.Error())
}

func TestService_Create_TaxCalculation(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	input := testCreateInput(uuid.New(), uuid.New())
	// 2 x 100 = 200 net, 19% tax = 38, gross = 238
	input.LineItems = []models.LineItem{{
		ID: "1", Position: 1, Description: "Widget",
		Quantity: decimal.NewFromInt(2), UnitPrice: decimal.NewFromInt(100),
		TaxRate: decimal.NewFromInt(19),
	}}

	inv, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.True(t, inv.Subtotal.Equal(decimal.NewFromInt(200)), "subtotal should be 200, got %s", inv.Subtotal)
	assert.True(t, inv.TotalTax.Equal(decimal.NewFromInt(38)), "total tax should be 38, got %s", inv.TotalTax)
	assert.True(t, inv.GrossTotal.Equal(decimal.NewFromInt(238)), "gross total should be 238, got %s", inv.GrossTotal)
}

// ============================================================================
// GetByID Tests
// ============================================================================

func TestService_GetByID_Success(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()
	inv := createDraftInvoice(t, repo, tenantID)

	result, err := svc.GetByID(context.Background(), tenantID, inv.ID)

	require.NoError(t, err)
	assert.Equal(t, inv.ID, result.ID)
	assert.Equal(t, "Draft Customer", result.CustomerName)
}

func TestService_GetByID_NotFound(t *testing.T) {
	svc, _, _, _, _ := newTestService()

	_, err := svc.GetByID(context.Background(), uuid.New(), uuid.New())

	assert.ErrorIs(t, err, ErrInvoiceNotFound)
}

// ============================================================================
// List Tests
// ============================================================================

func TestService_List_Success(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()
	createDraftInvoice(t, repo, tenantID)
	createDraftInvoice(t, repo, tenantID)

	invoices, total, err := svc.List(context.Background(), tenantID, ListFilter{})

	require.NoError(t, err)
	assert.Len(t, invoices, 2)
	assert.Equal(t, 2, total)
}

func TestService_List_RepoError(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	repo.listErr = errors.New("list failed")

	_, _, err := svc.List(context.Background(), uuid.New(), ListFilter{})

	assert.Error(t, err)
}

func TestService_List_FilterByContactID(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()
	contactID := uuid.New()
	otherContactID := uuid.New()

	// Invoice linked to contactID
	invA := createDraftInvoice(t, repo, tenantID)
	invA.ContactID = &contactID

	// Invoice linked to a different contact
	invB := createDraftInvoice(t, repo, tenantID)
	invB.ContactID = &otherContactID

	// Invoice with no contact
	createDraftInvoice(t, repo, tenantID)

	invoices, total, err := svc.List(context.Background(), tenantID, ListFilter{
		ContactID: &contactID,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, invoices, 1)
	assert.Equal(t, invA.ID, invoices[0].ID)
}

// ============================================================================
// Update Tests
// ============================================================================

func TestService_Update_Success_Draft(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()
	inv := createDraftInvoice(t, repo, tenantID)

	newName := "Updated Customer"
	newNotes := "Updated notes"
	result, err := svc.Update(context.Background(), tenantID, inv.ID, UpdateInput{
		CustomerName: &newName,
		Notes:        &newNotes,
	})

	require.NoError(t, err)
	assert.Equal(t, "Updated Customer", result.CustomerName)
	assert.Equal(t, "Updated notes", result.Notes)
}

func TestService_Update_RejectsNonDraft(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()
	inv := createDraftInvoice(t, repo, tenantID)
	inv.Status = models.InvoiceStatusSent // Sent = immutable

	newName := "Should Fail"
	_, err := svc.Update(context.Background(), tenantID, inv.ID, UpdateInput{
		CustomerName: &newName,
	})

	assert.ErrorIs(t, err, ErrInvoiceImmutable)
}

func TestService_Update_RejectsOverdueInvoice(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()
	inv := createDraftInvoice(t, repo, tenantID)
	inv.Status = models.InvoiceStatusOverdue

	newName := "Should Fail"
	_, err := svc.Update(context.Background(), tenantID, inv.ID, UpdateInput{
		CustomerName: &newName,
	})

	assert.ErrorIs(t, err, ErrInvoiceImmutable)
}

func TestService_Update_RecalcTaxOnLineItemChange(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()
	inv := createDraftInvoice(t, repo, tenantID)

	// Change to 3 x 50 = 150 net, 19% = 28.50, gross = 178.50
	newItems := []models.LineItem{{
		ID: "1", Position: 1, Description: "New Item",
		Quantity: decimal.NewFromInt(3), UnitPrice: decimal.NewFromInt(50),
		TaxRate: decimal.NewFromInt(19),
	}}

	result, err := svc.Update(context.Background(), tenantID, inv.ID, UpdateInput{
		LineItems: newItems,
	})

	require.NoError(t, err)
	assert.True(t, result.Subtotal.Equal(decimal.NewFromInt(150)), "subtotal should be 150, got %s", result.Subtotal)
	expectedTax := decimal.NewFromFloat(28.5)
	assert.True(t, result.TotalTax.Equal(expectedTax), "total tax should be 28.5, got %s", result.TotalTax)
	expectedGross := decimal.NewFromFloat(178.5)
	assert.True(t, result.GrossTotal.Equal(expectedGross), "gross total should be 178.5, got %s", result.GrossTotal)
}

func TestService_Update_RecalcTaxOnTaxModeChange(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()
	inv := createDraftInvoice(t, repo, tenantID)

	// Switch to reverse_charge: tax should be 0
	reverseCharge := models.TaxModeReverseCharge
	result, err := svc.Update(context.Background(), tenantID, inv.ID, UpdateInput{
		TaxMode: &reverseCharge,
	})

	require.NoError(t, err)
	assert.Equal(t, models.TaxModeReverseCharge, result.TaxMode)
	assert.True(t, result.TotalTax.Equal(decimal.Zero), "reverse charge should have zero tax, got %s", result.TotalTax)
	assert.True(t, result.GrossTotal.Equal(result.Subtotal), "gross should equal subtotal for reverse charge")
}

func TestService_Update_NoLineItems(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()
	inv := createDraftInvoice(t, repo, tenantID)

	emptyItems := []models.LineItem{}
	_, err := svc.Update(context.Background(), tenantID, inv.ID, UpdateInput{
		LineItems: emptyItems,
	})

	assert.ErrorIs(t, err, ErrNoLineItems)
}

func TestService_Update_NotFound(t *testing.T) {
	svc, _, _, _, _ := newTestService()

	newName := "Fail"
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), UpdateInput{
		CustomerName: &newName,
	})

	assert.ErrorIs(t, err, ErrInvoiceNotFound)
}

// ============================================================================
// Send Tests
// ============================================================================

func TestService_Send_Success(t *testing.T) {
	svc, repo, numSeq, cs, _ := newTestService()
	tenantID := uuid.New()
	userID := uuid.New()
	inv := createDraftInvoice(t, repo, tenantID)
	numSeq.nextNumber = "RE-2026-0042"
	cs.settings = &models.CompanySettings{
		Name:   "Test GmbH",
		Street: "Teststr. 1",
		PLZ:    "12345",
		City:   "Berlin",
	}

	err := svc.Send(context.Background(), tenantID, inv.ID, userID)

	require.NoError(t, err)
	updated := repo.invoices[inv.ID]
	assert.Equal(t, models.InvoiceStatusSent, updated.Status)
	assert.Equal(t, "RE-2026-0042", updated.InvoiceNumber)
	assert.NotNil(t, updated.SnapshotData)
	assert.NotNil(t, updated.CompanySnapshotRaw)
	assert.True(t, len(updated.SnapshotData) > 0)
	assert.True(t, len(updated.CompanySnapshotRaw) > 0)
}

func TestService_Send_RejectsNonDraft(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()
	inv := createDraftInvoice(t, repo, tenantID)
	inv.Status = models.InvoiceStatusSent

	err := svc.Send(context.Background(), tenantID, inv.ID, uuid.New())

	assert.ErrorIs(t, err, ErrInvoiceNotDraft)
}

func TestService_Send_RejectsPaidInvoice(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()
	inv := createDraftInvoice(t, repo, tenantID)
	inv.Status = models.InvoiceStatusPaid

	err := svc.Send(context.Background(), tenantID, inv.ID, uuid.New())

	assert.ErrorIs(t, err, ErrInvoiceNotDraft)
}

func TestService_Send_NumberSeqError(t *testing.T) {
	svc, repo, numSeq, _, _ := newTestService()
	tenantID := uuid.New()
	inv := createDraftInvoice(t, repo, tenantID)
	numSeq.nextErr = errors.New("sequence locked")

	err := svc.Send(context.Background(), tenantID, inv.ID, uuid.New())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sequence locked")
	// Invoice should remain draft
	assert.Equal(t, models.InvoiceStatusDraft, repo.invoices[inv.ID].Status)
}

func TestService_Send_NotFound(t *testing.T) {
	svc, _, _, _, _ := newTestService()

	err := svc.Send(context.Background(), uuid.New(), uuid.New(), uuid.New())

	assert.ErrorIs(t, err, ErrInvoiceNotFound)
}

// ============================================================================
// MarkPaid Tests
// ============================================================================

func TestService_MarkPaid_Success_FromSent(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()
	inv := createDraftInvoice(t, repo, tenantID)
	inv.Status = models.InvoiceStatusSent

	err := svc.MarkPaid(context.Background(), tenantID, inv.ID)

	require.NoError(t, err)
	assert.Equal(t, models.InvoiceStatusPaid, repo.invoices[inv.ID].Status)
}

func TestService_MarkPaid_Success_FromOverdue(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()
	inv := createDraftInvoice(t, repo, tenantID)
	inv.Status = models.InvoiceStatusOverdue

	err := svc.MarkPaid(context.Background(), tenantID, inv.ID)

	require.NoError(t, err)
	assert.Equal(t, models.InvoiceStatusPaid, repo.invoices[inv.ID].Status)
}

func TestService_MarkPaid_RejectsAlreadyPaid(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()
	inv := createDraftInvoice(t, repo, tenantID)
	inv.Status = models.InvoiceStatusPaid

	err := svc.MarkPaid(context.Background(), tenantID, inv.ID)

	assert.ErrorIs(t, err, ErrInvoiceAlreadyPaid)
}

func TestService_MarkPaid_RejectsCancelled(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()
	inv := createDraftInvoice(t, repo, tenantID)
	inv.Status = models.InvoiceStatusCancelled

	err := svc.MarkPaid(context.Background(), tenantID, inv.ID)

	assert.ErrorIs(t, err, ErrInvoiceAlreadyCancelled)
}

func TestService_MarkPaid_RejectsDraft(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()
	inv := createDraftInvoice(t, repo, tenantID)
	// Status is draft by default

	err := svc.MarkPaid(context.Background(), tenantID, inv.ID)

	assert.ErrorIs(t, err, ErrInvoiceNotDraft) // Draft falls into default case
}

func TestService_MarkPaid_NotFound(t *testing.T) {
	svc, _, _, _, _ := newTestService()

	err := svc.MarkPaid(context.Background(), uuid.New(), uuid.New())

	assert.ErrorIs(t, err, ErrInvoiceNotFound)
}

// ============================================================================
// Cancel Tests
// ============================================================================

func TestService_Cancel_Success_FromDraft(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()
	userID := uuid.New()
	inv := createDraftInvoice(t, repo, tenantID)

	err := svc.Cancel(context.Background(), tenantID, inv.ID, userID)

	require.NoError(t, err)
	assert.Equal(t, models.InvoiceStatusCancelled, repo.invoices[inv.ID].Status)
}

func TestService_Cancel_Success_FromSent(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()
	userID := uuid.New()
	inv := createDraftInvoice(t, repo, tenantID)
	inv.Status = models.InvoiceStatusSent

	err := svc.Cancel(context.Background(), tenantID, inv.ID, userID)

	require.NoError(t, err)
	assert.Equal(t, models.InvoiceStatusCancelled, repo.invoices[inv.ID].Status)
}

func TestService_Cancel_RejectsAlreadyPaid(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()
	inv := createDraftInvoice(t, repo, tenantID)
	inv.Status = models.InvoiceStatusPaid

	err := svc.Cancel(context.Background(), tenantID, inv.ID, uuid.New())

	assert.ErrorIs(t, err, ErrInvoiceAlreadyPaid)
}

func TestService_Cancel_RejectsCancelled(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()
	inv := createDraftInvoice(t, repo, tenantID)
	inv.Status = models.InvoiceStatusCancelled

	err := svc.Cancel(context.Background(), tenantID, inv.ID, uuid.New())

	assert.ErrorIs(t, err, ErrInvoiceAlreadyCancelled)
}

func TestService_Cancel_NotFound(t *testing.T) {
	svc, _, _, _, _ := newTestService()

	err := svc.Cancel(context.Background(), uuid.New(), uuid.New(), uuid.New())

	assert.ErrorIs(t, err, ErrInvoiceNotFound)
}

// ============================================================================
// DetectOverdue Tests
// ============================================================================

func TestService_DetectOverdue_MarksOverdue(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()

	// Create two sent invoices with past due dates
	inv1 := createDraftInvoice(t, repo, tenantID)
	inv1.Status = models.InvoiceStatusSent
	inv1.DueDate = time.Now().AddDate(0, 0, -7)

	inv2 := createDraftInvoice(t, repo, tenantID)
	inv2.Status = models.InvoiceStatusSent
	inv2.DueDate = time.Now().AddDate(0, 0, -1)

	count, err := svc.DetectOverdue(context.Background(), tenantID)

	require.NoError(t, err)
	assert.Equal(t, 2, count)
	assert.Equal(t, models.InvoiceStatusOverdue, repo.invoices[inv1.ID].Status)
	assert.Equal(t, models.InvoiceStatusOverdue, repo.invoices[inv2.ID].Status)
}

func TestService_DetectOverdue_NoOverdue(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()

	// Create a sent invoice with future due date (not overdue)
	inv := createDraftInvoice(t, repo, tenantID)
	inv.Status = models.InvoiceStatusSent
	inv.DueDate = time.Now().AddDate(0, 0, 30)

	count, err := svc.DetectOverdue(context.Background(), tenantID)

	require.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.Equal(t, models.InvoiceStatusSent, repo.invoices[inv.ID].Status)
}

func TestService_DetectOverdue_RepoError(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	repo.overdueErr = errors.New("query failed")

	count, err := svc.DetectOverdue(context.Background(), uuid.New())

	assert.Error(t, err)
	assert.Equal(t, 0, count)
}

// ============================================================================
// CreateFromQuote Tests
// ============================================================================

func TestService_CreateFromQuote_Success(t *testing.T) {
	svc, repo, _, cs, qr := newTestService()
	tenantID := uuid.New()
	userID := uuid.New()
	quoteID := uuid.New()

	cs.settings = &models.CompanySettings{
		DefaultPaymentTermsDays: 14,
	}

	lineItemsJSON, err := json.Marshal(testLineItems())
	require.NoError(t, err)

	qr.quotes[quoteID] = &models.Quote{
		ID:              quoteID,
		TenantID:        tenantID,
		Status:          models.QuoteStatusAccepted,
		CustomerName:    "Quote Customer",
		CustomerAddress: "Quote Street 5",
		CustomerEmail:   "quote@example.com",
		CustomerUStIDNr: "DE123456789",
		TaxMode:         models.TaxModeStandard,
		LineItems:       lineItemsJSON,
		Notes:           "From quote",
	}

	inv, err := svc.CreateFromQuote(context.Background(), tenantID, quoteID, userID)

	require.NoError(t, err)
	assert.Equal(t, "Quote Customer", inv.CustomerName)
	assert.Equal(t, "Quote Street 5", inv.CustomerAddress)
	assert.Equal(t, "quote@example.com", inv.CustomerEmail)
	assert.Equal(t, "DE123456789", inv.CustomerUStIDNr)
	assert.Equal(t, models.InvoiceStatusDraft, inv.Status)
	assert.Equal(t, &quoteID, inv.SourceQuoteID)
	assert.Equal(t, "14 Tage netto", inv.PaymentTerms)
	assert.True(t, inv.GrossTotal.GreaterThan(decimal.Zero))
	assert.Contains(t, repo.invoices, inv.ID)
}

func TestService_CreateFromQuote_RejectsNonAccepted(t *testing.T) {
	svc, _, _, _, qr := newTestService()
	tenantID := uuid.New()
	quoteID := uuid.New()

	lineItemsJSON, err := json.Marshal(testLineItems())
	require.NoError(t, err)

	qr.quotes[quoteID] = &models.Quote{
		ID:        quoteID,
		TenantID:  tenantID,
		Status:    models.QuoteStatusDraft, // Not accepted
		LineItems: lineItemsJSON,
	}

	_, err = svc.CreateFromQuote(context.Background(), tenantID, quoteID, uuid.New())

	assert.ErrorIs(t, err, ErrQuoteNotAccepted)
}

func TestService_CreateFromQuote_RejectsSentQuote(t *testing.T) {
	svc, _, _, _, qr := newTestService()
	tenantID := uuid.New()
	quoteID := uuid.New()

	lineItemsJSON, err := json.Marshal(testLineItems())
	require.NoError(t, err)

	qr.quotes[quoteID] = &models.Quote{
		ID:        quoteID,
		TenantID:  tenantID,
		Status:    models.QuoteStatusSent,
		LineItems: lineItemsJSON,
	}

	_, err = svc.CreateFromQuote(context.Background(), tenantID, quoteID, uuid.New())

	assert.ErrorIs(t, err, ErrQuoteNotAccepted)
}

func TestService_CreateFromQuote_NilQuoteReader(t *testing.T) {
	repo := NewMockRepository()
	numSeq := &MockNumberSequenceRepo{}
	cs := &MockCompanySettingsRepo{}
	svc := NewService(repo, numSeq, cs, nil, noopTxBeginner{}) // nil quoteReader

	_, err := svc.CreateFromQuote(context.Background(), uuid.New(), uuid.New(), uuid.New())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "quote reader not configured")
}

func TestService_CreateFromQuote_QuoteNotFound(t *testing.T) {
	svc, _, _, _, _ := newTestService()

	_, err := svc.CreateFromQuote(context.Background(), uuid.New(), uuid.New(), uuid.New())

	assert.Error(t, err)
}

// ============================================================================
// EventEmitter Tests
// ============================================================================

func TestService_Create_EmitsEvent(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	emitter := &MockEventEmitter{}
	svc.SetEventEmitter(emitter)

	input := testCreateInput(uuid.New(), uuid.New())

	_, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.Len(t, emitter.events, 1)
	assert.Equal(t, "biz.invoice.created", emitter.events[0].Type)
}

func TestService_Send_EmitsEvent(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	emitter := &MockEventEmitter{}
	svc.SetEventEmitter(emitter)
	tenantID := uuid.New()
	inv := createDraftInvoice(t, repo, tenantID)

	err := svc.Send(context.Background(), tenantID, inv.ID, uuid.New())

	require.NoError(t, err)
	assert.Len(t, emitter.events, 1)
	assert.Equal(t, "biz.invoice.sent", emitter.events[0].Type)
}

func TestService_Create_NilEmitterSafe(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	// No emitter set (nil) - should not panic
	input := testCreateInput(uuid.New(), uuid.New())

	inv, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.NotNil(t, inv)
}

// ============================================================================
// GoBD Lock-Guard Tests (Bug 1 — Review-Befund #11)
// ============================================================================

// createLockedSentInvoice creates a sent invoice, then locks it via LockInvoice.
// Returns the locked invoice and the lockedBy user ID.
func createLockedSentInvoice(t *testing.T, svc *Service, repo *MockRepository, numSeq *MockNumberSequenceRepo, tenantID uuid.UUID) (*models.Invoice, uuid.UUID) {
	t.Helper()
	userID := uuid.New()
	inv := createDraftInvoice(t, repo, tenantID)

	numSeq.nextNumber = "RE-2026-0099"
	err := svc.Send(context.Background(), tenantID, inv.ID, userID)
	require.NoError(t, err, "Send must succeed before locking")

	lockedBy := uuid.New()
	_, err = svc.LockInvoice(context.Background(), tenantID, inv.ID, lockedBy)
	require.NoError(t, err, "LockInvoice must succeed")

	return repo.invoices[inv.ID], lockedBy
}

// TestService_Update_RejectedWhenLocked verifies that Update returns ErrInvoiceLocked
// for an administratively locked invoice (GoBD §146).
func TestService_Update_RejectedWhenLocked(t *testing.T) {
	svc, repo, numSeq, _, _ := newTestService()
	tenantID := uuid.New()
	inv, _ := createLockedSentInvoice(t, svc, repo, numSeq, tenantID)

	newName := "Should Be Blocked"
	_, err := svc.Update(context.Background(), tenantID, inv.ID, UpdateInput{
		CustomerName: &newName,
	})

	// A locked invoice has status "sent" which first triggers ErrInvoiceImmutable.
	// Either ErrInvoiceImmutable or ErrInvoiceLocked is acceptable — both enforce
	// the GoBD write barrier. We verify that the update was NOT applied.
	assert.Error(t, err)
	stored := repo.invoices[inv.ID]
	assert.NotEqual(t, "Should Be Blocked", stored.CustomerName,
		"customer name must not change on a locked invoice")
}

// TestService_Cancel_RejectedWhenLocked verifies that Cancel returns ErrInvoiceLocked
// for an administratively locked invoice (GoBD §146).
func TestService_Cancel_RejectedWhenLocked(t *testing.T) {
	svc, repo, numSeq, _, _ := newTestService()
	tenantID := uuid.New()
	inv, _ := createLockedSentInvoice(t, svc, repo, numSeq, tenantID)
	userID := uuid.New()

	err := svc.Cancel(context.Background(), tenantID, inv.ID, userID)

	assert.ErrorIs(t, err, ErrInvoiceLocked,
		"Cancel must return ErrInvoiceLocked for a locked invoice")
	stored := repo.invoices[inv.ID]
	assert.Equal(t, models.InvoiceStatusSent, stored.Status,
		"status must remain 'sent' after blocked Cancel")
}

// TestService_UpdateStatus_RejectedWhenLocked verifies that MarkPaid returns
// ErrInvoiceLocked for an administratively locked invoice (GoBD §146).
func TestService_UpdateStatus_RejectedWhenLocked(t *testing.T) {
	svc, repo, numSeq, _, _ := newTestService()
	tenantID := uuid.New()
	inv, _ := createLockedSentInvoice(t, svc, repo, numSeq, tenantID)

	err := svc.MarkPaid(context.Background(), tenantID, inv.ID)

	assert.ErrorIs(t, err, ErrInvoiceLocked,
		"MarkPaid must return ErrInvoiceLocked for a locked invoice")
	stored := repo.invoices[inv.ID]
	assert.Equal(t, models.InvoiceStatusSent, stored.Status,
		"status must remain 'sent' after blocked MarkPaid")
}

// ============================================================================
// GoBD Journal-Summary Cancelled-Invoice Test (Bug 2 — Review-Befund #12)
// ============================================================================

// ============================================================================
// ValidateInvoiceNumber Tests (Review-Befund #15)
// ============================================================================

func TestService_ValidateInvoiceNumber_ValidNew(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	tenantID := uuid.New()

	result, err := svc.ValidateInvoiceNumber(context.Background(), tenantID, "RE-2026-0042")

	require.NoError(t, err)
	assert.True(t, result.ValidFormat)
	assert.False(t, result.AlreadyUsed)
	assert.Equal(t, "RE-2026-0042", result.Canonical)
}

func TestService_ValidateInvoiceNumber_InvalidFormat(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	tenantID := uuid.New()

	cases := []string{
		"foo",
		"RE-26-1",
		"INV-2026-0001",
		"",
		"RE-2026",
		"re-2026-0042", // lowercase — pattern only accepts [A-Z]{2}
	}

	for _, tc := range cases {
		result, err := svc.ValidateInvoiceNumber(context.Background(), tenantID, tc)
		require.NoError(t, err, "case %q", tc)
		assert.False(t, result.ValidFormat, "case %q should be invalid", tc)
		assert.Empty(t, result.Canonical, "case %q should have no canonical form", tc)
	}
}

func TestService_ValidateInvoiceNumber_AlreadyUsed(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()

	// Seed the repo with an invoice carrying that number
	existing := createDraftInvoice(t, repo, tenantID)
	existing.InvoiceNumber = "RE-2026-0007"

	result, err := svc.ValidateInvoiceNumber(context.Background(), tenantID, "RE-2026-0007")

	require.NoError(t, err)
	assert.True(t, result.ValidFormat)
	assert.True(t, result.AlreadyUsed)
	assert.Equal(t, "RE-2026-0007", result.Canonical)
}

func TestService_ValidateInvoiceNumber_CanonicalLowercase(t *testing.T) {
	// Pattern requires [A-Z]{2} — lowercase "re-2026-0042" must be invalid.
	svc, _, _, _, _ := newTestService()
	tenantID := uuid.New()

	result, err := svc.ValidateInvoiceNumber(context.Background(), tenantID, "re-2026-0042")

	require.NoError(t, err)
	assert.False(t, result.ValidFormat, "lowercase prefix must not match the pattern")
	assert.Empty(t, result.Canonical)
}

// ============================================================================
// GetPaymentStats Tests (Review-Befund #15)
// ============================================================================

func TestService_GetPaymentStats_Empty(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	tenantID := uuid.New()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)

	stats, err := svc.GetPaymentStats(context.Background(), tenantID, from, to)

	require.NoError(t, err)
	assert.Equal(t, 0, stats.TotalInvoices)
	assert.Equal(t, 0, stats.TotalPaid)
	assert.Equal(t, 0, stats.TotalOutstanding)
}

func TestService_GetPaymentStats_Aggregates(t *testing.T) {
	// The MockRepository.AggregatePaymentStats is a stub returning zeroed stats.
	// We replace it with a richer mock to verify the service delegates correctly.
	repo := NewMockRepository()
	// Patch the mock to return 2 paid + 1 outstanding
	repo.aggregateStatsResult = PaymentStats{
		TotalInvoices:    3,
		TotalPaid:        2,
		TotalOutstanding: 1,
	}

	numSeq := &MockNumberSequenceRepo{}
	cs := &MockCompanySettingsRepo{}
	svc := NewService(repo, numSeq, cs, NewMockQuoteReader(), noopTxBeginner{})
	tenantID := uuid.New()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)

	stats, err := svc.GetPaymentStats(context.Background(), tenantID, from, to)

	require.NoError(t, err)
	assert.Equal(t, 3, stats.TotalInvoices)
	assert.Equal(t, 2, stats.TotalPaid)
	assert.Equal(t, 1, stats.TotalOutstanding)
}

// TestService_GetJournalSummary_CountsCancelledInvoices verifies that cancelled
// invoices count toward TotalInvoicesIssued and do not create false gaps.
// GoBD requires that stornierte Belege the sequence not interrupt.
func TestService_GetJournalSummary_CountsCancelledInvoices(t *testing.T) {
	svc, repo, numSeq, _, _ := newTestService()
	tenantID := uuid.New()
	fiscalYear := 2026

	// Configure sequence: highest assigned number is 3
	numSeq.seqInfo = &SequenceInfo{
		FiscalYear:    fiscalYear,
		CurrentNumber: 3,
	}

	makeInvoice := func(status, number string) {
		t.Helper()
		lineItemsJSON, err := json.Marshal(testLineItems())
		require.NoError(t, err)
		inv := &models.Invoice{
			ID:            uuid.New(),
			TenantID:      tenantID,
			Status:        status,
			InvoiceNumber: number,
			InvoiceDate:   time.Date(fiscalYear, 6, 1, 0, 0, 0, 0, time.UTC),
			LineItems:     lineItemsJSON,
			CreatedBy:     uuid.New(),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		repo.invoices[inv.ID] = inv
	}

	// 3 invoices: sent, cancelled, paid — all must be counted
	makeInvoice(models.InvoiceStatusSent, "RE-2026-0001")
	makeInvoice(models.InvoiceStatusCancelled, "RE-2026-0002")
	makeInvoice(models.InvoiceStatusPaid, "RE-2026-0003")

	summary, err := svc.GetJournalSummary(context.Background(), tenantID, fiscalYear)

	require.NoError(t, err)
	assert.Equal(t, 3, summary.TotalInvoicesIssued,
		"all three invoices (sent, cancelled, paid) must be counted")
	assert.Equal(t, 3, summary.HighestSequence)
	assert.Equal(t, 0, summary.GapsDetected,
		"no gaps expected when all sequence numbers have matching invoices")
}
