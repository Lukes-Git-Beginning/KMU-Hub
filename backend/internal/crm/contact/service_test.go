package contact

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
	contacts      map[uuid.UUID]*models.Contact
	contactTags   map[uuid.UUID][]*models.Tag
	customFields  map[uuid.UUID]map[uuid.UUID]any
	companies     map[uuid.UUID]string // companyID -> name
	validTags     map[uuid.UUID]models.EntityType
	inUseContacts map[uuid.UUID]bool
	createErr     error
	getErr        error
	listErr       error
	updateErr     error
	deleteErr     error
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		contacts:      make(map[uuid.UUID]*models.Contact),
		contactTags:   make(map[uuid.UUID][]*models.Tag),
		customFields:  make(map[uuid.UUID]map[uuid.UUID]any),
		companies:     make(map[uuid.UUID]string),
		validTags:     make(map[uuid.UUID]models.EntityType),
		inUseContacts: make(map[uuid.UUID]bool),
	}
}

func (m *MockRepository) Create(ctx context.Context, contact *models.Contact) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.contacts[contact.ID] = contact
	return nil
}

func (m *MockRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Contact, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	contact, ok := m.contacts[id]
	if !ok {
		return nil, ErrContactNotFound
	}
	return contact, nil
}

func (m *MockRepository) GetByEmail(ctx context.Context, email string) (*models.Contact, error) {
	for _, c := range m.contacts {
		if c.Email != nil && *c.Email == email {
			return c, nil
		}
	}
	return nil, ErrContactNotFound
}

func (m *MockRepository) List(ctx context.Context, filter ListFilter, offset, limit int) ([]*models.Contact, int, error) {
	if m.listErr != nil {
		return nil, 0, m.listErr
	}

	var result []*models.Contact
	for _, c := range m.contacts {
		result = append(result, c)
	}

	total := len(result)
	if offset >= len(result) {
		return []*models.Contact{}, total, nil
	}

	end := offset + limit
	if end > len(result) {
		end = len(result)
	}

	return result[offset:end], total, nil
}

func (m *MockRepository) Update(ctx context.Context, contact *models.Contact) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.contacts[contact.ID] = contact
	return nil
}

func (m *MockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.contacts, id)
	return nil
}

func (m *MockRepository) GetCompanyName(ctx context.Context, companyID uuid.UUID) (string, error) {
	return m.companies[companyID], nil
}

func (m *MockRepository) GetTags(ctx context.Context, contactID uuid.UUID) ([]*models.Tag, error) {
	return m.contactTags[contactID], nil
}

func (m *MockRepository) AddTags(ctx context.Context, contactID uuid.UUID, tagIDs []uuid.UUID) error {
	return nil
}

func (m *MockRepository) RemoveTags(ctx context.Context, contactID uuid.UUID, tagIDs []uuid.UUID) error {
	return nil
}

func (m *MockRepository) GetCustomFieldValues(ctx context.Context, contactID uuid.UUID) ([]*models.CustomFieldValueRow, error) {
	return nil, nil
}

func (m *MockRepository) SetCustomFieldValues(ctx context.Context, contactID uuid.UUID, values map[uuid.UUID]any) error {
	m.customFields[contactID] = values
	return nil
}

func (m *MockRepository) IsInUse(ctx context.Context, id uuid.UUID) (bool, error) {
	return m.inUseContacts[id], nil
}

func (m *MockRepository) CompanyExists(ctx context.Context, companyID uuid.UUID) (bool, error) {
	_, exists := m.companies[companyID]
	return exists, nil
}

func (m *MockRepository) TagExists(ctx context.Context, tagID uuid.UUID, entityType models.EntityType) (bool, error) {
	et, exists := m.validTags[tagID]
	return exists && et == entityType, nil
}

func (m *MockRepository) ListWithVisibility(ctx context.Context, userID uuid.UUID, isAdmin bool, filter ListFilter, offset, limit int) ([]*models.Contact, int, error) {
	return m.List(ctx, filter, offset, limit)
}

func (m *MockRepository) ListByIDs(ctx context.Context, ids []uuid.UUID) ([]*models.Contact, error) {
	var result []*models.Contact
	for _, id := range ids {
		if c, ok := m.contacts[id]; ok {
			result = append(result, c)
		}
	}
	return result, nil
}

