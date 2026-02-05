package savedfilter

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// MockRepository implements Repository for testing
type MockRepository struct {
	filters      map[uuid.UUID]*models.SavedFilter
	createErr    error
	getErr       error
	updateErr    error
	deleteErr    error
	clearDefault bool
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		filters: make(map[uuid.UUID]*models.SavedFilter),
	}
}

func (m *MockRepository) Create(ctx context.Context, filter *models.SavedFilter) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.filters[filter.ID] = filter
	return nil
}

func (m *MockRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.SavedFilter, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	filter, ok := m.filters[id]
	if !ok {
		return nil, ErrFilterNotFound
	}
	return filter, nil
}

func (m *MockRepository) List(ctx context.Context, filter ListFilter) ([]*models.SavedFilter, error) {
	var result []*models.SavedFilter
	for _, f := range m.filters {
		if filter.EntityType != nil && f.EntityType != *filter.EntityType {
			continue
		}
		if filter.UserID != nil && f.CreatedBy != *filter.UserID {
			continue
		}
		result = append(result, f)
	}
	return result, nil
}

func (m *MockRepository) Update(ctx context.Context, filter *models.SavedFilter) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.filters[filter.ID] = filter
	return nil
}

func (m *MockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.filters, id)
	return nil
}

func (m *MockRepository) ClearDefault(ctx context.Context, userID uuid.UUID, entityType models.EntityType) error {
	m.clearDefault = true
	for _, f := range m.filters {
		if f.CreatedBy == userID && f.EntityType == entityType && f.IsDefault {
			f.IsDefault = false
		}
	}
	return nil
}

// ============================================================================
// Create Tests
// ============================================================================

func TestService_Create_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := CreateInput{
		Name:       "Hot Leads",
		EntityType: "contact",
		FilterJSON: `{"status":"hot"}`,
		IsDefault:  false,
		CreatedBy:  uuid.New(),
	}

	filter, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, filter.ID)
	assert.Equal(t, "Hot Leads", filter.Name)
	assert.Equal(t, models.EntityTypeContact, filter.EntityType)
	assert.Equal(t, `{"status":"hot"}`, filter.FilterJSON)
	assert.False(t, filter.IsDefault)
}

func TestService_Create_AllEntityTypes(t *testing.T) {
	types := []string{"contact", "company", "deal", "activity"}

	for _, entityType := range types {
		t.Run(entityType, func(t *testing.T) {
			repo := NewMockRepository()
			svc := NewService(repo)

			input := CreateInput{
				Name:       "Test Filter",
				EntityType: entityType,
				FilterJSON: `{}`,
				CreatedBy:  uuid.New(),
			}

			filter, err := svc.Create(context.Background(), input)

			require.NoError(t, err)
			assert.Equal(t, models.EntityType(entityType), filter.EntityType)
		})
	}
}

func TestService_Create_SetAsDefault(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	userID := uuid.New()

	// Create first default filter
	input1 := CreateInput{
		Name:       "First Default",
		EntityType: "deal",
		FilterJSON: `{"stage":"new"}`,
		IsDefault:  true,
		CreatedBy:  userID,
	}

	filter1, err := svc.Create(context.Background(), input1)
	require.NoError(t, err)
	assert.True(t, filter1.IsDefault)

	// Create second default filter - should clear first
	input2 := CreateInput{
		Name:       "Second Default",
		EntityType: "deal",
		FilterJSON: `{"stage":"won"}`,
		IsDefault:  true,
		CreatedBy:  userID,
	}

	filter2, err := svc.Create(context.Background(), input2)
	require.NoError(t, err)
	assert.True(t, filter2.IsDefault)
	assert.True(t, repo.clearDefault) // ClearDefault was called

	// Verify first filter is no longer default
	assert.False(t, repo.filters[filter1.ID].IsDefault)
}

func TestService_Create_NameRequired(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := CreateInput{
		Name:       "",
		EntityType: "contact",
		FilterJSON: `{}`,
		CreatedBy:  uuid.New(),
	}

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrNameRequired)
}

func TestService_Create_NameWhitespaceOnly(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := CreateInput{
		Name:       "   ",
		EntityType: "contact",
		FilterJSON: `{}`,
		CreatedBy:  uuid.New(),
	}

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrNameRequired)
}

func TestService_Create_InvalidEntityType(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := CreateInput{
		Name:       "Test",
		EntityType: "invalid",
		FilterJSON: `{}`,
		CreatedBy:  uuid.New(),
	}

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrInvalidEntityType)
}

