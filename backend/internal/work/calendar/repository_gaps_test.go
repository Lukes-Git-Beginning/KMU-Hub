package calendar

// Closes the coverage gap on postgres_repository.go and
// booking_postgres_repository.go beyond what tenant_write_test.go,
// tenant_isolation_phase2_test.go and booking_slug_unique_test.go already
// exercise: member permission/visibility lifecycle, discovery/subscription,
// event categories, preferences, EnsurePersonalCalendar idempotency, the full
// booking page + public booking write/read surface, and the
// GetCalendarEventsInRange overlap boundary used for staff availability.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// seedCalendarWithOwner creates a fresh user and a calendar of the given type
// owned by that user, in the given tenant.
func seedCalendarWithOwner(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, calendarType string) (calendarID, ownerID uuid.UUID) {
	t.Helper()
	ownerID = seedCalendarUser(t, pool, tenantID)
	calendarID = testutil.SeedRow(t, pool, "calendars", map[string]any{
		"tenant_id":     tenantID,
		"name":          "Repo-Gap-" + uuid.New().String()[:8],
		"owner_id":      ownerID,
		"calendar_type": calendarType,
	})
	return calendarID, ownerID
}

// TestCalendarMembers_PermissionAndVisibilityLifecycle covers the first
// priority named in this unit's backlog scope: a membership change must
// actually change what ListByUser returns, not just what the members table
// contains.
func TestCalendarMembers_PermissionAndVisibilityLifecycle(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Calendar Member Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenant)

	calID, ownerID := seedCalendarWithOwner(t, pool, tenant, models.CalendarTypeShared)
	defer testutil.CleanupRow(t, pool, "users", ownerID)
	defer testutil.CleanupRow(t, pool, "calendars", calID)

	member := seedCalendarUser(t, pool, tenant)
	defer testutil.CleanupRow(t, pool, "users", member)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	before, err := repo.ListByUser(ctx, member, tenant)
	require.NoError(t, err)
	for _, c := range before {
		if c.ID == calID {
			t.Fatalf("calendar visible before membership was granted")
		}
	}

	now := time.Now().UTC()
	err = repo.AddMember(ctx, &models.CalendarMember{
		CalendarID: calID, TenantID: tenant, UserID: member,
		Permission: models.CalendarPermissionView, IsVisible: true, CreatedAt: now,
	})
	require.NoError(t, err)

	err = repo.AddMember(ctx, &models.CalendarMember{
		CalendarID: calID, TenantID: tenant, UserID: member,
		Permission: models.CalendarPermissionView, IsVisible: true, CreatedAt: now,
	})
	require.ErrorIs(t, err, ErrMemberAlreadyExists)

	after, err := repo.ListByUser(ctx, member, tenant)
	require.NoError(t, err)
	found := false
	for _, c := range after {
		if c.ID == calID {
			found = true
			assert.Equal(t, models.CalendarPermissionView, c.Permission)
		}
	}
	assert.True(t, found, "calendar must be visible via ListByUser after membership was granted")

	got, err := repo.GetMember(ctx, calID, member)
	require.NoError(t, err)
	assert.Equal(t, models.CalendarPermissionView, got.Permission)
	assert.True(t, got.IsVisible)

	require.NoError(t, repo.UpdateMemberPermission(ctx, calID, member, models.CalendarPermissionEdit))
	require.NoError(t, repo.UpdateMemberVisibility(ctx, calID, member, false))
	color := "#ff0000"
	require.NoError(t, repo.UpdateMemberColorOverride(ctx, calID, member, &color))

	got, err = repo.GetMember(ctx, calID, member)
	require.NoError(t, err)
	assert.Equal(t, models.CalendarPermissionEdit, got.Permission)
	assert.False(t, got.IsVisible)
	require.NotNil(t, got.ColorOverride)
	assert.Equal(t, color, *got.ColorOverride)

	members, err := repo.ListMembers(ctx, calID)
	require.NoError(t, err)
	require.Len(t, members, 1)
	assert.Equal(t, member, members[0].UserID)

	require.NoError(t, repo.RemoveMember(ctx, calID, member))
	err = repo.RemoveMember(ctx, calID, member)
	require.ErrorIs(t, err, ErrMemberNotFound)

	_, err = repo.GetMember(ctx, calID, member)
	require.ErrorIs(t, err, ErrMemberNotFound)

	final, err := repo.ListByUser(ctx, member, tenant)
	require.NoError(t, err)
	for _, c := range final {
		if c.ID == calID {
			t.Fatalf("calendar still visible via ListByUser after RemoveMember — removal did not affect visibility")
		}
	}
}