func (m *MockRepository) ListAll(ctx context.Context, userID uuid.UUID, isAdmin bool) ([]*models.Contact, error) {
	var result []*models.Contact
	for _, c := range m.contacts {
		result = append(result, c)
	}
	return result, nil
}

func (m *MockRepository) UpdateVisibility(ctx context.Context, contactID uuid.UUID, visibility string, ownerID *uuid.UUID) error {
	if c, ok := m.contacts[contactID]; ok {
		c.Visibility = visibility
		c.OwnerID = ownerID
	}
	return nil
}

// Helpers for test setup
func (m *MockRepository) AddCompany(companyID uuid.UUID, name string) {
	m.companies[companyID] = name
}

func (m *MockRepository) AddValidTag(tagID uuid.UUID, entityType models.EntityType) {
	m.validTags[tagID] = entityType
}

func (m *MockRepository) SetInUse(contactID uuid.UUID, inUse bool) {
	m.inUseContacts[contactID] = inUse
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
		FirstName: "John",
		LastName:  "Doe",
		CreatedBy: uuid.New(),
	}

	contact, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, contact.ID)
	assert.Equal(t, "John", contact.FirstName)
	assert.Equal(t, "Doe", contact.LastName)
	assert.Nil(t, contact.Email)
	assert.NotZero(t, contact.CreatedAt)
	assert.NotZero(t, contact.UpdatedAt)
}

func TestService_Create_WithEmail(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	email := "john@example.com"
	input := CreateInput{
		FirstName: "John",
		LastName:  "Doe",
		Email:     &email,
		CreatedBy: uuid.New(),
	}

	contact, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, "john@example.com", *contact.Email)
}

func TestService_Create_WithCompany(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	companyID := uuid.New()
	repo.AddCompany(companyID, "Acme Corp")

	input := CreateInput{
		FirstName: "John",
		LastName:  "Doe",
		CompanyID: &companyID,
		CreatedBy: uuid.New(),
	}

	contact, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, companyID, *contact.CompanyID)
	assert.Equal(t, "Acme Corp", *contact.CompanyName)
}

func TestService_Create_WithTags(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	tagID := uuid.New()
	repo.AddValidTag(tagID, models.EntityTypeContact)

	input := CreateInput{
		FirstName: "John",
		LastName:  "Doe",
		TagIDs:    []uuid.UUID{tagID},
		CreatedBy: uuid.New(),
	}

	contact, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, "John", contact.FirstName)
}

func TestService_Create_AllFields(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	companyID := uuid.New()
	repo.AddCompany(companyID, "Acme Corp")

	email := "john@example.com"
	phone := "+49123456789"
	position := "CEO"
	notes := "Important contact"

	input := CreateInput{
		FirstName: "John",
		LastName:  "Doe",
		Email:     &email,
		Phone:     &phone,
		CompanyID: &companyID,
		Position:  &position,
		Notes:     &notes,
		CreatedBy: uuid.New(),
	}

	contact, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, "John", contact.FirstName)
	assert.Equal(t, "Doe", contact.LastName)
	assert.Equal(t, "john@example.com", *contact.Email)
	assert.Equal(t, "+49123456789", *contact.Phone)
	assert.Equal(t, companyID, *contact.CompanyID)
	assert.Equal(t, "CEO", *contact.Position)
	assert.Equal(t, "Important contact", *contact.Notes)
}

func TestService_Create_FirstNameRequired(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := CreateInput{
		FirstName: "",
		LastName:  "Doe",
		CreatedBy: uuid.New(),
	}

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrFirstNameRequired)
}

func TestService_Create_LastNameRequired(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := CreateInput{
		FirstName: "John",
		LastName:  "",
		CreatedBy: uuid.New(),
	}

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrLastNameRequired)
}

func TestService_Create_InvalidEmail(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	invalidEmail := "not-an-email"
	input := CreateInput{
		FirstName: "John",
		LastName:  "Doe",
		Email:     &invalidEmail,
		CreatedBy: uuid.New(),
	}

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrInvalidEmail)
}

func TestService_Create_DuplicateEmail(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	email := "john@example.com"
	existingID := uuid.New()
	repo.contacts[existingID] = &models.Contact{
		ID:        existingID,
		FirstName: "Existing",
		LastName:  "User",
		Email:     &email,
	}

	input := CreateInput{
		FirstName: "John",
		LastName:  "Doe",
		Email:     &email,
		CreatedBy: uuid.New(),
	}

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrEmailExists)
}