func TestService_Create_InvalidFilterJSON(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := CreateInput{
		Name:       "Test",
		EntityType: "contact",
		FilterJSON: `not valid json`,
		CreatedBy:  uuid.New(),
	}

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrInvalidFilterJSON)
}

func TestService_Create_TrimsName(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := CreateInput{
		Name:       "  Trimmed Name  ",
		EntityType: "contact",
		FilterJSON: `{}`,
		CreatedBy:  uuid.New(),
	}

	filter, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, "Trimmed Name", filter.Name)
}

// ============================================================================
// GetByID Tests
// ============================================================================

func TestService_GetByID_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	filterID := uuid.New()
	repo.filters[filterID] = &models.SavedFilter{
		ID:         filterID,
		Name:       "Test Filter",
		EntityType: models.EntityTypeContact,
		FilterJSON: `{"status":"active"}`,
		CreatedBy:  uuid.New(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	filter, err := svc.GetByID(context.Background(), filterID)

	require.NoError(t, err)
	assert.Equal(t, filterID, filter.ID)
	assert.Equal(t, "Test Filter", filter.Name)
}

func TestService_GetByID_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	_, err := svc.GetByID(context.Background(), uuid.New())

	assert.ErrorIs(t, err, ErrFilterNotFound)
}

// ============================================================================
// List Tests
// ============================================================================

