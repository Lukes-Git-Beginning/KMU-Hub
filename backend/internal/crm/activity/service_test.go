package activity

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
	activities   map[uuid.UUID]*models.Activity
	activityTags map[uuid.UUID][]*models.Tag
	customFields map[uuid.UUID]map[uuid.UUID]any
	contacts     map[uuid.UUID]string // contactID -> name
	companies    map[uuid.UUID]string // companyID -> name
	deals        map[uuid.UUID]string // dealID -> name
	users        map[uuid.UUID]string // userID -> name
	validTags    map[uuid.UUID]models.EntityType
	createErr    error
	getErr       error
	updateErr    error
	deleteErr    error
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		activities:   make(map[uuid.UUID]*models.Activity),
		activityTags: make(map[uuid.UUID][]*models.Tag),
		customFields: make(map[uuid.UUID]map[uuid.UUID]any),
		contacts:     make(map[uuid.UUID]string),
		companies:    make(map[uuid.UUID]string),
		deals:        make(map[uuid.UUID]string),
		users:        make(map[uuid.UUID]string),
		validTags:    make(map[uuid.UUID]models.EntityType),
	}
}

func (m *MockRepository) Create(ctx context.Context, activity *models.Activity) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.activities[activity.ID] = activity
	return nil
}

func (m *MockRepository) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*models.Activity, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	activity, ok := m.activities[id]
	if !ok {
		return nil, ErrActivityNotFound
	}
	return activity, nil
}

func (m *MockRepository) List(ctx context.Context, tenantID uuid.UUID, filter ListFilter, offset, limit int) ([]*models.Activity, int, error) {
	var result []*models.Activity
	for _, a := range m.activities {
		// Apply filter
		if filter.ActivityType != nil && a.ActivityType != *filter.ActivityType {
			continue
		}
		if filter.ContactID != nil && (a.ContactID == nil || *a.ContactID != *filter.ContactID) {
			continue
		}
		if filter.IsCompleted != nil && a.IsCompleted != *filter.IsCompleted {
			continue
		}
		result = append(result, a)
	}
	total := len(result)
	if offset >= len(result) {
		return []*models.Activity{}, total, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], total, nil
}

func (m *MockRepository) Update(ctx context.Context, activity *models.Activity) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.activities[activity.ID] = activity
	return nil
}

func (m *MockRepository) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.activities, id)
	return nil
}

func (m *MockRepository) GetContactName(ctx context.Context, contactID uuid.UUID) (string, error) {
	return m.contacts[contactID], nil
}

func (m *MockRepository) GetCompanyName(ctx context.Context, companyID uuid.UUID) (string, error) {
	return m.companies[companyID], nil
}

func (m *MockRepository) GetDealName(ctx context.Context, dealID uuid.UUID) (string, error) {
	return m.deals[dealID], nil
}

func (m *MockRepository) GetUserName(ctx context.Context, userID uuid.UUID) (string, error) {
	return m.users[userID], nil
}

func (m *MockRepository) GetTags(ctx context.Context, activityID uuid.UUID) ([]*models.Tag, error) {
	return m.activityTags[activityID], nil
}

func (m *MockRepository) AddTags(ctx context.Context, activityID uuid.UUID, tagIDs []uuid.UUID) error {
	return nil
}

func (m *MockRepository) RemoveTags(ctx context.Context, activityID uuid.UUID, tagIDs []uuid.UUID) error {
	return nil
}

func (m *MockRepository) GetCustomFieldValues(ctx context.Context, activityID uuid.UUID) ([]*models.CustomFieldValueRow, error) {
	return nil, nil
}

func (m *MockRepository) SetCustomFieldValues(ctx context.Context, activityID uuid.UUID, values map[uuid.UUID]any) error {
	m.customFields[activityID] = values
	return nil
}

func (m *MockRepository) ContactExists(ctx context.Context, contactID uuid.UUID) (bool, error) {
	_, exists := m.contacts[contactID]
	return exists, nil
}

func (m *MockRepository) CompanyExists(ctx context.Context, companyID uuid.UUID) (bool, error) {
	_, exists := m.companies[companyID]
	return exists, nil
}

func (m *MockRepository) DealExists(ctx context.Context, dealID uuid.UUID) (bool, error) {
	_, exists := m.deals[dealID]
	return exists, nil
}

func (m *MockRepository) UserExists(ctx context.Context, userID uuid.UUID) (bool, error) {
	_, exists := m.users[userID]
	return exists, nil
}

func (m *MockRepository) TagExists(ctx context.Context, tagID uuid.UUID, entityType models.EntityType) (bool, error) {
	et, exists := m.validTags[tagID]
	return exists && et == entityType, nil
}