func TestService_Create_CompanyNotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	companyID := uuid.New() // Non-existent company

	input := CreateInput{
		FirstName: "John",
		LastName:  "Doe",
		CompanyID: &companyID,
		CreatedBy: uuid.New(),
	}

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrCompanyNotFound)
}

func TestService_Create_InvalidTag(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := CreateInput{
		FirstName: "John",
		LastName:  "Doe",
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
	repo.AddValidTag(tagID, models.EntityTypeCompany) // Company tag, not contact

	input := CreateInput{
		FirstName: "John",
		LastName:  "Doe",
		TagIDs:    []uuid.UUID{tagID},
		CreatedBy: uuid.New(),
	}

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrTagNotFound)
}

func TestService_Create_NamesTrimmed(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := CreateInput{
		FirstName: "  John  ",
		LastName:  "  Doe  ",
		CreatedBy: uuid.New(),
	}

	contact, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, "John", contact.FirstName)
	assert.Equal(t, "Doe", contact.LastName)
}

// ============================================================================
// GetByID Tests
// ============================================================================

func TestService_GetByID_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	contactID := uuid.New()
	repo.contacts[contactID] = &models.Contact{
		ID:        contactID,
		FirstName: "John",
		LastName:  "Doe",
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	contact, err := svc.GetByID(context.Background(), contactID)

	require.NoError(t, err)
	assert.Equal(t, contactID, contact.ID)
	assert.Equal(t, "John", contact.FirstName)
}

func TestService_GetByID_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	_, err := svc.GetByID(context.Background(), uuid.New())

	assert.ErrorIs(t, err, ErrContactNotFound)
}

// ============================================================================
// List Tests
// ============================================================================

func TestService_List_Empty(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	contacts, total, err := svc.List(context.Background(), ListInput{})

	require.NoError(t, err)
	assert.Empty(t, contacts)
	assert.Equal(t, 0, total)
}

