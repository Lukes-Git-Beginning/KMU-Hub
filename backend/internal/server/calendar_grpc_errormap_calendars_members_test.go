package server

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/work/calendar"
	"github.com/kmuhub/kmuhub/internal/work/event"
	"github.com/kmuhub/kmuhub/internal/work/holiday"
	"github.com/kmuhub/kmuhub/internal/work/livekit"
	"github.com/kmuhub/kmuhub/internal/work/resource"
	calv1 "github.com/kmuhub/kmuhub/proto/calendar/v1"
)

// ============================================================================
// mapCalendarError — every sentinel individually against its expected gRPC code
// ============================================================================

func TestMapCalendarError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode codes.Code
	}{
		// Calendar errors
		{"calendar not found", calendar.ErrCalendarNotFound, codes.NotFound},
		{"calendar name required", calendar.ErrCalendarNameRequired, codes.InvalidArgument},
		{"invalid calendar type", calendar.ErrInvalidCalendarType, codes.InvalidArgument},
		{"member already exists", calendar.ErrMemberAlreadyExists, codes.AlreadyExists},
		{"member not found", calendar.ErrMemberNotFound, codes.NotFound},
		{"not calendar owner", calendar.ErrNotCalendarOwner, codes.PermissionDenied},
		{"not calendar admin", calendar.ErrNotCalendarAdmin, codes.PermissionDenied},
		{"insufficient permission", calendar.ErrInsufficientPermission, codes.PermissionDenied},
		{"cannot delete default calendar", calendar.ErrCannotDeleteDefaultCalendar, codes.FailedPrecondition},
		{"cannot delete personal calendar", calendar.ErrCannotDeletePersonalCalendar, codes.FailedPrecondition},
		{"category not found", calendar.ErrCategoryNotFound, codes.NotFound},
		{"category name required", calendar.ErrCategoryNameRequired, codes.InvalidArgument},
		{"duplicate category name", calendar.ErrDuplicateCategoryName, codes.AlreadyExists},
		{"subscription exists", calendar.ErrSubscriptionExists, codes.AlreadyExists},
		{"subscription not found", calendar.ErrSubscriptionNotFound, codes.NotFound},
		{"invalid permission", calendar.ErrInvalidPermission, codes.InvalidArgument},
		{"invalid color", calendar.ErrInvalidColor, codes.InvalidArgument},
		{"cannot subscribe personal", calendar.ErrCannotSubscribePersonal, codes.InvalidArgument},
		{"owner cannot be added", calendar.ErrOwnerCannotBeAdded, codes.InvalidArgument},
		// Event errors
		{"event not found", event.ErrEventNotFound, codes.NotFound},
		{"event title required", event.ErrEventTitleRequired, codes.InvalidArgument},
		{"invalid time range", event.ErrInvalidTimeRange, codes.InvalidArgument},
		{"invalid rrule", event.ErrInvalidRRule, codes.InvalidArgument},
		{"invalid rsvp status", event.ErrInvalidRSVPStatus, codes.InvalidArgument},
		{"already attendee", event.ErrAlreadyAttendee, codes.AlreadyExists},
		{"not attendee", event.ErrNotAttendee, codes.NotFound},
		{"not event creator", event.ErrNotEventCreator, codes.PermissionDenied},
		{"invalid recurring edit scope", event.ErrInvalidRecurringEditScope, codes.InvalidArgument},
		{"event not recurring", event.ErrEventNotRecurring, codes.FailedPrecondition},
		{"reminder limit exceeded", event.ErrReminderLimitExceeded, codes.InvalidArgument},
		{"invalid reminder minutes", event.ErrInvalidReminderMinutes, codes.InvalidArgument},
		{"event insufficient permission", event.ErrInsufficientPermission, codes.PermissionDenied},
		// Resource errors
		{"resource not found", resource.ErrResourceNotFound, codes.NotFound},
		{"resource name required", resource.ErrResourceNameRequired, codes.InvalidArgument},
		{"invalid resource type", resource.ErrInvalidResourceType, codes.InvalidArgument},
		{"resource inactive", resource.ErrResourceInactive, codes.FailedPrecondition},
		{"booking not found", resource.ErrBookingNotFound, codes.NotFound},
		{"booking conflict", resource.ErrBookingConflict, codes.AlreadyExists},
		{"booking already cancelled", resource.ErrBookingAlreadyCancelled, codes.FailedPrecondition},
		{"not booking owner", resource.ErrNotBookingOwner, codes.PermissionDenied},
		{"invalid booking time range", resource.ErrInvalidBookingTimeRange, codes.InvalidArgument},
		{"resource not authorized", resource.ErrNotAuthorized, codes.PermissionDenied},
		// Holiday errors
		{"invalid country code", holiday.ErrInvalidCountryCode, codes.InvalidArgument},
		{"invalid year", holiday.ErrInvalidYear, codes.InvalidArgument},
		{"nager api unavailable", holiday.ErrNagerAPIUnavailable, codes.Unavailable},
		// LiveKit errors
		{"livekit not configured", livekit.ErrLiveKitNotConfigured, codes.Unavailable},
		// Booking page errors
		{"booking page not found", calendar.ErrBookingPageNotFound, codes.NotFound},
		{"booking page slug required", calendar.ErrBookingPageSlugRequired, codes.InvalidArgument},
		{"booking page company name required", calendar.ErrBookingPageCompanyNameRequired, codes.InvalidArgument},
		{"booking page invalid rules", calendar.ErrBookingPageInvalidRules, codes.InvalidArgument},
		{"booking service not found", calendar.ErrBookingServiceNotFound, codes.NotFound},
		{"booking slot unavailable", calendar.ErrBookingSlotUnavailable, codes.FailedPrecondition},
		{"booking slot in past", calendar.ErrBookingSlotInPast, codes.FailedPrecondition},
		// Fallback
		{"unknown error", errors.New("boom"), codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireGRPCCode(t, mapCalendarError(tt.err), tt.wantCode)
		})
	}
}

