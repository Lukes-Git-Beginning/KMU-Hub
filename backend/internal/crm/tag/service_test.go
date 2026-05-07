package tag

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
	tags        map[uuid.UUID]*models.Tag
	inUseTagIDs map[uuid.UUID]bool
	createErr   error
	getErr      error
	listErr     error
	updateErr   error
	deleteErr   error
	isInUseErr  error
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		tags:        make(map[uuid.UUID]*models.Tag),
		inUseTagIDs: make(map[uuid.UUID]bool),
	}
}

func (m *MockRepository) Create(ctx context.Context, tag *models.Tag) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.tags[tag.ID] = tag
	return nil
}

func (m *MockRepository) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*models.Tag, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	tag, ok := m.tags[id]
	if !ok || tag.TenantID != tenantID {
		return nil, ErrTagNotFound
	}
	return tag, nil
}

func (m *MockRepository) GetByEntityAndName(ctx context.Context, tenantID uuid.UUID, entityType models.EntityType, name string) (*models.Tag, error) {
	for _, tag := range m.tags {
		if tag.TenantID == tenantID && tag.EntityType == entityType && tag.Name == name {
			return tag, nil
		}
	}
	return nil, ErrTagNotFound
}

func (m *MockRepository) List(ctx context.Context, tenantID uuid.UUID, entityType *models.EntityType, offset, limit int) ([]*models.Tag, int, error) {
	if m.listErr != nil {
		return nil, 0, m.listErr
	}

	var result []*models.Tag
	for _, tag := range m.tags {
		if tag.TenantID != tenantID {
			continue
		}
		if entityType == nil || tag.EntityType == *entityType {
			result = append(result, tag)
		}
	}

	total := len(result)
	if offset >= len(result) {
		return []*models.Tag{}, total, nil
	}

	end := offset + limit
	if end > len(result) {
		end = len(result)
	}

	return result[offset:end], total, nil
}

func (m *MockRepository) Update(ctx context.Context, tag *models.Tag) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.tags[tag.ID] = tag
	return nil
}

func (m *MockRepository) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.tags, id)
	return nil
}

func (m *MockRepository) IsInUse(ctx context.Context, id, tenantID uuid.UUID) (bool, error) {
	if m.isInUseErr != nil {
		return false, m.isInUseErr
	}
	return m.inUseTagIDs[id], nil
}

func (m *MockRepository) SetInUse(id uuid.UUID, inUse bool) {
	m.inUseTagIDs[id] = inUse
}

// ============================================================================
// Create Tests
// ============================================================================

func TestService_Create_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()

	input := CreateInput{
		Name:       "Important",
		Color:      "#ef4444",
		EntityType: models.EntityTypeContact,
	}

	tag, err := svc.Create(context.Background(), tenantID, input)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, tag.ID)
	assert.Equal(t, tenantID, tag.TenantID)
	assert.Equal(t, "Important", tag.Name)
	assert.Equal(t, "#ef4444", tag.Color)
	assert.Equal(t, models.EntityTypeContact, tag.EntityType)
	assert.WithinDuration(t, time.Now(), tag.CreatedAt, time.Second)
}

func TestService_Create_DefaultColor(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()

	input := CreateInput{
		Name:       "Test",
		EntityType: models.EntityTypeCompany,
		// No color specified
	}

	tag, err := svc.Create(context.Background(), tenantID, input)

	require.NoError(t, err)
	assert.Equal(t, "#6b7280", tag.Color) // Default gray
}

func TestService_Create_TrimName(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()

	input := CreateInput{
		Name:       "  Trimmed  ",
		Color:      "#3b82f6",
		EntityType: models.EntityTypeDeal,
	}

	tag, err := svc.Create(context.Background(), tenantID, input)

	require.NoError(t, err)
	assert.Equal(t, "Trimmed", tag.Name)
}

