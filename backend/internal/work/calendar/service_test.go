package calendar

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ============================================================================
// MockRepository
// ============================================================================

type MockRepository struct {
	calendars       map[uuid.UUID]*models.Calendar
	members         map[uuid.UUID]map[uuid.UUID]*models.CalendarMember // calendarID -> userID -> member
	categories      map[uuid.UUID]*models.EventCategory
	catsByUser      map[uuid.UUID][]uuid.UUID // userID -> category IDs
	preferences     map[uuid.UUID]*models.UserCalendarPreferences
	personalCreated map[uuid.UUID]bool // track if personal calendar was auto-created

	createErr    error
	getErr       error
	updateErr    error
	deleteErr    error
	addMemErr    error
	catCreateErr error
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		calendars:       make(map[uuid.UUID]*models.Calendar),
		members:         make(map[uuid.UUID]map[uuid.UUID]*models.CalendarMember),
		categories:      make(map[uuid.UUID]*models.EventCategory),
		catsByUser:      make(map[uuid.UUID][]uuid.UUID),
		preferences:     make(map[uuid.UUID]*models.UserCalendarPreferences),
		personalCreated: make(map[uuid.UUID]bool),
	}
}

func (m *MockRepository) Create(ctx context.Context, calendar *models.Calendar) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.calendars[calendar.ID] = calendar
	return nil
}

func (m *MockRepository) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*models.Calendar, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	c, ok := m.calendars[id]
	if !ok {
		return nil, ErrCalendarNotFound
	}
	// Tenant isolation check
	if c.TenantID != uuid.Nil && c.TenantID != tenantID {
		return nil, ErrCalendarNotFound
	}
	return c, nil
}

func (m *MockRepository) ListByUser(ctx context.Context, userID, tenantID uuid.UUID) ([]models.CalendarWithMemberInfo, error) {
	var result []models.CalendarWithMemberInfo
	for _, c := range m.calendars {
		// Tenant isolation
		if c.TenantID != uuid.Nil && c.TenantID != tenantID {
			continue
		}
		// Calendar owned by user
		if c.OwnerID == userID {
			result = append(result, models.CalendarWithMemberInfo{
				Calendar:   *c,
				Permission: models.CalendarPermissionAdmin,
				IsVisible:  true,
			})
			continue
		}
		// Calendar user is a member of
		if mems, ok := m.members[c.ID]; ok {
			if mem, exists := mems[userID]; exists {
				result = append(result, models.CalendarWithMemberInfo{
					Calendar:      *c,
					Permission:    mem.Permission,
					ColorOverride: mem.ColorOverride,
					IsVisible:     mem.IsVisible,
				})
			}
		}
	}
	return result, nil
}

func (m *MockRepository) Update(ctx context.Context, calendar *models.Calendar) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.calendars[calendar.ID] = calendar
	return nil
}

func (m *MockRepository) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.calendars, id)
	delete(m.members, id)
	return nil
}

func (m *MockRepository) AddMember(ctx context.Context, member *models.CalendarMember) error {
	if m.addMemErr != nil {
		return m.addMemErr
	}
	if _, ok := m.members[member.CalendarID]; !ok {
		m.members[member.CalendarID] = make(map[uuid.UUID]*models.CalendarMember)
	}
	if _, exists := m.members[member.CalendarID][member.UserID]; exists {
		return ErrMemberAlreadyExists
	}
	m.members[member.CalendarID][member.UserID] = member
	return nil
}

func (m *MockRepository) RemoveMember(ctx context.Context, calendarID, userID uuid.UUID) error {
	if mems, ok := m.members[calendarID]; ok {
		if _, exists := mems[userID]; exists {
			delete(mems, userID)
			return nil
		}
	}
	return ErrMemberNotFound
}

func (m *MockRepository) ListMembers(ctx context.Context, calendarID uuid.UUID) ([]models.CalendarMember, error) {
	var result []models.CalendarMember
	if mems, ok := m.members[calendarID]; ok {
		for _, mem := range mems {
			result = append(result, *mem)
		}
	}
	return result, nil
}

func (m *MockRepository) GetMember(ctx context.Context, calendarID, userID uuid.UUID) (*models.CalendarMember, error) {
	if mems, ok := m.members[calendarID]; ok {
		if mem, exists := mems[userID]; exists {
			return mem, nil
		}
	}
	return nil, ErrMemberNotFound
}

func (m *MockRepository) UpdateMemberPermission(ctx context.Context, calendarID, userID uuid.UUID, permission string) error {
	if mems, ok := m.members[calendarID]; ok {
		if mem, exists := mems[userID]; exists {
			mem.Permission = permission
			return nil
		}
	}
	return ErrMemberNotFound
}

func (m *MockRepository) UpdateMemberVisibility(ctx context.Context, calendarID, userID uuid.UUID, visible bool) error {
	if mems, ok := m.members[calendarID]; ok {
		if mem, exists := mems[userID]; exists {
			mem.IsVisible = visible
			return nil
		}
	}
	return ErrMemberNotFound
}

func (m *MockRepository) UpdateMemberColorOverride(ctx context.Context, calendarID, userID uuid.UUID, color *string) error {
	if mems, ok := m.members[calendarID]; ok {
		if mem, exists := mems[userID]; exists {
			mem.ColorOverride = color
			return nil
		}
	}
	return ErrMemberNotFound
}

func (m *MockRepository) ListBrowsable(ctx context.Context, userID, tenantID uuid.UUID) ([]models.Calendar, error) {
	var result []models.Calendar
	for _, c := range m.calendars {
		if c.TenantID != uuid.Nil && c.TenantID != tenantID {
			continue
		}
		if c.CalendarType != models.CalendarTypeShared {
			continue
		}
		if c.OwnerID == userID {
			continue
		}
		if mems, ok := m.members[c.ID]; ok {
			if _, isMember := mems[userID]; isMember {
				continue
			}
		}
		result = append(result, *c)
	}
	return result, nil
}