// ============================================================================
// stubCalendarRepo — implements calendar.Repository for the CRUD/member RPCs
// exercised below. Category/preference/visibility methods are no-ops: they
// belong to a later coverage unit and are never reached by these tests.
// ============================================================================

type stubCalendarRepo struct {
	calendars  map[uuid.UUID]*models.Calendar
	members    map[uuid.UUID]map[uuid.UUID]*models.CalendarMember
	browsable  []models.Calendar
	categories map[uuid.UUID]*models.EventCategory

	createErr          error
	getByIDErr         error
	listByUserErr      error
	updateErr          error
	deleteErr          error
	addMemberErr       error
	removeMemberErr    error
	listMembersErr     error
	getMemberErr       error
	updateMemPermErr   error
	listBrowsableErr   error
	subscribeErr       error
	unsubscribeErr     error
	ensurePersonalErr  error
	createCategoryErr  error
	listCategoriesErr  error
}

func newStubCalendarRepo() *stubCalendarRepo {
	return &stubCalendarRepo{
		calendars:  make(map[uuid.UUID]*models.Calendar),
		members:    make(map[uuid.UUID]map[uuid.UUID]*models.CalendarMember),
		categories: make(map[uuid.UUID]*models.EventCategory),
	}
}

func (r *stubCalendarRepo) Create(_ context.Context, c *models.Calendar) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.calendars[c.ID] = c
	return nil
}

func (r *stubCalendarRepo) GetByID(_ context.Context, id, tenantID uuid.UUID) (*models.Calendar, error) {
	if r.getByIDErr != nil {
		return nil, r.getByIDErr
	}
	c, ok := r.calendars[id]
	if !ok || c.TenantID != tenantID {
		return nil, calendar.ErrCalendarNotFound
	}
	return c, nil
}

func (r *stubCalendarRepo) ListByUser(_ context.Context, userID, tenantID uuid.UUID) ([]models.CalendarWithMemberInfo, error) {
	if r.listByUserErr != nil {
		return nil, r.listByUserErr
	}
	var out []models.CalendarWithMemberInfo
	for _, c := range r.calendars {
		if c.TenantID != tenantID || c.OwnerID != userID {
			continue
		}
		out = append(out, models.CalendarWithMemberInfo{Calendar: *c, Permission: models.CalendarPermissionAdmin, IsVisible: true})
	}
	return out, nil
}

