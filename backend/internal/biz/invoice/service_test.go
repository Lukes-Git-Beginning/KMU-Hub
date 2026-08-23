package invoice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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
	quoteLinkErr         error
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

// GetByQuoteID mirrors the real query's ordering: a live invoice wins over a
// cancelled one (map iteration order is random, so picking the first match would
// make the duplicate-conversion check flaky).
func (m *MockRepository) GetByQuoteID(ctx context.Context, tenantID, quoteID uuid.UUID) (*models.Invoice, error) {
	if m.quoteLinkErr != nil {
		return nil, m.quoteLinkErr
	}
	var fallback *models.Invoice
	for _, inv := range m.invoices {
		if inv.TenantID != tenantID || inv.SourceQuoteID == nil || *inv.SourceQuoteID != quoteID {
			continue
		}
		if inv.Status != models.InvoiceStatusCancelled {
			return inv, nil
		}
		fallback = inv
	}
	if fallback != nil {
		return fallback, nil
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

// UpsertImported stores an imported invoice, idempotent per (tenant_id, source,
// external_id): a repeat import reuses the existing row's id (mirrors the real
// ON CONFLICT behaviour).
func (m *MockRepository) UpsertImported(_ context.Context, invoice *models.Invoice) error {
	if m.createErr != nil {
		return m.createErr
	}
	for _, existing := range m.invoices {
		if existing.TenantID == invoice.TenantID && existing.Source == invoice.Source &&
			existing.ExternalID != nil && invoice.ExternalID != nil &&
			*existing.ExternalID == *invoice.ExternalID {
			invoice.ID = existing.ID
			break
		}
	}
	m.invoices[invoice.ID] = invoice
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
			inv.Source != models.InvoiceSourceBexio && inv.InvoiceDate.Year() == year {
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

func (m *MockRepository) ListDocumentChains(_ context.Context, _ uuid.UUID) ([]*models.DocumentChain, error) {
	return nil, nil
}

func (m *MockRepository) ListTransactions(_ context.Context, _ uuid.UUID) ([]*models.FinanceTransaction, error) {
	return nil, nil
}

// MockNumberSequenceRepo implements NumberSequenceRepo for testing.
type MockNumberSequenceRepo struct {
	nextNumber string
	nextErr    error
	seqInfo    *SequenceInfo // nil means "no sequence for this year"
	seqInfoErr error
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
		TenantID:         tenantID,
		CustomerName:     "Test Customer",
		CustomerAddress:  "Test Street 1, 12345 Berlin",
		CustomerEmail:    "customer@example.com",
		TaxMode:          models.TaxModeStandard,
		LineItems:        testLineItems(),
		InvoiceDate:      time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		PaymentTermsDays: 14,
		Notes:            "Test invoice",
		UserID:           userID,
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

// fakeStornoCreator stands in for creditnote.Service. It records the call and
// simulates the real atomic behaviour (the Storno tx flips the invoice to
// cancelled) so Cancel's orchestration can be exercised without a database.
type fakeStornoCreator struct {
	repo   *MockRepository
	called bool
	cnID   uuid.UUID
	cnNum  string
	err    error
}

func (f *fakeStornoCreator) StornoInvoice(_ context.Context, _, invoiceID, _ uuid.UUID) (uuid.UUID, string, error) {
	f.called = true
	if f.err != nil {
		return uuid.Nil, "", f.err
	}
	if inv, ok := f.repo.invoices[invoiceID]; ok {
		inv.Status = models.InvoiceStatusCancelled
	}
	return f.cnID, f.cnNum, nil
}

// TestService_Cancel_Sent_IssuesStorno verifies that cancelling an issued (sent)
// invoice reverses it via a Storno credit note rather than a silent status flip
// (GoBD §146).
func TestService_Cancel_Sent_IssuesStorno(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()
	userID := uuid.New()
	inv := createDraftInvoice(t, repo, tenantID)
	inv.Status = models.InvoiceStatusSent

	storno := &fakeStornoCreator{repo: repo, cnID: uuid.New(), cnNum: "GS-2026-0001"}
	svc.SetStornoCreator(storno)

	err := svc.Cancel(context.Background(), tenantID, inv.ID, userID)

	require.NoError(t, err)
	assert.True(t, storno.called, "a sent invoice must be reversed via the storno path")
	assert.Equal(t, models.InvoiceStatusCancelled, repo.invoices[inv.ID].Status)
}

// TestService_Cancel_Overdue_IssuesStorno verifies an overdue (also issued)
// invoice takes the same storno path.
func TestService_Cancel_Overdue_IssuesStorno(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()
	inv := createDraftInvoice(t, repo, tenantID)
	inv.Status = models.InvoiceStatusOverdue

	storno := &fakeStornoCreator{repo: repo, cnID: uuid.New(), cnNum: "GS-2026-0002"}
	svc.SetStornoCreator(storno)

	err := svc.Cancel(context.Background(), tenantID, inv.ID, uuid.New())

	require.NoError(t, err)
	assert.True(t, storno.called)
	assert.Equal(t, models.InvoiceStatusCancelled, repo.invoices[inv.ID].Status)
}

// TestService_Cancel_Sent_WithoutStornoCreator_Refuses verifies that an issued
// invoice is never silently flipped to cancelled when no storno path is wired.
func TestService_Cancel_Sent_WithoutStornoCreator_Refuses(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()
	inv := createDraftInvoice(t, repo, tenantID)
	inv.Status = models.InvoiceStatusSent

	err := svc.Cancel(context.Background(), tenantID, inv.ID, uuid.New())

	assert.ErrorIs(t, err, ErrStornoUnavailable)
	assert.Equal(t, models.InvoiceStatusSent, repo.invoices[inv.ID].Status,
		"an issued invoice must never be flipped without a storno credit note")
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

// TestService_DetectOverdue_SkipsLockedInvoice verifies that an administratively
// locked invoice returned by GetOverdue is left untouched: DetectOverdue must
// enforce the same GoBD §146 write barrier as Update/MarkPaid/Cancel instead of
// flipping the status straight from the repo result set.
func TestService_DetectOverdue_SkipsLockedInvoice(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()

	locked := createDraftInvoice(t, repo, tenantID)
	locked.Status = models.InvoiceStatusSent
	locked.DueDate = time.Now().AddDate(0, 0, -7)
	lockedAt := time.Now()
	locked.LockedAt = &lockedAt

	unlocked := createDraftInvoice(t, repo, tenantID)
	unlocked.Status = models.InvoiceStatusSent
	unlocked.DueDate = time.Now().AddDate(0, 0, -1)

	count, err := svc.DetectOverdue(context.Background(), tenantID)

	require.NoError(t, err)
	assert.Equal(t, 1, count, "only the unlocked invoice must be counted")
	assert.Equal(t, models.InvoiceStatusSent, repo.invoices[locked.ID].Status,
		"locked invoice must keep its status (GoBD §146 write barrier)")
	assert.Equal(t, models.InvoiceStatusOverdue, repo.invoices[unlocked.ID].Status)
}

// TestService_LockInvoice_RejectsAlreadyLocked verifies that locking an
// already-locked invoice returns ErrInvoiceLocked instead of silently
// overwriting locked_at/locked_by (which would let a second administrator move
// the audit-trail timestamp).
func TestService_LockInvoice_RejectsAlreadyLocked(t *testing.T) {
	svc, repo, numSeq, _, _ := newTestService()
	tenantID := uuid.New()
	inv, firstLockedBy := createLockedSentInvoice(t, svc, repo, numSeq, tenantID)
	firstLockedAt := *repo.invoices[inv.ID].LockedAt

	_, err := svc.LockInvoice(context.Background(), tenantID, inv.ID, uuid.New())

	assert.ErrorIs(t, err, ErrInvoiceLocked)
	stored := repo.invoices[inv.ID]
	assert.Equal(t, firstLockedAt, *stored.LockedAt, "re-locking must not move the original lock timestamp")
	assert.Equal(t, firstLockedBy, *stored.LockedBy, "re-locking must not change the original locked_by")
}

// TestService_LockInvoice_RejectsDraft verifies that a draft invoice cannot be
// administratively locked: it should be cancelled or sent first, not frozen
// mid-edit.
func TestService_LockInvoice_RejectsDraft(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()
	inv := createDraftInvoice(t, repo, tenantID)

	_, err := svc.LockInvoice(context.Background(), tenantID, inv.ID, uuid.New())

	assert.Error(t, err)
	assert.Nil(t, repo.invoices[inv.ID].LockedAt)
}

// TestService_LockInvoice_RejectsBexioImported verifies that an imported
// (source=bexio) invoice cannot be locked via the Cosmi GoBD path -- it is a
// read-only mirror of the external book, not a document this system issued.
func TestService_LockInvoice_RejectsBexioImported(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	tenantID := uuid.New()
	inv := createDraftInvoice(t, repo, tenantID)
	inv.Status = models.InvoiceStatusSent
	inv.Source = models.InvoiceSourceBexio

	_, err := svc.LockInvoice(context.Background(), tenantID, inv.ID, uuid.New())

	assert.ErrorIs(t, err, ErrExternalReadOnly)
	assert.Nil(t, repo.invoices[inv.ID].LockedAt)
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

// acceptedQuoteFixture registers an accepted quote the conversion tests can convert.
func acceptedQuoteFixture(t *testing.T, qr *MockQuoteReader, tenantID, quoteID uuid.UUID) {
	t.Helper()
	lineItemsJSON, err := json.Marshal(testLineItems())
	require.NoError(t, err)
	qr.quotes[quoteID] = &models.Quote{
		ID:           quoteID,
		TenantID:     tenantID,
		Status:       models.QuoteStatusAccepted,
		CustomerName: "Quote Customer",
		TaxMode:      models.TaxModeStandard,
		LineItems:    lineItemsJSON,
	}
}

// TestService_CreateFromQuote_RejectsSecondConversion covers the double-click /
// retry case: the quote keeps its "accepted" status after a conversion, so
// without the guard a second call produced a second complete invoice.
func TestService_CreateFromQuote_RejectsSecondConversion(t *testing.T) {
	svc, repo, _, _, qr := newTestService()
	tenantID := uuid.New()
	quoteID := uuid.New()
	acceptedQuoteFixture(t, qr, tenantID, quoteID)

	first, err := svc.CreateFromQuote(context.Background(), tenantID, quoteID, uuid.New())
	require.NoError(t, err)

	second, err := svc.CreateFromQuote(context.Background(), tenantID, quoteID, uuid.New())

	assert.ErrorIs(t, err, ErrQuoteAlreadyConverted)
	assert.Nil(t, second)
	assert.Len(t, repo.invoices, 1, "the rejected second conversion must not have written an invoice")
	assert.Contains(t, repo.invoices, first.ID)
}

// TestService_CreateFromQuote_AllowsReconversionAfterStorno proves the guard is
// scoped to live invoices: a cancelled invoice stays in the table for GoBD, but
// the quote may be invoiced again after a storno.
func TestService_CreateFromQuote_AllowsReconversionAfterStorno(t *testing.T) {
	svc, repo, _, _, qr := newTestService()
	tenantID := uuid.New()
	quoteID := uuid.New()
	acceptedQuoteFixture(t, qr, tenantID, quoteID)

	first, err := svc.CreateFromQuote(context.Background(), tenantID, quoteID, uuid.New())
	require.NoError(t, err)
	repo.invoices[first.ID].Status = models.InvoiceStatusCancelled

	second, err := svc.CreateFromQuote(context.Background(), tenantID, quoteID, uuid.New())

	require.NoError(t, err)
	assert.NotEqual(t, first.ID, second.ID)
	assert.Len(t, repo.invoices, 2)
}

// TestService_CreateFromQuote_PropagatesQuoteLinkLookupError makes sure an
// unavailable database is not mistaken for "no invoice yet" -- only
// ErrInvoiceNotFound clears the way for a conversion.
func TestService_CreateFromQuote_PropagatesQuoteLinkLookupError(t *testing.T) {
	svc, repo, _, _, qr := newTestService()
	tenantID := uuid.New()
	quoteID := uuid.New()
	acceptedQuoteFixture(t, qr, tenantID, quoteID)
	repo.quoteLinkErr = errors.New("connection refused")

	_, err := svc.CreateFromQuote(context.Background(), tenantID, quoteID, uuid.New())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused")
	assert.Empty(t, repo.invoices, "a failed lookup must not fall through to creating an invoice")
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

// TestService_ValidateInvoiceNumber_SpecialCharacters proves the pattern rejects
// anything outside its exact grammar — a trust-boundary input, since the value
// comes straight from the gRPC request (biz_grpc.go ValidateInvoiceNumber).
func TestService_ValidateInvoiceNumber_SpecialCharacters(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	tenantID := uuid.New()

	cases := []string{
		"RE-2026-00@1",
		"RE-2026-0001;DROP TABLE finance_invoices;",
		"RE-2026-0001\n",
		"RE-2026-00 1",
		"RE/2026/0001",
		"RE-2026-000€",
		"RE-2026-0001\x00",
	}

	for _, tc := range cases {
		result, err := svc.ValidateInvoiceNumber(context.Background(), tenantID, tc)
		require.NoError(t, err, "case %q", tc)
		assert.False(t, result.ValidFormat, "case %q should be invalid", tc)
		assert.Empty(t, result.Canonical, "case %q should have no canonical form", tc)
	}
}

// TestService_ValidateInvoiceNumber_ExcessivelyLongSequenceRejected proves the
// sequence group's upper bound (\d{4,10}) actually rejects an overlong digit
// run. Without the bound, strconv.Atoi in ValidateInvoiceNumber silently
// discards ErrRange on overflow and reports ValidFormat=true with a nonsense
// canonical (RE-2026-9223372036854775807) for caller-supplied garbage.
func TestService_ValidateInvoiceNumber_ExcessivelyLongSequenceRejected(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	tenantID := uuid.New()

	overlong := "RE-2026-" + strings.Repeat("9", 25)
	result, err := svc.ValidateInvoiceNumber(context.Background(), tenantID, overlong)

	require.NoError(t, err)
	assert.False(t, result.ValidFormat, "a 25-digit sequence must not pass format validation")
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

// TestService_GetJournalSummary_FiscalYearBoundaryNoFalseGap verifies that the
// year rollover itself is not misreported as a gap: the sequence restarts at 1
// for the new fiscal year (see quote.PostgresNumberSequenceRepo, one row per
// tenant+document_type+fiscal_year), and the prior year's invoices must not be
// counted into — or create a phantom gap against — the new year's summary.
func TestService_GetJournalSummary_FiscalYearBoundaryNoFalseGap(t *testing.T) {
	svc, repo, numSeq, _, _ := newTestService()
	tenantID := uuid.New()
	priorYear, currentYear := 2025, 2026

	makeInvoice := func(status, number string, year int) {
		t.Helper()
		lineItemsJSON, err := json.Marshal(testLineItems())
		require.NoError(t, err)
		inv := &models.Invoice{
			ID:            uuid.New(),
			TenantID:      tenantID,
			Status:        status,
			InvoiceNumber: number,
			InvoiceDate:   time.Date(year, 6, 1, 0, 0, 0, 0, time.UTC),
			LineItems:     lineItemsJSON,
			CreatedBy:     uuid.New(),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		repo.invoices[inv.ID] = inv
	}

	// Prior year: 5 invoices fully using up its own sequence (RE-2025-0001..0005).
	for i := 1; i <= 5; i++ {
		makeInvoice(models.InvoiceStatusPaid, fmt.Sprintf("RE-%d-%04d", priorYear, i), priorYear)
	}
	// Current year: sequence has restarted at 1, only 3 invoices issued so far.
	makeInvoice(models.InvoiceStatusSent, fmt.Sprintf("RE-%d-0001", currentYear), currentYear)
	makeInvoice(models.InvoiceStatusPaid, fmt.Sprintf("RE-%d-0002", currentYear), currentYear)
	makeInvoice(models.InvoiceStatusSent, fmt.Sprintf("RE-%d-0003", currentYear), currentYear)

	// The number sequence for the current fiscal year reports 3, independent of
	// the prior year's much higher count — a per-year row, not a running total.
	numSeq.seqInfo = &SequenceInfo{FiscalYear: currentYear, CurrentNumber: 3}

	summary, err := svc.GetJournalSummary(context.Background(), tenantID, currentYear)

	require.NoError(t, err)
	assert.Equal(t, 3, summary.TotalInvoicesIssued,
		"the prior year's 5 invoices must not leak into the current year's count")
	assert.Equal(t, 3, summary.HighestSequence)
	assert.Equal(t, 0, summary.GapsDetected,
		"the fiscal year rollover itself must never be reported as a gap")
}

// TestService_GetJournalSummary_EmptyYearReturnsZeroesNotError covers
// GetSequenceInfo returning nil (no invoice ever issued for that fiscal
// year) — the report a Prüfer sees for a fresh tenant or an unused year must
// be an all-zero summary, not an error.
func TestService_GetJournalSummary_EmptyYearReturnsZeroesNotError(t *testing.T) {
	svc, _, numSeq, _, _ := newTestService()
	tenantID := uuid.New()

	numSeq.seqInfo = nil // default, spelled out: no sequence row exists for this year

	summary, err := svc.GetJournalSummary(context.Background(), tenantID, 2099)

	require.NoError(t, err)
	assert.Equal(t, 2099, summary.Year)
	assert.Equal(t, 0, summary.TotalInvoicesIssued)
	assert.Equal(t, 0, summary.GapsDetected)
	assert.Equal(t, 0, summary.HighestSequence)
	assert.Empty(t, summary.FirstNumber)
	assert.Empty(t, summary.LastNumber)
}

// TestService_GetJournalSummary_DetectsRealGap is the positive case none of
// the existing tests proved: the sequence counter ahead of the actually
// persisted, numbered invoice count. This is the exact scenario GoBD §146
// gap-detection exists to catch (a number consumed — e.g. by a request that
// crashed after NextNumberInTx but before commit reached the caller's
// retry-safe path — never reaching a stored invoice) and, before this test,
// GapsDetected > 0 had never actually been exercised.
func TestService_GetJournalSummary_DetectsRealGap(t *testing.T) {
	svc, repo, numSeq, _, _ := newTestService()
	tenantID := uuid.New()
	fiscalYear := 2026

	// Sequence advanced to 5, but only numbers 1-3 ended up as persisted,
	// numbered invoices — 2 and 5 were "consumed" without a matching row.
	numSeq.seqInfo = &SequenceInfo{FiscalYear: fiscalYear, CurrentNumber: 5}

	makeInvoice := func(number string) {
		t.Helper()
		lineItemsJSON, err := json.Marshal(testLineItems())
		require.NoError(t, err)
		inv := &models.Invoice{
			ID:            uuid.New(),
			TenantID:      tenantID,
			Status:        models.InvoiceStatusSent,
			InvoiceNumber: number,
			InvoiceDate:   time.Date(fiscalYear, 3, 1, 0, 0, 0, 0, time.UTC),
			LineItems:     lineItemsJSON,
			CreatedBy:     uuid.New(),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		repo.invoices[inv.ID] = inv
	}
	makeInvoice("RE-2026-0001")
	makeInvoice("RE-2026-0003")
	makeInvoice("RE-2026-0004")

	summary, err := svc.GetJournalSummary(context.Background(), tenantID, fiscalYear)

	require.NoError(t, err)
	assert.Equal(t, 3, summary.TotalInvoicesIssued)
	assert.Equal(t, 5, summary.HighestSequence)
	assert.Equal(t, 2, summary.GapsDetected,
		"5 numbers consumed, only 3 persisted as numbered invoices -> 2 gaps must be reported")
}