func (m *MockRepository) Subscribe(ctx context.Context, calendarID, userID uuid.UUID) error {
	if _, ok := m.members[calendarID]; !ok {
		m.members[calendarID] = make(map[uuid.UUID]*models.CalendarMember)
	}
	if _, exists := m.members[calendarID][userID]; exists {
		return ErrSubscriptionExists
	}
	m.members[calendarID][userID] = &models.CalendarMember{
		CalendarID: calendarID,
		UserID:     userID,
		Permission: models.CalendarPermissionView,
		IsVisible:  true,
		CreatedAt:  time.Now(),
	}
	return nil
}

func (m *MockRepository) Unsubscribe(ctx context.Context, calendarID, userID uuid.UUID) error {
	if mems, ok := m.members[calendarID]; ok {
		if _, exists := mems[userID]; exists {
			delete(mems, userID)
			return nil
		}
	}
	return ErrSubscriptionNotFound
}

func (m *MockRepository) CreateCategory(ctx context.Context, category *models.EventCategory) error {
	if m.catCreateErr != nil {
		return m.catCreateErr
	}
	// Check for duplicate name within user+tenant
	for _, catID := range m.catsByUser[category.UserID] {
		existing := m.categories[catID]
		if existing != nil && existing.Name == category.Name {
			return ErrDuplicateCategoryName
		}
	}
	m.categories[category.ID] = category
	m.catsByUser[category.UserID] = append(m.catsByUser[category.UserID], category.ID)
	return nil
}

func (m *MockRepository) ListCategories(ctx context.Context, userID, tenantID uuid.UUID) ([]models.EventCategory, error) {
	var result []models.EventCategory
	for _, catID := range m.catsByUser[userID] {
		if cat, ok := m.categories[catID]; ok {
			if cat.TenantID == uuid.Nil || cat.TenantID == tenantID {
				result = append(result, *cat)
			}
		}
	}
	return result, nil
}

func (m *MockRepository) DeleteCategory(ctx context.Context, id, userID, tenantID uuid.UUID) error {
	cat, ok := m.categories[id]
	if !ok || cat.UserID != userID {
		return ErrCategoryNotFound
	}
	if cat.TenantID != uuid.Nil && cat.TenantID != tenantID {
		return ErrCategoryNotFound
	}
	delete(m.categories, id)
	// Remove from user's list
	ids := m.catsByUser[userID]
	for i, catID := range ids {
		if catID == id {
			m.catsByUser[userID] = append(ids[:i], ids[i+1:]...)
			break
		}
	}
	return nil
}

func (m *MockRepository) GetPreferences(ctx context.Context, userID uuid.UUID) (*models.UserCalendarPreferences, error) {
	prefs, ok := m.preferences[userID]
	if !ok {
		return nil, nil
	}
	return prefs, nil
}

func (m *MockRepository) UpsertPreferences(ctx context.Context, prefs *models.UserCalendarPreferences) error {
	m.preferences[prefs.UserID] = prefs
	return nil
}

func (m *MockRepository) EnsurePersonalCalendar(ctx context.Context, userID, tenantID uuid.UUID) (*models.Calendar, error) {
	// Check if personal calendar already exists
	for _, c := range m.calendars {
		if c.OwnerID == userID && c.IsDefault && c.CalendarType == models.CalendarTypePersonal {
			return c, nil
		}
	}

	// Create one
	now := time.Now()
	cal := &models.Calendar{
		ID:           uuid.New(),
		TenantID:     tenantID,
		Name:         "Mein Kalender",
		CalendarType: models.CalendarTypePersonal,
		Color:        "#4285F4",
		OwnerID:      userID,
		IsDefault:    true,
		Timezone:     "Europe/Berlin",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	m.calendars[cal.ID] = cal
	m.personalCreated[userID] = true
	return cal, nil
}

// Test helpers

func (m *MockRepository) addCalendar(c *models.Calendar) {
	m.calendars[c.ID] = c
}

func (m *MockRepository) addMemberDirect(calendarID, userID uuid.UUID, permission string) {
	if _, ok := m.members[calendarID]; !ok {
		m.members[calendarID] = make(map[uuid.UUID]*models.CalendarMember)
	}
	m.members[calendarID][userID] = &models.CalendarMember{
		CalendarID: calendarID,
		UserID:     userID,
		Permission: permission,
		IsVisible:  true,
		CreatedAt:  time.Now(),
	}
}

// tenantID used by all tests
var testTenantID = uuid.New()

// ============================================================================
// Create Tests
// ============================================================================

func TestService_Create_PersonalCalendar(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	userID := uuid.New()

	cal, err := svc.Create(context.Background(), userID, testTenantID, CreateInput{
		Name:         "Work Calendar",
		CalendarType: models.CalendarTypePersonal,
		Color:        "#FF5733",
	})

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, cal.ID)
	assert.Equal(t, "Work Calendar", cal.Name)
	assert.Equal(t, models.CalendarTypePersonal, cal.CalendarType)
	assert.Equal(t, "#FF5733", cal.Color)
	assert.Equal(t, userID, cal.OwnerID)
	assert.Equal(t, testTenantID, cal.TenantID)
	assert.False(t, cal.IsDefault)
	assert.Equal(t, "Europe/Berlin", cal.Timezone)
}

func TestService_Create_SharedCalendar(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	userID := uuid.New()

	cal, err := svc.Create(context.Background(), userID, testTenantID, CreateInput{
		Name:         "Team Calendar",
		CalendarType: models.CalendarTypeShared,
		Color:        "#34A853",
		Timezone:     "Europe/Zurich",
	})

	require.NoError(t, err)
	assert.Equal(t, models.CalendarTypeShared, cal.CalendarType)
	assert.Equal(t, "Europe/Zurich", cal.Timezone)
}

func TestService_Create_DefaultColor(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	cal, err := svc.Create(context.Background(), uuid.New(), testTenantID, CreateInput{
		Name:         "Calendar",
		CalendarType: models.CalendarTypePersonal,
	})

	require.NoError(t, err)
	assert.Equal(t, "#4285F4", cal.Color)
}