func TestService_Create_InvalidEntityType(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()

	input := CreateInput{
		Name:       "Test",
		Color:      "#ef4444",
		EntityType: "invalid",
	}

	tag, err := svc.Create(context.Background(), tenantID, input)

	assert.Nil(t, tag)
	assert.ErrorIs(t, err, ErrInvalidEntityType)
}

func TestService_Create_EmptyName(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()

	input := CreateInput{
		Name:       "",
		Color:      "#ef4444",
		EntityType: models.EntityTypeContact,
	}

	tag, err := svc.Create(context.Background(), tenantID, input)

	assert.Nil(t, tag)
	assert.ErrorIs(t, err, ErrTagNameRequired)
}

func TestService_Create_WhitespaceOnlyName(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()

	input := CreateInput{
		Name:       "   ",
		Color:      "#ef4444",
		EntityType: models.EntityTypeContact,
	}

	tag, err := svc.Create(context.Background(), tenantID, input)

	assert.Nil(t, tag)
	assert.ErrorIs(t, err, ErrTagNameRequired)
}

func TestService_Create_InvalidColor(t *testing.T) {
	testCases := []struct {
		name  string
		color string
	}{
		{"short", "#fff"},
		{"no hash", "ff5733"},
		{"too long", "#ff5733aa"},
		{"invalid chars", "#gggggg"},
		{"random", "red"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repo := NewMockRepository()
			svc := NewService(repo)
			tenantID := uuid.New()

			input := CreateInput{
				Name:       "Test",
				Color:      tc.color,
				EntityType: models.EntityTypeContact,
			}

			tag, err := svc.Create(context.Background(), tenantID, input)

			assert.Nil(t, tag)
			assert.ErrorIs(t, err, ErrInvalidColor)
		})
	}
}

func TestService_Create_ValidColors(t *testing.T) {
	validColors := []string{
		"#ef4444",
		"#FFFFFF",
		"#000000",
		"#AbCdEf",
	}

	for _, color := range validColors {
		t.Run(color, func(t *testing.T) {
			repo := NewMockRepository()
			svc := NewService(repo)
			tenantID := uuid.New()

			input := CreateInput{
				Name:       "Test",
				Color:      color,
				EntityType: models.EntityTypeContact,
			}

			tag, err := svc.Create(context.Background(), tenantID, input)

			require.NoError(t, err)
			assert.Equal(t, color, tag.Color)
		})
	}
}

func TestService_Create_Duplicate(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()

	// Create first tag
	input := CreateInput{
		Name:       "Important",
		Color:      "#ef4444",
		EntityType: models.EntityTypeContact,
	}
	_, err := svc.Create(context.Background(), tenantID, input)
	require.NoError(t, err)

	// Try to create duplicate within same tenant
	tag, err := svc.Create(context.Background(), tenantID, input)

	assert.Nil(t, tag)
	assert.ErrorIs(t, err, ErrTagExists)
}

func TestService_Create_SameNameDifferentTenant(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	tenantA := uuid.New()
	tenantB := uuid.New()

	input := CreateInput{
		Name:       "Important",
		Color:      "#ef4444",
		EntityType: models.EntityTypeContact,
	}

	// Create for tenant A
	tagA, err := svc.Create(context.Background(), tenantA, input)
	require.NoError(t, err)

	// Same name same entity type — different tenant, should succeed
	tagB, err := svc.Create(context.Background(), tenantB, input)
	require.NoError(t, err)
	assert.NotEqual(t, tagA.ID, tagB.ID)
	assert.Equal(t, tenantA, tagA.TenantID)
	assert.Equal(t, tenantB, tagB.TenantID)
}

