package deal

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// MockRepository implements Repository for testing
type MockRepository struct {
	deals          map[uuid.UUID]*models.Deal
	dealTags       map[uuid.UUID][]*models.Tag
	customFields   map[uuid.UUID]map[uuid.UUID]any
	stages         map[uuid.UUID]*models.PipelineStage
	contacts       map[uuid.UUID]string // contactID -> name
	companies      map[uuid.UUID]string // companyID -> name
	owners         map[uuid.UUID]string // ownerID -> name
	validTags      map[uuid.UUID]models.EntityType
	createErr      error
	getErr         error
	updateErr      error
	deleteErr      error
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		deals:        make(map[uuid.UUID]*models.Deal),
		dealTags:     make(map[uuid.UUID][]*models.Tag),
		customFields: make(map[uuid.UUID]map[uuid.UUID]any),
		stages:       make(map[uuid.UUID]*models.PipelineStage),
		contacts:     make(map[uuid.UUID]string),
		companies:    make(map[uuid.UUID]string),
		owners:       make(map[uuid.UUID]string),
		validTags:    make(map[uuid.UUID]models.EntityType),
	}
}

func (m *MockRepository) Create(ctx context.Context, deal *models.Deal) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.deals[deal.ID] = deal
	return nil
}

func (m *MockRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Deal, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	deal, ok := m.deals[id]
	if !ok {
		return nil, ErrDealNotFound
	}
	return deal, nil
}

func (m *MockRepository) List(ctx context.Context, filter ListFilter, offset, limit int) ([]*models.Deal, int, error) {
	var result []*models.Deal
	for _, d := range m.deals {
		result = append(result, d)
	}
	total := len(result)
	if offset >= len(result) {
		return []*models.Deal{}, total, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], total, nil
}

func (m *MockRepository) Update(ctx context.Context, deal *models.Deal) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.deals[deal.ID] = deal
	return nil
}

func (m *MockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.deals, id)
	return nil
}

func (m *MockRepository) GetStageName(ctx context.Context, stageID uuid.UUID) (string, error) {
	if stage, ok := m.stages[stageID]; ok {
		return stage.Name, nil
	}
	return "", nil
}

func (m *MockRepository) GetContactName(ctx context.Context, contactID uuid.UUID) (string, error) {
	return m.contacts[contactID], nil
}

func (m *MockRepository) GetCompanyName(ctx context.Context, companyID uuid.UUID) (string, error) {
	return m.companies[companyID], nil
}

func (m *MockRepository) GetOwnerName(ctx context.Context, ownerID uuid.UUID) (string, error) {
	return m.owners[ownerID], nil
}

func (m *MockRepository) GetTags(ctx context.Context, dealID uuid.UUID) ([]*models.Tag, error) {
	return m.dealTags[dealID], nil
}

func (m *MockRepository) AddTags(ctx context.Context, dealID uuid.UUID, tagIDs []uuid.UUID) error {
	return nil
}

func (m *MockRepository) RemoveTags(ctx context.Context, dealID uuid.UUID, tagIDs []uuid.UUID) error {
	return nil
}

func (m *MockRepository) GetCustomFieldValues(ctx context.Context, dealID uuid.UUID) ([]*models.CustomFieldValueRow, error) {
	return nil, nil
}

func (m *MockRepository) SetCustomFieldValues(ctx context.Context, dealID uuid.UUID, values map[uuid.UUID]any) error {
	m.customFields[dealID] = values
	return nil
}

func (m *MockRepository) StageExists(ctx context.Context, stageID uuid.UUID) (bool, error) {
	_, exists := m.stages[stageID]
	return exists, nil
}

func (m *MockRepository) GetStage(ctx context.Context, stageID uuid.UUID) (*models.PipelineStage, error) {
	stage, ok := m.stages[stageID]
	if !ok {
		return nil, ErrStageNotFound
	}
	return stage, nil
}

func (m *MockRepository) ContactExists(ctx context.Context, contactID uuid.UUID) (bool, error) {
	_, exists := m.contacts[contactID]
	return exists, nil
}

func (m *MockRepository) CompanyExists(ctx context.Context, companyID uuid.UUID) (bool, error) {
	_, exists := m.companies[companyID]
	return exists, nil
}

func (m *MockRepository) OwnerExists(ctx context.Context, ownerID uuid.UUID) (bool, error) {
	_, exists := m.owners[ownerID]
	return exists, nil
}

func (m *MockRepository) TagExists(ctx context.Context, tagID uuid.UUID, entityType models.EntityType) (bool, error) {
	et, exists := m.validTags[tagID]
	return exists && et == entityType, nil
}