func TestService_Create_EmptyName(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	_, err := svc.Create(context.Background(), uuid.New(), testTenantID, CreateInput{
		Name:         "",
		CalendarType: models.CalendarTypePersonal,
	})

	assert.ErrorIs(t, err, ErrCalendarNameRequired)
}

func TestService_Create_InvalidType(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	_, err := svc.Create(context.Background(), uuid.New(), testTenantID, CreateInput{
		Name:         "Calendar",
		CalendarType: "invalid",
	})

	assert.ErrorIs(t, err, ErrInvalidCalendarType)
}

func TestService_Create_InvalidColor(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	_, err := svc.Create(context.Background(), uuid.New(), testTenantID, CreateInput{
		Name:         "Calendar",
		CalendarType: models.CalendarTypePersonal,
		Color:        "not-a-color",
	})

	assert.ErrorIs(t, err, ErrInvalidColor)
}

// ============================================================================
// ListByUser Tests
// ============================================================================

func TestService_ListByUser_CreatesPersonalCalendar(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	userID := uuid.New()

	calendars, err := svc.ListByUser(context.Background(), userID, testTenantID)

	require.NoError(t, err)
	assert.True(t, repo.personalCreated[userID])
	// Should include the auto-created personal calendar
	assert.GreaterOrEqual(t, len(calendars), 1)
}

