package deal

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// MockRepository implements Repository for testing
type MockRepository struct {
	deals        map[uuid.UUID]*models.Deal
	dealTags     map[uuid.UUID][]*models.Tag
	customFields map[uuid.UUID]map[uuid.UUID]any
	stages       map[uuid.UUID]*models.PipelineStage
	contacts     map[uuid.UUID]string // contactID -> name
	companies    map[uuid.UUID]string // companyID -> name
	owners       map[uuid.UUID]string // ownerID -> name
	validTags    map[uuid.UUID]models.EntityType
	managers     map[uuid.UUID]uuid.UUID // userID -> manager_user_id
	createErr    error
	getErr       error
	updateErr    error
	deleteErr    error
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
		managers:     make(map[uuid.UUID]uuid.UUID),
	}
}

func (m *MockRepository) Create(ctx context.Context, deal *models.Deal) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.deals[deal.ID] = deal
	return nil
}

func (m *MockRepository) GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.Deal, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	deal, ok := m.deals[id]
	if !ok {
		return nil, ErrDealNotFound
	}
	return deal, nil
}

func (m *MockRepository) List(ctx context.Context, tenantID uuid.UUID, filter ListFilter, offset, limit int) ([]*models.Deal, int, error) {
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

func (m *MockRepository) Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
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

func (m *MockRepository) StageExists(ctx context.Context, stageID, tenantID uuid.UUID) (bool, error) {
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

func (m *MockRepository) IsDirectReport(_ context.Context, _, managerID, userID uuid.UUID) (bool, error) {
	mgr, ok := m.managers[userID]
	return ok && mgr == managerID, nil
}

func (m *MockRepository) TagExists(ctx context.Context, tagID uuid.UUID, entityType models.EntityType) (bool, error) {
	et, exists := m.validTags[tagID]
	return exists && et == entityType, nil
}

func (m *MockRepository) SetClosedAt(ctx context.Context, dealID, tenantID uuid.UUID, closedAt *time.Time) error {
	if deal, ok := m.deals[dealID]; ok {
		deal.ClosedAt = closedAt
	}
	return nil
}

func (m *MockRepository) GetStageNames(ctx context.Context, stageIDs []uuid.UUID) (map[uuid.UUID]string, error) {
	result := make(map[uuid.UUID]string)
	for _, id := range stageIDs {
		if stage, ok := m.stages[id]; ok {
			result[id] = stage.Name
		}
	}
	return result, nil
}

func (m *MockRepository) GetContactNames(ctx context.Context, contactIDs []uuid.UUID) (map[uuid.UUID]string, error) {
	result := make(map[uuid.UUID]string)
	for _, id := range contactIDs {
		if name, ok := m.contacts[id]; ok {
			result[id] = name
		}
	}
	return result, nil
}

func (m *MockRepository) GetCompanyNames(ctx context.Context, companyIDs []uuid.UUID) (map[uuid.UUID]string, error) {
	result := make(map[uuid.UUID]string)
	for _, id := range companyIDs {
		if name, ok := m.companies[id]; ok {
			result[id] = name
		}
	}
	return result, nil
}

func (m *MockRepository) GetOwnerNames(ctx context.Context, ownerIDs []uuid.UUID) (map[uuid.UUID]string, error) {
	result := make(map[uuid.UUID]string)
	for _, id := range ownerIDs {
		if name, ok := m.owners[id]; ok {
			result[id] = name
		}
	}
	return result, nil
}

func (m *MockRepository) GetTagsBatch(ctx context.Context, dealIDs []uuid.UUID) (map[uuid.UUID][]*models.Tag, error) {
	result := make(map[uuid.UUID][]*models.Tag)
	for _, id := range dealIDs {
		if tags, ok := m.dealTags[id]; ok {
			result[id] = tags
		}
	}
	return result, nil
}

func (m *MockRepository) GetCustomFieldValuesBatch(ctx context.Context, dealIDs []uuid.UUID) (map[uuid.UUID][]*models.CustomFieldValueRow, error) {
	return nil, nil
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

	createdBy := uuid.New()
	input := CreateInput{
		Name:      "Big Deal",
		Value:     50000.00,
		Currency:  "EUR",
		StageID:   stageID,
		CreatedBy: createdBy,
	}

	deal, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, deal.ID)
	assert.Equal(t, "Big Deal", deal.Name)
	assert.Equal(t, decimal.NewFromFloat(50000.00), deal.Value)
	assert.Equal(t, "EUR", deal.Currency)
	assert.Equal(t, stageID, deal.StageID)
	assert.Equal(t, "Lead", deal.StageName)
	assert.Equal(t, createdBy, deal.CreatedBy)
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

	deal, err := svc.GetByID(context.Background(), uuid.Nil, dealID)

	require.NoError(t, err)
	assert.Equal(t, dealID, deal.ID)
	assert.Equal(t, "Test Deal", deal.Name)
	assert.Equal(t, "Lead", deal.StageName)
}

func TestService_GetByID_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	_, err := svc.GetByID(context.Background(), uuid.Nil, uuid.New())

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

	deal, err := svc.Update(context.Background(), uuid.Nil, dealID, UpdateInput{
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
	_, err := svc.Update(context.Background(), uuid.Nil, uuid.New(), UpdateInput{
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
	_, err := svc.Update(context.Background(), uuid.Nil, dealID, UpdateInput{
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
	deal, err := svc.Update(context.Background(), uuid.Nil, dealID, UpdateInput{
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

	err := svc.Delete(context.Background(), uuid.Nil, dealID)

	require.NoError(t, err)
	assert.NotContains(t, repo.deals, dealID)
}

func TestService_Delete_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	err := svc.Delete(context.Background(), uuid.Nil, uuid.New())

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

	deal, err := svc.MoveToStage(context.Background(), uuid.Nil, dealID, newStageID, nil)

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

	deal, err := svc.MoveToStage(context.Background(), uuid.Nil, dealID, wonStageID, nil)

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

	deal, err := svc.MoveToStage(context.Background(), uuid.Nil, dealID, leadStageID, nil)

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

	_, err := svc.MoveToStage(context.Background(), uuid.Nil, dealID, uuid.New(), nil)

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

	deal, err := svc.AddTags(context.Background(), uuid.Nil, dealID, []uuid.UUID{tagID}, nil)

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

	_, err := svc.AddTags(context.Background(), uuid.Nil, dealID, []uuid.UUID{uuid.New()}, nil)

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

	deal, err := svc.RemoveTags(context.Background(), uuid.Nil, dealID, []uuid.UUID{uuid.New()}, nil)

	require.NoError(t, err)
	assert.Equal(t, dealID, deal.ID)
}

func TestService_RemoveTags_DealNotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	_, err := svc.RemoveTags(context.Background(), uuid.New(), uuid.Nil, []uuid.UUID{uuid.New()}, nil)

	assert.ErrorIs(t, err, ErrDealNotFound)
}

// ============================================================================
// List Tests
// ============================================================================

func TestService_List_Empty(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	deals, total, err := svc.List(context.Background(), ListInput{})

	require.NoError(t, err)
	assert.Empty(t, deals)
	assert.Equal(t, 0, total)
}

func TestService_List_WithDeals(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.AddStage(&models.PipelineStage{ID: stageID, Name: "Lead"})

	for i := 0; i < 3; i++ {
		id := uuid.New()
		repo.deals[id] = &models.Deal{
			ID:        id,
			Name:      "Deal",
			StageID:   stageID,
			Currency:  "EUR",
			Value:     decimal.NewFromFloat(1000),
			CreatedBy: uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	deals, total, err := svc.List(context.Background(), ListInput{})

	require.NoError(t, err)
	assert.Len(t, deals, 3)
	assert.Equal(t, 3, total)
}

func TestService_List_DefaultPagination(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.AddStage(&models.PipelineStage{ID: stageID, Name: "Lead"})

	for i := 0; i < 25; i++ {
		id := uuid.New()
		repo.deals[id] = &models.Deal{
			ID:        id,
			Name:      "Deal",
			StageID:   stageID,
			Currency:  "EUR",
			Value:     decimal.NewFromFloat(1000),
			CreatedBy: uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	deals, total, err := svc.List(context.Background(), ListInput{
		Page:     0,
		PageSize: 0,
	})

	require.NoError(t, err)
	assert.Len(t, deals, 20)
	assert.Equal(t, 25, total)
}

func TestService_List_PageSizeUpToTwoHundred(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.AddStage(&models.PipelineStage{ID: stageID, Name: "Lead"})
	for i := 0; i < 150; i++ {
		id := uuid.New()
		repo.deals[id] = &models.Deal{
			ID:        id,
			Name:      "Deal",
			StageID:   stageID,
			Currency:  "EUR",
			Value:     decimal.NewFromFloat(1000),
			CreatedBy: uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	deals, total, err := svc.List(context.Background(), ListInput{PageSize: 200})
	require.NoError(t, err)
	assert.Len(t, deals, 150)
	assert.Equal(t, 150, total)

	// Above the ceiling the page falls back to the default of 20.
	deals, _, err = svc.List(context.Background(), ListInput{PageSize: 201})
	require.NoError(t, err)
	assert.Len(t, deals, 20)
}

// ============================================================================
// Create - Lost Stage Tests
// ============================================================================

func TestService_Create_LostStage_SetsClosedAt(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.AddStage(&models.PipelineStage{
		ID:     stageID,
		Name:   "Lost",
		IsLost: true,
	})

	deal, err := svc.Create(context.Background(), CreateInput{
		Name:      "Lost Deal",
		Value:     5000.00,
		StageID:   stageID,
		CreatedBy: uuid.New(),
	})

	require.NoError(t, err)
	assert.NotNil(t, deal.ClosedAt)
}

func TestService_Create_WithNotes(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.AddStage(&models.PipelineStage{ID: stageID, Name: "Lead"})

	notes := "Important deal notes"
	deal, err := svc.Create(context.Background(), CreateInput{
		Name:      "Deal",
		StageID:   stageID,
		Notes:     &notes,
		CreatedBy: uuid.New(),
	})

	require.NoError(t, err)
	assert.Equal(t, "Important deal notes", *deal.Notes)
}

func TestService_Create_WithCustomFields(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.AddStage(&models.PipelineStage{ID: stageID, Name: "Lead"})

	fieldID := uuid.New()
	deal, err := svc.Create(context.Background(), CreateInput{
		Name:         "Deal",
		StageID:      stageID,
		CustomFields: map[uuid.UUID]any{fieldID: "custom value"},
		CreatedBy:    uuid.New(),
	})

	require.NoError(t, err)
	assert.NotNil(t, deal)
	assert.Contains(t, repo.customFields[deal.ID], fieldID)
}

func TestService_Create_RepoError(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.AddStage(&models.PipelineStage{ID: stageID, Name: "Lead"})
	repo.createErr = errors.New("db error")

	_, err := svc.Create(context.Background(), CreateInput{
		Name:      "Deal",
		StageID:   stageID,
		CreatedBy: uuid.New(),
	})

	assert.Error(t, err)
}

// ============================================================================
// Update - Additional Tests
// ============================================================================

func TestService_Update_Currency(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.AddStage(&models.PipelineStage{ID: stageID, Name: "Lead"})

	dealID := uuid.New()
	repo.deals[dealID] = &models.Deal{
		ID:        dealID,
		Name:      "Deal",
		Currency:  "EUR",
		StageID:   stageID,
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	currency := "USD"
	deal, err := svc.Update(context.Background(), uuid.Nil, dealID, UpdateInput{
		Currency: &currency,
	})

	require.NoError(t, err)
	assert.Equal(t, "USD", deal.Currency)
}

func TestService_Update_InvalidCurrency(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.AddStage(&models.PipelineStage{ID: stageID, Name: "Lead"})

	dealID := uuid.New()
	repo.deals[dealID] = &models.Deal{
		ID:        dealID,
		Name:      "Deal",
		Currency:  "EUR",
		StageID:   stageID,
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	currency := "INVALID"
	_, err := svc.Update(context.Background(), uuid.Nil, dealID, UpdateInput{
		Currency: &currency,
	})

	assert.ErrorIs(t, err, ErrInvalidCurrency)
}

func TestService_Update_ClearCompany(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.AddStage(&models.PipelineStage{ID: stageID, Name: "Lead"})

	companyID := uuid.New()
	repo.AddCompany(companyID, "Acme Corp")

	dealID := uuid.New()
	repo.deals[dealID] = &models.Deal{
		ID:        dealID,
		Name:      "Deal",
		StageID:   stageID,
		CompanyID: &companyID,
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	nilCompany := uuid.Nil
	deal, err := svc.Update(context.Background(), uuid.Nil, dealID, UpdateInput{
		CompanyID: &nilCompany,
	})

	require.NoError(t, err)
	assert.Nil(t, deal.CompanyID)
}

func TestService_Update_SetCompany(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.AddStage(&models.PipelineStage{ID: stageID, Name: "Lead"})

	companyID := uuid.New()
	repo.AddCompany(companyID, "New Corp")

	dealID := uuid.New()
	repo.deals[dealID] = &models.Deal{
		ID:        dealID,
		Name:      "Deal",
		StageID:   stageID,
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	deal, err := svc.Update(context.Background(), uuid.Nil, dealID, UpdateInput{
		CompanyID: &companyID,
	})

	require.NoError(t, err)
	assert.Equal(t, companyID, *deal.CompanyID)
}

func TestService_Update_CompanyNotFound(t *testing.T) {
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

	badCompanyID := uuid.New()
	_, err := svc.Update(context.Background(), uuid.Nil, dealID, UpdateInput{
		CompanyID: &badCompanyID,
	})

	assert.ErrorIs(t, err, ErrCompanyNotFound)
}

func TestService_Update_ClearOwner(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.AddStage(&models.PipelineStage{ID: stageID, Name: "Lead"})

	ownerID := uuid.New()
	repo.AddOwner(ownerID, "Sales Rep")

	dealID := uuid.New()
	repo.deals[dealID] = &models.Deal{
		ID:        dealID,
		Name:      "Deal",
		StageID:   stageID,
		OwnerID:   &ownerID,
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	nilOwner := uuid.Nil
	deal, err := svc.Update(context.Background(), uuid.Nil, dealID, UpdateInput{
		OwnerID: &nilOwner,
	})

	require.NoError(t, err)
	assert.Nil(t, deal.OwnerID)
}

func TestService_Update_SetOwner(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.AddStage(&models.PipelineStage{ID: stageID, Name: "Lead"})

	ownerID := uuid.New()
	repo.AddOwner(ownerID, "New Owner")

	dealID := uuid.New()
	repo.deals[dealID] = &models.Deal{
		ID:        dealID,
		Name:      "Deal",
		StageID:   stageID,
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	deal, err := svc.Update(context.Background(), uuid.Nil, dealID, UpdateInput{
		OwnerID: &ownerID,
	})

	require.NoError(t, err)
	assert.Equal(t, ownerID, *deal.OwnerID)
}

func TestService_Update_OwnerNotFound(t *testing.T) {
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

	badOwnerID := uuid.New()
	_, err := svc.Update(context.Background(), uuid.Nil, dealID, UpdateInput{
		OwnerID: &badOwnerID,
	})

	assert.ErrorIs(t, err, ErrOwnerNotFound)
}

func TestService_Update_ContactNotFound(t *testing.T) {
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

	badContactID := uuid.New()
	_, err := svc.Update(context.Background(), uuid.Nil, dealID, UpdateInput{
		ContactID: &badContactID,
	})

	assert.ErrorIs(t, err, ErrContactNotFound)
}

func TestService_Update_SetContact(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.AddStage(&models.PipelineStage{ID: stageID, Name: "Lead"})

	contactID := uuid.New()
	repo.AddContact(contactID, "Jane Doe")

	dealID := uuid.New()
	repo.deals[dealID] = &models.Deal{
		ID:        dealID,
		Name:      "Deal",
		StageID:   stageID,
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	deal, err := svc.Update(context.Background(), uuid.Nil, dealID, UpdateInput{
		ContactID: &contactID,
	})

	require.NoError(t, err)
	assert.Equal(t, contactID, *deal.ContactID)
}

func TestService_Update_ExpectedCloseDate(t *testing.T) {
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

	closeDate := time.Now().AddDate(0, 1, 0)
	deal, err := svc.Update(context.Background(), uuid.Nil, dealID, UpdateInput{
		ExpectedCloseDate: &closeDate,
	})

	require.NoError(t, err)
	assert.NotNil(t, deal.ExpectedCloseDate)
}

func TestService_Update_Notes(t *testing.T) {
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

	notes := "Updated notes"
	deal, err := svc.Update(context.Background(), uuid.Nil, dealID, UpdateInput{
		Notes: &notes,
	})

	require.NoError(t, err)
	assert.Equal(t, "Updated notes", *deal.Notes)
}

func TestService_Update_CustomFields(t *testing.T) {
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

	fieldID := uuid.New()
	deal, err := svc.Update(context.Background(), uuid.Nil, dealID, UpdateInput{
		CustomFields: map[uuid.UUID]any{fieldID: "value"},
	})

	require.NoError(t, err)
	assert.NotNil(t, deal)
	assert.Contains(t, repo.customFields[dealID], fieldID)
}

func TestService_Update_RepoError(t *testing.T) {
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

	repo.updateErr = errors.New("db error")
	newName := "Updated"
	_, err := svc.Update(context.Background(), uuid.Nil, dealID, UpdateInput{
		Name: &newName,
	})

	assert.Error(t, err)
}

func TestService_Delete_RepoError(t *testing.T) {
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

	repo.deleteErr = errors.New("db error")
	err := svc.Delete(context.Background(), uuid.Nil, dealID)

	assert.Error(t, err)
}

// ============================================================================
// MoveToStage - Additional Tests
// ============================================================================

func TestService_MoveToStage_ToLost_SetsClosedAt(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	leadStageID := uuid.New()
	repo.AddStage(&models.PipelineStage{ID: leadStageID, Name: "Lead"})

	lostStageID := uuid.New()
	repo.AddStage(&models.PipelineStage{
		ID:     lostStageID,
		Name:   "Lost",
		IsLost: true,
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

	deal, err := svc.MoveToStage(context.Background(), uuid.Nil, dealID, lostStageID, nil)

	require.NoError(t, err)
	assert.NotNil(t, deal.ClosedAt)
}

func TestService_MoveToStage_DealNotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.AddStage(&models.PipelineStage{ID: stageID, Name: "Lead"})

	_, err := svc.MoveToStage(context.Background(), uuid.Nil, uuid.New(), stageID, nil)

	assert.ErrorIs(t, err, ErrDealNotFound)
}

// ============================================================================
// EventEmitter Tests
// ============================================================================

type mockEventEmitter struct {
	events []models.EventPayload
}

func (m *mockEventEmitter) EmitDealEvent(_ context.Context, payload models.EventPayload) error {
	m.events = append(m.events, payload)
	return nil
}

func TestService_SetEventEmitter(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	emitter := &mockEventEmitter{}
	svc.SetEventEmitter(emitter)

	assert.NotNil(t, svc.eventEmitter)
}

func TestService_Update_OwnerChange_EmitsEvent(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	emitter := &mockEventEmitter{}
	svc.SetEventEmitter(emitter)

	stageID := uuid.New()
	repo.AddStage(&models.PipelineStage{ID: stageID, Name: "Lead"})

	ownerID := uuid.New()
	repo.AddOwner(ownerID, "New Owner")

	dealID := uuid.New()
	repo.deals[dealID] = &models.Deal{
		ID:        dealID,
		Name:      "Deal",
		StageID:   stageID,
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err := svc.Update(context.Background(), uuid.Nil, dealID, UpdateInput{
		OwnerID: &ownerID,
	})

	require.NoError(t, err)
	require.Len(t, emitter.events, 1)
	assert.Equal(t, "crm.deal.assigned", emitter.events[0].Type)
	assert.Contains(t, emitter.events[0].TargetUserIDs, ownerID.String())
}

func TestService_MoveToStage_EmitsEvent(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	emitter := &mockEventEmitter{}
	svc.SetEventEmitter(emitter)

	leadStageID := uuid.New()
	repo.AddStage(&models.PipelineStage{ID: leadStageID, Name: "Lead"})

	wonStageID := uuid.New()
	repo.AddStage(&models.PipelineStage{ID: wonStageID, Name: "Won", IsWon: true})

	ownerID := uuid.New()
	repo.AddOwner(ownerID, "Sales Rep")

	dealID := uuid.New()
	repo.deals[dealID] = &models.Deal{
		ID:        dealID,
		Name:      "Big Deal",
		StageID:   leadStageID,
		OwnerID:   &ownerID,
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err := svc.MoveToStage(context.Background(), uuid.Nil, dealID, wonStageID, nil)

	require.NoError(t, err)
	require.Len(t, emitter.events, 1)
	assert.Equal(t, "crm.deal.stage_changed", emitter.events[0].Type)
	assert.Contains(t, emitter.events[0].Title, "Won")
}

// ============================================================================
// AddTags - Additional Tests
// ============================================================================

func TestService_AddTags_DealNotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	tagID := uuid.New()
	repo.AddValidTag(tagID, models.EntityTypeDeal)

	_, err := svc.AddTags(context.Background(), uuid.Nil, uuid.New(), []uuid.UUID{tagID}, nil)

	assert.ErrorIs(t, err, ErrDealNotFound)
}

// ============================================================================
// List with Relations Tests (enrichWithRelationsBatch)
// ============================================================================

func TestService_List_WithRelations(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.AddStage(&models.PipelineStage{ID: stageID, Name: "Proposal"})

	contactID := uuid.New()
	repo.AddContact(contactID, "Jane Doe")

	companyID := uuid.New()
	repo.AddCompany(companyID, "Acme Corp")

	ownerID := uuid.New()
	repo.AddOwner(ownerID, "Sales Rep")

	dealID := uuid.New()
	repo.deals[dealID] = &models.Deal{
		ID:        dealID,
		Name:      "Full Deal",
		StageID:   stageID,
		ContactID: &contactID,
		CompanyID: &companyID,
		OwnerID:   &ownerID,
		Currency:  "EUR",
		Value:     decimal.NewFromFloat(50000),
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tagID := uuid.New()
	repo.dealTags[dealID] = []*models.Tag{{ID: tagID, Name: "VIP"}}

	deals, total, err := svc.List(context.Background(), ListInput{Page: 1, PageSize: 10})

	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, deals, 1)
	assert.Equal(t, "Proposal", deals[0].StageName)
	assert.Equal(t, "Jane Doe", *deals[0].ContactName)
	assert.Equal(t, "Acme Corp", *deals[0].CompanyName)
	assert.Equal(t, "Sales Rep", *deals[0].OwnerName)
	assert.Len(t, deals[0].Tags, 1)
}

func TestService_List_Pagination(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.AddStage(&models.PipelineStage{ID: stageID, Name: "Lead"})

	for i := 0; i < 5; i++ {
		id := uuid.New()
		repo.deals[id] = &models.Deal{
			ID:        id,
			Name:      "Deal",
			StageID:   stageID,
			Currency:  "EUR",
			Value:     decimal.NewFromFloat(1000),
			CreatedBy: uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	deals, total, err := svc.List(context.Background(), ListInput{Page: 2, PageSize: 3})

	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, deals, 2)
}

// ============================================================================
// GetByID with Relations
// ============================================================================

func TestService_GetByID_WithAllRelations(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.AddStage(&models.PipelineStage{ID: stageID, Name: "Qualified"})

	contactID := uuid.New()
	repo.AddContact(contactID, "John Doe")

	companyID := uuid.New()
	repo.AddCompany(companyID, "Test Corp")

	ownerID := uuid.New()
	repo.AddOwner(ownerID, "Owner")

	dealID := uuid.New()
	repo.deals[dealID] = &models.Deal{
		ID:        dealID,
		Name:      "Deal",
		StageID:   stageID,
		ContactID: &contactID,
		CompanyID: &companyID,
		OwnerID:   &ownerID,
		Currency:  "EUR",
		Value:     decimal.NewFromFloat(1000),
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	deal, err := svc.GetByID(context.Background(), uuid.Nil, dealID)

	require.NoError(t, err)
	assert.Equal(t, "Qualified", deal.StageName)
	assert.Equal(t, "John Doe", *deal.ContactName)
	assert.Equal(t, "Test Corp", *deal.CompanyName)
	assert.Equal(t, "Owner", *deal.OwnerName)
}

// TestService_EditorScope pins the team boundary of crm:deal:create/edit: a
// scoped editor writes deals owned by themselves or a direct report, nothing else.
func TestService_EditorScope(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	lead, report, peer := uuid.New(), uuid.New(), uuid.New()

	setup := func() (*Service, *MockRepository, uuid.UUID) {
		repo := NewMockRepository()
		svc := NewService(repo)
		stageID := uuid.New()
		repo.AddStage(&models.PipelineStage{ID: stageID, Name: "Lead"})
		repo.owners[lead], repo.owners[report], repo.owners[peer] = "Lead", "Report", "Peer"
		repo.managers[report] = lead
		return svc, repo, stageID
	}
	addDeal := func(repo *MockRepository, stageID, createdBy uuid.UUID, owner *uuid.UUID) uuid.UUID {
		id := uuid.New()
		repo.deals[id] = &models.Deal{ID: id, Name: "Deal", StageID: stageID, CreatedBy: createdBy, OwnerID: owner, Currency: "EUR"}
		return id
	}
	team := &Editor{UserID: lead, Team: true}
	own := &Editor{UserID: lead}

	t.Run("team editor writes own deals and deals of a direct report", func(t *testing.T) {
		svc, repo, stageID := setup()
		mine := addDeal(repo, stageID, lead, nil)
		reports := addDeal(repo, stageID, peer, &report)
		for _, id := range []uuid.UUID{mine, reports} {
			name := "renamed"
			_, err := svc.Update(ctx, tenantID, id, UpdateInput{Name: &name, Editor: team})
			require.NoError(t, err)
			_, err = svc.MoveToStage(ctx, tenantID, id, stageID, team)
			require.NoError(t, err)
			_, err = svc.AddTags(ctx, tenantID, id, nil, team)
			require.NoError(t, err)
			_, err = svc.RemoveTags(ctx, tenantID, id, nil, team)
			require.NoError(t, err)
		}
	})

	t.Run("deal of a peer is outside the team for every write", func(t *testing.T) {
		svc, repo, stageID := setup()
		// Created by the lead but owned by a peer: the owner decides, not the creator.
		foreign := addDeal(repo, stageID, lead, &peer)
		name := "renamed"
		_, err := svc.Update(ctx, tenantID, foreign, UpdateInput{Name: &name, Editor: team})
		assert.ErrorIs(t, err, ErrOutsideEditorScope)
		_, err = svc.MoveToStage(ctx, tenantID, foreign, stageID, team)
		assert.ErrorIs(t, err, ErrOutsideEditorScope)
		_, err = svc.AddTags(ctx, tenantID, foreign, nil, team)
		assert.ErrorIs(t, err, ErrOutsideEditorScope)
		_, err = svc.RemoveTags(ctx, tenantID, foreign, nil, team)
		assert.ErrorIs(t, err, ErrOutsideEditorScope)
		assert.Equal(t, "Deal", repo.deals[foreign].Name)
		// Unscoped (manager, admin) still writes it.
		_, err = svc.Update(ctx, tenantID, foreign, UpdateInput{Name: &name})
		require.NoError(t, err)
	})

	t.Run("own editor does not reach a direct report", func(t *testing.T) {
		svc, repo, stageID := setup()
		reports := addDeal(repo, stageID, report, nil)
		_, err := svc.MoveToStage(ctx, tenantID, reports, stageID, own)
		assert.ErrorIs(t, err, ErrOutsideEditorScope)
	})

	t.Run("team editor cannot hand a deal out of the team", func(t *testing.T) {
		svc, repo, stageID := setup()
		mine := addDeal(repo, stageID, lead, nil)
		_, err := svc.Update(ctx, tenantID, mine, UpdateInput{OwnerID: &peer, Editor: team})
		assert.ErrorIs(t, err, ErrOutsideEditorScope)
		// The mock hands out its stored struct, so the refused write above left it
		// mutated; the handover to a report gets a deal of its own.
		other := addDeal(repo, stageID, lead, nil)
		_, err = svc.Update(ctx, tenantID, other, UpdateInput{OwnerID: &report, Editor: team})
		require.NoError(t, err)
	})

	t.Run("team editor creates for themselves or a report, not for a peer", func(t *testing.T) {
		svc, _, stageID := setup()
		_, err := svc.Create(ctx, CreateInput{TenantID: tenantID, Name: "A", StageID: stageID, CreatedBy: lead, Editor: team})
		require.NoError(t, err)
		_, err = svc.Create(ctx, CreateInput{TenantID: tenantID, Name: "B", StageID: stageID, CreatedBy: lead, OwnerID: &report, Editor: team})
		require.NoError(t, err)
		_, err = svc.Create(ctx, CreateInput{TenantID: tenantID, Name: "C", StageID: stageID, CreatedBy: lead, OwnerID: &peer, Editor: team})
		assert.ErrorIs(t, err, ErrOutsideEditorScope)
	})
}