func (m *MockRepository) SetClosedAt(ctx context.Context, dealID uuid.UUID, closedAt *time.Time) error {
	if deal, ok := m.deals[dealID]; ok {
		deal.ClosedAt = closedAt
	}
	return nil
}

// Test helpers
func (m *MockRepository) AddStage(stage *models.PipelineStage) {
	m.stages[stage.ID] = stage
}

func (m *MockRepository) AddContact(contactID uuid.UUID, name string) {
	m.contacts[contactID] = name
}

func (m *MockRepository) AddCompany(companyID uuid.UUID, name string) {
	m.companies[companyID] = name
}

func (m *MockRepository) AddOwner(ownerID uuid.UUID, name string) {
	m.owners[ownerID] = name
}

func (m *MockRepository) AddValidTag(tagID uuid.UUID, entityType models.EntityType) {
	m.validTags[tagID] = entityType
}

// ============================================================================
// Create Tests
// ============================================================================

func TestService_Create_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.AddStage(&models.PipelineStage{
		ID:          stageID,
		Name:        "Lead",
		Probability: decimal.NewFromFloat(10.0),
	})

	input := CreateInput{
		Name:      "Big Deal",
		Value:     50000.00,
		Currency:  "EUR",
		StageID:   stageID,
		CreatedBy: uuid.New(),
	}

	deal, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, deal.ID)
	assert.Equal(t, "Big Deal", deal.Name)
	assert.Equal(t, decimal.NewFromFloat(50000.00), deal.Value)
	assert.Equal(t, "EUR", deal.Currency)
	assert.Equal(t, stageID, deal.StageID)
	assert.Equal(t, "Lead", deal.StageName)
	assert.Nil(t, deal.ClosedAt)
}

func TestService_Create_WithAllRelations(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.AddStage(&models.PipelineStage{
		ID:          stageID,
		Name:        "Proposal",
		Probability: decimal.NewFromFloat(50.0),
	})

	contactID := uuid.New()
	repo.AddContact(contactID, "John Doe")

	companyID := uuid.New()
	repo.AddCompany(companyID, "Acme Corp")

	ownerID := uuid.New()
	repo.AddOwner(ownerID, "Sales Rep")

	expectedClose := time.Now().AddDate(0, 1, 0)

	input := CreateInput{
		Name:              "Enterprise Deal",
		Value:             100000.00,
		Currency:          "USD",
		StageID:           stageID,
		ContactID:         &contactID,
		CompanyID:         &companyID,
		OwnerID:           &ownerID,
		ExpectedCloseDate: &expectedClose,
		CreatedBy:         uuid.New(),
	}

	deal, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, "Enterprise Deal", deal.Name)
	assert.Equal(t, contactID, *deal.ContactID)
	assert.Equal(t, "John Doe", *deal.ContactName)
	assert.Equal(t, companyID, *deal.CompanyID)
	assert.Equal(t, "Acme Corp", *deal.CompanyName)
	assert.Equal(t, ownerID, *deal.OwnerID)
	assert.Equal(t, "Sales Rep", *deal.OwnerName)
}

func TestService_Create_WonStage_SetsClosedAt(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.AddStage(&models.PipelineStage{
		ID:          stageID,
		Name:        "Won",
		IsWon:       true,
		Probability: decimal.NewFromFloat(100.0),
	})

	input := CreateInput{
		Name:      "Won Deal",
		Value:     10000.00,
		StageID:   stageID,
		CreatedBy: uuid.New(),
	}

	deal, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.NotNil(t, deal.ClosedAt)
}

func TestService_Create_NameRequired(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.AddStage(&models.PipelineStage{ID: stageID, Name: "Lead"})

	input := CreateInput{
		Name:      "",
		StageID:   stageID,
		CreatedBy: uuid.New(),
	}

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrNameRequired)
}

func TestService_Create_InvalidCurrency(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.AddStage(&models.PipelineStage{ID: stageID, Name: "Lead"})

	input := CreateInput{
		Name:      "Deal",
		Currency:  "XYZ",
		StageID:   stageID,
		CreatedBy: uuid.New(),
	}

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrInvalidCurrency)
}

func TestService_Create_DefaultCurrency(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.AddStage(&models.PipelineStage{ID: stageID, Name: "Lead"})

	input := CreateInput{
		Name:      "Deal",
		Currency:  "", // Empty, should default to EUR
		StageID:   stageID,
		CreatedBy: uuid.New(),
	}

	deal, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, "EUR", deal.Currency)
}

func TestService_Create_StageNotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := CreateInput{
		Name:      "Deal",
		StageID:   uuid.New(), // Non-existent
		CreatedBy: uuid.New(),
	}

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrStageNotFound)
}

