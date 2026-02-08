package status

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// MockRepository implements Repository for testing
type MockRepository struct {
	statuses   map[uuid.UUID]*models.ProjectStatus
	taskCounts map[uuid.UUID]int // statusID -> task count
	sortOrders map[uuid.UUID]int // projectID -> next sort order

	createErr error
	getErr    error
	updateErr error
	deleteErr error
	copyErr   error
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		statuses:   make(map[uuid.UUID]*models.ProjectStatus),
		taskCounts: make(map[uuid.UUID]int),
		sortOrders: make(map[uuid.UUID]int),
	}
}

func (m *MockRepository) Create(ctx context.Context, status *models.ProjectStatus) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.statuses[status.ID] = status
	// Track sort order
	if status.SortOrder >= m.sortOrders[status.ProjectID] {
		m.sortOrders[status.ProjectID] = status.SortOrder + 1
	}
	return nil
}

func (m *MockRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.ProjectStatus, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	s, ok := m.statuses[id]
	if !ok {
		return nil, ErrNotFound
	}
	return s, nil
}

func (m *MockRepository) ListByProject(ctx context.Context, projectID uuid.UUID) ([]models.ProjectStatus, error) {
	var result []models.ProjectStatus
	for _, s := range m.statuses {
		if s.ProjectID == projectID {
			result = append(result, *s)
		}
	}
	return result, nil
}

func (m *MockRepository) Update(ctx context.Context, status *models.ProjectStatus) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.statuses[status.ID] = status
	return nil
}

func (m *MockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.statuses, id)
	return nil
}

func (m *MockRepository) Reorder(ctx context.Context, projectID uuid.UUID, statusIDs []uuid.UUID) error {
	for i, id := range statusIDs {
		if s, ok := m.statuses[id]; ok {
			s.SortOrder = i
		}
	}
	return nil
}

func (m *MockRepository) CountTasksWithStatus(ctx context.Context, statusID uuid.UUID) (int, error) {
	return m.taskCounts[statusID], nil
}

func (m *MockRepository) HasDefault(ctx context.Context, projectID uuid.UUID) (bool, error) {
	for _, s := range m.statuses {
		if s.ProjectID == projectID && s.IsDefault {
			return true, nil
		}
	}
	return false, nil
}

func (m *MockRepository) GetNextSortOrder(ctx context.Context, projectID uuid.UUID) (int, error) {
	return m.sortOrders[projectID], nil
}

func (m *MockRepository) NameExistsInProject(ctx context.Context, projectID uuid.UUID, name string, excludeID *uuid.UUID) (bool, error) {
	for _, s := range m.statuses {
		if s.ProjectID == projectID && strings.EqualFold(s.Name, name) {
			if excludeID != nil && s.ID == *excludeID {
				continue
			}
			return true, nil
		}
	}
	return false, nil
}

func (m *MockRepository) CopyStatusesForProject(ctx context.Context, sourceProjectID, targetProjectID uuid.UUID) error {
	if m.copyErr != nil {
		return m.copyErr
	}
	for _, s := range m.statuses {
		if s.ProjectID == sourceProjectID {
			newStatus := &models.ProjectStatus{
				ID:        uuid.New(),
				ProjectID: targetProjectID,
				Name:      s.Name,
				Color:     s.Color,
				SortOrder: s.SortOrder,
				IsDefault: s.IsDefault,
				IsClosed:  s.IsClosed,
				CreatedAt: time.Now(),
			}
			m.statuses[newStatus.ID] = newStatus
		}
	}
	return nil
}

// Test helper to add a status directly
func (m *MockRepository) addStatus(s *models.ProjectStatus) {
	m.statuses[s.ID] = s
	if s.SortOrder >= m.sortOrders[s.ProjectID] {
		m.sortOrders[s.ProjectID] = s.SortOrder + 1
	}
}

// ============================================================================
// Create Tests
// ============================================================================

func TestService_Create_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	status, err := svc.Create(context.Background(), CreateInput{
		ProjectID: projectID,
		Name:      "To Do",
		IsDefault: true,
	})

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, status.ID)
	assert.Equal(t, projectID, status.ProjectID)
	assert.Equal(t, "To Do", status.Name)
	assert.Equal(t, 0, status.SortOrder)
	assert.True(t, status.IsDefault)
	assert.False(t, status.IsClosed)
}