func TestService_ListByUser_IncludesOwnedAndSubscribed(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	userID := uuid.New()
	otherID := uuid.New()

	// User's personal calendar
	personalCal := &models.Calendar{
		ID: uuid.New(), TenantID: testTenantID, Name: "Mine", CalendarType: models.CalendarTypePersonal,
		OwnerID: userID, IsDefault: true, Color: "#4285F4", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	repo.addCalendar(personalCal)

	// Shared calendar user is subscribed to
	sharedCal := &models.Calendar{
		ID: uuid.New(), TenantID: testTenantID, Name: "Team", CalendarType: models.CalendarTypeShared,
		OwnerID: otherID, Color: "#34A853", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	repo.addCalendar(sharedCal)
	repo.addMemberDirect(sharedCal.ID, userID, models.CalendarPermissionView)

	// Other calendar user is NOT subscribed to
	otherCal := &models.Calendar{
		ID: uuid.New(), TenantID: testTenantID, Name: "Other Team", CalendarType: models.CalendarTypeShared,
		OwnerID: otherID, Color: "#EA4335", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	repo.addCalendar(otherCal)

	calendars, err := svc.ListByUser(context.Background(), userID, testTenantID)

	require.NoError(t, err)
	// Should have: personal + subscribed shared (not the other)
	assert.Len(t, calendars, 2)
}

// ============================================================================
// Update Tests
// ============================================================================

func TestService_Update_AsOwner(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	ownerID := uuid.New()
	calID := uuid.New()

	repo.addCalendar(&models.Calendar{
		ID: calID, TenantID: testTenantID, Name: "Old Name", CalendarType: models.CalendarTypePersonal,
		OwnerID: ownerID, Color: "#4285F4", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	newName := "New Name"
	cal, err := svc.Update(context.Background(), calID, ownerID, testTenantID, UpdateInput{
		Name: &newName,
	})

	require.NoError(t, err)
	assert.Equal(t, "New Name", cal.Name)
}

func TestService_Update_AsAdmin(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	ownerID := uuid.New()
	adminID := uuid.New()
	calID := uuid.New()

	repo.addCalendar(&models.Calendar{
		ID: calID, TenantID: testTenantID, Name: "Calendar", CalendarType: models.CalendarTypeShared,
		OwnerID: ownerID, Color: "#4285F4", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
	repo.addMemberDirect(calID, adminID, models.CalendarPermissionAdmin)

	newName := "Updated"
	cal, err := svc.Update(context.Background(), calID, adminID, testTenantID, UpdateInput{
		Name: &newName,
	})

	require.NoError(t, err)
	assert.Equal(t, "Updated", cal.Name)
}

func TestService_Update_AsViewer_Fails(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	ownerID := uuid.New()
	viewerID := uuid.New()
	calID := uuid.New()

	repo.addCalendar(&models.Calendar{
		ID: calID, TenantID: testTenantID, Name: "Calendar", CalendarType: models.CalendarTypeShared,
		OwnerID: ownerID, Color: "#4285F4", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
	repo.addMemberDirect(calID, viewerID, models.CalendarPermissionView)

	newName := "Hacked"
	_, err := svc.Update(context.Background(), calID, viewerID, testTenantID, UpdateInput{
		Name: &newName,
	})

	assert.ErrorIs(t, err, ErrInsufficientPermission)
}

func TestService_Update_EmptyName(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	ownerID := uuid.New()
	calID := uuid.New()

	repo.addCalendar(&models.Calendar{
		ID: calID, TenantID: testTenantID, Name: "Calendar", CalendarType: models.CalendarTypePersonal,
		OwnerID: ownerID, Color: "#4285F4", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	emptyName := ""
	_, err := svc.Update(context.Background(), calID, ownerID, testTenantID, UpdateInput{
		Name: &emptyName,
	})

	assert.ErrorIs(t, err, ErrCalendarNameRequired)
}

func TestService_Update_InvalidColor(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	ownerID := uuid.New()
	calID := uuid.New()

	repo.addCalendar(&models.Calendar{
		ID: calID, TenantID: testTenantID, Name: "Calendar", CalendarType: models.CalendarTypePersonal,
		OwnerID: ownerID, Color: "#4285F4", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	badColor := "red"
	_, err := svc.Update(context.Background(), calID, ownerID, testTenantID, UpdateInput{
		Color: &badColor,
	})

	assert.ErrorIs(t, err, ErrInvalidColor)
}

func TestService_Update_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	newName := "Test"
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), testTenantID, UpdateInput{
		Name: &newName,
	})

	assert.ErrorIs(t, err, ErrCalendarNotFound)
}

// ============================================================================
// Delete Tests
// ============================================================================

func TestService_Delete_AsOwner(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	ownerID := uuid.New()
	calID := uuid.New()

	repo.addCalendar(&models.Calendar{
		ID: calID, TenantID: testTenantID, Name: "To Delete", CalendarType: models.CalendarTypePersonal,
		OwnerID: ownerID, Color: "#4285F4", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	err := svc.Delete(context.Background(), calID, ownerID, testTenantID)

	require.NoError(t, err)
	assert.NotContains(t, repo.calendars, calID)
}

func TestService_Delete_NotOwner_Fails(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	calID := uuid.New()

	repo.addCalendar(&models.Calendar{
		ID: calID, TenantID: testTenantID, Name: "Calendar", CalendarType: models.CalendarTypeShared,
		OwnerID: uuid.New(), Color: "#4285F4", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	err := svc.Delete(context.Background(), calID, uuid.New(), testTenantID)

	assert.ErrorIs(t, err, ErrNotCalendarOwner)
}

func TestService_Delete_DefaultCalendar_Fails(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	ownerID := uuid.New()
	calID := uuid.New()

	repo.addCalendar(&models.Calendar{
		ID: calID, TenantID: testTenantID, Name: "Default", CalendarType: models.CalendarTypePersonal,
		OwnerID: ownerID, IsDefault: true, Color: "#4285F4", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	err := svc.Delete(context.Background(), calID, ownerID, testTenantID)

	assert.ErrorIs(t, err, ErrCannotDeleteDefaultCalendar)
}

func TestService_Delete_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	err := svc.Delete(context.Background(), uuid.New(), uuid.New(), testTenantID)

	assert.ErrorIs(t, err, ErrCalendarNotFound)
}

// ============================================================================
// AddMember Tests
// ============================================================================

func TestService_AddMember_AsOwner(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	ownerID := uuid.New()
	targetID := uuid.New()
	calID := uuid.New()

	repo.addCalendar(&models.Calendar{
		ID: calID, TenantID: testTenantID, Name: "Shared", CalendarType: models.CalendarTypeShared,
		OwnerID: ownerID, Color: "#4285F4", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	err := svc.AddMember(context.Background(), calID, ownerID, targetID, testTenantID, models.CalendarPermissionEdit)

	require.NoError(t, err)
	assert.NotNil(t, repo.members[calID][targetID])
	assert.Equal(t, models.CalendarPermissionEdit, repo.members[calID][targetID].Permission)
}

func TestService_AddMember_AsAdmin(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	ownerID := uuid.New()
	adminID := uuid.New()
	targetID := uuid.New()
	calID := uuid.New()

	repo.addCalendar(&models.Calendar{
		ID: calID, TenantID: testTenantID, Name: "Shared", CalendarType: models.CalendarTypeShared,
		OwnerID: ownerID, Color: "#4285F4", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
	repo.addMemberDirect(calID, adminID, models.CalendarPermissionAdmin)

	err := svc.AddMember(context.Background(), calID, adminID, targetID, testTenantID, models.CalendarPermissionView)

	require.NoError(t, err)
}

func TestService_AddMember_AsViewer_Fails(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	ownerID := uuid.New()
	viewerID := uuid.New()
	calID := uuid.New()

	repo.addCalendar(&models.Calendar{
		ID: calID, TenantID: testTenantID, Name: "Shared", CalendarType: models.CalendarTypeShared,
		OwnerID: ownerID, Color: "#4285F4", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
	repo.addMemberDirect(calID, viewerID, models.CalendarPermissionView)

	err := svc.AddMember(context.Background(), calID, viewerID, uuid.New(), testTenantID, models.CalendarPermissionView)

	assert.ErrorIs(t, err, ErrInsufficientPermission)
}

func TestService_AddMember_CannotAddOwner(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	ownerID := uuid.New()
	calID := uuid.New()

	repo.addCalendar(&models.Calendar{
		ID: calID, TenantID: testTenantID, Name: "Shared", CalendarType: models.CalendarTypeShared,
		OwnerID: ownerID, Color: "#4285F4", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	err := svc.AddMember(context.Background(), calID, ownerID, ownerID, testTenantID, models.CalendarPermissionAdmin)

	assert.ErrorIs(t, err, ErrOwnerCannotBeAdded)
}

func TestService_AddMember_InvalidPermission(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	ownerID := uuid.New()
	calID := uuid.New()

	repo.addCalendar(&models.Calendar{
		ID: calID, TenantID: testTenantID, Name: "Shared", CalendarType: models.CalendarTypeShared,
		OwnerID: ownerID, Color: "#4285F4", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	err := svc.AddMember(context.Background(), calID, ownerID, uuid.New(), testTenantID, "superadmin")

	assert.ErrorIs(t, err, ErrInvalidPermission)
}

func TestService_AddMember_AlreadyMember(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	ownerID := uuid.New()
	targetID := uuid.New()
	calID := uuid.New()

	repo.addCalendar(&models.Calendar{
		ID: calID, TenantID: testTenantID, Name: "Shared", CalendarType: models.CalendarTypeShared,
		OwnerID: ownerID, Color: "#4285F4", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
	repo.addMemberDirect(calID, targetID, models.CalendarPermissionView)

	err := svc.AddMember(context.Background(), calID, ownerID, targetID, testTenantID, models.CalendarPermissionEdit)

	assert.ErrorIs(t, err, ErrMemberAlreadyExists)
}

// ============================================================================
// RemoveMember Tests
// ============================================================================

func TestService_RemoveMember_AsOwner(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	ownerID := uuid.New()
	targetID := uuid.New()
	calID := uuid.New()

	repo.addCalendar(&models.Calendar{
		ID: calID, TenantID: testTenantID, Name: "Shared", CalendarType: models.CalendarTypeShared,
		OwnerID: ownerID, Color: "#4285F4", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
	repo.addMemberDirect(calID, targetID, models.CalendarPermissionView)

	err := svc.RemoveMember(context.Background(), calID, ownerID, targetID, testTenantID)

	require.NoError(t, err)
	_, exists := repo.members[calID][targetID]
	assert.False(t, exists)
}

func TestService_RemoveMember_CannotRemoveOwner(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	ownerID := uuid.New()
	adminID := uuid.New()
	calID := uuid.New()

	repo.addCalendar(&models.Calendar{
		ID: calID, TenantID: testTenantID, Name: "Shared", CalendarType: models.CalendarTypeShared,
		OwnerID: ownerID, Color: "#4285F4", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
	repo.addMemberDirect(calID, adminID, models.CalendarPermissionAdmin)

	err := svc.RemoveMember(context.Background(), calID, adminID, ownerID, testTenantID)

	assert.ErrorIs(t, err, ErrNotCalendarOwner)
}

func TestService_RemoveMember_AsViewer_Fails(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	ownerID := uuid.New()
	viewerID := uuid.New()
	targetID := uuid.New()
	calID := uuid.New()

	repo.addCalendar(&models.Calendar{
		ID: calID, TenantID: testTenantID, Name: "Shared", CalendarType: models.CalendarTypeShared,
		OwnerID: ownerID, Color: "#4285F4", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
	repo.addMemberDirect(calID, viewerID, models.CalendarPermissionView)
	repo.addMemberDirect(calID, targetID, models.CalendarPermissionView)

	err := svc.RemoveMember(context.Background(), calID, viewerID, targetID, testTenantID)

	assert.ErrorIs(t, err, ErrInsufficientPermission)
}

// ============================================================================
// UpdateMemberPermission Tests
// ============================================================================

func TestService_UpdateMemberPermission_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	ownerID := uuid.New()
	targetID := uuid.New()
	calID := uuid.New()

	repo.addCalendar(&models.Calendar{
		ID: calID, TenantID: testTenantID, Name: "Shared", CalendarType: models.CalendarTypeShared,
		OwnerID: ownerID, Color: "#4285F4", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
	repo.addMemberDirect(calID, targetID, models.CalendarPermissionView)

	err := svc.UpdateMemberPermission(context.Background(), calID, ownerID, targetID, testTenantID, models.CalendarPermissionEdit)

	require.NoError(t, err)
	assert.Equal(t, models.CalendarPermissionEdit, repo.members[calID][targetID].Permission)
}

func TestService_UpdateMemberPermission_CannotChangeOwner(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	ownerID := uuid.New()
	adminID := uuid.New()
	calID := uuid.New()

	repo.addCalendar(&models.Calendar{
		ID: calID, TenantID: testTenantID, Name: "Shared", CalendarType: models.CalendarTypeShared,
		OwnerID: ownerID, Color: "#4285F4", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
	repo.addMemberDirect(calID, adminID, models.CalendarPermissionAdmin)

	err := svc.UpdateMemberPermission(context.Background(), calID, adminID, ownerID, testTenantID, models.CalendarPermissionView)

	assert.ErrorIs(t, err, ErrOwnerCannotBeAdded)
}

func TestService_UpdateMemberPermission_InvalidPermission(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	ownerID := uuid.New()
	targetID := uuid.New()
	calID := uuid.New()

	repo.addCalendar(&models.Calendar{
		ID: calID, TenantID: testTenantID, Name: "Shared", CalendarType: models.CalendarTypeShared,
		OwnerID: ownerID, Color: "#4285F4", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
	repo.addMemberDirect(calID, targetID, models.CalendarPermissionView)

	err := svc.UpdateMemberPermission(context.Background(), calID, ownerID, targetID, testTenantID, "superuser")

	assert.ErrorIs(t, err, ErrInvalidPermission)
}

func TestService_UpdateMemberPermission_NotAMember(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	ownerID := uuid.New()
	calID := uuid.New()

	repo.addCalendar(&models.Calendar{
		ID: calID, TenantID: testTenantID, Name: "Shared", CalendarType: models.CalendarTypeShared,
		OwnerID: ownerID, Color: "#4285F4", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	err := svc.UpdateMemberPermission(context.Background(), calID, ownerID, uuid.New(), testTenantID, models.CalendarPermissionEdit)

	assert.ErrorIs(t, err, ErrMemberNotFound)
}

// ============================================================================
// ListMembers Tests
// ============================================================================

func TestService_ListMembers_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	ownerID := uuid.New()
	calID := uuid.New()

	repo.addCalendar(&models.Calendar{
		ID: calID, TenantID: testTenantID, Name: "Shared", CalendarType: models.CalendarTypeShared,
		OwnerID: ownerID, Color: "#4285F4", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
	repo.addMemberDirect(calID, uuid.New(), models.CalendarPermissionView)
	repo.addMemberDirect(calID, uuid.New(), models.CalendarPermissionEdit)

	members, err := svc.ListMembers(context.Background(), calID, testTenantID)

	require.NoError(t, err)
	assert.Len(t, members, 2)
}

func TestService_ListMembers_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	_, err := svc.ListMembers(context.Background(), uuid.New(), testTenantID)

	assert.ErrorIs(t, err, ErrCalendarNotFound)
}

// ============================================================================
// Subscribe / Unsubscribe Tests
// ============================================================================

func TestService_Subscribe_SharedCalendar(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	ownerID := uuid.New()
	userID := uuid.New()
	calID := uuid.New()

	repo.addCalendar(&models.Calendar{
		ID: calID, TenantID: testTenantID, Name: "Team", CalendarType: models.CalendarTypeShared,
		OwnerID: ownerID, Color: "#4285F4", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	err := svc.Subscribe(context.Background(), calID, userID, testTenantID)

	require.NoError(t, err)
	assert.NotNil(t, repo.members[calID][userID])
	assert.Equal(t, models.CalendarPermissionView, repo.members[calID][userID].Permission)
}

func TestService_Subscribe_PersonalCalendar_Fails(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	ownerID := uuid.New()
	calID := uuid.New()

	repo.addCalendar(&models.Calendar{
		ID: calID, TenantID: testTenantID, Name: "Personal", CalendarType: models.CalendarTypePersonal,
		OwnerID: ownerID, Color: "#4285F4", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	err := svc.Subscribe(context.Background(), calID, uuid.New(), testTenantID)

	assert.ErrorIs(t, err, ErrCannotSubscribePersonal)
}

func TestService_Subscribe_OwnCalendar_Fails(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	ownerID := uuid.New()
	calID := uuid.New()

	repo.addCalendar(&models.Calendar{
		ID: calID, TenantID: testTenantID, Name: "My Shared", CalendarType: models.CalendarTypeShared,
		OwnerID: ownerID, Color: "#4285F4", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	err := svc.Subscribe(context.Background(), calID, ownerID, testTenantID)

	assert.ErrorIs(t, err, ErrSubscriptionExists)
}

func TestService_Subscribe_AlreadySubscribed_Fails(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	ownerID := uuid.New()
	userID := uuid.New()
	calID := uuid.New()

	repo.addCalendar(&models.Calendar{
		ID: calID, TenantID: testTenantID, Name: "Team", CalendarType: models.CalendarTypeShared,
		OwnerID: ownerID, Color: "#4285F4", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	err := svc.Subscribe(context.Background(), calID, userID, testTenantID)
	require.NoError(t, err)

	err = svc.Subscribe(context.Background(), calID, userID, testTenantID)
	assert.ErrorIs(t, err, ErrSubscriptionExists)
}

func TestService_Unsubscribe_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	ownerID := uuid.New()
	userID := uuid.New()
	calID := uuid.New()

	repo.addCalendar(&models.Calendar{
		ID: calID, TenantID: testTenantID, Name: "Team", CalendarType: models.CalendarTypeShared,
		OwnerID: ownerID, Color: "#4285F4", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
	repo.addMemberDirect(calID, userID, models.CalendarPermissionView)

	err := svc.Unsubscribe(context.Background(), calID, userID, testTenantID)

	require.NoError(t, err)
	_, exists := repo.members[calID][userID]
	assert.False(t, exists)
}

func TestService_Unsubscribe_OwnCalendar_Fails(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	ownerID := uuid.New()
	calID := uuid.New()

	repo.addCalendar(&models.Calendar{
		ID: calID, TenantID: testTenantID, Name: "My Shared", CalendarType: models.CalendarTypeShared,
		OwnerID: ownerID, Color: "#4285F4", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	err := svc.Unsubscribe(context.Background(), calID, ownerID, testTenantID)

	assert.ErrorIs(t, err, ErrNotCalendarOwner)
}

func TestService_Unsubscribe_NotSubscribed_Fails(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	ownerID := uuid.New()
	calID := uuid.New()

	repo.addCalendar(&models.Calendar{
		ID: calID, TenantID: testTenantID, Name: "Team", CalendarType: models.CalendarTypeShared,
		OwnerID: ownerID, Color: "#4285F4", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	err := svc.Unsubscribe(context.Background(), calID, uuid.New(), testTenantID)

	assert.ErrorIs(t, err, ErrSubscriptionNotFound)
}

// ============================================================================
// ListBrowsable Tests
// ============================================================================

func TestService_ListBrowsable(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	userID := uuid.New()
	otherID := uuid.New()

	// Shared calendar NOT subscribed to
	browsable := &models.Calendar{
		ID: uuid.New(), TenantID: testTenantID, Name: "Browsable", CalendarType: models.CalendarTypeShared,
		OwnerID: otherID, Color: "#4285F4", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	repo.addCalendar(browsable)

	// Shared calendar already subscribed to
	subscribed := &models.Calendar{
		ID: uuid.New(), TenantID: testTenantID, Name: "Subscribed", CalendarType: models.CalendarTypeShared,
		OwnerID: otherID, Color: "#34A853", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	repo.addCalendar(subscribed)
	repo.addMemberDirect(subscribed.ID, userID, models.CalendarPermissionView)

	// Personal calendar (should not appear)
	personal := &models.Calendar{
		ID: uuid.New(), TenantID: testTenantID, Name: "Other's Personal", CalendarType: models.CalendarTypePersonal,
		OwnerID: otherID, Color: "#EA4335", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	repo.addCalendar(personal)

	calendars, err := svc.ListBrowsable(context.Background(), userID, testTenantID)

	require.NoError(t, err)
	require.Len(t, calendars, 1)
	assert.Equal(t, "Browsable", calendars[0].Name)
}

// ============================================================================
// Category Tests
// ============================================================================

func TestService_CreateCategory_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	userID := uuid.New()

	cat, err := svc.CreateCategory(context.Background(), userID, testTenantID, "Meeting", "#FF5733")

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, cat.ID)
	assert.Equal(t, "Meeting", cat.Name)
	assert.Equal(t, "#FF5733", cat.Color)
	assert.Equal(t, userID, cat.UserID)
	assert.Equal(t, testTenantID, cat.TenantID)
}

func TestService_CreateCategory_DefaultColor(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	cat, err := svc.CreateCategory(context.Background(), uuid.New(), testTenantID, "Focus", "")

	require.NoError(t, err)
	assert.Equal(t, "#7986CB", cat.Color)
}

func TestService_CreateCategory_EmptyName(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	_, err := svc.CreateCategory(context.Background(), uuid.New(), testTenantID, "", "#FF5733")

	assert.ErrorIs(t, err, ErrCategoryNameRequired)
}

func TestService_CreateCategory_InvalidColor(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	_, err := svc.CreateCategory(context.Background(), uuid.New(), testTenantID, "Bad", "red")

	assert.ErrorIs(t, err, ErrInvalidColor)
}

func TestService_CreateCategory_DuplicateName(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	userID := uuid.New()

	_, err := svc.CreateCategory(context.Background(), userID, testTenantID, "Meeting", "#FF5733")
	require.NoError(t, err)

	_, err = svc.CreateCategory(context.Background(), userID, testTenantID, "Meeting", "#34A853")
	assert.ErrorIs(t, err, ErrDuplicateCategoryName)
}

func TestService_ListCategories(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	userID := uuid.New()

	_, err := svc.CreateCategory(context.Background(), userID, testTenantID, "Meeting", "#FF5733")
	require.NoError(t, err)
	_, err = svc.CreateCategory(context.Background(), userID, testTenantID, "Focus", "#34A853")
	require.NoError(t, err)

	cats, err := svc.ListCategories(context.Background(), userID, testTenantID)

	require.NoError(t, err)
	assert.Len(t, cats, 2)
}

func TestService_DeleteCategory_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	userID := uuid.New()

	cat, err := svc.CreateCategory(context.Background(), userID, testTenantID, "Meeting", "#FF5733")
	require.NoError(t, err)

	err = svc.DeleteCategory(context.Background(), cat.ID, userID, testTenantID)

	require.NoError(t, err)
	cats, _ := svc.ListCategories(context.Background(), userID, testTenantID)
	assert.Len(t, cats, 0)
}

func TestService_DeleteCategory_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	err := svc.DeleteCategory(context.Background(), uuid.New(), uuid.New(), testTenantID)

	assert.ErrorIs(t, err, ErrCategoryNotFound)
}

func TestService_DeleteCategory_WrongUser(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	userID := uuid.New()

	cat, err := svc.CreateCategory(context.Background(), userID, testTenantID, "Meeting", "#FF5733")
	require.NoError(t, err)

	err = svc.DeleteCategory(context.Background(), cat.ID, uuid.New(), testTenantID)

	assert.ErrorIs(t, err, ErrCategoryNotFound)
}

// ============================================================================
// Preferences Tests
// ============================================================================

func TestService_GetPreferences_Default(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	userID := uuid.New()

	prefs, err := svc.GetPreferences(context.Background(), userID)

	require.NoError(t, err)
	assert.Equal(t, userID, prefs.UserID)
	assert.Equal(t, "week", prefs.DefaultView)
	assert.Equal(t, 5, prefs.WeekDays)
	assert.Equal(t, 15, prefs.DefaultReminderMinutes)
	assert.Equal(t, 1440, prefs.DefaultAllDayReminderMinutes)
	assert.True(t, prefs.ShowTaskDeadlines)
}

func TestService_UpsertPreferences_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	userID := uuid.New()

	err := svc.UpsertPreferences(context.Background(), &models.UserCalendarPreferences{
		UserID:                       userID,
		DefaultView:                  "month",
		WeekDays:                     7,
		DefaultReminderMinutes:       30,
		DefaultAllDayReminderMinutes: 720,
		ShowTaskDeadlines:            false,
	})

	require.NoError(t, err)

	prefs, err := svc.GetPreferences(context.Background(), userID)
	require.NoError(t, err)
	assert.Equal(t, "month", prefs.DefaultView)
	assert.Equal(t, 7, prefs.WeekDays)
	assert.Equal(t, 30, prefs.DefaultReminderMinutes)
	assert.False(t, prefs.ShowTaskDeadlines)
}

func TestService_UpsertPreferences_InvalidView(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	err := svc.UpsertPreferences(context.Background(), &models.UserCalendarPreferences{
		UserID:      uuid.New(),
		DefaultView: "timeline",
		WeekDays:    5,
	})

	assert.ErrorIs(t, err, ErrInvalidCalendarType)
}

func TestService_UpsertPreferences_InvalidWeekDays_Normalized(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	userID := uuid.New()

	err := svc.UpsertPreferences(context.Background(), &models.UserCalendarPreferences{
		UserID:      userID,
		DefaultView: "week",
		WeekDays:    3, // invalid, normalized to 5
	})

	require.NoError(t, err)
	prefs, _ := svc.GetPreferences(context.Background(), userID)
	assert.Equal(t, 5, prefs.WeekDays)
}

// ============================================================================
// Permission hierarchy tests
// ============================================================================

func TestService_PermissionHierarchy_EditCannotAddMembers(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	ownerID := uuid.New()
	editorID := uuid.New()
	calID := uuid.New()

	repo.addCalendar(&models.Calendar{
		ID: calID, TenantID: testTenantID, Name: "Shared", CalendarType: models.CalendarTypeShared,
		OwnerID: ownerID, Color: "#4285F4", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
	repo.addMemberDirect(calID, editorID, models.CalendarPermissionEdit)

	err := svc.AddMember(context.Background(), calID, editorID, uuid.New(), testTenantID, models.CalendarPermissionView)

	assert.ErrorIs(t, err, ErrInsufficientPermission)
}

func TestService_PermissionHierarchy_EditorCanUpdate(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	ownerID := uuid.New()
	editorID := uuid.New()
	calID := uuid.New()

	repo.addCalendar(&models.Calendar{
		ID: calID, TenantID: testTenantID, Name: "Shared", CalendarType: models.CalendarTypeShared,
		OwnerID: ownerID, Color: "#4285F4", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
	repo.addMemberDirect(calID, editorID, models.CalendarPermissionEdit)

	// Editor should NOT be able to update (requires admin)
	newName := "Updated by Editor"
	_, err := svc.Update(context.Background(), calID, editorID, testTenantID, UpdateInput{
		Name: &newName,
	})

	assert.ErrorIs(t, err, ErrInsufficientPermission)
}

// ============================================================================
// Get Tests
// ============================================================================

func TestService_Get_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	calID := uuid.New()

	repo.addCalendar(&models.Calendar{
		ID: calID, TenantID: testTenantID, Name: "Calendar", CalendarType: models.CalendarTypePersonal,
		OwnerID: uuid.New(), Color: "#4285F4", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	cal, err := svc.Get(context.Background(), calID, testTenantID)

	require.NoError(t, err)
	assert.Equal(t, calID, cal.ID)
	assert.Equal(t, "Calendar", cal.Name)
}

func TestService_Get_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	_, err := svc.Get(context.Background(), uuid.New(), testTenantID)

	assert.ErrorIs(t, err, ErrCalendarNotFound)
}

// ============================================================================
// Additional edge case tests for coverage
// ============================================================================

func TestService_Subscribe_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	err := svc.Subscribe(context.Background(), uuid.New(), uuid.New(), testTenantID)

	assert.ErrorIs(t, err, ErrCalendarNotFound)
}

func TestService_Unsubscribe_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	err := svc.Unsubscribe(context.Background(), uuid.New(), uuid.New(), testTenantID)

	assert.ErrorIs(t, err, ErrCalendarNotFound)
}

func TestService_RemoveMember_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	err := svc.RemoveMember(context.Background(), uuid.New(), uuid.New(), uuid.New(), testTenantID)

	assert.ErrorIs(t, err, ErrCalendarNotFound)
}

func TestService_AddMember_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	err := svc.AddMember(context.Background(), uuid.New(), uuid.New(), uuid.New(), testTenantID, models.CalendarPermissionView)

	assert.ErrorIs(t, err, ErrCalendarNotFound)
}

func TestService_UpdateMemberPermission_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	err := svc.UpdateMemberPermission(context.Background(), uuid.New(), uuid.New(), uuid.New(), testTenantID, models.CalendarPermissionEdit)

	assert.ErrorIs(t, err, ErrCalendarNotFound)
}

func TestService_Update_Color(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	ownerID := uuid.New()
	calID := uuid.New()

	repo.addCalendar(&models.Calendar{
		ID: calID, TenantID: testTenantID, Name: "Cal", CalendarType: models.CalendarTypePersonal,
		OwnerID: ownerID, Color: "#4285F4", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	newColor := "#FF0000"
	newTz := "Europe/Zurich"
	cal, err := svc.Update(context.Background(), calID, ownerID, testTenantID, UpdateInput{
		Color:    &newColor,
		Timezone: &newTz,
	})

	require.NoError(t, err)
	assert.Equal(t, "#FF0000", cal.Color)
	assert.Equal(t, "Europe/Zurich", cal.Timezone)
}

func TestService_Update_Description(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	ownerID := uuid.New()
	calID := uuid.New()

	repo.addCalendar(&models.Calendar{
		ID: calID, TenantID: testTenantID, Name: "Cal", CalendarType: models.CalendarTypePersonal,
		OwnerID: ownerID, Color: "#4285F4", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	desc := "My calendar description"
	cal, err := svc.Update(context.Background(), calID, ownerID, testTenantID, UpdateInput{
		Description: &desc,
	})

	require.NoError(t, err)
	require.NotNil(t, cal.Description)
	assert.Equal(t, "My calendar description", *cal.Description)
}

func TestService_Subscribe_ResourceCalendar(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	ownerID := uuid.New()
	calID := uuid.New()

	repo.addCalendar(&models.Calendar{
		ID: calID, TenantID: testTenantID, Name: "Room", CalendarType: models.CalendarTypeResource,
		OwnerID: ownerID, Color: "#4285F4", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	// Resource calendars should be subscribable (not personal)
	err := svc.Subscribe(context.Background(), calID, uuid.New(), testTenantID)

	require.NoError(t, err)
}

// ============================================================================
// Tenant Isolation Test
// ============================================================================

func TestCalendarRepo_TenantIsolation(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	tenantA := uuid.New()
	tenantB := uuid.New()
	ownerA := uuid.New()

	// Create calendar as Tenant A
	calA, err := svc.Create(context.Background(), ownerA, tenantA, CreateInput{
		Name:         "Tenant A Calendar",
		CalendarType: models.CalendarTypePersonal,
	})
	require.NoError(t, err)

	// Query as Tenant B -> should not find it
	_, err = svc.Get(context.Background(), calA.ID, tenantB)
	assert.ErrorIs(t, err, ErrCalendarNotFound, "calendar from tenant A must not be visible to tenant B")

	// Query as Tenant A -> should find it
	found, err := svc.Get(context.Background(), calA.ID, tenantA)
	require.NoError(t, err)
	assert.Equal(t, calA.ID, found.ID)
}

// ============================================================================
// F6 — DB-Level Cross-Tenant Isolation Tests (Sprint 3 Welle 1)
//
// These tests verify that the repository mock correctly enforces tenant_id
// scoping at the data layer. They simulate the same guarantee that the real
// Postgres repository provides via the WHERE tenant_id = $N predicates added
// in Migration 000109 (Welle 4B).
//
// Pattern: Tenant A inserts → Tenant B reads → result must be empty / not found.
// ============================================================================

// TestCalendarCrossTenantIsolation_DBLevel verifies that a calendar created
// under Tenant A is never returned when listing calendars for Tenant B.
// This covers both ListByUser (owned/subscribed list) and ListBrowsable
// (shared calendar discovery), ensuring two independently scoped repository
// calls cannot leak data across tenant boundaries.
func TestCalendarCrossTenantIsolation_DBLevel(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	tenantA := uuid.New()
	tenantB := uuid.New()
	userA := uuid.New()
	userB := uuid.New()

	// Tenant A creates a personal calendar.
	_, err := svc.Create(context.Background(), userA, tenantA, CreateInput{
		Name:         "Tenant A Private Calendar",
		CalendarType: models.CalendarTypePersonal,
	})
	require.NoError(t, err)

	// Tenant A creates a shared calendar (could be discovered if browsable).
	_, err = svc.Create(context.Background(), userA, tenantA, CreateInput{
		Name:         "Tenant A Shared Calendar",
		CalendarType: models.CalendarTypeShared,
	})
	require.NoError(t, err)

	// Tenant B lists their own calendars — must not see any of Tenant A's calendars.
	calsByB, err := svc.ListByUser(context.Background(), userB, tenantB)
	require.NoError(t, err)
	for _, c := range calsByB {
		assert.NotEqual(t, tenantA, c.TenantID,
			"calendar from tenant A must not appear in tenant B's list: %s", c.Name)
	}

	// Tenant B browses discoverable shared calendars — must not see Tenant A's shared calendar.
	browsable, err := svc.ListBrowsable(context.Background(), userB, tenantB)
	require.NoError(t, err)
	for _, c := range browsable {
		assert.NotEqual(t, tenantA, c.TenantID,
			"shared calendar from tenant A must not appear in tenant B's browse list: %s", c.Name)
	}
}

// Suppress unused import for fmt
var _ = fmt.Sprintf