func TestService_Create_ContactNotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.AddStage(&models.PipelineStage{ID: stageID, Name: "Lead"})

	contactID := uuid.New() // Non-existent

	input := CreateInput{
		Name:      "Deal",
		StageID:   stageID,
		ContactID: &contactID,
		CreatedBy: uuid.New(),
	}

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrContactNotFound)
}

func TestService_Create_CompanyNotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.AddStage(&models.PipelineStage{ID: stageID, Name: "Lead"})

	companyID := uuid.New() // Non-existent

	input := CreateInput{
		Name:      "Deal",
		StageID:   stageID,
		CompanyID: &companyID,
		CreatedBy: uuid.New(),
	}

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrCompanyNotFound)
}

func TestService_Create_OwnerNotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.AddStage(&models.PipelineStage{ID: stageID, Name: "Lead"})

	ownerID := uuid.New() // Non-existent

	input := CreateInput{
		Name:      "Deal",
		StageID:   stageID,
		OwnerID:   &ownerID,
		CreatedBy: uuid.New(),
	}

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrOwnerNotFound)
}

func TestService_Create_TagNotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.AddStage(&models.PipelineStage{ID: stageID, Name: "Lead"})

	input := CreateInput{
		Name:      "Deal",
		StageID:   stageID,
		TagIDs:    []uuid.UUID{uuid.New()}, // Non-existent
		CreatedBy: uuid.New(),
	}

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrTagNotFound)
}

// ============================================================================
// GetByID Tests
// ============================================================================

func TestService_GetByID_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.AddStage(&models.PipelineStage{ID: stageID, Name: "Lead"})

	dealID := uuid.New()
	repo.deals[dealID] = &models.Deal{
		ID:        dealID,
		Name:      "Test Deal",
		Value:     decimal.NewFromFloat(10000),
		Currency:  "EUR",
		StageID:   stageID,
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	deal, err := svc.GetByID(context.Background(), dealID)

	require.NoError(t, err)
	assert.Equal(t, dealID, deal.ID)
	assert.Equal(t, "Test Deal", deal.Name)
	assert.Equal(t, "Lead", deal.StageName)
}

func TestService_GetByID_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	_, err := svc.GetByID(context.Background(), uuid.New())

	assert.ErrorIs(t, err, ErrDealNotFound)
}

// ============================================================================
// Update Tests
// ============================================================================

func TestService_Update_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.AddStage(&models.PipelineStage{ID: stageID, Name: "Lead"})

	dealID := uuid.New()
	repo.deals[dealID] = &models.Deal{
		ID:        dealID,
		Name:      "Old Name",
		Value:     decimal.NewFromFloat(10000),
		Currency:  "EUR",
		StageID:   stageID,
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	newName := "New Name"
	newValue := 25000.00

	deal, err := svc.Update(context.Background(), dealID, UpdateInput{
		Name:  &newName,
		Value: &newValue,
	})

	require.NoError(t, err)
	assert.Equal(t, "New Name", deal.Name)
	assert.Equal(t, decimal.NewFromFloat(25000.00), deal.Value)
}

func TestService_Update_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	newName := "New Name"
	_, err := svc.Update(context.Background(), uuid.New(), UpdateInput{
		Name: &newName,
	})

	assert.ErrorIs(t, err, ErrDealNotFound)
}

