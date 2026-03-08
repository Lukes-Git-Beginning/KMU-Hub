package quote

import (
	"context"
	"encoding/json"
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
// Mock Repository
// ============================================================================

type MockRepository struct {
	quotes          map[uuid.UUID]*models.Quote
	createErr       error
	getErr          error
	listErr         error
	updateErr       error
	deleteErr       error
	updateStatusErr error
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		quotes: make(map[uuid.UUID]*models.Quote),
	}
}

func (m *MockRepository) Create(ctx context.Context, quote *models.Quote) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.quotes[quote.ID] = quote
	return nil
}

func (m *MockRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Quote, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	q, ok := m.quotes[id]
	if !ok {
		return nil, ErrQuoteNotFound
	}
	return q, nil
}

func (m *MockRepository) List(ctx context.Context, tenantID uuid.UUID, filter ListFilter) ([]*models.Quote, int, error) {
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	var result []*models.Quote
	for _, q := range m.quotes {
		if q.TenantID != tenantID {
			continue
		}
		if filter.Status != "" && q.Status != filter.Status {
			continue
		}
		result = append(result, q)
	}
	total := len(result)
	if filter.Offset >= len(result) {
		return []*models.Quote{}, total, nil
	}
	end := filter.Offset + filter.Limit
	if end > len(result) || filter.Limit == 0 {
		end = len(result)
	}
	return result[filter.Offset:end], total, nil
}

func (m *MockRepository) Update(ctx context.Context, quote *models.Quote) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.quotes[quote.ID] = quote
	return nil
}

func (m *MockRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.quotes, id)
	return nil
}

func (m *MockRepository) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status string) error {
	if m.updateStatusErr != nil {
		return m.updateStatusErr
	}
	if q, ok := m.quotes[id]; ok {
		q.Status = status
	}
	return nil
}

func (m *MockRepository) GetByDealID(ctx context.Context, tenantID, dealID uuid.UUID) ([]*models.Quote, error) {
	var result []*models.Quote
	for _, q := range m.quotes {
		if q.DealID != nil && *q.DealID == dealID && q.TenantID == tenantID {
			result = append(result, q)
		}
	}
	return result, nil
}

// ============================================================================
// Mock NumberSequenceRepo
// ============================================================================

type MockNumberSequenceRepo struct {
	nextNumber string
	nextErr    error
}

func (m *MockNumberSequenceRepo) NextNumber(ctx context.Context, tenantID uuid.UUID, documentType string, fiscalYear int, prefix string) (string, error) {
	if m.nextErr != nil {
		return "", m.nextErr
	}
	if m.nextNumber != "" {
		return m.nextNumber, nil
	}
	return "AN-2026-0001", nil
}

// ============================================================================
// Mock CompanySettingsRepo
// ============================================================================

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

// ============================================================================
// Mock DealValueUpdater
// ============================================================================

type MockDealValueUpdater struct {
	calledWith  map[uuid.UUID]decimal.Decimal
	updateErr   error
	callCount   int
}

func NewMockDealValueUpdater() *MockDealValueUpdater {
	return &MockDealValueUpdater{
		calledWith: make(map[uuid.UUID]decimal.Decimal),
	}
}

func (m *MockDealValueUpdater) UpdateDealValue(ctx context.Context, dealID uuid.UUID, grossTotal decimal.Decimal) error {
	m.callCount++
	if m.updateErr != nil {
		return m.updateErr
	}
	m.calledWith[dealID] = grossTotal
	return nil
}

// ============================================================================
// Mock EventEmitter
// ============================================================================

type MockEventEmitter struct {
	events []models.EventPayload
	err    error
}

func (m *MockEventEmitter) EmitBizEvent(ctx context.Context, payload models.EventPayload) error {
	if m.err != nil {
		return m.err
	}
	m.events = append(m.events, payload)
	return nil
}

// ============================================================================
// Test Helpers
// ============================================================================