func (r *stubCalendarRepo) Update(_ context.Context, c *models.Calendar) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	r.calendars[c.ID] = c
	return nil
}

func (r *stubCalendarRepo) Delete(_ context.Context, id, _ uuid.UUID) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	delete(r.calendars, id)
	return nil
}

func (r *stubCalendarRepo) AddMember(_ context.Context, m *models.CalendarMember) error {
	if r.addMemberErr != nil {
		return r.addMemberErr
	}
	if _, ok := r.members[m.CalendarID]; !ok {
		r.members[m.CalendarID] = make(map[uuid.UUID]*models.CalendarMember)
	}
	r.members[m.CalendarID][m.UserID] = m
	return nil
}

func (r *stubCalendarRepo) RemoveMember(_ context.Context, calendarID, userID uuid.UUID) error {
	if r.removeMemberErr != nil {
		return r.removeMemberErr
	}
	if mems, ok := r.members[calendarID]; ok {
		delete(mems, userID)
	}
	return nil
}

func (r *stubCalendarRepo) ListMembers(_ context.Context, calendarID uuid.UUID) ([]models.CalendarMember, error) {
	if r.listMembersErr != nil {
		return nil, r.listMembersErr
	}
	var out []models.CalendarMember
	for _, m := range r.members[calendarID] {
		out = append(out, *m)
	}
	return out, nil
}

func (r *stubCalendarRepo) GetMember(_ context.Context, calendarID, userID uuid.UUID) (*models.CalendarMember, error) {
	if r.getMemberErr != nil {
		return nil, r.getMemberErr
	}
	m, ok := r.members[calendarID][userID]
	if !ok {
		return nil, calendar.ErrMemberNotFound
	}
	return m, nil
}

func (r *stubCalendarRepo) UpdateMemberPermission(_ context.Context, calendarID, userID uuid.UUID, permission string) error {
	if r.updateMemPermErr != nil {
		return r.updateMemPermErr
	}
	if m, ok := r.members[calendarID][userID]; ok {
		m.Permission = permission
	}
	return nil
}

func (r *stubCalendarRepo) UpdateMemberVisibility(_ context.Context, _, _ uuid.UUID, _ bool) error {
	return nil
}

func (r *stubCalendarRepo) UpdateMemberColorOverride(_ context.Context, _, _ uuid.UUID, _ *string) error {
	return nil
}

func (r *stubCalendarRepo) ListBrowsable(_ context.Context, _, _ uuid.UUID) ([]models.Calendar, error) {
	if r.listBrowsableErr != nil {
		return nil, r.listBrowsableErr
	}
	return r.browsable, nil
}

func (r *stubCalendarRepo) Subscribe(_ context.Context, _, _ uuid.UUID) error {
	return r.subscribeErr
}

func (r *stubCalendarRepo) Unsubscribe(_ context.Context, _, _ uuid.UUID) error {
	return r.unsubscribeErr
}

func (r *stubCalendarRepo) CreateCategory(_ context.Context, c *models.EventCategory) error {
	if r.createCategoryErr != nil {
		return r.createCategoryErr
	}
	r.categories[c.ID] = c
	return nil
}

func (r *stubCalendarRepo) ListCategories(_ context.Context, userID, tenantID uuid.UUID) ([]models.EventCategory, error) {
	if r.listCategoriesErr != nil {
		return nil, r.listCategoriesErr
	}
	var out []models.EventCategory
	for _, c := range r.categories {
		if c.UserID == userID && c.TenantID == tenantID {
			out = append(out, *c)
		}
	}
	return out, nil
}