// Test helpers
func (m *MockRepository) AddContact(contactID uuid.UUID, name string) {
	m.contacts[contactID] = name
}

func (m *MockRepository) AddCompany(companyID uuid.UUID, name string) {
	m.companies[companyID] = name
}

func (m *MockRepository) AddDeal(dealID uuid.UUID, name string) {
	m.deals[dealID] = name
}

func (m *MockRepository) AddUser(userID uuid.UUID, name string) {
	m.users[userID] = name
}

func (m *MockRepository) AddValidTag(tagID uuid.UUID, entityType models.EntityType) {
	m.validTags[tagID] = entityType
}

func (m *MockRepository) GetContactTimeline(_ context.Context, _ uuid.UUID, _, _ int) ([]*TimelineEvent, int, error) {
	return nil, 0, nil
}

// ============================================================================
// Create Tests
// ============================================================================

func TestService_Create_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := CreateInput{
		ActivityType: models.ActivityTypeCall,
		Subject:      "Sales Call",
		CreatedBy:    uuid.New(),
	}

	activity, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, activity.ID)
	assert.Equal(t, models.ActivityTypeCall, activity.ActivityType)
	assert.Equal(t, "Sales Call", activity.Subject)
	assert.False(t, activity.IsCompleted)
	assert.Nil(t, activity.CompletedAt)
}

func TestService_Create_WithAllRelations(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	contactID := uuid.New()
	repo.AddContact(contactID, "John Doe")

	companyID := uuid.New()
	repo.AddCompany(companyID, "Acme Corp")

	dealID := uuid.New()
	repo.AddDeal(dealID, "Big Deal")

	assignedTo := uuid.New()
	repo.AddUser(assignedTo, "Sales Rep")

	dueDate := time.Now().AddDate(0, 0, 7)
	description := "Follow up on proposal"

	input := CreateInput{
		ActivityType: models.ActivityTypeMeeting,
		Subject:      "Follow Up Meeting",
		Description:  &description,
		ContactID:    &contactID,
		CompanyID:    &companyID,
		DealID:       &dealID,
		AssignedTo:   &assignedTo,
		DueDate:      &dueDate,
		CreatedBy:    uuid.New(),
	}

	activity, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, models.ActivityTypeMeeting, activity.ActivityType)
	assert.Equal(t, "Follow Up Meeting", activity.Subject)
	assert.Equal(t, "Follow up on proposal", *activity.Description)
	assert.Equal(t, contactID, *activity.ContactID)
	assert.Equal(t, "John Doe", *activity.ContactName)
	assert.Equal(t, companyID, *activity.CompanyID)
	assert.Equal(t, "Acme Corp", *activity.CompanyName)
	assert.Equal(t, dealID, *activity.DealID)
	assert.Equal(t, "Big Deal", *activity.DealName)
	assert.Equal(t, assignedTo, *activity.AssignedTo)
	assert.Equal(t, "Sales Rep", *activity.AssignedToName)
}

func TestService_Create_AllActivityTypes(t *testing.T) {
	types := []models.ActivityType{
		models.ActivityTypeCall,
		models.ActivityTypeMeeting,
		models.ActivityTypeNote,
		models.ActivityTypeEmail,
		models.ActivityTypeTask,
	}

	for _, actType := range types {
		t.Run(string(actType), func(t *testing.T) {
			repo := NewMockRepository()
			svc := NewService(repo)

			input := CreateInput{
				ActivityType: actType,
				Subject:      "Test Activity",
				CreatedBy:    uuid.New(),
			}

			activity, err := svc.Create(context.Background(), input)

			require.NoError(t, err)
			assert.Equal(t, actType, activity.ActivityType)
		})
	}
}

func TestService_Create_SubjectRequired(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := CreateInput{
		ActivityType: models.ActivityTypeCall,
		Subject:      "",
		CreatedBy:    uuid.New(),
	}

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrSubjectRequired)
}

func TestService_Create_SubjectWhitespaceOnly(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := CreateInput{
		ActivityType: models.ActivityTypeCall,
		Subject:      "   ",
		CreatedBy:    uuid.New(),
	}

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrSubjectRequired)
}

func TestService_Create_InvalidActivityType(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := CreateInput{
		ActivityType: models.ActivityType("invalid"),
		Subject:      "Test",
		CreatedBy:    uuid.New(),
	}

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrInvalidActivityType)
}

