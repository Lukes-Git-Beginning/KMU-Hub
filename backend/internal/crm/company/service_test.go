package company

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
	companies     map[uuid.UUID]*models.Company
	companyTags   map[uuid.UUID][]*models.Tag
	customFields  map[uuid.UUID]map[uuid.UUID]any
	contactCounts map[uuid.UUID]int
	validTags     map[uuid.UUID]models.EntityType
	createErr     error
	getErr        error
	listErr       error
	updateErr     error
	deleteErr     error
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		companies:     make(map[uuid.UUID]*models.Company),
		companyTags:   make(map[uuid.UUID][]*models.Tag),
		customFields:  make(map[uuid.UUID]map[uuid.UUID]any),
		contactCounts: make(map[uuid.UUID]int),
		validTags:     make(map[uuid.UUID]models.EntityType),
	}
}

func (m *MockRepository) Create(ctx context.Context, company *models.Company) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.companies[company.ID] = company
	return nil
}

func (m *MockRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Company, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	company, ok := m.companies[id]
	if !ok {
		return nil, ErrCompanyNotFound
	}
	return company, nil
}

func (m *MockRepository) List(ctx context.Context, filter ListFilter, offset, limit int) ([]*models.Company, int, error) {
	if m.listErr != nil {
		return nil, 0, m.listErr
	}

	var result []*models.Company
	for _, c := range m.companies {
		result = append(result, c)
	}

	total := len(result)
	if offset >= len(result) {
		return []*models.Company{}, total, nil
	}

	end := offset + limit
	if end > len(result) {
		end = len(result)
	}

	return result[offset:end], total, nil
}

func (m *MockRepository) Update(ctx context.Context, company *models.Company) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.companies[company.ID] = company
	return nil
}

func (m *MockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.companies, id)
	return nil
}

func (m *MockRepository) GetContactCount(ctx context.Context, companyID uuid.UUID) (int, error) {
	return m.contactCounts[companyID], nil
}

func (m *MockRepository) GetTags(ctx context.Context, companyID uuid.UUID) ([]*models.Tag, error) {
	return m.companyTags[companyID], nil
}

func (m *MockRepository) AddTags(ctx context.Context, companyID uuid.UUID, tagIDs []uuid.UUID) error {
	return nil
}

func (m *MockRepository) RemoveTags(ctx context.Context, companyID uuid.UUID, tagIDs []uuid.UUID) error {
	return nil
}

func (m *MockRepository) GetCustomFieldValues(ctx context.Context, companyID uuid.UUID) ([]*models.CustomFieldValueRow, error) {
	return nil, nil
}

func (m *MockRepository) SetCustomFieldValues(ctx context.Context, companyID uuid.UUID, values map[uuid.UUID]any) error {
	m.customFields[companyID] = values
	return nil
}

func (m *MockRepository) HasContacts(ctx context.Context, companyID uuid.UUID) (bool, error) {
	return m.contactCounts[companyID] > 0, nil
}

func (m *MockRepository) TagExists(ctx context.Context, tagID uuid.UUID, entityType models.EntityType) (bool, error) {
	et, exists := m.validTags[tagID]
	return exists && et == entityType, nil
}

// Helper to add a valid tag for testing
func (m *MockRepository) AddValidTag(tagID uuid.UUID, entityType models.EntityType) {
	m.validTags[tagID] = entityType
}

// Helper to set contact count for testing
func (m *MockRepository) SetContactCount(companyID uuid.UUID, count int) {
	m.contactCounts[companyID] = count
}

func (m *MockRepository) FindDuplicateCandidates(_ context.Context, _ uuid.UUID) ([]*DuplicateCandidate, error) {
	return nil, nil
}

func (m *MockRepository) MergeInto(_ context.Context, _, _ uuid.UUID) error {
	return nil
}

// ============================================================================
// Create Tests
// ============================================================================

func TestService_Create_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := CreateInput{
		Name:      "Acme Corp",
		CreatedBy: uuid.New(),
	}

	company, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, company.ID)
	assert.Equal(t, "Acme Corp", company.Name)
	assert.NotZero(t, company.CreatedAt)
	assert.NotZero(t, company.UpdatedAt)
}

