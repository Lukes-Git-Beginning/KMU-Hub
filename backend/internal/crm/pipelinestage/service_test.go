package pipelinestage

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
	stages        map[uuid.UUID]*models.PipelineStage
	hasWon        bool
	hasLost       bool
	dealsPerStage map[uuid.UUID]int
	createErr     error
	getErr        error
	updateErr     error
	deleteErr     error
	reorderErr    error
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		stages:        make(map[uuid.UUID]*models.PipelineStage),
		dealsPerStage: make(map[uuid.UUID]int),
	}
}

func (m *MockRepository) Create(ctx context.Context, stage *models.PipelineStage) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.stages[stage.ID] = stage
	return nil
}

func (m *MockRepository) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*models.PipelineStage, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	stage, ok := m.stages[id]
	if !ok {
		return nil, ErrStageNotFound
	}
	return stage, nil
}

func (m *MockRepository) List(ctx context.Context, tenantID uuid.UUID) ([]*models.PipelineStage, error) {
	var result []*models.PipelineStage
	for _, s := range m.stages {
		result = append(result, s)
	}
	return result, nil
}

func (m *MockRepository) ListWithStats(ctx context.Context, tenantID uuid.UUID) ([]*models.PipelineStageWithStats, error) {
	var result []*models.PipelineStageWithStats
	for _, s := range m.stages {
		result = append(result, &models.PipelineStageWithStats{
			PipelineStage: *s,
			DealCount:     m.dealsPerStage[s.ID],
			TotalValue:    decimal.NewFromInt(int64(m.dealsPerStage[s.ID] * 1000)),
		})
	}
	return result, nil
}

func (m *MockRepository) Update(ctx context.Context, stage *models.PipelineStage) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.stages[stage.ID] = stage
	return nil
}

func (m *MockRepository) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.stages, id)
	return nil
}

func (m *MockRepository) Reorder(ctx context.Context, tenantID uuid.UUID, stageIDs []uuid.UUID) error {
	if m.reorderErr != nil {
		return m.reorderErr
	}
	for i, id := range stageIDs {
		if stage, ok := m.stages[id]; ok {
			stage.SortOrder = i + 1
		}
	}
	return nil
}

func (m *MockRepository) HasDeals(ctx context.Context, id, tenantID uuid.UUID) (bool, error) {
	return m.dealsPerStage[id] > 0, nil
}

func (m *MockRepository) HasWonStage(ctx context.Context, tenantID uuid.UUID, excludeID *uuid.UUID) (bool, error) {
	if !m.hasWon {
		return false, nil
	}
	// Check if we need to exclude a specific stage
	if excludeID != nil {
		for _, s := range m.stages {
			if s.IsWon && s.ID != *excludeID {
				return true, nil
			}
		}
		return false, nil
	}
	return true, nil
}

func (m *MockRepository) HasLostStage(ctx context.Context, tenantID uuid.UUID, excludeID *uuid.UUID) (bool, error) {
	if !m.hasLost {
		return false, nil
	}
	// Check if we need to exclude a specific stage
	if excludeID != nil {
		for _, s := range m.stages {
			if s.IsLost && s.ID != *excludeID {
				return true, nil
			}
		}
		return false, nil
	}
	return true, nil
}

func (m *MockRepository) GetNextSortOrder(ctx context.Context, tenantID uuid.UUID) (int, error) {
	return len(m.stages) + 1, nil
}

func (m *MockRepository) CountStages(ctx context.Context, tenantID uuid.UUID) (int, error) {
	return len(m.stages), nil
}

// Test helpers
func (m *MockRepository) SetHasWon(val bool) {
	m.hasWon = val
}

func (m *MockRepository) SetHasLost(val bool) {
	m.hasLost = val
}

func (m *MockRepository) SetDeals(stageID uuid.UUID, count int) {
	m.dealsPerStage[stageID] = count
}

// ============================================================================
// Create Tests
// ============================================================================