// DeleteCategory mirrors PostgresRepository.DeleteCategory (postgres_repository.go:337-349):
// a DELETE ... WHERE id=$1 AND user_id=$2 AND tenant_id=$3 that maps zero rows affected to
// ErrCategoryNotFound. That 0-row semantics is exactly what exposes the DeleteEventCategory
// gateway-auth bug below (calendar_grpc.go:791-792 passes uuid.Nil as userID).
func (r *stubCalendarRepo) DeleteCategory(_ context.Context, categoryID, userID, tenantID uuid.UUID) error {
	c, ok := r.categories[categoryID]
	if !ok || c.UserID != userID || c.TenantID != tenantID {
		return calendar.ErrCategoryNotFound
	}
	delete(r.categories, categoryID)
	return nil
}

func (r *stubCalendarRepo) GetPreferences(_ context.Context, _ uuid.UUID) (*models.UserCalendarPreferences, error) {
	return nil, nil
}

func (r *stubCalendarRepo) UpsertPreferences(_ context.Context, _ *models.UserCalendarPreferences) error {
	return nil
}

func (r *stubCalendarRepo) EnsurePersonalCalendar(_ context.Context, userID, tenantID uuid.UUID) (*models.Calendar, error) {
	if r.ensurePersonalErr != nil {
		return nil, r.ensurePersonalErr
	}
	return &models.Calendar{ID: uuid.New(), TenantID: tenantID, OwnerID: userID, CalendarType: models.CalendarTypePersonal, IsDefault: true}, nil
}

func newTestCalendarServerWithRepo(repo calendar.Repository) *CalendarGRPCServer {
	return NewCalendarGRPCServer(calendar.NewService(repo), nil, nil, nil, nil)
}

// ============================================================================
// Calendar CRUD
// ============================================================================