func TestService_Update_EmptyName(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.AddStage(&models.PipelineStage{ID: stageID, Name: "Lead"})

	dealID := uuid.New()
	repo.deals[dealID] = &models.Deal{
		ID:        dealID,
		Name:      "Deal",
		StageID:   stageID,
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	emptyName := ""
	_, err := svc.Update(context.Background(), dealID, UpdateInput{
		Name: &emptyName,
	})

	assert.ErrorIs(t, err, ErrNameRequired)
}

func TestService_Update_ClearContact(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.AddStage(&models.PipelineStage{ID: stageID, Name: "Lead"})

	contactID := uuid.New()
	repo.AddContact(contactID, "John Doe")

	dealID := uuid.New()
	repo.deals[dealID] = &models.Deal{
		ID:        dealID,
		Name:      "Deal",
		StageID:   stageID,
		ContactID: &contactID,
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	nilContact := uuid.Nil
	deal, err := svc.Update(context.Background(), dealID, UpdateInput{
		ContactID: &nilContact,
	})

	require.NoError(t, err)
	assert.Nil(t, deal.ContactID)
}

// ============================================================================
// Delete Tests
// ============================================================================

func TestService_Delete_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	dealID := uuid.New()
	repo.deals[dealID] = &models.Deal{
		ID:        dealID,
		Name:      "Deal",
		StageID:   uuid.New(),
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := svc.Delete(context.Background(), dealID)

	require.NoError(t, err)
	assert.NotContains(t, repo.deals, dealID)
}

func TestService_Delete_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	err := svc.Delete(context.Background(), uuid.New())

	assert.ErrorIs(t, err, ErrDealNotFound)
}

// ============================================================================
// MoveToStage Tests
// ============================================================================

func TestService_MoveToStage_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	oldStageID := uuid.New()
	repo.AddStage(&models.PipelineStage{
		ID:   oldStageID,
		Name: "Lead",
	})

	newStageID := uuid.New()
	repo.AddStage(&models.PipelineStage{
		ID:   newStageID,
		Name: "Qualified",
	})

	dealID := uuid.New()
	repo.deals[dealID] = &models.Deal{
		ID:        dealID,
		Name:      "Deal",
		StageID:   oldStageID,
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	deal, err := svc.MoveToStage(context.Background(), dealID, newStageID)

	require.NoError(t, err)
	assert.Equal(t, newStageID, deal.StageID)
	assert.Equal(t, "Qualified", deal.StageName)
	assert.Nil(t, deal.ClosedAt)
}

func TestService_MoveToStage_ToWon_SetsClosedAt(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	leadStageID := uuid.New()
	repo.AddStage(&models.PipelineStage{ID: leadStageID, Name: "Lead"})

	wonStageID := uuid.New()
	repo.AddStage(&models.PipelineStage{
		ID:    wonStageID,
		Name:  "Won",
		IsWon: true,
	})

	dealID := uuid.New()
	repo.deals[dealID] = &models.Deal{
		ID:        dealID,
		Name:      "Deal",
		StageID:   leadStageID,
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	deal, err := svc.MoveToStage(context.Background(), dealID, wonStageID)

	require.NoError(t, err)
	assert.NotNil(t, deal.ClosedAt)
}

func TestService_MoveToStage_FromWonToOpen_ClearsClosedAt(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	leadStageID := uuid.New()
	repo.AddStage(&models.PipelineStage{ID: leadStageID, Name: "Lead"})

	wonStageID := uuid.New()
	repo.AddStage(&models.PipelineStage{
		ID:    wonStageID,
		Name:  "Won",
		IsWon: true,
	})

	closedAt := time.Now()
	dealID := uuid.New()
	repo.deals[dealID] = &models.Deal{
		ID:        dealID,
		Name:      "Deal",
		StageID:   wonStageID,
		ClosedAt:  &closedAt,
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	deal, err := svc.MoveToStage(context.Background(), dealID, leadStageID)

	require.NoError(t, err)
	assert.Nil(t, deal.ClosedAt)
}

func TestService_MoveToStage_StageNotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	leadStageID := uuid.New()
	repo.AddStage(&models.PipelineStage{ID: leadStageID, Name: "Lead"})

	dealID := uuid.New()
	repo.deals[dealID] = &models.Deal{
		ID:        dealID,
		Name:      "Deal",
		StageID:   leadStageID,
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err := svc.MoveToStage(context.Background(), dealID, uuid.New())

	assert.ErrorIs(t, err, ErrStageNotFound)
}

// ============================================================================
// Tags Tests
// ============================================================================

func TestService_AddTags_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.AddStage(&models.PipelineStage{ID: stageID, Name: "Lead"})

	dealID := uuid.New()
	repo.deals[dealID] = &models.Deal{
		ID:        dealID,
		Name:      "Deal",
		StageID:   stageID,
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tagID := uuid.New()
	repo.AddValidTag(tagID, models.EntityTypeDeal)

	deal, err := svc.AddTags(context.Background(), dealID, []uuid.UUID{tagID})

	require.NoError(t, err)
	assert.Equal(t, dealID, deal.ID)
}

func TestService_AddTags_TagNotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.AddStage(&models.PipelineStage{ID: stageID, Name: "Lead"})

	dealID := uuid.New()
	repo.deals[dealID] = &models.Deal{
		ID:        dealID,
		Name:      "Deal",
		StageID:   stageID,
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err := svc.AddTags(context.Background(), dealID, []uuid.UUID{uuid.New()})

	assert.ErrorIs(t, err, ErrTagNotFound)
}

func TestService_RemoveTags_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.AddStage(&models.PipelineStage{ID: stageID, Name: "Lead"})

	dealID := uuid.New()
	repo.deals[dealID] = &models.Deal{
		ID:        dealID,
		Name:      "Deal",
		StageID:   stageID,
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	deal, err := svc.RemoveTags(context.Background(), dealID, []uuid.UUID{uuid.New()})

	require.NoError(t, err)
	assert.Equal(t, dealID, deal.ID)
}