func newTestService(repo *MockRepository) (*Service, *MockNumberSequenceRepo, *MockCompanySettingsRepo) {
	numRepo := &MockNumberSequenceRepo{}
	csRepo := &MockCompanySettingsRepo{}
	svc := NewService(repo, numRepo, csRepo, nil)
	return svc, numRepo, csRepo
}

func newTestServiceWithDealUpdater(repo *MockRepository, du *MockDealValueUpdater) (*Service, *MockNumberSequenceRepo, *MockCompanySettingsRepo) {
	numRepo := &MockNumberSequenceRepo{}
	csRepo := &MockCompanySettingsRepo{}
	svc := NewService(repo, numRepo, csRepo, du)
	return svc, numRepo, csRepo
}

func sampleLineItems() []models.LineItem {
	return []models.LineItem{
		{
			ID:          "item-1",
			Position:    1,
			Description: "Consulting",
			Quantity:    decimal.NewFromInt(2),
			UnitPrice:   decimal.NewFromInt(100),
			TaxRate:     decimal.NewFromInt(19),
		},
	}
}

func sampleCreateInput(tenantID uuid.UUID) CreateInput {
	return CreateInput{
		TenantID:        tenantID,
		CustomerName:    "Acme GmbH",
		CustomerAddress: "Musterstr. 1, 10115 Berlin",
		CustomerEmail:   "info@acme.de",
		TaxMode:         models.TaxModeStandard,
		LineItems:       sampleLineItems(),
		UserID:          uuid.New(),
	}
}

func seedDraftQuote(repo *MockRepository, tenantID uuid.UUID) *models.Quote {
	lineItems, _ := json.Marshal(sampleLineItems())
	q := &models.Quote{
		ID:           uuid.New(),
		TenantID:     tenantID,
		Status:       models.QuoteStatusDraft,
		CustomerName: "Acme GmbH",
		TaxMode:      models.TaxModeStandard,
		LineItems:    lineItems,
		Subtotal:     decimal.NewFromInt(200),
		TotalTax:     decimal.NewFromInt(38),
		GrossTotal:   decimal.NewFromInt(238),
		CreatedBy:    uuid.New(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	repo.quotes[q.ID] = q
	return q
}

func seedSentQuote(repo *MockRepository, tenantID uuid.UUID, validUntil *time.Time) *models.Quote {
	q := seedDraftQuote(repo, tenantID)
	q.Status = models.QuoteStatusSent
	q.QuoteNumber = "AN-2026-0001"
	q.ValidUntil = validUntil
	return q
}

// ============================================================================
// Create Tests
// ============================================================================

func TestService_Create_Success(t *testing.T) {
	repo := NewMockRepository()
	svc, _, _ := newTestService(repo)
	tenantID := uuid.New()

	input := sampleCreateInput(tenantID)
	quote, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, quote.ID)
	assert.Equal(t, tenantID, quote.TenantID)
	assert.Equal(t, models.QuoteStatusDraft, quote.Status)
	assert.Equal(t, "", quote.QuoteNumber) // Assigned on send
	assert.Equal(t, "Acme GmbH", quote.CustomerName)
	assert.NotZero(t, quote.CreatedAt)
	assert.NotZero(t, quote.UpdatedAt)

	// Tax calculation: 2 * 100 = 200 net, 200 * 0.19 = 38 tax, 238 gross
	assert.True(t, quote.Subtotal.Equal(decimal.NewFromInt(200)), "subtotal should be 200, got %s", quote.Subtotal)
	assert.True(t, quote.TotalTax.Equal(decimal.NewFromInt(38)), "total tax should be 38, got %s", quote.TotalTax)
	assert.True(t, quote.GrossTotal.Equal(decimal.NewFromInt(238)), "gross total should be 238, got %s", quote.GrossTotal)
}

func TestService_Create_NoLineItems(t *testing.T) {
	repo := NewMockRepository()
	svc, _, _ := newTestService(repo)

	input := sampleCreateInput(uuid.New())
	input.LineItems = nil

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrNoLineItems)
}

