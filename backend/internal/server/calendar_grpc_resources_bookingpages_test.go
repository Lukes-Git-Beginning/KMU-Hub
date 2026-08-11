package server

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/work/calendar"
	"github.com/kmuhub/kmuhub/internal/work/holiday"
	"github.com/kmuhub/kmuhub/internal/work/resource"
	calv1 "github.com/kmuhub/kmuhub/proto/calendar/v1"
)

// ============================================================================
// stubResourceRepo — implements resource.Repository for resource/booking RPCs.
// ============================================================================

type stubResourceRepo struct {
	resources map[uuid.UUID]*models.Resource
	bookings  map[uuid.UUID]*models.ResourceBooking

	listErr          error
	createErr        error
	updateErr        error
	deleteErr        error
	conflictOnCreate bool
	alternatives     []resource.AlternativeResource
}

func newStubResourceRepo() *stubResourceRepo {
	return &stubResourceRepo{
		resources: make(map[uuid.UUID]*models.Resource),
		bookings:  make(map[uuid.UUID]*models.ResourceBooking),
	}
}

func (r *stubResourceRepo) Create(_ context.Context, res *models.Resource) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.resources[res.ID] = res
	return nil
}

func (r *stubResourceRepo) GetByID(_ context.Context, id, tenantID uuid.UUID) (*models.Resource, error) {
	res, ok := r.resources[id]
	if !ok || res.TenantID != tenantID {
		return nil, resource.ErrResourceNotFound
	}
	return res, nil
}

func (r *stubResourceRepo) List(_ context.Context, _ resource.ResourceFilters) ([]models.Resource, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	var out []models.Resource
	for _, res := range r.resources {
		out = append(out, *res)
	}
	return out, nil
}

func (r *stubResourceRepo) Update(_ context.Context, res *models.Resource) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	r.resources[res.ID] = res
	return nil
}

func (r *stubResourceRepo) Delete(_ context.Context, id, _ uuid.UUID) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	delete(r.resources, id)
	return nil
}

func (r *stubResourceRepo) SetTags(_ context.Context, _ uuid.UUID, _ []string) error { return nil }

func (r *stubResourceRepo) CreateBooking(_ context.Context, b *models.ResourceBooking) error {
	if r.conflictOnCreate {
		return resource.ErrBookingConflict
	}
	r.bookings[b.ID] = b
	return nil
}

func (r *stubResourceRepo) CancelBooking(_ context.Context, bookingID, _ uuid.UUID) error {
	b, ok := r.bookings[bookingID]
	if !ok {
		return resource.ErrBookingNotFound
	}
	now := time.Now()
	b.CancelledAt = &now
	return nil
}

func (r *stubResourceRepo) ListBookings(_ context.Context, resourceID uuid.UUID, _, _ time.Time) ([]models.ResourceBooking, error) {
	var out []models.ResourceBooking
	for _, b := range r.bookings {
		if b.ResourceID == resourceID {
			out = append(out, *b)
		}
	}
	return out, nil
}

func (r *stubResourceRepo) ListBookingsByEvent(_ context.Context, _ uuid.UUID) ([]models.ResourceBooking, error) {
	return nil, nil
}

func (r *stubResourceRepo) GetBooking(_ context.Context, bookingID, _ uuid.UUID) (*models.ResourceBooking, error) {
	b, ok := r.bookings[bookingID]
	if !ok {
		return nil, resource.ErrBookingNotFound
	}
	return b, nil
}

func (r *stubResourceRepo) FindAvailableResources(_ context.Context, _, _ time.Time, _ resource.ResourceFilters) ([]models.Resource, error) {
	return nil, nil
}

func (r *stubResourceRepo) FindAlternatives(_ context.Context, _ uuid.UUID, _, _ time.Time, _ string, _ uuid.UUID) ([]resource.AlternativeResource, error) {
	return r.alternatives, nil
}

// ============================================================================
// stubHolidayRepo — implements holiday.Repository.
// ============================================================================

type stubHolidayRepo struct {
	holidays map[string][]models.PublicHoliday // keyed by country code
	hasData  bool
	listErr  error
}