func TestService_Create_WithAllFields(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	domain := "acme.com"
	industry := "Technology"
	employees := 100
	address := "123 Main St"
	city := "Berlin"
	country := "Germany"
	notes := "Important client"

	input := CreateInput{
		Name:          "Acme Corp",
		Domain:        &domain,
		Industry:      &industry,
		EmployeeCount: &employees,
		Address:       &address,
		City:          &city,
		Country:       &country,
		Notes:         &notes,
		CreatedBy:     uuid.New(),
	}

	company, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, "Acme Corp", company.Name)
	assert.Equal(t, "acme.com", *company.Domain)
	assert.Equal(t, "Technology", *company.Industry)
	assert.Equal(t, 100, *company.EmployeeCount)
	assert.Equal(t, "123 Main St", *company.Address)
	assert.Equal(t, "Berlin", *company.City)
	assert.Equal(t, "Germany", *company.Country)
	assert.Equal(t, "Important client", *company.Notes)
}

func TestService_Create_WithTags(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	tagID := uuid.New()
	repo.AddValidTag(tagID, models.EntityTypeCompany)

	input := CreateInput{
		Name:      "Acme Corp",
		TagIDs:    []uuid.UUID{tagID},
		CreatedBy: uuid.New(),
	}

	company, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, "Acme Corp", company.Name)
}

func TestService_Create_NameRequired(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := CreateInput{
		Name:      "",
		CreatedBy: uuid.New(),
	}

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrNameRequired)
}

func TestService_Create_NameTrimmed(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := CreateInput{
		Name:      "  Acme Corp  ",
		CreatedBy: uuid.New(),
	}

	company, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, "Acme Corp", company.Name)
}

func TestService_Create_InvalidTag(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := CreateInput{
		Name:      "Acme Corp",
		TagIDs:    []uuid.UUID{uuid.New()}, // Non-existent tag
		CreatedBy: uuid.New(),
	}

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrTagNotFound)
}