// TestCalendarGetByID_CrossTenantNotFound covers the cross-tenant read path
// required by this unit's done_when: RLS, not just the explicit tenant_id
// filter, must block the read even when the caller supplies the victim's
// real tenant ID as the method argument.
func TestCalendarGetByID_CrossTenantNotFound(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "GetByID Own Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "GetByID Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	calID, ownerID := seedCalendarWithOwner(t, pool, tenantOwn, models.CalendarTypePersonal)
	defer testutil.CleanupRow(t, pool, "users", ownerID)
	defer testutil.CleanupRow(t, pool, "calendars", calID)

	repo := NewPostgresRepository(pool)

	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	_, err := repo.GetByID(ctxOther, calID, tenantOther)
	require.ErrorIs(t, err, ErrCalendarNotFound)

	// Even passing the victim's real tenantID as the SQL argument must stay
	// blocked — RLS scopes on the connection's app.tenant_id (from ctx), not
	// on this parameter.
	_, err = repo.GetByID(ctxOther, calID, tenantOwn)
	require.ErrorIs(t, err, ErrCalendarNotFound)

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	got, err := repo.GetByID(ctxOwn, calID, tenantOwn)
	require.NoError(t, err)
	assert.Equal(t, calID, got.ID)
}

// TestListBrowsable_UnusedSecondParameter_DocumentsCurrentGap proves
// ListBrowsable is unconditionally broken: its query binds three arguments
// (userID, userID, tenantID) for placeholders $1 and $3, but $2 never
// appears anywhere in the SQL text. Postgres's extended query protocol
// cannot infer a type for a parameter the statement never references, so
// EVERY call fails with SQLSTATE 42P18 regardless of input — this has
// nothing to do with RLS or fixture data. Not fixed here — a real behavior
// change is out of scope for a coverage unit; filed as
// fix-work-calendar-listbrowsable-broken-query in BACKLOG.yml.
func TestListBrowsable_UnusedSecondParameter_DocumentsCurrentGap(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "ListBrowsable Gap Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenant)

	seeker := seedCalendarUser(t, pool, tenant)
	defer testutil.CleanupRow(t, pool, "users", seeker)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	_, err := repo.ListBrowsable(ctx, seeker, tenant)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "42P18",
		"ListBrowsable binds 3 params (userID, userID, tenantID) but the query only references $1 and $3 — $2 is always unused")
}

// TestCalendarSubscription_SubscribeAndUnsubscribe covers Subscribe and
// Unsubscribe directly (via GetMember/calendar_members), independent of the
// broken ListBrowsable — see TestListBrowsable_UnusedSecondParameter_DocumentsCurrentGap.
func TestCalendarSubscription_SubscribeAndUnsubscribe(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Calendar Subscription Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenant)

	sharedCalID, sharedOwnerID := seedCalendarWithOwner(t, pool, tenant, models.CalendarTypeShared)
	defer testutil.CleanupRow(t, pool, "users", sharedOwnerID)
	defer testutil.CleanupRow(t, pool, "calendars", sharedCalID)

	seeker := seedCalendarUser(t, pool, tenant)
	defer testutil.CleanupRow(t, pool, "users", seeker)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	require.NoError(t, repo.Subscribe(ctx, sharedCalID, seeker))

	err := repo.Subscribe(ctx, sharedCalID, seeker)
	require.ErrorIs(t, err, ErrSubscriptionExists)

	member, err := repo.GetMember(ctx, sharedCalID, seeker)
	require.NoError(t, err)
	assert.Equal(t, models.CalendarPermissionView, member.Permission)
	assert.True(t, member.IsVisible)

	require.NoError(t, repo.Unsubscribe(ctx, sharedCalID, seeker))
	err = repo.Unsubscribe(ctx, sharedCalID, seeker)
	require.ErrorIs(t, err, ErrSubscriptionNotFound)
}