func TestService_Create_ContactNotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	contactID := uuid.New() // Non-existent

	input := CreateInput{
		ActivityType: models.ActivityTypeCall,
		Subject:      "Test",
		ContactID:    &contactID,
		CreatedBy:    uuid.New(),
	}

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrContactNotFound)
}

func TestService_Create_CompanyNotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	companyID := uuid.New() // Non-existent

	input := CreateInput{
		ActivityType: models.ActivityTypeCall,
		Subject:      "Test",
		CompanyID:    &companyID,
		CreatedBy:    uuid.New(),
	}

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrCompanyNotFound)
}

func TestService_Create_DealNotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	dealID := uuid.New() // Non-existent

	input := CreateInput{
		ActivityType: models.ActivityTypeCall,
		Subject:      "Test",
		DealID:       &dealID,
		CreatedBy:    uuid.New(),
	}

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrDealNotFound)
}

func TestService_Create_UserNotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	userID := uuid.New() // Non-existent

	input := CreateInput{
		ActivityType: models.ActivityTypeTask,
		Subject:      "Test",
		AssignedTo:   &userID,
		CreatedBy:    uuid.New(),
	}

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestService_Create_TagNotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	input := CreateInput{
		ActivityType: models.ActivityTypeCall,
		Subject:      "Test",
		TagIDs:       []uuid.UUID{uuid.New()}, // Non-existent
		CreatedBy:    uuid.New(),
	}

	_, err := svc.Create(context.Background(), input)

	assert.ErrorIs(t, err, ErrTagNotFound)
}

func TestService_Create_WithTags(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	tagID := uuid.New()
	repo.AddValidTag(tagID, models.EntityTypeActivity)

	input := CreateInput{
		ActivityType: models.ActivityTypeCall,
		Subject:      "Test",
		TagIDs:       []uuid.UUID{tagID},
		CreatedBy:    uuid.New(),
	}

	activity, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, activity.ID)
}

// ============================================================================
// GetByID Tests
// ============================================================================

func TestService_GetByID_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	activityID := uuid.New()
	repo.activities[activityID] = &models.Activity{
		ID:           activityID,
		ActivityType: models.ActivityTypeCall,
		Subject:      "Test Call",
		CreatedBy:    uuid.New(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	activity, err := svc.GetByID(context.Background(), uuid.Nil, activityID)

	require.NoError(t, err)
	assert.Equal(t, activityID, activity.ID)
	assert.Equal(t, "Test Call", activity.Subject)
}

func TestService_GetByID_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	_, err := svc.GetByID(context.Background(), uuid.Nil, uuid.New())

	assert.ErrorIs(t, err, ErrActivityNotFound)
}