func TestService_List_All(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	userID := uuid.New()
	for i := 0; i < 5; i++ {
		id := uuid.New()
		repo.filters[id] = &models.SavedFilter{
			ID:         id,
			Name:       "Filter",
			EntityType: models.EntityTypeContact,
			FilterJSON: `{}`,
			CreatedBy:  userID,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
	}

	filters, err := svc.List(context.Background(), ListInput{
		UserID: &userID,
	})

	require.NoError(t, err)
	assert.Len(t, filters, 5)
}

func TestService_List_FilterByEntityType(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	userID := uuid.New()

	// Add 3 contact filters
	for i := 0; i < 3; i++ {
		id := uuid.New()
		repo.filters[id] = &models.SavedFilter{
			ID:         id,
			Name:       "Contact Filter",
			EntityType: models.EntityTypeContact,
			FilterJSON: `{}`,
			CreatedBy:  userID,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
	}

	// Add 2 deal filters
	for i := 0; i < 2; i++ {
		id := uuid.New()
		repo.filters[id] = &models.SavedFilter{
			ID:         id,
			Name:       "Deal Filter",
			EntityType: models.EntityTypeDeal,
			FilterJSON: `{}`,
			CreatedBy:  userID,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
	}

	contactType := "contact"
	filters, err := svc.List(context.Background(), ListInput{
		EntityType: &contactType,
		UserID:     &userID,
	})

	require.NoError(t, err)
	assert.Len(t, filters, 3)
}

func TestService_List_InvalidEntityType(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	invalidType := "invalid"
	_, err := svc.List(context.Background(), ListInput{
		EntityType: &invalidType,
	})

	assert.ErrorIs(t, err, ErrInvalidEntityType)
}

func TestService_List_FilterByUserID(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	userID1 := uuid.New()
	userID2 := uuid.New()

	// Add 3 filters for user1
	for i := 0; i < 3; i++ {
		id := uuid.New()
		repo.filters[id] = &models.SavedFilter{
			ID:         id,
			Name:       "User1 Filter",
			EntityType: models.EntityTypeContact,
			FilterJSON: `{}`,
			CreatedBy:  userID1,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
	}

	// Add 2 filters for user2
	for i := 0; i < 2; i++ {
		id := uuid.New()
		repo.filters[id] = &models.SavedFilter{
			ID:         id,
			Name:       "User2 Filter",
			EntityType: models.EntityTypeContact,
			FilterJSON: `{}`,
			CreatedBy:  userID2,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
	}

	filters, err := svc.List(context.Background(), ListInput{
		UserID: &userID1,
	})

	require.NoError(t, err)
	assert.Len(t, filters, 3)
}

// ============================================================================
// Update Tests
// ============================================================================

func TestService_Update_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	userID := uuid.New()
	filterID := uuid.New()
	repo.filters[filterID] = &models.SavedFilter{
		ID:         filterID,
		Name:       "Old Name",
		EntityType: models.EntityTypeContact,
		FilterJSON: `{"old":"value"}`,
		IsDefault:  false,
		CreatedBy:  userID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	newName := "New Name"
	newJSON := `{"new":"value"}`

	filter, err := svc.Update(context.Background(), filterID, userID, UpdateInput{
		Name:       &newName,
		FilterJSON: &newJSON,
	})

	require.NoError(t, err)
	assert.Equal(t, "New Name", filter.Name)
	assert.Equal(t, `{"new":"value"}`, filter.FilterJSON)
}

func TestService_Update_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	newName := "New Name"
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), UpdateInput{
		Name: &newName,
	})

	assert.ErrorIs(t, err, ErrFilterNotFound)
}

func TestService_Update_NotOwned(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	ownerID := uuid.New()
	otherUserID := uuid.New()
	filterID := uuid.New()
	repo.filters[filterID] = &models.SavedFilter{
		ID:         filterID,
		Name:       "Test",
		EntityType: models.EntityTypeContact,
		FilterJSON: `{}`,
		CreatedBy:  ownerID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	newName := "New Name"
	_, err := svc.Update(context.Background(), filterID, otherUserID, UpdateInput{
		Name: &newName,
	})

	assert.ErrorIs(t, err, ErrFilterNotOwned)
}

func TestService_Update_EmptyName(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	userID := uuid.New()
	filterID := uuid.New()
	repo.filters[filterID] = &models.SavedFilter{
		ID:         filterID,
		Name:       "Test",
		EntityType: models.EntityTypeContact,
		FilterJSON: `{}`,
		CreatedBy:  userID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	emptyName := ""
	_, err := svc.Update(context.Background(), filterID, userID, UpdateInput{
		Name: &emptyName,
	})

	assert.ErrorIs(t, err, ErrNameRequired)
}

func TestService_Update_InvalidFilterJSON(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	userID := uuid.New()
	filterID := uuid.New()
	repo.filters[filterID] = &models.SavedFilter{
		ID:         filterID,
		Name:       "Test",
		EntityType: models.EntityTypeContact,
		FilterJSON: `{}`,
		CreatedBy:  userID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	invalidJSON := "not valid"
	_, err := svc.Update(context.Background(), filterID, userID, UpdateInput{
		FilterJSON: &invalidJSON,
	})

	assert.ErrorIs(t, err, ErrInvalidFilterJSON)
}

func TestService_Update_SetAsDefault(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	userID := uuid.New()

	// Create existing default
	existingID := uuid.New()
	repo.filters[existingID] = &models.SavedFilter{
		ID:         existingID,
		Name:       "Existing Default",
		EntityType: models.EntityTypeDeal,
		FilterJSON: `{}`,
		IsDefault:  true,
		CreatedBy:  userID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Create filter to be updated
	filterID := uuid.New()
	repo.filters[filterID] = &models.SavedFilter{
		ID:         filterID,
		Name:       "Test",
		EntityType: models.EntityTypeDeal,
		FilterJSON: `{}`,
		IsDefault:  false,
		CreatedBy:  userID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	isDefault := true
	filter, err := svc.Update(context.Background(), filterID, userID, UpdateInput{
		IsDefault: &isDefault,
	})

	require.NoError(t, err)
	assert.True(t, filter.IsDefault)
	assert.True(t, repo.clearDefault) // ClearDefault was called
	assert.False(t, repo.filters[existingID].IsDefault)
}

// ============================================================================
// Delete Tests
// ============================================================================

func TestService_Delete_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	userID := uuid.New()
	filterID := uuid.New()
	repo.filters[filterID] = &models.SavedFilter{
		ID:         filterID,
		Name:       "Test",
		EntityType: models.EntityTypeContact,
		FilterJSON: `{}`,
		CreatedBy:  userID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	err := svc.Delete(context.Background(), filterID, userID)

	require.NoError(t, err)
	assert.NotContains(t, repo.filters, filterID)
}

func TestService_Delete_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	err := svc.Delete(context.Background(), uuid.New(), uuid.New())

	assert.ErrorIs(t, err, ErrFilterNotFound)
}

func TestService_Delete_NotOwned(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	ownerID := uuid.New()
	otherUserID := uuid.New()
	filterID := uuid.New()
	repo.filters[filterID] = &models.SavedFilter{
		ID:         filterID,
		Name:       "Test",
		EntityType: models.EntityTypeContact,
		FilterJSON: `{}`,
		CreatedBy:  ownerID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	err := svc.Delete(context.Background(), filterID, otherUserID)

	assert.ErrorIs(t, err, ErrFilterNotOwned)
	assert.Contains(t, repo.filters, filterID) // Should not be deleted
}