func TestService_Create_EmptyLineItems(t *testing.T) {
	repo := NewMockRepository()
	svc, _, _ := newTestService(repo)

	input := sampleCreateInput(uuid.New())
	input.LineItems = []models.LineItem{}

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrNoLineItems)
}

func TestService_Create_WithDealSync(t *testing.T) {
	repo := NewMockRepository()
	du := NewMockDealValueUpdater()
	svc, _, _ := newTestServiceWithDealUpdater(repo, du)
	tenantID := uuid.New()
	dealID := uuid.New()

	input := sampleCreateInput(tenantID)
	input.DealID = &dealID

	quote, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, 1, du.callCount)
	syncedValue, ok := du.calledWith[dealID]
	assert.True(t, ok, "deal updater should have been called with dealID")
	assert.True(t, syncedValue.Equal(quote.GrossTotal), "synced value should match gross total")
}

func TestService_Create_NilDealUpdater(t *testing.T) {
	repo := NewMockRepository()
	svc, _, _ := newTestService(repo) // nil dealUpdater
	tenantID := uuid.New()
	dealID := uuid.New()

	input := sampleCreateInput(tenantID)
	input.DealID = &dealID

	quote, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.NotNil(t, quote)
	// No panic, no error — graceful without deal updater
}

func TestService_Create_ValidUntilFromSettings(t *testing.T) {
	repo := NewMockRepository()
	svc, _, csRepo := newTestService(repo)
	tenantID := uuid.New()

	csRepo.settings = &models.CompanySettings{
		TenantID:                 tenantID,
		DefaultQuoteValidityDays: 30,
	}

	input := sampleCreateInput(tenantID)
	input.ValidUntil = nil // Let settings decide

	quote, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	require.NotNil(t, quote.ValidUntil)
	// Should be approximately 30 days from now
	expectedMin := time.Now().AddDate(0, 0, 29)
	expectedMax := time.Now().AddDate(0, 0, 31)
	assert.True(t, quote.ValidUntil.After(expectedMin), "valid_until should be ~30 days out")
	assert.True(t, quote.ValidUntil.Before(expectedMax), "valid_until should be ~30 days out")
}

func TestService_Create_ExplicitValidUntilOverridesSettings(t *testing.T) {
	repo := NewMockRepository()
	svc, _, csRepo := newTestService(repo)
	tenantID := uuid.New()

	csRepo.settings = &models.CompanySettings{
		TenantID:                 tenantID,
		DefaultQuoteValidityDays: 30,
	}

	explicitDate := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	input := sampleCreateInput(tenantID)
	input.ValidUntil = &explicitDate

	quote, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	require.NotNil(t, quote.ValidUntil)
	assert.Equal(t, explicitDate, *quote.ValidUntil)
}

// ============================================================================
// GetByID Tests
// ============================================================================

func TestService_GetByID_Success(t *testing.T) {
	repo := NewMockRepository()
	svc, _, _ := newTestService(repo)
	tenantID := uuid.New()

	existing := seedDraftQuote(repo, tenantID)

	quote, err := svc.GetByID(context.Background(), tenantID, existing.ID)

	require.NoError(t, err)
	assert.Equal(t, existing.ID, quote.ID)
	assert.Equal(t, "Acme GmbH", quote.CustomerName)
}

func TestService_GetByID_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc, _, _ := newTestService(repo)

	_, err := svc.GetByID(context.Background(), uuid.New(), uuid.New())

	assert.ErrorIs(t, err, ErrQuoteNotFound)
}

// ============================================================================
// Update Tests
// ============================================================================