func TestService_Create_SameNameDifferentEntity(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()

	// Create tag for contact
	input1 := CreateInput{
		Name:       "Important",
		Color:      "#ef4444",
		EntityType: models.EntityTypeContact,
	}
	tag1, err := svc.Create(context.Background(), tenantID, input1)
	require.NoError(t, err)

	// Create tag with same name for company - should work
	input2 := CreateInput{
		Name:       "Important",
		Color:      "#3b82f6",
		EntityType: models.EntityTypeCompany,
	}
	tag2, err := svc.Create(context.Background(), tenantID, input2)

	require.NoError(t, err)
	assert.NotEqual(t, tag1.ID, tag2.ID)
	assert.Equal(t, models.EntityTypeContact, tag1.EntityType)
	assert.Equal(t, models.EntityTypeCompany, tag2.EntityType)
}

// ============================================================================
// GetByID Tests
// ============================================================================

func TestService_GetByID_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()

	existingTag := &models.Tag{
		ID:         uuid.New(),
		TenantID:   tenantID,
		Name:       "Test",
		Color:      "#ef4444",
		EntityType: models.EntityTypeContact,
		CreatedAt:  time.Now(),
	}
	repo.tags[existingTag.ID] = existingTag

	tag, err := svc.GetByID(context.Background(), existingTag.ID, tenantID)

	require.NoError(t, err)
	assert.Equal(t, existingTag.ID, tag.ID)
	assert.Equal(t, existingTag.Name, tag.Name)
}

func TestService_GetByID_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()

	tag, err := svc.GetByID(context.Background(), uuid.New(), tenantID)

	assert.Nil(t, tag)
	assert.ErrorIs(t, err, ErrTagNotFound)
}

func TestService_GetByID_CrossTenantIsolation(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	tenantA := uuid.New()
	tenantB := uuid.New()

	// Insert tag for tenant A
	existingTag := &models.Tag{
		ID:         uuid.New(),
		TenantID:   tenantA,
		Name:       "SecretTag",
		Color:      "#ef4444",
		EntityType: models.EntityTypeContact,
		CreatedAt:  time.Now(),
	}
	repo.tags[existingTag.ID] = existingTag

	// Tenant B must not see it
	tag, err := svc.GetByID(context.Background(), existingTag.ID, tenantB)
	assert.Nil(t, tag)
	assert.ErrorIs(t, err, ErrTagNotFound)
}

// ============================================================================
// List Tests
// ============================================================================

func TestService_List_All(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()

	// Create multiple tags for tenantID
	for i := 0; i < 5; i++ {
		id := uuid.New()
		repo.tags[id] = &models.Tag{
			ID:         id,
			TenantID:   tenantID,
			Name:       "Tag",
			EntityType: models.EntityTypeContact,
		}
	}
	// Add one tag for a different tenant — must not appear
	otherId := uuid.New()
	repo.tags[otherId] = &models.Tag{
		ID:         otherId,
		TenantID:   uuid.New(),
		Name:       "OtherTag",
		EntityType: models.EntityTypeContact,
	}

	tags, total, err := svc.List(context.Background(), tenantID, ListInput{
		Page:     1,
		PageSize: 50,
	})

	require.NoError(t, err)
	assert.Len(t, tags, 5)
	assert.Equal(t, 5, total)
}

func TestService_List_ByEntityType(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()

	// Create tags for different entities
	for i := 0; i < 3; i++ {
		id := uuid.New()
		repo.tags[id] = &models.Tag{
			ID:         id,
			TenantID:   tenantID,
			Name:       "Contact Tag",
			EntityType: models.EntityTypeContact,
		}
	}
	for i := 0; i < 2; i++ {
		id := uuid.New()
		repo.tags[id] = &models.Tag{
			ID:         id,
			TenantID:   tenantID,
			Name:       "Company Tag",
			EntityType: models.EntityTypeCompany,
		}
	}

	entityType := models.EntityTypeContact
	tags, total, err := svc.List(context.Background(), tenantID, ListInput{
		EntityType: &entityType,
		Page:       1,
		PageSize:   50,
	})

	require.NoError(t, err)
	assert.Len(t, tags, 3)
	assert.Equal(t, 3, total)
}