func TestCreateCalendar(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		repo := newStubCalendarRepo()
		srv := newTestCalendarServerWithRepo(repo)
		tenantID := uuid.New()
		ownerID := uuid.New()

		resp, err := srv.CreateCalendar(ctxWithTenant(tenantID), &calv1.CreateCalendarRequest{
			OwnerId:      ownerID.String(),
			Name:         "Team Calendar",
			CalendarType: models.CalendarTypeShared,
		})
		require.NoError(t, err)
		assert.Equal(t, "Team Calendar", resp.Calendar.Name)
		assert.Len(t, repo.calendars, 1)
	})

	t.Run("invalid owner_id", func(t *testing.T) {
		repo := newStubCalendarRepo()
		srv := newTestCalendarServerWithRepo(repo)

		_, err := srv.CreateCalendar(ctxWithTenant(uuid.New()), &calv1.CreateCalendarRequest{
			OwnerId:      "not-a-uuid",
			Name:         "Team Calendar",
			CalendarType: models.CalendarTypeShared,
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

func TestGetCalendar(t *testing.T) {
	tenantID := uuid.New()
	ownerID := uuid.New()
	calID := uuid.New()

	t.Run("happy path", func(t *testing.T) {
		repo := newStubCalendarRepo()
		repo.calendars[calID] = &models.Calendar{
			ID: calID, TenantID: tenantID, OwnerID: ownerID, Name: "Cal",
			CalendarType: models.CalendarTypeShared, Color: "#4285F4", Timezone: "Europe/Berlin",
		}
		srv := newTestCalendarServerWithRepo(repo)

		resp, err := srv.GetCalendar(ctxWithTenant(tenantID), &calv1.GetCalendarRequest{Id: calID.String()})
		require.NoError(t, err)
		assert.Equal(t, calID.String(), resp.Calendar.Id)
	})

	t.Run("not found", func(t *testing.T) {
		repo := newStubCalendarRepo()
		srv := newTestCalendarServerWithRepo(repo)

		_, err := srv.GetCalendar(ctxWithTenant(tenantID), &calv1.GetCalendarRequest{Id: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})
}

func TestListCalendars(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("happy path", func(t *testing.T) {
		repo := newStubCalendarRepo()
		calID := uuid.New()
		repo.calendars[calID] = &models.Calendar{ID: calID, TenantID: tenantID, OwnerID: userID, Name: "Personal", CalendarType: models.CalendarTypePersonal}
		srv := newTestCalendarServerWithRepo(repo)

		resp, err := srv.ListCalendars(ctxWithTenant(tenantID), &calv1.ListCalendarsRequest{UserId: userID.String()})
		require.NoError(t, err)
		assert.Len(t, resp.Calendars, 1)
	})

	t.Run("invalid user_id", func(t *testing.T) {
		repo := newStubCalendarRepo()
		srv := newTestCalendarServerWithRepo(repo)

		_, err := srv.ListCalendars(ctxWithTenant(tenantID), &calv1.ListCalendarsRequest{UserId: "nope"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

func TestUpdateCalendar(t *testing.T) {
	tenantID := uuid.New()
	calID := uuid.New()

	t.Run("happy path", func(t *testing.T) {
		repo := newStubCalendarRepo()
		repo.calendars[calID] = &models.Calendar{
			ID: calID, TenantID: tenantID, OwnerID: uuid.Nil, Name: "Old",
			CalendarType: models.CalendarTypeShared, Color: "#4285F4", Timezone: "Europe/Berlin",
		}
		srv := newTestCalendarServerWithRepo(repo)
		newName := "New Name"

		resp, err := srv.UpdateCalendar(ctxWithTenant(tenantID), &calv1.UpdateCalendarRequest{Id: calID.String(), Name: &newName})
		require.NoError(t, err)
		assert.Equal(t, "New Name", resp.Calendar.Name)
	})

	t.Run("not found", func(t *testing.T) {
		repo := newStubCalendarRepo()
		srv := newTestCalendarServerWithRepo(repo)
		newName := "New"

		_, err := srv.UpdateCalendar(ctxWithTenant(tenantID), &calv1.UpdateCalendarRequest{Id: uuid.New().String(), Name: &newName})
		requireGRPCCode(t, err, codes.NotFound)
	})
}

func TestDeleteCalendar(t *testing.T) {
	tenantID := uuid.New()

	t.Run("happy path", func(t *testing.T) {
		repo := newStubCalendarRepo()
		calID := uuid.New()
		repo.calendars[calID] = &models.Calendar{ID: calID, TenantID: tenantID, OwnerID: uuid.Nil, IsDefault: false}
		srv := newTestCalendarServerWithRepo(repo)

		_, err := srv.DeleteCalendar(ctxWithTenant(tenantID), &calv1.DeleteCalendarRequest{Id: calID.String()})
		require.NoError(t, err)
		_, exists := repo.calendars[calID]
		assert.False(t, exists)
	})

	t.Run("cannot delete default calendar", func(t *testing.T) {
		repo := newStubCalendarRepo()
		calID := uuid.New()
		repo.calendars[calID] = &models.Calendar{ID: calID, TenantID: tenantID, OwnerID: uuid.Nil, IsDefault: true}
		srv := newTestCalendarServerWithRepo(repo)

		_, err := srv.DeleteCalendar(ctxWithTenant(tenantID), &calv1.DeleteCalendarRequest{Id: calID.String()})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})
}

// ============================================================================
// Calendar Members
// ============================================================================

func TestAddCalendarMember(t *testing.T) {
	tenantID := uuid.New()
	calID := uuid.New()

	t.Run("happy path", func(t *testing.T) {
		repo := newStubCalendarRepo()
		repo.calendars[calID] = &models.Calendar{ID: calID, TenantID: tenantID, OwnerID: uuid.Nil}
		srv := newTestCalendarServerWithRepo(repo)
		targetID := uuid.New()

		_, err := srv.AddCalendarMember(ctxWithTenant(tenantID), &calv1.AddCalendarMemberRequest{
			CalendarId: calID.String(),
			UserId:     targetID.String(),
			Permission: models.CalendarPermissionView,
		})
		require.NoError(t, err)
		assert.NotNil(t, repo.members[calID][targetID])
	})

	t.Run("owner cannot be added", func(t *testing.T) {
		repo := newStubCalendarRepo()
		ownerID := uuid.New()
		repo.calendars[calID] = &models.Calendar{ID: calID, TenantID: tenantID, OwnerID: ownerID}
		srv := newTestCalendarServerWithRepo(repo)

		_, err := srv.AddCalendarMember(ctxWithTenant(tenantID), &calv1.AddCalendarMemberRequest{
			CalendarId: calID.String(),
			UserId:     ownerID.String(),
			Permission: models.CalendarPermissionView,
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

func TestRemoveCalendarMember(t *testing.T) {
	tenantID := uuid.New()
	calID := uuid.New()

	t.Run("happy path", func(t *testing.T) {
		repo := newStubCalendarRepo()
		targetID := uuid.New()
		repo.calendars[calID] = &models.Calendar{ID: calID, TenantID: tenantID, OwnerID: uuid.Nil}
		repo.members[calID] = map[uuid.UUID]*models.CalendarMember{
			targetID: {CalendarID: calID, UserID: targetID, Permission: models.CalendarPermissionView},
		}
		srv := newTestCalendarServerWithRepo(repo)

		_, err := srv.RemoveCalendarMember(ctxWithTenant(tenantID), &calv1.RemoveCalendarMemberRequest{
			CalendarId: calID.String(),
			UserId:     targetID.String(),
		})
		require.NoError(t, err)
		_, exists := repo.members[calID][targetID]
		assert.False(t, exists)
	})

	t.Run("cannot remove owner", func(t *testing.T) {
		repo := newStubCalendarRepo()
		ownerID := uuid.New()
		repo.calendars[calID] = &models.Calendar{ID: calID, TenantID: tenantID, OwnerID: ownerID}
		srv := newTestCalendarServerWithRepo(repo)

		_, err := srv.RemoveCalendarMember(ctxWithTenant(tenantID), &calv1.RemoveCalendarMemberRequest{
			CalendarId: calID.String(),
			UserId:     ownerID.String(),
		})
		requireGRPCCode(t, err, codes.PermissionDenied)
	})
}

func TestListCalendarMembers(t *testing.T) {
	tenantID := uuid.New()
	calID := uuid.New()

	t.Run("happy path", func(t *testing.T) {
		repo := newStubCalendarRepo()
		memberID := uuid.New()
		repo.calendars[calID] = &models.Calendar{ID: calID, TenantID: tenantID}
		repo.members[calID] = map[uuid.UUID]*models.CalendarMember{
			memberID: {CalendarID: calID, UserID: memberID, Permission: models.CalendarPermissionEdit},
		}
		srv := newTestCalendarServerWithRepo(repo)

		resp, err := srv.ListCalendarMembers(ctxWithTenant(tenantID), &calv1.ListCalendarMembersRequest{CalendarId: calID.String()})
		require.NoError(t, err)
		assert.Len(t, resp.Members, 1)
	})

	t.Run("calendar not found", func(t *testing.T) {
		repo := newStubCalendarRepo()
		srv := newTestCalendarServerWithRepo(repo)

		_, err := srv.ListCalendarMembers(ctxWithTenant(tenantID), &calv1.ListCalendarMembersRequest{CalendarId: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})
}

func TestUpdateCalendarMemberPermission(t *testing.T) {
	tenantID := uuid.New()
	calID := uuid.New()

	t.Run("happy path", func(t *testing.T) {
		repo := newStubCalendarRepo()
		targetID := uuid.New()
		repo.calendars[calID] = &models.Calendar{ID: calID, TenantID: tenantID, OwnerID: uuid.Nil}
		repo.members[calID] = map[uuid.UUID]*models.CalendarMember{
			targetID: {CalendarID: calID, UserID: targetID, Permission: models.CalendarPermissionView},
		}
		srv := newTestCalendarServerWithRepo(repo)

		_, err := srv.UpdateCalendarMemberPermission(ctxWithTenant(tenantID), &calv1.UpdateCalendarMemberPermissionRequest{
			CalendarId: calID.String(),
			UserId:     targetID.String(),
			Permission: models.CalendarPermissionEdit,
		})
		require.NoError(t, err)
		assert.Equal(t, models.CalendarPermissionEdit, repo.members[calID][targetID].Permission)
	})

	t.Run("target not a member", func(t *testing.T) {
		repo := newStubCalendarRepo()
		repo.calendars[calID] = &models.Calendar{ID: calID, TenantID: tenantID, OwnerID: uuid.Nil}
		srv := newTestCalendarServerWithRepo(repo)

		_, err := srv.UpdateCalendarMemberPermission(ctxWithTenant(tenantID), &calv1.UpdateCalendarMemberPermissionRequest{
			CalendarId: calID.String(),
			UserId:     uuid.New().String(),
			Permission: models.CalendarPermissionEdit,
		})
		requireGRPCCode(t, err, codes.NotFound)
	})
}

// ============================================================================
// Calendar Discovery / Subscription
// ============================================================================

func TestListBrowsableCalendars(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("happy path", func(t *testing.T) {
		repo := newStubCalendarRepo()
		repo.browsable = []models.Calendar{{ID: uuid.New(), TenantID: tenantID, Name: "Shared", CalendarType: models.CalendarTypeShared}}
		srv := newTestCalendarServerWithRepo(repo)

		resp, err := srv.ListBrowsableCalendars(ctxWithTenant(tenantID), &calv1.ListBrowsableCalendarsRequest{UserId: userID.String()})
		require.NoError(t, err)
		assert.Len(t, resp.Calendars, 1)
	})

	t.Run("invalid user_id", func(t *testing.T) {
		repo := newStubCalendarRepo()
		srv := newTestCalendarServerWithRepo(repo)

		_, err := srv.ListBrowsableCalendars(ctxWithTenant(tenantID), &calv1.ListBrowsableCalendarsRequest{UserId: "nope"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

func TestSubscribeToCalendar(t *testing.T) {
	tenantID := uuid.New()
	calID := uuid.New()

	t.Run("happy path", func(t *testing.T) {
		repo := newStubCalendarRepo()
		repo.calendars[calID] = &models.Calendar{ID: calID, TenantID: tenantID, OwnerID: uuid.New(), CalendarType: models.CalendarTypeShared}
		srv := newTestCalendarServerWithRepo(repo)

		_, err := srv.SubscribeToCalendar(ctxWithTenant(tenantID), &calv1.SubscribeToCalendarRequest{
			CalendarId: calID.String(),
			UserId:     uuid.New().String(),
		})
		require.NoError(t, err)
	})

	t.Run("cannot subscribe to personal calendar", func(t *testing.T) {
		repo := newStubCalendarRepo()
		repo.calendars[calID] = &models.Calendar{ID: calID, TenantID: tenantID, OwnerID: uuid.New(), CalendarType: models.CalendarTypePersonal}
		srv := newTestCalendarServerWithRepo(repo)

		_, err := srv.SubscribeToCalendar(ctxWithTenant(tenantID), &calv1.SubscribeToCalendarRequest{
			CalendarId: calID.String(),
			UserId:     uuid.New().String(),
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

func TestUnsubscribeFromCalendar(t *testing.T) {
	tenantID := uuid.New()
	calID := uuid.New()

	t.Run("happy path", func(t *testing.T) {
		repo := newStubCalendarRepo()
		ownerID := uuid.New()
		repo.calendars[calID] = &models.Calendar{ID: calID, TenantID: tenantID, OwnerID: ownerID, CalendarType: models.CalendarTypeShared}
		srv := newTestCalendarServerWithRepo(repo)

		_, err := srv.UnsubscribeFromCalendar(ctxWithTenant(tenantID), &calv1.UnsubscribeFromCalendarRequest{
			CalendarId: calID.String(),
			UserId:     uuid.New().String(),
		})
		require.NoError(t, err)
	})

	t.Run("cannot unsubscribe from own calendar", func(t *testing.T) {
		repo := newStubCalendarRepo()
		ownerID := uuid.New()
		repo.calendars[calID] = &models.Calendar{ID: calID, TenantID: tenantID, OwnerID: ownerID, CalendarType: models.CalendarTypeShared}
		srv := newTestCalendarServerWithRepo(repo)

		_, err := srv.UnsubscribeFromCalendar(ctxWithTenant(tenantID), &calv1.UnsubscribeFromCalendarRequest{
			CalendarId: calID.String(),
			UserId:     ownerID.String(),
		})
		requireGRPCCode(t, err, codes.PermissionDenied)
	})
}