func TestService_Update_Success_Draft(t *testing.T) {
	repo := NewMockRepository()
	svc, _, _ := newTestService(repo)
	tenantID := uuid.New()

	existing := seedDraftQuote(repo, tenantID)
	newName := "Updated GmbH"
	newNotes := "Updated notes"

	updated, err := svc.Update(context.Background(), tenantID, existing.ID, UpdateInput{
		CustomerName: &newName,
		Notes:        &newNotes,
	})

	require.NoError(t, err)
	assert.Equal(t, "Updated GmbH", updated.CustomerName)
	assert.Equal(t, "Updated notes", updated.Notes)
	assert.True(t, updated.UpdatedAt.After(existing.CreatedAt) || updated.UpdatedAt.Equal(existing.CreatedAt))
}

func TestService_Update_RejectsNonDraft(t *testing.T) {
	repo := NewMockRepository()
	svc, _, _ := newTestService(repo)
	tenantID := uuid.New()

	sent := seedSentQuote(repo, tenantID, nil)
	newName := "Should Fail"

	_, err := svc.Update(context.Background(), tenantID, sent.ID, UpdateInput{
		CustomerName: &newName,
	})

	assert.ErrorIs(t, err, ErrQuoteNotDraft)
}

func TestService_Update_RecalcTaxOnLineItemChange(t *testing.T) {
	repo := NewMockRepository()
	svc, _, _ := newTestService(repo)
	tenantID := uuid.New()

	existing := seedDraftQuote(repo, tenantID)

	// Change line items: 1 x 500 at 19%
	newItems := []models.LineItem{
		{
			ID:          "item-new",
			Position:    1,
			Description: "Premium Consulting",
			Quantity:    decimal.NewFromInt(1),
			UnitPrice:   decimal.NewFromInt(500),
			TaxRate:     decimal.NewFromInt(19),
		},
	}

	updated, err := svc.Update(context.Background(), tenantID, existing.ID, UpdateInput{
		LineItems: newItems,
	})

	require.NoError(t, err)
	// 1 * 500 = 500 net, 500 * 0.19 = 95 tax, 595 gross
	assert.True(t, updated.Subtotal.Equal(decimal.NewFromInt(500)), "subtotal should be 500, got %s", updated.Subtotal)
	assert.True(t, updated.TotalTax.Equal(decimal.NewFromInt(95)), "total tax should be 95, got %s", updated.TotalTax)
	assert.True(t, updated.GrossTotal.Equal(decimal.NewFromInt(595)), "gross total should be 595, got %s", updated.GrossTotal)
}

func TestService_Update_EmptyLineItems(t *testing.T) {
	repo := NewMockRepository()
	svc, _, _ := newTestService(repo)
	tenantID := uuid.New()

	existing := seedDraftQuote(repo, tenantID)

	_, err := svc.Update(context.Background(), tenantID, existing.ID, UpdateInput{
		LineItems: []models.LineItem{}, // empty = error
	})

	assert.ErrorIs(t, err, ErrNoLineItems)
}