func (r *stubHolidayRepo) BulkUpsert(_ context.Context, _ []models.PublicHoliday) error { return nil }

func (r *stubHolidayRepo) ListByDateRange(_ context.Context, _, _ time.Time, countryCode string, _ *string) ([]models.PublicHoliday, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	return r.holidays[countryCode], nil
}

func (r *stubHolidayRepo) HasDataForYear(_ context.Context, _ int, _ string) (bool, error) {
	return r.hasData, nil
}

func (r *stubHolidayRepo) DeleteByYear(_ context.Context, _ int, _ string) error { return nil }

// ============================================================================
// stubBookingRepo — implements calendar.BookingRepository. GetCalendarEventsInRange
// and GetBookedSlotsForPage are intentionally left unimplemented (delegated to the
// embedded nil interface): both take an unexported eventSlot/models type that can't
// be spelled from this package, and every test case below returns before either is
// reached (page-not-found / validation / lead-time errors all short-circuit first).
// ============================================================================

type stubBookingRepo struct {
	calendar.BookingRepository

	pages       map[uuid.UUID]*models.BookingPage
	pagesBySlug map[string]*models.BookingPage

	createPageErr error
	getPageErr    error
	updatePageErr error
	listPagesErr  error
}

func newStubBookingRepo() *stubBookingRepo {
	return &stubBookingRepo{
		pages:       make(map[uuid.UUID]*models.BookingPage),
		pagesBySlug: make(map[string]*models.BookingPage),
	}
}

func (r *stubBookingRepo) CreateBookingPage(_ context.Context, p *models.BookingPage) error {
	if r.createPageErr != nil {
		return r.createPageErr
	}
	r.pages[p.ID] = p
	r.pagesBySlug[p.Slug] = p
	return nil
}

func (r *stubBookingRepo) GetBookingPageByID(_ context.Context, id, tenantID uuid.UUID) (*models.BookingPage, error) {
	if r.getPageErr != nil {
		return nil, r.getPageErr
	}
	p, ok := r.pages[id]
	if !ok || p.TenantID != tenantID {
		return nil, calendar.ErrBookingPageNotFound
	}
	return p, nil
}

func (r *stubBookingRepo) GetBookingPageBySlug(_ context.Context, slug string) (*models.BookingPage, error) {
	p, ok := r.pagesBySlug[slug]
	if !ok {
		return nil, calendar.ErrBookingPageNotFound
	}
	return p, nil
}

func (r *stubBookingRepo) UpdateBookingPage(_ context.Context, p *models.BookingPage) error {
	if r.updatePageErr != nil {
		return r.updatePageErr
	}
	r.pages[p.ID] = p
	r.pagesBySlug[p.Slug] = p
	return nil
}

func (r *stubBookingRepo) DeleteBookingPage(_ context.Context, id, tenantID uuid.UUID) error {
	p, ok := r.pages[id]
	if !ok || p.TenantID != tenantID {
		return calendar.ErrBookingPageNotFound
	}
	delete(r.pages, id)
	delete(r.pagesBySlug, p.Slug)
	return nil
}

func (r *stubBookingRepo) ListBookingPages(_ context.Context, tenantID uuid.UUID, _ bool) ([]models.BookingPage, error) {
	if r.listPagesErr != nil {
		return nil, r.listPagesErr
	}
	var out []models.BookingPage
	for _, p := range r.pages {
		if p.TenantID == tenantID {
			out = append(out, *p)
		}
	}
	return out, nil
}

func (r *stubBookingRepo) CreatePublicBooking(_ context.Context, _ *models.PublicBooking) error {
	return nil
}

// ============================================================================
// Resources
// ============================================================================

func TestCreateResource_Success(t *testing.T) {
	repo := newStubResourceRepo()
	srv := NewCalendarGRPCServer(nil, nil, resource.NewService(repo), nil, nil)

	cap := int32(4)
	resp, err := srv.CreateResource(ctxWithTenant(uuid.New()), &calv1.CreateResourceRequest{
		Name:         "Meeting Room 1",
		ResourceType: models.ResourceTypeRoom,
		Capacity:     &cap,
		CreatedBy:    uuid.New().String(),
		Tags:         []string{"quiet"},
	})
	require.NoError(t, err)
	assert.Equal(t, "Meeting Room 1", resp.Resource.Name)
	assert.Equal(t, int32(4), *resp.Resource.Capacity)
}