func TestService_Create_WithColor(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	color := "#3b82f6"
	status, err := svc.Create(context.Background(), CreateInput{
		ProjectID: uuid.New(),
		Name:      "In Progress",
		Color:     &color,
	})

	require.NoError(t, err)
	require.NotNil(t, status.Color)
	assert.Equal(t, "#3b82f6", *status.Color)
}

func TestService_Create_AutoSortOrder(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	projectID := uuid.New()

	// Create first status
	s1, err := svc.Create(context.Background(), CreateInput{
		ProjectID: projectID,
		Name:      "First",
	})
	require.NoError(t, err)
	assert.Equal(t, 0, s1.SortOrder)

	// Create second status
	s2, err := svc.Create(context.Background(), CreateInput{
		ProjectID: projectID,
		Name:      "Second",
	})
	require.NoError(t, err)
	assert.Equal(t, 1, s2.SortOrder)
}

func TestService_Create_EmptyName(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	_, err := svc.Create(context.Background(), CreateInput{
		ProjectID: uuid.New(),
		Name:      "",
	})

	assert.ErrorIs(t, err, ErrNameRequired)
}

func TestService_Create_WhitespaceName(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	_, err := svc.Create(context.Background(), CreateInput{
		ProjectID: uuid.New(),
		Name:      "   ",
	})

	assert.ErrorIs(t, err, ErrNameRequired)
}

func TestService_Create_NameTooLong(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	longName := strings.Repeat("a", 101)
	_, err := svc.Create(context.Background(), CreateInput{
		ProjectID: uuid.New(),
		Name:      longName,
	})

	assert.ErrorIs(t, err, ErrNameTooLong)
}

func TestService_Create_DuplicateName(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	projectID := uuid.New()

	// Create first status
	_, err := svc.Create(context.Background(), CreateInput{
		ProjectID: projectID,
		Name:      "To Do",
	})
	require.NoError(t, err)

	// Try duplicate (same case)
	_, err = svc.Create(context.Background(), CreateInput{
		ProjectID: projectID,
		Name:      "To Do",
	})
	assert.ErrorIs(t, err, ErrDuplicateName)
}

func TestService_Create_DuplicateName_CaseInsensitive(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	projectID := uuid.New()

	// Create first status
	_, err := svc.Create(context.Background(), CreateInput{
		ProjectID: projectID,
		Name:      "To Do",
	})
	require.NoError(t, err)

	// Try duplicate (different case)
	_, err = svc.Create(context.Background(), CreateInput{
		ProjectID: projectID,
		Name:      "to do",
	})
	assert.ErrorIs(t, err, ErrDuplicateName)
}

func TestService_Create_SameNameDifferentProject(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	// Same name in different projects should be fine
	_, err := svc.Create(context.Background(), CreateInput{
		ProjectID: uuid.New(),
		Name:      "To Do",
	})
	require.NoError(t, err)

	_, err = svc.Create(context.Background(), CreateInput{
		ProjectID: uuid.New(),
		Name:      "To Do",
	})
	require.NoError(t, err)
}

// ============================================================================
// GetByID Tests
// ============================================================================

func TestService_GetByID_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	statusID := uuid.New()
	repo.addStatus(&models.ProjectStatus{
		ID:        statusID,
		ProjectID: uuid.New(),
		Name:      "In Progress",
		SortOrder: 1,
		CreatedAt: time.Now(),
	})

	status, err := svc.GetByID(context.Background(), statusID)

	require.NoError(t, err)
	assert.Equal(t, "In Progress", status.Name)
}

func TestService_GetByID_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	_, err := svc.GetByID(context.Background(), uuid.New())

	assert.ErrorIs(t, err, ErrNotFound)
}

// ============================================================================
// ListByProject Tests
// ============================================================================

func TestService_ListByProject(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	otherProjectID := uuid.New()

	repo.addStatus(&models.ProjectStatus{
		ID: uuid.New(), ProjectID: projectID, Name: "To Do", SortOrder: 0, CreatedAt: time.Now(),
	})
	repo.addStatus(&models.ProjectStatus{
		ID: uuid.New(), ProjectID: projectID, Name: "Done", SortOrder: 1, CreatedAt: time.Now(),
	})
	repo.addStatus(&models.ProjectStatus{
		ID: uuid.New(), ProjectID: otherProjectID, Name: "Other", SortOrder: 0, CreatedAt: time.Now(),
	})

	statuses, err := svc.ListByProject(context.Background(), projectID)

	require.NoError(t, err)
	assert.Len(t, statuses, 2)
}

// ============================================================================
// Update Tests
// ============================================================================