func TestService_List_WithContacts(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	for i := 0; i < 3; i++ {
		id := uuid.New()
		repo.contacts[id] = &models.Contact{
			ID:        id,
			FirstName: "User",
			LastName:  string(rune('A' + i)),
			CreatedBy: uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	contacts, total, err := svc.List(context.Background(), ListInput{})

	require.NoError(t, err)
	assert.Len(t, contacts, 3)
	assert.Equal(t, 3, total)
}

func TestService_List_DefaultPagination(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	for i := 0; i < 25; i++ {
		id := uuid.New()
		repo.contacts[id] = &models.Contact{
			ID:        id,
			FirstName: "User",
			LastName:  "Test",
			CreatedBy: uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	contacts, total, err := svc.List(context.Background(), ListInput{
		Page:     0, // Should default to 1
		PageSize: 0, // Should default to 20
	})

	require.NoError(t, err)
	assert.Len(t, contacts, 20)
	assert.Equal(t, 25, total)
}

// ============================================================================
// Update Tests
// ============================================================================

func TestService_Update_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	contactID := uuid.New()
	repo.contacts[contactID] = &models.Contact{
		ID:        contactID,
		FirstName: "Old",
		LastName:  "Name",
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	newFirst := "New"
	newLast := "Updated"
	contact, err := svc.Update(context.Background(), contactID, UpdateInput{
		FirstName: &newFirst,
		LastName:  &newLast,
	})

	require.NoError(t, err)
	assert.Equal(t, "New", contact.FirstName)
	assert.Equal(t, "Updated", contact.LastName)
}

func TestService_Update_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	newFirst := "New"
	_, err := svc.Update(context.Background(), uuid.New(), UpdateInput{
		FirstName: &newFirst,
	})

	assert.ErrorIs(t, err, ErrContactNotFound)
}

func TestService_Update_EmptyFirstName(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	contactID := uuid.New()
	repo.contacts[contactID] = &models.Contact{
		ID:        contactID,
		FirstName: "John",
		LastName:  "Doe",
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	emptyName := ""
	_, err := svc.Update(context.Background(), contactID, UpdateInput{
		FirstName: &emptyName,
	})

	assert.ErrorIs(t, err, ErrFirstNameRequired)
}

func TestService_Update_ClearCompany(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	companyID := uuid.New()
	repo.AddCompany(companyID, "Acme Corp")

	contactID := uuid.New()
	repo.contacts[contactID] = &models.Contact{
		ID:        contactID,
		FirstName: "John",
		LastName:  "Doe",
		CompanyID: &companyID,
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	nilCompany := uuid.Nil
	contact, err := svc.Update(context.Background(), contactID, UpdateInput{
		CompanyID: &nilCompany,
	})

	require.NoError(t, err)
	assert.Nil(t, contact.CompanyID)
}

func TestService_Update_DuplicateEmail(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	existingEmail := "existing@example.com"
	existingID := uuid.New()
	repo.contacts[existingID] = &models.Contact{
		ID:        existingID,
		FirstName: "Existing",
		LastName:  "User",
		Email:     &existingEmail,
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	contactID := uuid.New()
	repo.contacts[contactID] = &models.Contact{
		ID:        contactID,
		FirstName: "John",
		LastName:  "Doe",
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err := svc.Update(context.Background(), contactID, UpdateInput{
		Email: &existingEmail,
	})

	assert.ErrorIs(t, err, ErrEmailExists)
}

func TestService_Update_SameEmailSameContact(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	email := "john@example.com"
	contactID := uuid.New()
	repo.contacts[contactID] = &models.Contact{
		ID:        contactID,
		FirstName: "John",
		LastName:  "Doe",
		Email:     &email,
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Updating with same email should work
	contact, err := svc.Update(context.Background(), contactID, UpdateInput{
		Email: &email,
	})

	require.NoError(t, err)
	assert.Equal(t, email, *contact.Email)
}

// ============================================================================
// Delete Tests
// ============================================================================

func TestService_Delete_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	contactID := uuid.New()
	repo.contacts[contactID] = &models.Contact{
		ID:        contactID,
		FirstName: "John",
		LastName:  "Doe",
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := svc.Delete(context.Background(), contactID)

	require.NoError(t, err)
	assert.NotContains(t, repo.contacts, contactID)
}

func TestService_Delete_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	err := svc.Delete(context.Background(), uuid.New())

	assert.ErrorIs(t, err, ErrContactNotFound)
}

func TestService_Delete_InUse(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	contactID := uuid.New()
	repo.contacts[contactID] = &models.Contact{
		ID:        contactID,
		FirstName: "John",
		LastName:  "Doe",
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.SetInUse(contactID, true)

	err := svc.Delete(context.Background(), contactID)

	assert.ErrorIs(t, err, ErrContactInUse)
	assert.Contains(t, repo.contacts, contactID) // Not deleted
}

// ============================================================================
// AddTags Tests
// ============================================================================

func TestService_AddTags_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	contactID := uuid.New()
	repo.contacts[contactID] = &models.Contact{
		ID:        contactID,
		FirstName: "John",
		LastName:  "Doe",
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tagID := uuid.New()
	repo.AddValidTag(tagID, models.EntityTypeContact)

	contact, err := svc.AddTags(context.Background(), contactID, []uuid.UUID{tagID})

	require.NoError(t, err)
	assert.Equal(t, contactID, contact.ID)
}

func TestService_AddTags_ContactNotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	tagID := uuid.New()
	repo.AddValidTag(tagID, models.EntityTypeContact)

	_, err := svc.AddTags(context.Background(), uuid.New(), []uuid.UUID{tagID})

	assert.ErrorIs(t, err, ErrContactNotFound)
}

func TestService_AddTags_InvalidTag(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	contactID := uuid.New()
	repo.contacts[contactID] = &models.Contact{
		ID:        contactID,
		FirstName: "John",
		LastName:  "Doe",
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err := svc.AddTags(context.Background(), contactID, []uuid.UUID{uuid.New()})

	assert.ErrorIs(t, err, ErrTagNotFound)
}

// ============================================================================
// RemoveTags Tests
// ============================================================================

func TestService_RemoveTags_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	contactID := uuid.New()
	repo.contacts[contactID] = &models.Contact{
		ID:        contactID,
		FirstName: "John",
		LastName:  "Doe",
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	contact, err := svc.RemoveTags(context.Background(), contactID, []uuid.UUID{uuid.New()})

	require.NoError(t, err)
	assert.Equal(t, contactID, contact.ID)
}

func TestService_RemoveTags_ContactNotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	_, err := svc.RemoveTags(context.Background(), uuid.New(), []uuid.UUID{uuid.New()})

	assert.ErrorIs(t, err, ErrContactNotFound)
}