// TestEventCategories_CreateListDelete covers the category CRUD surface and
// the case-insensitive per-user uniqueness index (idx_event_categories_name).
func TestEventCategories_CreateListDelete(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Event Category Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenant)

	user := seedCalendarUser(t, pool, tenant)
	defer testutil.CleanupRow(t, pool, "users", user)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	now := time.Now().UTC()
	cat := &models.EventCategory{
		ID: uuid.New(), TenantID: tenant, UserID: user,
		Name: "Arbeit", Color: "#3d8abf", SortOrder: 1, CreatedAt: now,
	}
	require.NoError(t, repo.CreateCategory(ctx, cat))
	defer testutil.CleanupRow(t, pool, "event_categories", cat.ID)

	dup := &models.EventCategory{
		ID: uuid.New(), TenantID: tenant, UserID: user,
		Name: "ARBEIT", Color: "#000000", SortOrder: 2, CreatedAt: now,
	}
	err := repo.CreateCategory(ctx, dup)
	require.ErrorIs(t, err, ErrDuplicateCategoryName)

	cats, err := repo.ListCategories(ctx, user, tenant)
	require.NoError(t, err)
	require.Len(t, cats, 1)
	assert.Equal(t, "Arbeit", cats[0].Name)

	otherUser := seedCalendarUser(t, pool, tenant)
	defer testutil.CleanupRow(t, pool, "users", otherUser)

	err = repo.DeleteCategory(ctx, cat.ID, otherUser, tenant)
	require.ErrorIs(t, err, ErrCategoryNotFound, "another user's category delete must not succeed")

	require.NoError(t, repo.DeleteCategory(ctx, cat.ID, user, tenant))
	err = repo.DeleteCategory(ctx, cat.ID, user, tenant)
	require.ErrorIs(t, err, ErrCategoryNotFound)
}

// TestUserCalendarPreferences_UpsertAndGet covers the nil-on-missing GetPreferences
// contract and the ON CONFLICT update path of UpsertPreferences.
func TestUserCalendarPreferences_UpsertAndGet(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Calendar Prefs Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenant)

	user := seedCalendarUser(t, pool, tenant)
	// user_calendar_preferences.user_id is PK with ON DELETE CASCADE from
	// users — no separate row cleanup needed (and CleanupRow assumes an "id"
	// column, which this table doesn't have).
	defer testutil.CleanupRow(t, pool, "users", user)

	repo := NewPostgresRepository(pool)
	// UpsertPreferences' INSERT derives tenant_id from a subquery on users,
	// but the WITH CHECK clause of the RLS policy still evaluates against
	// app.tenant_id from ctx — a bare context.Background() has none set and
	// the insert is rejected before the subquery is even considered.
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	got, err := repo.GetPreferences(ctx, user)
	require.NoError(t, err)
	assert.Nil(t, got, "no row yet must return nil, nil — not an error")

	now := time.Now().UTC()
	prefs := &models.UserCalendarPreferences{
		UserID: user, DefaultView: "week", WeekDays: 7,
		DefaultReminderMinutes: 15, DefaultAllDayReminderMinutes: 0,
		ShowTaskDeadlines: true, UpdatedAt: now,
	}
	require.NoError(t, repo.UpsertPreferences(ctx, prefs))

	got, err = repo.GetPreferences(ctx, user)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "week", got.DefaultView)
	assert.True(t, got.ShowTaskDeadlines)

	prefs.DefaultView = "day"
	prefs.ShowTaskDeadlines = false
	require.NoError(t, repo.UpsertPreferences(ctx, prefs))

	got, err = repo.GetPreferences(ctx, user)
	require.NoError(t, err)
	assert.Equal(t, "day", got.DefaultView, "second Upsert must update in place, not fail on the PK conflict")
	assert.False(t, got.ShowTaskDeadlines)
}