func TestService_List_Pagination(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()

	// Create 10 tags
	for i := 0; i < 10; i++ {
		id := uuid.New()
		repo.tags[id] = &models.Tag{
			ID:         id,
			TenantID:   tenantID,
			Name:       "Tag",
			EntityType: models.EntityTypeContact,
		}
	}

	// Get page 2 with page size 3
	tags, total, err := svc.List(context.Background(), tenantID, ListInput{
		Page:     2,
		PageSize: 3,
	})

	require.NoError(t, err)
	assert.Len(t, tags, 3)
	assert.Equal(t, 10, total)
}

func TestService_List_DefaultPagination(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()

	tags, _, err := svc.List(context.Background(), tenantID, ListInput{
		Page:     0,  // Invalid, should default to 1
		PageSize: -1, // Invalid, should default to 50
	})

	require.NoError(t, err)
	assert.NotNil(t, tags)
}

func TestService_List_MaxPageSize(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()

	tags, _, err := svc.List(context.Background(), tenantID, ListInput{
		Page:     1,
		PageSize: 200, // Over max, should cap at 100
	})

	require.NoError(t, err)
	assert.NotNil(t, tags)
}

// ============================================================================
// Update Tests
// ============================================================================

func TestService_Update_Name(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()

	existingTag := &models.Tag{
		ID:         uuid.New(),
		TenantID:   tenantID,
		Name:       "Old Name",
		Color:      "#ef4444",
		EntityType: models.EntityTypeContact,
		CreatedAt:  time.Now(),
	}
	repo.tags[existingTag.ID] = existingTag

	newName := "New Name"
	tag, err := svc.Update(context.Background(), existingTag.ID, tenantID, UpdateInput{
		Name: &newName,
	})

	require.NoError(t, err)
	assert.Equal(t, "New Name", tag.Name)
	assert.Equal(t, "#ef4444", tag.Color) // Unchanged
}

func TestService_Update_Color(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()

	existingTag := &models.Tag{
		ID:         uuid.New(),
		TenantID:   tenantID,
		Name:       "Test",
		Color:      "#ef4444",
		EntityType: models.EntityTypeContact,
		CreatedAt:  time.Now(),
	}
	repo.tags[existingTag.ID] = existingTag

	newColor := "#3b82f6"
	tag, err := svc.Update(context.Background(), existingTag.ID, tenantID, UpdateInput{
		Color: &newColor,
	})

	require.NoError(t, err)
	assert.Equal(t, "Test", tag.Name) // Unchanged
	assert.Equal(t, "#3b82f6", tag.Color)
}

func TestService_Update_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()

	newName := "New Name"
	tag, err := svc.Update(context.Background(), uuid.New(), tenantID, UpdateInput{
		Name: &newName,
	})

	assert.Nil(t, tag)
	assert.ErrorIs(t, err, ErrTagNotFound)
}

func TestService_Update_EmptyName(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()

	existingTag := &models.Tag{
		ID:         uuid.New(),
		TenantID:   tenantID,
		Name:       "Test",
		Color:      "#ef4444",
		EntityType: models.EntityTypeContact,
		CreatedAt:  time.Now(),
	}
	repo.tags[existingTag.ID] = existingTag

	emptyName := ""
	tag, err := svc.Update(context.Background(), existingTag.ID, tenantID, UpdateInput{
		Name: &emptyName,
	})

	assert.Nil(t, tag)
	assert.ErrorIs(t, err, ErrTagNameRequired)
}

func TestService_Update_InvalidColor(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()

	existingTag := &models.Tag{
		ID:         uuid.New(),
		TenantID:   tenantID,
		Name:       "Test",
		Color:      "#ef4444",
		EntityType: models.EntityTypeContact,
		CreatedAt:  time.Now(),
	}
	repo.tags[existingTag.ID] = existingTag

	invalidColor := "red"
	tag, err := svc.Update(context.Background(), existingTag.ID, tenantID, UpdateInput{
		Color: &invalidColor,
	})

	assert.Nil(t, tag)
	assert.ErrorIs(t, err, ErrInvalidColor)
}