func TestService_Update_Name(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	statusID := uuid.New()
	repo.addStatus(&models.ProjectStatus{
		ID:        statusID,
		ProjectID: uuid.New(),
		Name:      "Old Name",
		SortOrder: 0,
		CreatedAt: time.Now(),
	})

	newName := "New Name"
	status, err := svc.Update(context.Background(), statusID, UpdateInput{
		Name: &newName,
	})

	require.NoError(t, err)
	assert.Equal(t, "New Name", status.Name)
}

func TestService_Update_Color(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	statusID := uuid.New()
	repo.addStatus(&models.ProjectStatus{
		ID:        statusID,
		ProjectID: uuid.New(),
		Name:      "Status",
		SortOrder: 0,
		CreatedAt: time.Now(),
	})

	newColor := "#ff0000"
	status, err := svc.Update(context.Background(), statusID, UpdateInput{
		Color: &newColor,
	})

	require.NoError(t, err)
	require.NotNil(t, status.Color)
	assert.Equal(t, "#ff0000", *status.Color)
}

func TestService_Update_EmptyName(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	statusID := uuid.New()
	repo.addStatus(&models.ProjectStatus{
		ID:        statusID,
		ProjectID: uuid.New(),
		Name:      "Status",
		SortOrder: 0,
		CreatedAt: time.Now(),
	})

	emptyName := ""
	_, err := svc.Update(context.Background(), statusID, UpdateInput{
		Name: &emptyName,
	})

	assert.ErrorIs(t, err, ErrNameRequired)
}

func TestService_Update_DuplicateName(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	status1ID := uuid.New()
	status2ID := uuid.New()

	repo.addStatus(&models.ProjectStatus{
		ID:        status1ID,
		ProjectID: projectID,
		Name:      "To Do",
		SortOrder: 0,
		CreatedAt: time.Now(),
	})
	repo.addStatus(&models.ProjectStatus{
		ID:        status2ID,
		ProjectID: projectID,
		Name:      "Done",
		SortOrder: 1,
		CreatedAt: time.Now(),
	})

	// Rename "Done" to "To Do" -- should fail
	dupName := "To Do"
	_, err := svc.Update(context.Background(), status2ID, UpdateInput{
		Name: &dupName,
	})

	assert.ErrorIs(t, err, ErrDuplicateName)
}

func TestService_Update_SameName_SameStatus_OK(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	statusID := uuid.New()
	repo.addStatus(&models.ProjectStatus{
		ID:        statusID,
		ProjectID: uuid.New(),
		Name:      "To Do",
		SortOrder: 0,
		CreatedAt: time.Now(),
	})

	// Rename "To Do" to "To Do" (no actual change) -- should succeed
	sameName := "To Do"
	status, err := svc.Update(context.Background(), statusID, UpdateInput{
		Name: &sameName,
	})

	require.NoError(t, err)
	assert.Equal(t, "To Do", status.Name)
}

func TestService_Update_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	newName := "Updated"
	_, err := svc.Update(context.Background(), uuid.New(), UpdateInput{
		Name: &newName,
	})

	assert.ErrorIs(t, err, ErrNotFound)
}

// ============================================================================
// Delete Tests
// ============================================================================

func TestService_Delete_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	statusID := uuid.New()
	repo.addStatus(&models.ProjectStatus{
		ID:        statusID,
		ProjectID: uuid.New(),
		Name:      "In Progress",
		SortOrder: 1,
		IsDefault: false,
		CreatedAt: time.Now(),
	})

	err := svc.Delete(context.Background(), statusID)

	require.NoError(t, err)
	_, exists := repo.statuses[statusID]
	assert.False(t, exists)
}

func TestService_Delete_Default_Fails(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	statusID := uuid.New()
	repo.addStatus(&models.ProjectStatus{
		ID:        statusID,
		ProjectID: uuid.New(),
		Name:      "To Do",
		SortOrder: 0,
		IsDefault: true,
		CreatedAt: time.Now(),
	})

	err := svc.Delete(context.Background(), statusID)

	assert.ErrorIs(t, err, ErrCannotDeleteDefault)
	// Verify it was NOT deleted
	_, exists := repo.statuses[statusID]
	assert.True(t, exists)
}

func TestService_Delete_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	err := svc.Delete(context.Background(), uuid.New())

	assert.ErrorIs(t, err, ErrNotFound)
}

// ============================================================================
// Reorder Tests
// ============================================================================