// TestEnsurePersonalCalendar_IdempotentAndCreates covers the find-or-create
// semantics: a second call must return the SAME calendar, not a duplicate.
func TestEnsurePersonalCalendar_IdempotentAndCreates(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Ensure Personal Calendar Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenant)

	user := seedCalendarUser(t, pool, tenant)
	defer testutil.CleanupRow(t, pool, "users", user)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	first, err := repo.EnsurePersonalCalendar(ctx, user, tenant)
	require.NoError(t, err)
	defer testutil.CleanupRow(t, pool, "calendars", first.ID)
	assert.Equal(t, models.CalendarTypePersonal, first.CalendarType)
	assert.True(t, first.IsDefault)

	second, err := repo.EnsurePersonalCalendar(ctx, user, tenant)
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID, "second call must return the same calendar, not create a duplicate")

	var count int
	err = pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT count(*) FROM calendars WHERE owner_id = $1 AND is_default = true AND calendar_type = 'personal'`,
		user,
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

// TestBookingPages_CRUDAndListFiltering covers the booking page CRUD surface
// beyond the global-slug-uniqueness test in booking_slug_unique_test.go.
func TestBookingPages_CRUDAndListFiltering(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Booking Page CRUD Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenant)

	calID, ownerID := seedCalendarWithOwner(t, pool, tenant, models.CalendarTypeShared)
	defer testutil.CleanupRow(t, pool, "users", ownerID)
	defer testutil.CleanupRow(t, pool, "calendars", calID)

	repo := NewPostgresBookingRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	now := time.Now().UTC()
	slug := "gap-test-" + uuid.New().String()[:8]
	page := &models.BookingPage{
		ID: uuid.New(), TenantID: tenant, CalendarID: calID, Slug: slug,
		CompanyName: "Gap GmbH",
		Services:    []models.BookingService{{ID: "svc-1", Name: "Beratung", DurationMin: 30, Price: "50.00"}},
		AvailabilityRules: models.AvailabilityRules{
			Weekdays: []int{1, 2, 3, 4, 5}, SlotDurationMin: 30, BufferMin: 0, LeadTimeHours: 24,
		},
		Active: true, CreatedAt: now, UpdatedAt: now,
	}
	require.NoError(t, repo.CreateBookingPage(ctx, page))
	defer testutil.CleanupRow(t, pool, "booking_pages", page.ID)

	got, err := repo.GetBookingPageByID(ctx, page.ID, tenant)
	require.NoError(t, err)
	assert.Equal(t, "Gap GmbH", got.CompanyName)
	require.Len(t, got.Services, 1)
	assert.Equal(t, "Beratung", got.Services[0].Name)

	_, err = repo.GetBookingPageByID(ctx, uuid.New(), tenant)
	require.ErrorIs(t, err, ErrBookingPageNotFound)

	bySlug, err := repo.GetBookingPageBySlug(ctx, slug)
	require.NoError(t, err)
	assert.Equal(t, page.ID, bySlug.ID)

	page.CompanyName = "Gap GmbH Renamed"
	page.Active = false
	page.UpdatedAt = time.Now().UTC()
	require.NoError(t, repo.UpdateBookingPage(ctx, page))

	got, err = repo.GetBookingPageByID(ctx, page.ID, tenant)
	require.NoError(t, err)
	assert.Equal(t, "Gap GmbH Renamed", got.CompanyName)
	assert.False(t, got.Active)

	// An inactive page must no longer resolve via the unauth public slug lookup.
	_, err = repo.GetBookingPageBySlug(ctx, slug)
	require.ErrorIs(t, err, ErrBookingPageNotFound)

	activeOnly, err := repo.ListBookingPages(ctx, tenant, false)
	require.NoError(t, err)
	for _, p := range activeOnly {
		if p.ID == page.ID {
			t.Fatalf("inactive page must be excluded when includeInactive=false")
		}
	}

	withInactive, err := repo.ListBookingPages(ctx, tenant, true)
	require.NoError(t, err)
	found := false
	for _, p := range withInactive {
		if p.ID == page.ID {
			found = true
		}
	}
	assert.True(t, found, "includeInactive=true must still return the inactive page")

	fakeTenant := uuid.New()
	err = repo.UpdateBookingPage(ctx, &models.BookingPage{
		ID: page.ID, TenantID: fakeTenant,
		Services: page.Services, AvailabilityRules: page.AvailabilityRules,
	})
	require.ErrorIs(t, err, ErrBookingPageNotFound)

	err = repo.DeleteBookingPage(ctx, page.ID, fakeTenant)
	require.ErrorIs(t, err, ErrBookingPageNotFound)

	require.NoError(t, repo.DeleteBookingPage(ctx, page.ID, tenant))
	_, err = repo.GetBookingPageByID(ctx, page.ID, tenant)
	require.ErrorIs(t, err, ErrBookingPageNotFound)
}

// TestPublicBookings_CreateAndGetBookedSlots covers CreatePublicBooking,
// GetBookedSlotsForPage's cancelled-status exclusion, and
// UpdatePublicBookingCalendarEventID's write-back.
func TestPublicBookings_CreateAndGetBookedSlots(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Public Booking Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenant)

	calID, ownerID := seedCalendarWithOwner(t, pool, tenant, models.CalendarTypeShared)
	defer testutil.CleanupRow(t, pool, "users", ownerID)
	defer testutil.CleanupRow(t, pool, "calendars", calID)

	pageID := testutil.SeedRow(t, pool, "booking_pages", map[string]any{
		"tenant_id": tenant, "calendar_id": calID,
		"slug": "slots-" + uuid.New().String()[:8], "company_name": "Slots GmbH",
	})
	defer testutil.CleanupRow(t, pool, "booking_pages", pageID)

	repo := NewPostgresBookingRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	day := time.Now().UTC().Truncate(24 * time.Hour)
	confirmed := &models.PublicBooking{
		ID: uuid.New(), TenantID: tenant, BookingPageID: pageID, ServiceID: "svc-1",
		CustomerName: "Max Muster", CustomerEmail: "max@example.invalid",
		Date: day, TimeSlot: "09:00", Status: models.PublicBookingStatusConfirmed, CreatedAt: time.Now().UTC(),
	}
	require.NoError(t, repo.CreatePublicBooking(ctx, confirmed))
	defer testutil.CleanupRow(t, pool, "public_bookings", confirmed.ID)

	cancelled := &models.PublicBooking{
		ID: uuid.New(), TenantID: tenant, BookingPageID: pageID, ServiceID: "svc-1",
		CustomerName: "Erika Muster", CustomerEmail: "erika@example.invalid",
		Date: day, TimeSlot: "10:00", Status: models.PublicBookingStatusCancelled, CreatedAt: time.Now().UTC(),
	}
	require.NoError(t, repo.CreatePublicBooking(ctx, cancelled))
	defer testutil.CleanupRow(t, pool, "public_bookings", cancelled.ID)

	slots, err := repo.GetBookedSlotsForPage(ctx, pageID, day, day)
	require.NoError(t, err)
	require.Len(t, slots, 1, "a cancelled booking must not count as an occupied slot")
	assert.Equal(t, "09:00", slots[0].TimeSlot)

	eventID := testutil.SeedRow(t, pool, "calendar_events", map[string]any{
		"tenant_id": tenant, "calendar_id": calID, "title": "Beratung",
		"start_time": time.Now().UTC(), "end_time": time.Now().UTC().Add(30 * time.Minute),
		"created_by": ownerID,
	})
	defer testutil.CleanupRow(t, pool, "calendar_events", eventID)

	require.NoError(t, repo.UpdatePublicBookingCalendarEventID(ctx, confirmed.ID, tenant, eventID))

	var linked *uuid.UUID
	err = pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT calendar_event_id FROM public_bookings WHERE id = $1`, confirmed.ID,
	).Scan(&linked)
	require.NoError(t, err)
	require.NotNil(t, linked)
	assert.Equal(t, eventID, *linked)
}