func TestService_Update_DealSync(t *testing.T) {
	repo := NewMockRepository()
	du := NewMockDealValueUpdater()
	svc, _, _ := newTestServiceWithDealUpdater(repo, du)
	tenantID := uuid.New()

	existing := seedDraftQuote(repo, tenantID)
	dealID := uuid.New()
	existing.DealID = &dealID

	newItems := []models.LineItem{
		{
			ID:          "item-1",
			Position:    1,
			Description: "Service",
			Quantity:    decimal.NewFromInt(3),
			UnitPrice:   decimal.NewFromInt(100),
			TaxRate:     decimal.NewFromInt(19),
		},
	}

	updated, err := svc.Update(context.Background(), tenantID, existing.ID, UpdateInput{
		LineItems: newItems,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, du.callCount)
	syncedValue, ok := du.calledWith[dealID]
	assert.True(t, ok, "deal updater should have been called")
	assert.True(t, syncedValue.Equal(updated.GrossTotal), "synced value should match updated gross total")
}

// ============================================================================
// Delete Tests
// ============================================================================

func TestService_Delete_Success_Draft(t *testing.T) {
	repo := NewMockRepository()
	svc, _, _ := newTestService(repo)
	tenantID := uuid.New()

	existing := seedDraftQuote(repo, tenantID)

	err := svc.Delete(context.Background(), tenantID, existing.ID)

	require.NoError(t, err)
	assert.NotContains(t, repo.quotes, existing.ID)
}

func TestService_Delete_RejectsNonDraft(t *testing.T) {
	repo := NewMockRepository()
	svc, _, _ := newTestService(repo)
	tenantID := uuid.New()

	sent := seedSentQuote(repo, tenantID, nil)

	err := svc.Delete(context.Background(), tenantID, sent.ID)

	assert.ErrorIs(t, err, ErrQuoteNotDraft)
	assert.Contains(t, repo.quotes, sent.ID) // Not deleted
}

// ============================================================================
// Send Tests
// ============================================================================

func TestService_Send_Success(t *testing.T) {
	repo := NewMockRepository()
	svc, numRepo, _ := newTestService(repo)
	tenantID := uuid.New()
	userID := uuid.New()

	numRepo.nextNumber = "AN-2026-0042"
	existing := seedDraftQuote(repo, tenantID)

	err := svc.Send(context.Background(), tenantID, existing.ID, userID)

	require.NoError(t, err)
	updated := repo.quotes[existing.ID]
	assert.Equal(t, models.QuoteStatusSent, updated.Status)
	assert.Equal(t, "AN-2026-0042", updated.QuoteNumber)
}

func TestService_Send_RejectsNonDraft(t *testing.T) {
	repo := NewMockRepository()
	svc, _, _ := newTestService(repo)
	tenantID := uuid.New()
	userID := uuid.New()

	sent := seedSentQuote(repo, tenantID, nil)

	err := svc.Send(context.Background(), tenantID, sent.ID, userID)

	assert.ErrorIs(t, err, ErrQuoteNotDraft)
}

func TestService_Send_NumberSequenceError(t *testing.T) {
	repo := NewMockRepository()
	svc, numRepo, _ := newTestService(repo)
	tenantID := uuid.New()
	userID := uuid.New()

	numRepo.nextErr = errors.New("sequence exhausted")
	existing := seedDraftQuote(repo, tenantID)

	err := svc.Send(context.Background(), tenantID, existing.ID, userID)

	assert.Error(t, err)
	assert.Equal(t, models.QuoteStatusDraft, repo.quotes[existing.ID].Status) // Status unchanged
}

// ============================================================================
// Accept Tests
// ============================================================================

func TestService_Accept_Success_FromSent(t *testing.T) {
	repo := NewMockRepository()
	svc, _, _ := newTestService(repo)
	tenantID := uuid.New()

	sent := seedSentQuote(repo, tenantID, nil)

	err := svc.Accept(context.Background(), tenantID, sent.ID)

	require.NoError(t, err)
	assert.Equal(t, models.QuoteStatusAccepted, repo.quotes[sent.ID].Status)
}

func TestService_Accept_RejectsNonSent(t *testing.T) {
	repo := NewMockRepository()
	svc, _, _ := newTestService(repo)
	tenantID := uuid.New()

	draft := seedDraftQuote(repo, tenantID)

	err := svc.Accept(context.Background(), tenantID, draft.ID)

	assert.ErrorIs(t, err, ErrQuoteNotSent)
}

func TestService_Accept_RejectsAlreadyAccepted(t *testing.T) {
	repo := NewMockRepository()
	svc, _, _ := newTestService(repo)
	tenantID := uuid.New()

	sent := seedSentQuote(repo, tenantID, nil)
	sent.Status = models.QuoteStatusAccepted

	err := svc.Accept(context.Background(), tenantID, sent.ID)

	assert.ErrorIs(t, err, ErrQuoteNotSent)
}

// ============================================================================
// Reject Tests
// ============================================================================

func TestService_Reject_Success_FromSent(t *testing.T) {
	repo := NewMockRepository()
	svc, _, _ := newTestService(repo)
	tenantID := uuid.New()

	sent := seedSentQuote(repo, tenantID, nil)

	err := svc.Reject(context.Background(), tenantID, sent.ID)

	require.NoError(t, err)
	assert.Equal(t, models.QuoteStatusRejected, repo.quotes[sent.ID].Status)
}

func TestService_Reject_RejectsNonSent(t *testing.T) {
	repo := NewMockRepository()
	svc, _, _ := newTestService(repo)
	tenantID := uuid.New()

	draft := seedDraftQuote(repo, tenantID)

	err := svc.Reject(context.Background(), tenantID, draft.ID)

	assert.ErrorIs(t, err, ErrQuoteNotSent)
}

// ============================================================================
// ExpireQuotes Tests
// ============================================================================

func TestService_ExpireQuotes_ExpiresOld(t *testing.T) {
	repo := NewMockRepository()
	svc, _, _ := newTestService(repo)
	tenantID := uuid.New()

	pastDate := time.Now().Add(-48 * time.Hour)
	expired1 := seedSentQuote(repo, tenantID, &pastDate)
	expired2 := seedSentQuote(repo, tenantID, &pastDate)

	count, err := svc.ExpireQuotes(context.Background(), tenantID)

	require.NoError(t, err)
	assert.Equal(t, 2, count)
	assert.Equal(t, models.QuoteStatusExpired, repo.quotes[expired1.ID].Status)
	assert.Equal(t, models.QuoteStatusExpired, repo.quotes[expired2.ID].Status)
}

func TestService_ExpireQuotes_SkipsFutureValidUntil(t *testing.T) {
	repo := NewMockRepository()
	svc, _, _ := newTestService(repo)
	tenantID := uuid.New()

	futureDate := time.Now().Add(48 * time.Hour)
	future := seedSentQuote(repo, tenantID, &futureDate)

	count, err := svc.ExpireQuotes(context.Background(), tenantID)

	require.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.Equal(t, models.QuoteStatusSent, repo.quotes[future.ID].Status) // Unchanged
}

func TestService_ExpireQuotes_NoSentQuotes(t *testing.T) {
	repo := NewMockRepository()
	svc, _, _ := newTestService(repo)
	tenantID := uuid.New()

	// Only a draft quote exists
	seedDraftQuote(repo, tenantID)

	count, err := svc.ExpireQuotes(context.Background(), tenantID)

	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestService_ExpireQuotes_SkipsNilValidUntil(t *testing.T) {
	repo := NewMockRepository()
	svc, _, _ := newTestService(repo)
	tenantID := uuid.New()

	noExpiry := seedSentQuote(repo, tenantID, nil) // No valid_until

	count, err := svc.ExpireQuotes(context.Background(), tenantID)

	require.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.Equal(t, models.QuoteStatusSent, repo.quotes[noExpiry.ID].Status)
}

// ============================================================================
// Event Emitter Tests
// ============================================================================

func TestService_Create_EmitsEvent(t *testing.T) {
	repo := NewMockRepository()
	svc, _, _ := newTestService(repo)
	emitter := &MockEventEmitter{}
	svc.SetEventEmitter(emitter)
	tenantID := uuid.New()

	input := sampleCreateInput(tenantID)
	_, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.Len(t, emitter.events, 1)
	assert.Equal(t, "Acme GmbH", emitter.events[0].Body)
}

func TestService_Create_NoEmitterNoPanic(t *testing.T) {
	repo := NewMockRepository()
	svc, _, _ := newTestService(repo)
	// No emitter set
	tenantID := uuid.New()

	input := sampleCreateInput(tenantID)
	quote, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.NotNil(t, quote)
}