func TestService_Create_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := CreateInput{
		Name:        "New Lead",
		Color:       "#3b82f6",
		Probability: 10.0,
	}

	stage, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, stage.ID)
	assert.Equal(t, "New Lead", stage.Name)
	assert.Equal(t, "#3b82f6", stage.Color)
	assert.Equal(t, decimal.NewFromFloat(10.0), stage.Probability)
	assert.False(t, stage.IsWon)
	assert.False(t, stage.IsLost)
	assert.Equal(t, 1, stage.SortOrder)
	assert.NotZero(t, stage.CreatedAt)
	assert.NotZero(t, stage.UpdatedAt)
}

func TestService_Create_WithDefaultColor(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := CreateInput{
		Name:        "New Lead",
		Color:       "", // Empty, should use default
		Probability: 10.0,
	}

	stage, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, "#3b82f6", stage.Color)
}

func TestService_Create_WonStage(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := CreateInput{
		Name:        "Won",
		Color:       "#22c55e",
		IsWon:       true,
		Probability: 100.0,
	}

	stage, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.True(t, stage.IsWon)
	assert.False(t, stage.IsLost)
}

func TestService_Create_LostStage(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := CreateInput{
		Name:        "Lost",
		Color:       "#6b7280",
		IsLost:      true,
		Probability: 0.0,
	}

	stage, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.True(t, stage.IsLost)
	assert.False(t, stage.IsWon)
}

func TestService_Create_NameRequired(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := CreateInput{
		Name:  "",
		Color: "#3b82f6",
	}

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrNameRequired)
}

func TestService_Create_NameTrimmed(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := CreateInput{
		Name:  "  Qualified  ",
		Color: "#3b82f6",
	}

	stage, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, "Qualified", stage.Name)
}

func TestService_Create_InvalidColor(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := CreateInput{
		Name:  "Test",
		Color: "red", // Invalid format
	}

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrInvalidColor)
}

func TestService_Create_InvalidProbability_Negative(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := CreateInput{
		Name:        "Test",
		Color:       "#3b82f6",
		Probability: -10.0,
	}

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrInvalidProbability)
}

func TestService_Create_InvalidProbability_Over100(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := CreateInput{
		Name:        "Test",
		Color:       "#3b82f6",
		Probability: 150.0,
	}

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrInvalidProbability)
}

func TestService_Create_WonStageAlreadyExists(t *testing.T) {
	repo := NewMockRepository()
	repo.SetHasWon(true)
	svc := NewService(repo)

	input := CreateInput{
		Name:        "Another Won",
		Color:       "#22c55e",
		IsWon:       true,
		Probability: 100.0,
	}

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrWonStageExists)
}

func TestService_Create_LostStageAlreadyExists(t *testing.T) {
	repo := NewMockRepository()
	repo.SetHasLost(true)
	svc := NewService(repo)

	input := CreateInput{
		Name:        "Another Lost",
		Color:       "#6b7280",
		IsLost:      true,
		Probability: 0.0,
	}

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrLostStageExists)
}

// ============================================================================
// GetByID Tests
// ============================================================================

func TestService_GetByID_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.stages[stageID] = &models.PipelineStage{
		ID:          stageID,
		Name:        "Lead",
		Color:       "#3b82f6",
		SortOrder:   1,
		Probability: decimal.NewFromFloat(10.0),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	stage, err := svc.GetByID(context.Background(), stageID, uuid.Nil)

	require.NoError(t, err)
	assert.Equal(t, stageID, stage.ID)
	assert.Equal(t, "Lead", stage.Name)
}

func TestService_GetByID_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	_, err := svc.GetByID(context.Background(), uuid.New(), uuid.Nil)

	assert.ErrorIs(t, err, ErrStageNotFound)
}

// ============================================================================
// List Tests
// ============================================================================

func TestService_List_Empty(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stages, err := svc.List(context.Background(), uuid.Nil)

	require.NoError(t, err)
	assert.Empty(t, stages)
}