func TestService_GetByID_LoadsRelations(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	contactID := uuid.New()
	repo.AddContact(contactID, "John Doe")

	companyID := uuid.New()
	repo.AddCompany(companyID, "Acme Corp")

	activityID := uuid.New()
	repo.activities[activityID] = &models.Activity{
		ID:           activityID,
		ActivityType: models.ActivityTypeMeeting,
		Subject:      "Test Meeting",
		ContactID:    &contactID,
		CompanyID:    &companyID,
		CreatedBy:    uuid.New(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	activity, err := svc.GetByID(context.Background(), uuid.Nil, activityID)

	require.NoError(t, err)
	assert.Equal(t, "John Doe", *activity.ContactName)
	assert.Equal(t, "Acme Corp", *activity.CompanyName)
}

// ============================================================================
// List Tests
// ============================================================================

func TestService_List_All(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	for i := 0; i < 5; i++ {
		id := uuid.New()
		repo.activities[id] = &models.Activity{
			ID:           id,
			ActivityType: models.ActivityTypeCall,
			Subject:      "Activity",
			CreatedBy:    uuid.New(),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
	}

	activities, total, err := svc.List(context.Background(), ListInput{
		Page:     1,
		PageSize: 20,
	})

	require.NoError(t, err)
	assert.Len(t, activities, 5)
	assert.Equal(t, 5, total)
}

func TestService_List_FilterByType(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	// Add 3 calls
	for i := 0; i < 3; i++ {
		id := uuid.New()
		repo.activities[id] = &models.Activity{
			ID:           id,
			ActivityType: models.ActivityTypeCall,
			Subject:      "Call",
			CreatedBy:    uuid.New(),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
	}

	// Add 2 meetings
	for i := 0; i < 2; i++ {
		id := uuid.New()
		repo.activities[id] = &models.Activity{
			ID:           id,
			ActivityType: models.ActivityTypeMeeting,
			Subject:      "Meeting",
			CreatedBy:    uuid.New(),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
	}

	callType := models.ActivityTypeCall
	activities, total, err := svc.List(context.Background(), ListInput{
		ActivityType: &callType,
		Page:         1,
		PageSize:     20,
	})

	require.NoError(t, err)
	assert.Len(t, activities, 3)
	assert.Equal(t, 3, total)
}

func TestService_List_FilterByIsCompleted(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	// Add 2 completed
	for i := 0; i < 2; i++ {
		id := uuid.New()
		completedAt := time.Now()
		repo.activities[id] = &models.Activity{
			ID:           id,
			ActivityType: models.ActivityTypeTask,
			Subject:      "Completed Task",
			IsCompleted:  true,
			CompletedAt:  &completedAt,
			CreatedBy:    uuid.New(),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
	}

	// Add 3 not completed
	for i := 0; i < 3; i++ {
		id := uuid.New()
		repo.activities[id] = &models.Activity{
			ID:           id,
			ActivityType: models.ActivityTypeTask,
			Subject:      "Pending Task",
			IsCompleted:  false,
			CreatedBy:    uuid.New(),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
	}

	completed := true
	activities, total, err := svc.List(context.Background(), ListInput{
		IsCompleted: &completed,
		Page:        1,
		PageSize:    20,
	})

	require.NoError(t, err)
	assert.Len(t, activities, 2)
	assert.Equal(t, 2, total)
}

func TestService_List_Pagination(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	for i := 0; i < 15; i++ {
		id := uuid.New()
		repo.activities[id] = &models.Activity{
			ID:           id,
			ActivityType: models.ActivityTypeCall,
			Subject:      "Activity",
			CreatedBy:    uuid.New(),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
	}

	activities, total, err := svc.List(context.Background(), ListInput{
		Page:     1,
		PageSize: 5,
	})

	require.NoError(t, err)
	assert.Len(t, activities, 5)
	assert.Equal(t, 15, total)
}

// ============================================================================
// Update Tests
// ============================================================================

func TestService_Update_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	activityID := uuid.New()
	repo.activities[activityID] = &models.Activity{
		ID:           activityID,
		ActivityType: models.ActivityTypeCall,
		Subject:      "Old Subject",
		CreatedBy:    uuid.New(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	newSubject := "New Subject"
	newDescription := "Updated description"

	activity, err := svc.Update(context.Background(), uuid.Nil, activityID, UpdateInput{
		Subject:     &newSubject,
		Description: &newDescription,
	})

	require.NoError(t, err)
	assert.Equal(t, "New Subject", activity.Subject)
	assert.Equal(t, "Updated description", *activity.Description)
}

func TestService_Update_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	newSubject := "New Subject"
	_, err := svc.Update(context.Background(), uuid.Nil, uuid.New(), UpdateInput{
		Subject: &newSubject,
	})

	assert.ErrorIs(t, err, ErrActivityNotFound)
}

func TestService_Update_EmptySubject(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	activityID := uuid.New()
	repo.activities[activityID] = &models.Activity{
		ID:           activityID,
		ActivityType: models.ActivityTypeCall,
		Subject:      "Subject",
		CreatedBy:    uuid.New(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	emptySubject := ""
	_, err := svc.Update(context.Background(), uuid.Nil, activityID, UpdateInput{
		Subject: &emptySubject,
	})

	assert.ErrorIs(t, err, ErrSubjectRequired)
}

func TestService_Update_ClearContact(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	contactID := uuid.New()
	repo.AddContact(contactID, "John Doe")

	activityID := uuid.New()
	repo.activities[activityID] = &models.Activity{
		ID:           activityID,
		ActivityType: models.ActivityTypeCall,
		Subject:      "Call",
		ContactID:    &contactID,
		CreatedBy:    uuid.New(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	nilContact := uuid.Nil
	activity, err := svc.Update(context.Background(), uuid.Nil, activityID, UpdateInput{
		ContactID: &nilContact,
	})

	require.NoError(t, err)
	assert.Nil(t, activity.ContactID)
}

// ============================================================================
// Delete Tests
// ============================================================================

func TestService_Delete_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	activityID := uuid.New()
	repo.activities[activityID] = &models.Activity{
		ID:           activityID,
		ActivityType: models.ActivityTypeCall,
		Subject:      "Call",
		CreatedBy:    uuid.New(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err := svc.Delete(context.Background(), uuid.Nil, activityID)

	require.NoError(t, err)
	assert.NotContains(t, repo.activities, activityID)
}

func TestService_Delete_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	err := svc.Delete(context.Background(), uuid.Nil, uuid.New())

	assert.ErrorIs(t, err, ErrActivityNotFound)
}

// ============================================================================
// Complete/Uncomplete Tests
// ============================================================================

func TestService_Complete_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	activityID := uuid.New()
	repo.activities[activityID] = &models.Activity{
		ID:           activityID,
		ActivityType: models.ActivityTypeTask,
		Subject:      "Task",
		IsCompleted:  false,
		CreatedBy:    uuid.New(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	activity, err := svc.Complete(context.Background(), uuid.Nil, activityID)

	require.NoError(t, err)
	assert.True(t, activity.IsCompleted)
	assert.NotNil(t, activity.CompletedAt)
}

func TestService_Complete_AlreadyCompleted(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	completedAt := time.Now().Add(-time.Hour)
	activityID := uuid.New()
	repo.activities[activityID] = &models.Activity{
		ID:           activityID,
		ActivityType: models.ActivityTypeTask,
		Subject:      "Task",
		IsCompleted:  true,
		CompletedAt:  &completedAt,
		CreatedBy:    uuid.New(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	activity, err := svc.Complete(context.Background(), uuid.Nil, activityID)

	require.NoError(t, err)
	assert.True(t, activity.IsCompleted)
	// Should keep original completed_at
	assert.Equal(t, completedAt.Unix(), activity.CompletedAt.Unix())
}

func TestService_Complete_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	_, err := svc.Complete(context.Background(), uuid.Nil, uuid.New())

	assert.ErrorIs(t, err, ErrActivityNotFound)
}

func TestService_Uncomplete_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	completedAt := time.Now()
	activityID := uuid.New()
	repo.activities[activityID] = &models.Activity{
		ID:           activityID,
		ActivityType: models.ActivityTypeTask,
		Subject:      "Task",
		IsCompleted:  true,
		CompletedAt:  &completedAt,
		CreatedBy:    uuid.New(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	activity, err := svc.Uncomplete(context.Background(), uuid.Nil, activityID)

	require.NoError(t, err)
	assert.False(t, activity.IsCompleted)
	assert.Nil(t, activity.CompletedAt)
}

func TestService_Uncomplete_AlreadyNotCompleted(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	activityID := uuid.New()
	repo.activities[activityID] = &models.Activity{
		ID:           activityID,
		ActivityType: models.ActivityTypeTask,
		Subject:      "Task",
		IsCompleted:  false,
		CreatedBy:    uuid.New(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	activity, err := svc.Uncomplete(context.Background(), uuid.Nil, activityID)

	require.NoError(t, err)
	assert.False(t, activity.IsCompleted)
}

// ============================================================================
// Tags Tests
// ============================================================================

func TestService_AddTags_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	activityID := uuid.New()
	repo.activities[activityID] = &models.Activity{
		ID:           activityID,
		ActivityType: models.ActivityTypeCall,
		Subject:      "Call",
		CreatedBy:    uuid.New(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	tagID := uuid.New()
	repo.AddValidTag(tagID, models.EntityTypeActivity)

	activity, err := svc.AddTags(context.Background(), uuid.Nil, activityID, []uuid.UUID{tagID})

	require.NoError(t, err)
	assert.Equal(t, activityID, activity.ID)
}

func TestService_AddTags_TagNotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	activityID := uuid.New()
	repo.activities[activityID] = &models.Activity{
		ID:           activityID,
		ActivityType: models.ActivityTypeCall,
		Subject:      "Call",
		CreatedBy:    uuid.New(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	_, err := svc.AddTags(context.Background(), uuid.Nil, activityID, []uuid.UUID{uuid.New()})

	assert.ErrorIs(t, err, ErrTagNotFound)
}

func TestService_AddTags_WrongEntityType(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	activityID := uuid.New()
	repo.activities[activityID] = &models.Activity{
		ID:           activityID,
		ActivityType: models.ActivityTypeCall,
		Subject:      "Call",
		CreatedBy:    uuid.New(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	tagID := uuid.New()
	repo.AddValidTag(tagID, models.EntityTypeDeal) // Wrong type - should be activity

	_, err := svc.AddTags(context.Background(), uuid.Nil, activityID, []uuid.UUID{tagID})

	assert.ErrorIs(t, err, ErrTagNotFound)
}

func TestService_RemoveTags_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	activityID := uuid.New()
	repo.activities[activityID] = &models.Activity{
		ID:           activityID,
		ActivityType: models.ActivityTypeCall,
		Subject:      "Call",
		CreatedBy:    uuid.New(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	activity, err := svc.RemoveTags(context.Background(), uuid.Nil, activityID, []uuid.UUID{uuid.New()})

	require.NoError(t, err)
	assert.Equal(t, activityID, activity.ID)
}