func TestCreateResource_InvalidResourceType(t *testing.T) {
	repo := newStubResourceRepo()
	srv := NewCalendarGRPCServer(nil, nil, resource.NewService(repo), nil, nil)

	_, err := srv.CreateResource(ctxWithTenant(uuid.New()), &calv1.CreateResourceRequest{
		Name:         "Broken",
		ResourceType: "spaceship",
		CreatedBy:    uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetResource_Success(t *testing.T) {
	repo := newStubResourceRepo()
	tenantID := uuid.New()
	res := &models.Resource{ID: uuid.New(), TenantID: tenantID, Name: "Room A", ResourceType: models.ResourceTypeRoom, IsActive: true}
	repo.resources[res.ID] = res
	srv := NewCalendarGRPCServer(nil, nil, resource.NewService(repo), nil, nil)

	resp, err := srv.GetResource(ctxWithTenant(tenantID), &calv1.GetResourceRequest{Id: res.ID.String()})
	require.NoError(t, err)
	assert.Equal(t, "Room A", resp.Resource.Name)
}

func TestGetResource_NotFound(t *testing.T) {
	repo := newStubResourceRepo()
	srv := NewCalendarGRPCServer(nil, nil, resource.NewService(repo), nil, nil)

	_, err := srv.GetResource(ctxWithTenant(uuid.New()), &calv1.GetResourceRequest{Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestListResources_Success(t *testing.T) {
	repo := newStubResourceRepo()
	tenantID := uuid.New()
	repo.resources[uuid.New()] = &models.Resource{ID: uuid.New(), TenantID: tenantID, Name: "Room A", ResourceType: models.ResourceTypeRoom, IsActive: true}
	srv := NewCalendarGRPCServer(nil, nil, resource.NewService(repo), nil, nil)

	resp, err := srv.ListResources(ctxWithTenant(tenantID), &calv1.ListResourcesRequest{})
	require.NoError(t, err)
	assert.Len(t, resp.Resources, 1)
}

// Regression for fix-calendar-grpc-nil-slice-wire-shape.
func TestListResources_EmptyIsNotNil(t *testing.T) {
	repo := newStubResourceRepo()
	srv := NewCalendarGRPCServer(nil, nil, resource.NewService(repo), nil, nil)

	resp, err := srv.ListResources(ctxWithTenant(uuid.New()), &calv1.ListResourcesRequest{})
	require.NoError(t, err)
	require.NotNil(t, resp.Resources)
	assert.Empty(t, resp.Resources)
}

func TestListResources_RepoError(t *testing.T) {
	repo := newStubResourceRepo()
	repo.listErr = assert.AnError
	srv := NewCalendarGRPCServer(nil, nil, resource.NewService(repo), nil, nil)

	_, err := srv.ListResources(ctxWithTenant(uuid.New()), &calv1.ListResourcesRequest{})
	requireGRPCCode(t, err, codes.Internal)
}

func TestUpdateResource_Success(t *testing.T) {
	repo := newStubResourceRepo()
	tenantID := uuid.New()
	res := &models.Resource{ID: uuid.New(), TenantID: tenantID, Name: "Old Name", ResourceType: models.ResourceTypeRoom, IsActive: true}
	repo.resources[res.ID] = res
	srv := NewCalendarGRPCServer(nil, nil, resource.NewService(repo), nil, nil)

	newName := "New Name"
	resp, err := srv.UpdateResource(ctxWithTenant(tenantID), &calv1.UpdateResourceRequest{
		Id:   res.ID.String(),
		Name: &newName,
	})
	require.NoError(t, err)
	assert.Equal(t, "New Name", resp.Resource.Name)
}

func TestUpdateResource_NotFound(t *testing.T) {
	repo := newStubResourceRepo()
	srv := NewCalendarGRPCServer(nil, nil, resource.NewService(repo), nil, nil)

	newName := "New Name"
	_, err := srv.UpdateResource(ctxWithTenant(uuid.New()), &calv1.UpdateResourceRequest{
		Id:   uuid.New().String(),
		Name: &newName,
	})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestDeleteResource_Success(t *testing.T) {
	repo := newStubResourceRepo()
	tenantID := uuid.New()
	res := &models.Resource{ID: uuid.New(), TenantID: tenantID, Name: "Room A", ResourceType: models.ResourceTypeRoom, IsActive: true}
	repo.resources[res.ID] = res
	srv := NewCalendarGRPCServer(nil, nil, resource.NewService(repo), nil, nil)

	_, err := srv.DeleteResource(ctxWithTenant(tenantID), &calv1.DeleteResourceRequest{Id: res.ID.String()})
	require.NoError(t, err)
	_, stillThere := repo.resources[res.ID]
	assert.False(t, stillThere)
}

func TestDeleteResource_RepoError(t *testing.T) {
	repo := newStubResourceRepo()
	repo.deleteErr = resource.ErrResourceNotFound
	srv := NewCalendarGRPCServer(nil, nil, resource.NewService(repo), nil, nil)

	_, err := srv.DeleteResource(ctxWithTenant(uuid.New()), &calv1.DeleteResourceRequest{Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestListResourceAvailability_Success(t *testing.T) {
	repo := newStubResourceRepo()
	tenantID := uuid.New()
	res := &models.Resource{ID: uuid.New(), TenantID: tenantID, Name: "Room A", ResourceType: models.ResourceTypeRoom, IsActive: true}
	repo.resources[res.ID] = res
	srv := NewCalendarGRPCServer(nil, nil, resource.NewService(repo), nil, nil)

	resp, err := srv.ListResourceAvailability(ctxWithTenant(tenantID), &calv1.ListResourceAvailabilityRequest{
		ResourceId: res.ID.String(),
		Start:      timestamppb.New(time.Now()),
		End:        timestamppb.New(time.Now().Add(24 * time.Hour)),
	})
	require.NoError(t, err)
	// Regression for fix-calendar-grpc-nil-slice-wire-shape: the empty result
	// must serialize as [] over the wire, not null.
	require.NotNil(t, resp.Bookings)
	assert.Empty(t, resp.Bookings)
}

func TestListResourceAvailability_InvalidResourceID(t *testing.T) {
	repo := newStubResourceRepo()
	srv := NewCalendarGRPCServer(nil, nil, resource.NewService(repo), nil, nil)

	_, err := srv.ListResourceAvailability(ctxWithTenant(uuid.New()), &calv1.ListResourceAvailabilityRequest{
		ResourceId: "not-a-uuid",
		Start:      timestamppb.New(time.Now()),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestBookResource_Success(t *testing.T) {
	repo := newStubResourceRepo()
	tenantID := uuid.New()
	res := &models.Resource{ID: uuid.New(), TenantID: tenantID, Name: "Room A", ResourceType: models.ResourceTypeRoom, IsActive: true}
	repo.resources[res.ID] = res
	srv := NewCalendarGRPCServer(nil, nil, resource.NewService(repo), nil, nil)

	start := time.Now().Add(time.Hour)
	resp, err := srv.BookResource(ctxWithTenant(tenantID), &calv1.BookResourceRequest{
		ResourceId: res.ID.String(),
		EventId:    uuid.New().String(),
		BookedBy:   uuid.New().String(),
		StartTime:  timestamppb.New(start),
		EndTime:    timestamppb.New(start.Add(time.Hour)),
	})
	require.NoError(t, err)
	assert.Equal(t, "Room A", resp.Booking.ResourceName)
}

func TestBookResource_ConflictWithAlternatives(t *testing.T) {
	repo := newStubResourceRepo()
	tenantID := uuid.New()
	res := &models.Resource{ID: uuid.New(), TenantID: tenantID, Name: "Room A", ResourceType: models.ResourceTypeRoom, IsActive: true}
	repo.resources[res.ID] = res
	repo.conflictOnCreate = true
	repo.alternatives = []resource.AlternativeResource{{ResourceID: uuid.New(), ResourceName: "Room B"}}
	srv := NewCalendarGRPCServer(nil, nil, resource.NewService(repo), nil, nil)

	start := time.Now().Add(time.Hour)
	_, err := srv.BookResource(ctxWithTenant(tenantID), &calv1.BookResourceRequest{
		ResourceId: res.ID.String(),
		EventId:    uuid.New().String(),
		BookedBy:   uuid.New().String(),
		StartTime:  timestamppb.New(start),
		EndTime:    timestamppb.New(start.Add(time.Hour)),
	})
	requireGRPCCode(t, err, codes.AlreadyExists)
	assert.Contains(t, err.Error(), "Room B")
}

func TestListResourceBookings_Success(t *testing.T) {
	repo := newStubResourceRepo()
	tenantID := uuid.New()
	res := &models.Resource{ID: uuid.New(), TenantID: tenantID, Name: "Room A", ResourceType: models.ResourceTypeRoom, IsActive: true}
	repo.resources[res.ID] = res
	srv := NewCalendarGRPCServer(nil, nil, resource.NewService(repo), nil, nil)

	resourceID := res.ID.String()
	resp, err := srv.ListResourceBookings(ctxWithTenant(tenantID), &calv1.ListResourceBookingsRequest{
		ResourceId: &resourceID,
		Start:      timestamppb.New(time.Now()),
		End:        timestamppb.New(time.Now().Add(24 * time.Hour)),
	})
	require.NoError(t, err)
	// Regression for fix-calendar-grpc-nil-slice-wire-shape: the empty result
	// must serialize as [] over the wire, not null.
	require.NotNil(t, resp.Bookings)
	assert.Empty(t, resp.Bookings)
}

func TestListResourceBookings_MissingResourceID(t *testing.T) {
	repo := newStubResourceRepo()
	srv := NewCalendarGRPCServer(nil, nil, resource.NewService(repo), nil, nil)

	_, err := srv.ListResourceBookings(ctxWithTenant(uuid.New()), &calv1.ListResourceBookingsRequest{
		Start: timestamppb.New(time.Now()),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// ============================================================================
// Holidays
// ============================================================================

func TestListHolidays_Success(t *testing.T) {
	repo := &stubHolidayRepo{
		hasData: true,
		holidays: map[string][]models.PublicHoliday{
			"DE": {{ID: uuid.New(), Name: "Tag der Arbeit", CountryCode: "DE", Year: time.Now().Year()}},
		},
	}
	srv := NewCalendarGRPCServer(nil, nil, nil, holiday.NewService(repo, nil), nil)

	resp, err := srv.ListHolidays(context.Background(), &calv1.ListHolidaysRequest{
		CountryCode: "DE",
		Start:       timestamppb.New(time.Now()),
		End:         timestamppb.New(time.Now().Add(10 * 24 * time.Hour)),
	})
	require.NoError(t, err)
	require.Len(t, resp.Holidays, 1)
	assert.Equal(t, "Tag der Arbeit", resp.Holidays[0].Name)
}

// Regression for fix-calendar-grpc-nil-slice-wire-shape.
func TestListHolidays_EmptyIsNotNil(t *testing.T) {
	repo := &stubHolidayRepo{hasData: true}
	srv := NewCalendarGRPCServer(nil, nil, nil, holiday.NewService(repo, nil), nil)

	resp, err := srv.ListHolidays(context.Background(), &calv1.ListHolidaysRequest{
		CountryCode: "DE",
		Start:       timestamppb.New(time.Now()),
		End:         timestamppb.New(time.Now().Add(10 * 24 * time.Hour)),
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Holidays)
	assert.Empty(t, resp.Holidays)
}

func TestListHolidays_InvalidCountryCode(t *testing.T) {
	repo := &stubHolidayRepo{hasData: true}
	srv := NewCalendarGRPCServer(nil, nil, nil, holiday.NewService(repo, nil), nil)

	_, err := srv.ListHolidays(context.Background(), &calv1.ListHolidaysRequest{
		CountryCode: "XX",
		Start:       timestamppb.New(time.Now()),
		End:         timestamppb.New(time.Now().Add(24 * time.Hour)),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestSeedHolidays_InvalidYear(t *testing.T) {
	repo := &stubHolidayRepo{}
	srv := NewCalendarGRPCServer(nil, nil, nil, holiday.NewService(repo, nil), nil)

	_, err := srv.SeedHolidays(context.Background(), &calv1.SeedHolidaysRequest{
		CountryCode: "DE",
		Year:        1500,
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestSeedHolidays_NagerUnavailable(t *testing.T) {
	repo := &stubHolidayRepo{}
	srv := NewCalendarGRPCServer(nil, nil, nil, holiday.NewService(repo, nil), nil)

	_, err := srv.SeedHolidays(context.Background(), &calv1.SeedHolidaysRequest{
		CountryCode: "DE",
		Year:        2026,
	})
	requireGRPCCode(t, err, codes.Unavailable)
}

// ============================================================================
// Preferences
// ============================================================================

func TestGetCalendarPreferences_Success(t *testing.T) {
	repo := newStubCalendarRepo()
	userID := uuid.New()
	repo.prefs[userID] = &models.UserCalendarPreferences{UserID: userID, DefaultView: "month", WeekDays: 7}
	srv := NewCalendarGRPCServer(calendar.NewService(repo), nil, nil, nil, nil)

	resp, err := srv.GetCalendarPreferences(context.Background(), &calv1.GetCalendarPreferencesRequest{UserId: userID.String()})
	require.NoError(t, err)
	assert.Equal(t, "month", resp.Preferences.DefaultView)
	assert.Equal(t, int32(7), resp.Preferences.WeekDays)
}

func TestGetCalendarPreferences_InvalidUserID(t *testing.T) {
	repo := newStubCalendarRepo()
	srv := NewCalendarGRPCServer(calendar.NewService(repo), nil, nil, nil, nil)

	_, err := srv.GetCalendarPreferences(context.Background(), &calv1.GetCalendarPreferencesRequest{UserId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestUpdateCalendarPreferences_Success(t *testing.T) {
	repo := newStubCalendarRepo()
	userID := uuid.New()
	srv := NewCalendarGRPCServer(calendar.NewService(repo), nil, nil, nil, nil)

	view := "day"
	resp, err := srv.UpdateCalendarPreferences(context.Background(), &calv1.UpdateCalendarPreferencesRequest{
		UserId:      userID.String(),
		DefaultView: &view,
	})
	require.NoError(t, err)
	assert.Equal(t, "day", resp.Preferences.DefaultView)
	assert.Equal(t, "day", repo.prefs[userID].DefaultView)
}

func TestUpdateCalendarPreferences_InvalidUserID(t *testing.T) {
	repo := newStubCalendarRepo()
	srv := NewCalendarGRPCServer(calendar.NewService(repo), nil, nil, nil, nil)

	_, err := srv.UpdateCalendarPreferences(context.Background(), &calv1.UpdateCalendarPreferencesRequest{UserId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// ============================================================================
// Booking Pages — admin
// ============================================================================

func newBookingServer(repo calendar.BookingRepository) *CalendarGRPCServer {
	srv := NewCalendarGRPCServer(nil, nil, nil, nil, nil)
	srv.SetBookingService(calendar.NewBookingService(repo))
	return srv
}

func TestCreateBookingPage_ServiceNotConfigured(t *testing.T) {
	srv := NewCalendarGRPCServer(nil, nil, nil, nil, nil) // bookingService left nil

	_, err := srv.CreateBookingPage(ctxWithTenant(uuid.New()), &calv1.CreateBookingPageRequest{
		CalendarId: uuid.New().String(),
		Slug:       "team-standup",
	})
	requireGRPCCode(t, err, codes.Unimplemented)
}

func TestCreateBookingPage_Success(t *testing.T) {
	repo := newStubBookingRepo()
	srv := newBookingServer(repo)

	resp, err := srv.CreateBookingPage(ctxWithTenant(uuid.New()), &calv1.CreateBookingPageRequest{
		CalendarId:  uuid.New().String(),
		Slug:        "Team-Standup",
		CompanyName: "Acme GmbH",
		Services: []*calv1.BookingServiceProto{
			{Id: "svc1", Name: "Standup", DurationMin: 30, Price: "0.00"},
		},
		AvailabilityRulesJson: `{"weekdays":[1,2,3,4,5],"slots_per_weekday":[{"start":"09:00","end":"17:00"}],"slot_duration_min":30}`,
	})
	require.NoError(t, err)
	assert.Equal(t, "team-standup", resp.Page.Slug)
	assert.Equal(t, "Acme GmbH", resp.Page.CompanyName)
}

func TestGetBookingPage_Success(t *testing.T) {
	repo := newStubBookingRepo()
	tenantID := uuid.New()
	page := &models.BookingPage{ID: uuid.New(), TenantID: tenantID, Slug: "team-standup", CompanyName: "Acme"}
	repo.pages[page.ID] = page
	repo.pagesBySlug[page.Slug] = page
	srv := newBookingServer(repo)

	resp, err := srv.GetBookingPage(ctxWithTenant(tenantID), &calv1.GetBookingPageRequest{Id: page.ID.String()})
	require.NoError(t, err)
	assert.Equal(t, "Acme", resp.Page.CompanyName)
}

func TestGetBookingPage_NotFound(t *testing.T) {
	repo := newStubBookingRepo()
	srv := newBookingServer(repo)

	_, err := srv.GetBookingPage(ctxWithTenant(uuid.New()), &calv1.GetBookingPageRequest{Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestListBookingPages_Success(t *testing.T) {
	repo := newStubBookingRepo()
	tenantID := uuid.New()
	page := &models.BookingPage{ID: uuid.New(), TenantID: tenantID, Slug: "team-standup", CompanyName: "Acme"}
	repo.pages[page.ID] = page
	srv := newBookingServer(repo)

	resp, err := srv.ListBookingPages(ctxWithTenant(tenantID), &calv1.ListBookingPagesRequest{})
	require.NoError(t, err)
	assert.Len(t, resp.Pages, 1)
}

// Regression for fix-calendar-grpc-nil-slice-wire-shape.
func TestListBookingPages_EmptyIsNotNil(t *testing.T) {
	repo := newStubBookingRepo()
	srv := newBookingServer(repo)

	resp, err := srv.ListBookingPages(ctxWithTenant(uuid.New()), &calv1.ListBookingPagesRequest{})
	require.NoError(t, err)
	require.NotNil(t, resp.Pages)
	assert.Empty(t, resp.Pages)
}

func TestListBookingPages_RepoError(t *testing.T) {
	repo := newStubBookingRepo()
	repo.listPagesErr = assert.AnError
	srv := newBookingServer(repo)

	_, err := srv.ListBookingPages(ctxWithTenant(uuid.New()), &calv1.ListBookingPagesRequest{})
	requireGRPCCode(t, err, codes.Internal)
}

func TestUpdateBookingPage_Success(t *testing.T) {
	repo := newStubBookingRepo()
	tenantID := uuid.New()
	page := &models.BookingPage{ID: uuid.New(), TenantID: tenantID, Slug: "team-standup", CompanyName: "Old Name"}
	repo.pages[page.ID] = page
	repo.pagesBySlug[page.Slug] = page
	srv := newBookingServer(repo)

	newName := "New Name"
	resp, err := srv.UpdateBookingPage(ctxWithTenant(tenantID), &calv1.UpdateBookingPageRequest{
		Id:          page.ID.String(),
		CompanyName: &newName,
	})
	require.NoError(t, err)
	assert.Equal(t, "New Name", resp.Page.CompanyName)
}

func TestUpdateBookingPage_NotFound(t *testing.T) {
	repo := newStubBookingRepo()
	srv := newBookingServer(repo)

	newName := "New Name"
	_, err := srv.UpdateBookingPage(ctxWithTenant(uuid.New()), &calv1.UpdateBookingPageRequest{
		Id:          uuid.New().String(),
		CompanyName: &newName,
	})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestDeleteBookingPage_Success(t *testing.T) {
	repo := newStubBookingRepo()
	tenantID := uuid.New()
	page := &models.BookingPage{ID: uuid.New(), TenantID: tenantID, Slug: "team-standup", CompanyName: "Acme"}
	repo.pages[page.ID] = page
	repo.pagesBySlug[page.Slug] = page
	srv := newBookingServer(repo)

	_, err := srv.DeleteBookingPage(ctxWithTenant(tenantID), &calv1.DeleteBookingPageRequest{Id: page.ID.String()})
	require.NoError(t, err)
	_, stillThere := repo.pages[page.ID]
	assert.False(t, stillThere)
}

func TestDeleteBookingPage_NotFound(t *testing.T) {
	repo := newStubBookingRepo()
	srv := newBookingServer(repo)

	_, err := srv.DeleteBookingPage(ctxWithTenant(uuid.New()), &calv1.DeleteBookingPageRequest{Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

// ============================================================================
// Booking Pages — public
// ============================================================================

func TestGetPublicBookingPage_Success(t *testing.T) {
	repo := newStubBookingRepo()
	page := &models.BookingPage{ID: uuid.New(), TenantID: uuid.New(), Slug: "team-standup", CompanyName: "Acme"}
	repo.pages[page.ID] = page
	repo.pagesBySlug[page.Slug] = page
	srv := newBookingServer(repo)

	resp, err := srv.GetPublicBookingPage(context.Background(), &calv1.GetPublicBookingPageRequest{Slug: "team-standup"})
	require.NoError(t, err)
	assert.Equal(t, "Acme", resp.Page.CompanyName)
}

func TestGetPublicBookingPage_NotFound(t *testing.T) {
	repo := newStubBookingRepo()
	srv := newBookingServer(repo)

	_, err := srv.GetPublicBookingPage(context.Background(), &calv1.GetPublicBookingPageRequest{Slug: "missing"})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestGetAvailability_InvalidDateFrom(t *testing.T) {
	repo := newStubBookingRepo()
	srv := newBookingServer(repo)

	_, err := srv.GetAvailability(context.Background(), &calv1.GetAvailabilityRequest{
		Slug:     "team-standup",
		DateFrom: "not-a-date",
		DateTo:   "2026-01-10",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetAvailability_PageNotFound(t *testing.T) {
	repo := newStubBookingRepo()
	srv := newBookingServer(repo)

	_, err := srv.GetAvailability(context.Background(), &calv1.GetAvailabilityRequest{
		Slug:     "missing",
		DateFrom: "2026-01-05",
		DateTo:   "2026-01-10",
	})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestCreatePublicBooking_CustomerRequired(t *testing.T) {
	repo := newStubBookingRepo()
	srv := newBookingServer(repo)

	_, err := srv.CreatePublicBooking(context.Background(), &calv1.CreatePublicBookingRequest{
		Slug:     "team-standup",
		Date:     "2026-01-05",
		TimeSlot: "10:00",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreatePublicBooking_PageNotFound(t *testing.T) {
	repo := newStubBookingRepo()
	srv := newBookingServer(repo)

	_, err := srv.CreatePublicBooking(context.Background(), &calv1.CreatePublicBookingRequest{
		Slug:     "missing",
		Date:     "2026-01-05",
		TimeSlot: "10:00",
		Customer: &calv1.PublicBookingCustomerProto{Name: "Jane Doe", Email: "jane@example.com"},
	})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestCreatePublicBooking_SlotInPast(t *testing.T) {
	repo := newStubBookingRepo()
	page := &models.BookingPage{
		ID:          uuid.New(),
		TenantID:    uuid.New(),
		Slug:        "team-standup",
		CompanyName: "Acme",
		Services:    []models.BookingService{{ID: "svc1", Name: "Standup", DurationMin: 30}},
		AvailabilityRules: models.AvailabilityRules{
			LeadTimeHours: 999999, // guarantees any slot is "too soon" regardless of the date below
		},
		Active: true,
	}
	repo.pages[page.ID] = page
	repo.pagesBySlug[page.Slug] = page
	srv := newBookingServer(repo)

	_, err := srv.CreatePublicBooking(context.Background(), &calv1.CreatePublicBookingRequest{
		Slug:      "team-standup",
		ServiceId: "svc1",
		Date:      time.Now().Format("2006-01-02"),
		TimeSlot:  "10:00",
		Customer:  &calv1.PublicBookingCustomerProto{Name: "Jane Doe", Email: "jane@example.com"},
	})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}