func TestService_List_WithStages(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	for i := 0; i < 3; i++ {
		id := uuid.New()
		repo.stages[id] = &models.PipelineStage{
			ID:          id,
			Name:        "Stage",
			Color:       "#3b82f6",
			SortOrder:   i + 1,
			Probability: decimal.NewFromFloat(float64((i + 1) * 10)),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
	}

	stages, err := svc.List(context.Background(), uuid.Nil)

	require.NoError(t, err)
	assert.Len(t, stages, 3)
}

func TestService_ListWithStats(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.stages[stageID] = &models.PipelineStage{
		ID:          stageID,
		Name:        "Lead",
		Color:       "#3b82f6",
		SortOrder:   1,
		Probability: decimal.NewFromFloat(10.0),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	repo.SetDeals(stageID, 5)

	stages, err := svc.ListWithStats(context.Background(), uuid.Nil)

	require.NoError(t, err)
	require.Len(t, stages, 1)
	assert.Equal(t, 5, stages[0].DealCount)
	assert.Equal(t, decimal.NewFromInt(5000), stages[0].TotalValue)
}

// ============================================================================
// Update Tests
// ============================================================================

func TestService_Update_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.stages[stageID] = &models.PipelineStage{
		ID:          stageID,
		Name:        "Old Name",
		Color:       "#3b82f6",
		SortOrder:   1,
		Probability: decimal.NewFromFloat(10.0),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	newName := "New Name"
	newColor := "#14b8a6"
	newProb := 25.0

	stage, err := svc.Update(context.Background(), stageID, uuid.Nil, UpdateInput{
		Name:        &newName,
		Color:       &newColor,
		Probability: &newProb,
	})

	require.NoError(t, err)
	assert.Equal(t, "New Name", stage.Name)
	assert.Equal(t, "#14b8a6", stage.Color)
	assert.Equal(t, decimal.NewFromFloat(25.0), stage.Probability)
}

func TestService_Update_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	newName := "New Name"
	_, err := svc.Update(context.Background(), uuid.New(), uuid.Nil, UpdateInput{
		Name: &newName,
	})

	assert.ErrorIs(t, err, ErrStageNotFound)
}

func TestService_Update_EmptyName(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.stages[stageID] = &models.PipelineStage{
		ID:        stageID,
		Name:      "Lead",
		Color:     "#3b82f6",
		SortOrder: 1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	emptyName := ""
	_, err := svc.Update(context.Background(), stageID, uuid.Nil, UpdateInput{
		Name: &emptyName,
	})

	assert.ErrorIs(t, err, ErrNameRequired)
}

func TestService_Update_InvalidColor(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.stages[stageID] = &models.PipelineStage{
		ID:        stageID,
		Name:      "Lead",
		Color:     "#3b82f6",
		SortOrder: 1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	invalidColor := "blue"
	_, err := svc.Update(context.Background(), stageID, uuid.Nil, UpdateInput{
		Color: &invalidColor,
	})

	assert.ErrorIs(t, err, ErrInvalidColor)
}

func TestService_Update_SetIsWon(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.stages[stageID] = &models.PipelineStage{
		ID:        stageID,
		Name:      "Stage",
		Color:     "#3b82f6",
		SortOrder: 1,
		IsWon:     false,
		IsLost:    false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	isWon := true
	stage, err := svc.Update(context.Background(), stageID, uuid.Nil, UpdateInput{
		IsWon: &isWon,
	})

	require.NoError(t, err)
	assert.True(t, stage.IsWon)
	assert.False(t, stage.IsLost)
}

func TestService_Update_SetIsWon_AlreadyExists(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	existingWonID := uuid.New()
	repo.stages[existingWonID] = &models.PipelineStage{
		ID:        existingWonID,
		Name:      "Won",
		Color:     "#22c55e",
		SortOrder: 5,
		IsWon:     true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.SetHasWon(true)

	stageID := uuid.New()
	repo.stages[stageID] = &models.PipelineStage{
		ID:        stageID,
		Name:      "Stage",
		Color:     "#3b82f6",
		SortOrder: 1,
		IsWon:     false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	isWon := true
	_, err := svc.Update(context.Background(), stageID, uuid.Nil, UpdateInput{
		IsWon: &isWon,
	})

	assert.ErrorIs(t, err, ErrWonStageExists)
}

func TestService_Update_SetIsLost(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.stages[stageID] = &models.PipelineStage{
		ID:        stageID,
		Name:      "Stage",
		Color:     "#6b7280",
		SortOrder: 1,
		IsWon:     false,
		IsLost:    false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	isLost := true
	stage, err := svc.Update(context.Background(), stageID, uuid.Nil, UpdateInput{
		IsLost: &isLost,
	})

	require.NoError(t, err)
	assert.True(t, stage.IsLost)
	assert.False(t, stage.IsWon)
}

func TestService_Update_ClearIsWon(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.stages[stageID] = &models.PipelineStage{
		ID:        stageID,
		Name:      "Won",
		Color:     "#22c55e",
		SortOrder: 5,
		IsWon:     true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.SetHasWon(true)

	isWon := false
	stage, err := svc.Update(context.Background(), stageID, uuid.Nil, UpdateInput{
		IsWon: &isWon,
	})

	require.NoError(t, err)
	assert.False(t, stage.IsWon)
}

// ============================================================================
// Delete Tests
// ============================================================================

func TestService_Delete_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.stages[stageID] = &models.PipelineStage{
		ID:        stageID,
		Name:      "Lead",
		Color:     "#3b82f6",
		SortOrder: 1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := svc.Delete(context.Background(), stageID, uuid.Nil)

	require.NoError(t, err)
	assert.NotContains(t, repo.stages, stageID)
}

func TestService_Delete_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	err := svc.Delete(context.Background(), uuid.New(), uuid.Nil)

	assert.ErrorIs(t, err, ErrStageNotFound)
}

func TestService_Delete_HasDeals(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	stageID := uuid.New()
	repo.stages[stageID] = &models.PipelineStage{
		ID:        stageID,
		Name:      "Lead",
		Color:     "#3b82f6",
		SortOrder: 1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.SetDeals(stageID, 3)

	err := svc.Delete(context.Background(), stageID, uuid.Nil)

	assert.ErrorIs(t, err, ErrStageHasDeals)
	assert.Contains(t, repo.stages, stageID) // Not deleted
}

// ============================================================================
// Reorder Tests
// ============================================================================

func TestService_Reorder_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	id1, id2, id3 := uuid.New(), uuid.New(), uuid.New()
	repo.stages[id1] = &models.PipelineStage{ID: id1, Name: "Lead", SortOrder: 1}
	repo.stages[id2] = &models.PipelineStage{ID: id2, Name: "Qualified", SortOrder: 2}
	repo.stages[id3] = &models.PipelineStage{ID: id3, Name: "Proposal", SortOrder: 3}

	// Reorder: move Qualified to first
	err := svc.Reorder(context.Background(), uuid.Nil, []uuid.UUID{id2, id1, id3})

	require.NoError(t, err)
	assert.Equal(t, 2, repo.stages[id1].SortOrder)
	assert.Equal(t, 1, repo.stages[id2].SortOrder)
	assert.Equal(t, 3, repo.stages[id3].SortOrder)
}

func TestService_Reorder_MissingStage(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	id1, id2 := uuid.New(), uuid.New()
	repo.stages[id1] = &models.PipelineStage{ID: id1, Name: "Lead", SortOrder: 1}
	repo.stages[id2] = &models.PipelineStage{ID: id2, Name: "Qualified", SortOrder: 2}

	// Missing id2
	err := svc.Reorder(context.Background(), uuid.Nil, []uuid.UUID{id1})

	assert.ErrorIs(t, err, ErrInvalidReorder)
}

func TestService_Reorder_DuplicateStage(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	id1, id2 := uuid.New(), uuid.New()
	repo.stages[id1] = &models.PipelineStage{ID: id1, Name: "Lead", SortOrder: 1}
	repo.stages[id2] = &models.PipelineStage{ID: id2, Name: "Qualified", SortOrder: 2}

	// Duplicate id1
	err := svc.Reorder(context.Background(), uuid.Nil, []uuid.UUID{id1, id1})

	assert.ErrorIs(t, err, ErrInvalidReorder)
}

func TestService_Reorder_InvalidStageID(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	id1 := uuid.New()
	repo.stages[id1] = &models.PipelineStage{ID: id1, Name: "Lead", SortOrder: 1}

	// Non-existent stage ID
	err := svc.Reorder(context.Background(), uuid.Nil, []uuid.UUID{uuid.New()})

	assert.ErrorIs(t, err, ErrStageNotFound)
}