func TestService_Reorder_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	s1 := &models.ProjectStatus{ID: uuid.New(), ProjectID: projectID, Name: "A", SortOrder: 0, CreatedAt: time.Now()}
	s2 := &models.ProjectStatus{ID: uuid.New(), ProjectID: projectID, Name: "B", SortOrder: 1, CreatedAt: time.Now()}
	s3 := &models.ProjectStatus{ID: uuid.New(), ProjectID: projectID, Name: "C", SortOrder: 2, CreatedAt: time.Now()}
	repo.addStatus(s1)
	repo.addStatus(s2)
	repo.addStatus(s3)

	// Reverse order: C, B, A
	err := svc.Reorder(context.Background(), projectID, []uuid.UUID{s3.ID, s2.ID, s1.ID})

	require.NoError(t, err)
	assert.Equal(t, 0, repo.statuses[s3.ID].SortOrder)
	assert.Equal(t, 1, repo.statuses[s2.ID].SortOrder)
	assert.Equal(t, 2, repo.statuses[s1.ID].SortOrder)
}

func TestService_Reorder_MismatchedProject(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	projectA := uuid.New()
	projectB := uuid.New()

	s1 := &models.ProjectStatus{ID: uuid.New(), ProjectID: projectA, Name: "A", SortOrder: 0, CreatedAt: time.Now()}
	s2 := &models.ProjectStatus{ID: uuid.New(), ProjectID: projectB, Name: "B", SortOrder: 0, CreatedAt: time.Now()}
	repo.addStatus(s1)
	repo.addStatus(s2)

	err := svc.Reorder(context.Background(), projectA, []uuid.UUID{s1.ID, s2.ID})

	assert.ErrorIs(t, err, ErrInvalidSortOrder)
}

func TestService_Reorder_DuplicateIDs(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	projectID := uuid.New()
	s1 := &models.ProjectStatus{ID: uuid.New(), ProjectID: projectID, Name: "A", SortOrder: 0, CreatedAt: time.Now()}
	repo.addStatus(s1)

	err := svc.Reorder(context.Background(), projectID, []uuid.UUID{s1.ID, s1.ID})

	assert.ErrorIs(t, err, ErrInvalidSortOrder)
}

func TestService_Reorder_StatusNotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	err := svc.Reorder(context.Background(), uuid.New(), []uuid.UUID{uuid.New()})

	assert.ErrorIs(t, err, ErrNotFound)
}

// ============================================================================
// CopyForProject Tests
// ============================================================================

func TestService_CopyForProject_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	sourceID := uuid.New()
	targetID := uuid.New()

	repo.addStatus(&models.ProjectStatus{
		ID: uuid.New(), ProjectID: sourceID, Name: "To Do", SortOrder: 0, IsDefault: true, CreatedAt: time.Now(),
	})
	repo.addStatus(&models.ProjectStatus{
		ID: uuid.New(), ProjectID: sourceID, Name: "In Progress", SortOrder: 1, CreatedAt: time.Now(),
	})
	repo.addStatus(&models.ProjectStatus{
		ID: uuid.New(), ProjectID: sourceID, Name: "Done", SortOrder: 2, IsClosed: true, CreatedAt: time.Now(),
	})

	err := svc.CopyForProject(context.Background(), sourceID, targetID)

	require.NoError(t, err)

	// Count statuses for target project
	targetStatuses, _ := svc.ListByProject(context.Background(), targetID)
	assert.Len(t, targetStatuses, 3)
}

func TestService_CopyForProject_EmptySource(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	// Copy from project with no statuses
	err := svc.CopyForProject(context.Background(), uuid.New(), uuid.New())

	require.NoError(t, err)
}

// ============================================================================
// IsClosed Flag Tests
// ============================================================================

func TestService_Create_WithIsClosed(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	status, err := svc.Create(context.Background(), CreateInput{
		ProjectID: uuid.New(),
		Name:      "Done",
		IsClosed:  true,
	})

	require.NoError(t, err)
	assert.True(t, status.IsClosed)
}

func TestService_Update_SetClosed(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	statusID := uuid.New()
	repo.addStatus(&models.ProjectStatus{
		ID:        statusID,
		ProjectID: uuid.New(),
		Name:      "Done",
		SortOrder: 2,
		CreatedAt: time.Now(),
	})

	closed := true
	status, err := svc.Update(context.Background(), statusID, UpdateInput{
		IsClosed: &closed,
	})

	require.NoError(t, err)
	assert.True(t, status.IsClosed)
}