func TestService_Update_DuplicateName(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()

	// Create two tags
	tag1 := &models.Tag{
		ID:         uuid.New(),
		TenantID:   tenantID,
		Name:       "Tag One",
		Color:      "#ef4444",
		EntityType: models.EntityTypeContact,
		CreatedAt:  time.Now(),
	}
	repo.tags[tag1.ID] = tag1

	tag2 := &models.Tag{
		ID:         uuid.New(),
		TenantID:   tenantID,
		Name:       "Tag Two",
		Color:      "#3b82f6",
		EntityType: models.EntityTypeContact,
		CreatedAt:  time.Now(),
	}
	repo.tags[tag2.ID] = tag2

	// Try to rename tag2 to tag1's name
	duplicateName := "Tag One"
	tag, err := svc.Update(context.Background(), tag2.ID, tenantID, UpdateInput{
		Name: &duplicateName,
	})

	assert.Nil(t, tag)
	assert.ErrorIs(t, err, ErrTagExists)
}

func TestService_Update_SameNameAllowed(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()

	existingTag := &models.Tag{
		ID:         uuid.New(),
		TenantID:   tenantID,
		Name:       "Same Name",
		Color:      "#ef4444",
		EntityType: models.EntityTypeContact,
		CreatedAt:  time.Now(),
	}
	repo.tags[existingTag.ID] = existingTag

	// Update with same name - should work
	sameName := "Same Name"
	newColor := "#3b82f6"
	tag, err := svc.Update(context.Background(), existingTag.ID, tenantID, UpdateInput{
		Name:  &sameName,
		Color: &newColor,
	})

	require.NoError(t, err)
	assert.Equal(t, "Same Name", tag.Name)
	assert.Equal(t, "#3b82f6", tag.Color)
}

// ============================================================================
// Delete Tests
// ============================================================================

func TestService_Delete_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()

	existingTag := &models.Tag{
		ID:         uuid.New(),
		TenantID:   tenantID,
		Name:       "To Delete",
		Color:      "#ef4444",
		EntityType: models.EntityTypeContact,
		CreatedAt:  time.Now(),
	}
	repo.tags[existingTag.ID] = existingTag

	err := svc.Delete(context.Background(), existingTag.ID, tenantID)

	require.NoError(t, err)
	assert.Len(t, repo.tags, 0)
}

func TestService_Delete_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()

	err := svc.Delete(context.Background(), uuid.New(), tenantID)

	assert.ErrorIs(t, err, ErrTagNotFound)
}

func TestService_Delete_InUse(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()

	existingTag := &models.Tag{
		ID:         uuid.New(),
		TenantID:   tenantID,
		Name:       "In Use",
		Color:      "#ef4444",
		EntityType: models.EntityTypeContact,
		CreatedAt:  time.Now(),
	}
	repo.tags[existingTag.ID] = existingTag
	repo.SetInUse(existingTag.ID, true)

	err := svc.Delete(context.Background(), existingTag.ID, tenantID)

	assert.ErrorIs(t, err, ErrTagInUse)
	assert.Len(t, repo.tags, 1) // Tag should not be deleted
}

// ============================================================================
// EntityType Tests
// ============================================================================

func TestService_Create_AllEntityTypes(t *testing.T) {
	entityTypes := []models.EntityType{
		models.EntityTypeContact,
		models.EntityTypeCompany,
		models.EntityTypeDeal,
		models.EntityTypeActivity,
	}

	for _, et := range entityTypes {
		t.Run(string(et), func(t *testing.T) {
			repo := NewMockRepository()
			svc := NewService(repo)
			tenantID := uuid.New()

			input := CreateInput{
				Name:       "Test",
				Color:      "#ef4444",
				EntityType: et,
			}

			tag, err := svc.Create(context.Background(), tenantID, input)

			require.NoError(t, err)
			assert.Equal(t, et, tag.EntityType)
		})
	}
}