// TestGetCalendarEventsInRange_OverlapBoundaryAndCrossTenant covers the
// second priority named in this unit's backlog scope: an event that ends
// exactly when the query window starts is free, not blocked, and a foreign
// tenant must see zero events even though the query itself carries no
// explicit tenant_id filter (RLS-only protection).
func TestGetCalendarEventsInRange_OverlapBoundaryAndCrossTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Overlap Own Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Overlap Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	calID, ownerID := seedCalendarWithOwner(t, pool, tenantOwn, models.CalendarTypeShared)
	defer testutil.CleanupRow(t, pool, "users", ownerID)
	defer testutil.CleanupRow(t, pool, "calendars", calID)

	base := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)

	// Adjacent: ends exactly when the query window starts.
	adjacentID := testutil.SeedRow(t, pool, "calendar_events", map[string]any{
		"tenant_id": tenantOwn, "calendar_id": calID, "title": "Adjacent",
		"start_time": base.Add(-time.Hour), "end_time": base, "created_by": ownerID,
	})
	defer testutil.CleanupRow(t, pool, "calendar_events", adjacentID)

	// Overlapping: starts inside the window.
	overlappingID := testutil.SeedRow(t, pool, "calendar_events", map[string]any{
		"tenant_id": tenantOwn, "calendar_id": calID, "title": "Overlapping",
		"start_time": base.Add(30 * time.Minute), "end_time": base.Add(90 * time.Minute), "created_by": ownerID,
	})
	defer testutil.CleanupRow(t, pool, "calendar_events", overlappingID)

	repo := NewPostgresBookingRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	slots, err := repo.GetCalendarEventsInRange(ctxOwn, calID, base, base.Add(time.Hour))
	require.NoError(t, err)
	require.Len(t, slots, 1, "an adjacent event (end == window start) must be excluded, only the overlapping one counts")
	assert.True(t, slots[0].StartTime.Equal(base.Add(30*time.Minute)))

	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	crossSlots, err := repo.GetCalendarEventsInRange(ctxOther, calID, base.Add(-2*time.Hour), base.Add(2*time.Hour))
	require.NoError(t, err)
	assert.Empty(t, crossSlots, "a foreign tenant must not see another tenant's calendar events")
}