func TestService_Create_WrongTagType(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	tagID := uuid.New()
	repo.AddValidTag(tagID, models.EntityTypeContact) // Contact tag, not company

	input := CreateInput{
		Name:      "Acme Corp",
		TagIDs:    []uuid.UUID{tagID},
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

	companyID := uuid.New()
	repo.companies[companyID] = &models.Company{
		ID:        companyID,
		Name:      "Acme Corp",
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	company, err := svc.GetByID(context.Background(), companyID)

	require.NoError(t, err)
	assert.Equal(t, companyID, company.ID)
	assert.Equal(t, "Acme Corp", company.Name)
}

func TestService_GetByID_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	_, err := svc.GetByID(context.Background(), uuid.New())

	assert.ErrorIs(t, err, ErrCompanyNotFound)
}

// ============================================================================
// List Tests
// ============================================================================

func TestService_List_Empty(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	companies, total, err := svc.List(context.Background(), ListInput{})

	require.NoError(t, err)
	assert.Empty(t, companies)
	assert.Equal(t, 0, total)
}

func TestService_List_WithCompanies(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	for i := 0; i < 3; i++ {
		id := uuid.New()
		repo.companies[id] = &models.Company{
			ID:        id,
			Name:      "Company " + string(rune('A'+i)),
			CreatedBy: uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	companies, total, err := svc.List(context.Background(), ListInput{})

	require.NoError(t, err)
	assert.Len(t, companies, 3)
	assert.Equal(t, 3, total)
}

func TestService_List_DefaultPagination(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	// Create 25 companies
	for i := 0; i < 25; i++ {
		id := uuid.New()
		repo.companies[id] = &models.Company{
			ID:        id,
			Name:      "Company",
			CreatedBy: uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	companies, total, err := svc.List(context.Background(), ListInput{
		Page:     0, // Should default to 1
		PageSize: 0, // Should default to 20
	})

	require.NoError(t, err)
	assert.Len(t, companies, 20)
	assert.Equal(t, 25, total)
}

func TestService_List_Pagination(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	for i := 0; i < 5; i++ {
		id := uuid.New()
		repo.companies[id] = &models.Company{
			ID:        id,
			Name:      "Company",
			CreatedBy: uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	companies, total, err := svc.List(context.Background(), ListInput{
		Page:     2,
		PageSize: 2,
	})

	require.NoError(t, err)
	assert.Len(t, companies, 2)
	assert.Equal(t, 5, total)
}

// ============================================================================
// Update Tests
// ============================================================================

func TestService_Update_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	companyID := uuid.New()
	repo.companies[companyID] = &models.Company{
		ID:        companyID,
		Name:      "Old Name",
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	newName := "New Name"
	company, err := svc.Update(context.Background(), companyID, UpdateInput{
		Name: &newName,
	})

	require.NoError(t, err)
	assert.Equal(t, "New Name", company.Name)
}

func TestService_Update_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	newName := "New Name"
	_, err := svc.Update(context.Background(), uuid.New(), UpdateInput{
		Name: &newName,
	})

	assert.ErrorIs(t, err, ErrCompanyNotFound)
}

func TestService_Update_EmptyName(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	companyID := uuid.New()
	repo.companies[companyID] = &models.Company{
		ID:        companyID,
		Name:      "Old Name",
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	emptyName := ""
	_, err := svc.Update(context.Background(), companyID, UpdateInput{
		Name: &emptyName,
	})

	assert.ErrorIs(t, err, ErrNameRequired)
}

func TestService_Update_AllFields(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	companyID := uuid.New()
	repo.companies[companyID] = &models.Company{
		ID:        companyID,
		Name:      "Old Name",
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	newName := "New Name"
	newDomain := "new.com"
	newIndustry := "Finance"
	newEmployees := 500
	newAddress := "456 New St"
	newCity := "Munich"
	newCountry := "Austria"
	newNotes := "Updated notes"

	company, err := svc.Update(context.Background(), companyID, UpdateInput{
		Name:          &newName,
		Domain:        &newDomain,
		Industry:      &newIndustry,
		EmployeeCount: &newEmployees,
		Address:       &newAddress,
		City:          &newCity,
		Country:       &newCountry,
		Notes:         &newNotes,
	})

	require.NoError(t, err)
	assert.Equal(t, "New Name", company.Name)
	assert.Equal(t, "new.com", *company.Domain)
	assert.Equal(t, "Finance", *company.Industry)
	assert.Equal(t, 500, *company.EmployeeCount)
	assert.Equal(t, "456 New St", *company.Address)
	assert.Equal(t, "Munich", *company.City)
	assert.Equal(t, "Austria", *company.Country)
	assert.Equal(t, "Updated notes", *company.Notes)
}

// ============================================================================
// Delete Tests
// ============================================================================

func TestService_Delete_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	companyID := uuid.New()
	repo.companies[companyID] = &models.Company{
		ID:        companyID,
		Name:      "Acme Corp",
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := svc.Delete(context.Background(), companyID)

	require.NoError(t, err)
	assert.NotContains(t, repo.companies, companyID)
}

func TestService_Delete_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	err := svc.Delete(context.Background(), uuid.New())

	assert.ErrorIs(t, err, ErrCompanyNotFound)
}

func TestService_Delete_HasContacts(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	companyID := uuid.New()
	repo.companies[companyID] = &models.Company{
		ID:        companyID,
		Name:      "Acme Corp",
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.SetContactCount(companyID, 5)

	err := svc.Delete(context.Background(), companyID)

	assert.ErrorIs(t, err, ErrCompanyInUse)
	assert.Contains(t, repo.companies, companyID) // Not deleted
}

// ============================================================================
// AddTags Tests
// ============================================================================

func TestService_AddTags_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	companyID := uuid.New()
	repo.companies[companyID] = &models.Company{
		ID:        companyID,
		Name:      "Acme Corp",
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tagID := uuid.New()
	repo.AddValidTag(tagID, models.EntityTypeCompany)

	company, err := svc.AddTags(context.Background(), companyID, []uuid.UUID{tagID})

	require.NoError(t, err)
	assert.Equal(t, companyID, company.ID)
}

func TestService_AddTags_CompanyNotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	tagID := uuid.New()
	repo.AddValidTag(tagID, models.EntityTypeCompany)

	_, err := svc.AddTags(context.Background(), uuid.New(), []uuid.UUID{tagID})

	assert.ErrorIs(t, err, ErrCompanyNotFound)
}

func TestService_AddTags_InvalidTag(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	companyID := uuid.New()
	repo.companies[companyID] = &models.Company{
		ID:        companyID,
		Name:      "Acme Corp",
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err := svc.AddTags(context.Background(), companyID, []uuid.UUID{uuid.New()})

	assert.ErrorIs(t, err, ErrTagNotFound)
}

// ============================================================================
// RemoveTags Tests
// ============================================================================

func TestService_RemoveTags_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	companyID := uuid.New()
	repo.companies[companyID] = &models.Company{
		ID:        companyID,
		Name:      "Acme Corp",
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	company, err := svc.RemoveTags(context.Background(), companyID, []uuid.UUID{uuid.New()})

	require.NoError(t, err)
	assert.Equal(t, companyID, company.ID)
}

func TestService_RemoveTags_CompanyNotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	_, err := svc.RemoveTags(context.Background(), uuid.New(), []uuid.UUID{uuid.New()})

	assert.ErrorIs(t, err, ErrCompanyNotFound)
}
