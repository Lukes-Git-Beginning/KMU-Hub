package resource

// DB tests for the PostgresRepository methods that tenant_write_test.go and
// tenant_isolation_phase2_test.go do not exercise: List, SetTags,
// CreateBooking/CancelBooking, ListBookings/ListBookingsByEvent/GetBooking,
// FindAvailableResources and FindAlternatives. service_test.go covers the
// business rules against an in-memory fake; these tests exercise the real
// SQL (filter combinations, the EXCLUDE GIST conflict path, tenant scoping
// on the read side).

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// resourceFixture seeds a tenant, a user and a calendar (needed as the FK
// parent for calendar_events, which resource_bookings.event_id requires).
type resourceFixture struct {
	tenantID uuid.UUID
	userID   uuid.UUID
	calID    uuid.UUID
}

func seedResourceFixture(t *testing.T, pool *pgxpool.Pool, label string) resourceFixture {
	t.Helper()
	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Resource DB Test "+label)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("resource-db-%s@test.invalid", uuid.New()),
		"password_hash": "x",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })

	calID := testutil.SeedRow(t, pool, "calendars", map[string]any{
		"tenant_id": tenantID,
		"name":      "Resource DB Test Calendar",
		"owner_id":  userID,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "calendars", calID) })

	return resourceFixture{tenantID: tenantID, userID: userID, calID: calID}
}

// seedResourceEvent creates a calendar_events row to satisfy resource_bookings.event_id.
func seedResourceEvent(t *testing.T, pool *pgxpool.Pool, f resourceFixture, start, end time.Time) uuid.UUID {
	t.Helper()
	eventID := testutil.SeedRow(t, pool, "calendar_events", map[string]any{
		"tenant_id":   f.tenantID,
		"calendar_id": f.calID,
		"title":       "Resource DB Test Event",
		"start_time":  start,
		"end_time":    end,
		"created_by":  f.userID,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "calendar_events", eventID) })
	return eventID
}

func mustCreateResource(t *testing.T, ctx context.Context, repo *PostgresRepository, pool *pgxpool.Pool, f resourceFixture, name, resourceType string, capacity *int, floor *string) *models.Resource {
	t.Helper()
	now := time.Now().UTC()
	res := &models.Resource{
		ID:           uuid.New(),
		TenantID:     f.tenantID,
		Name:         name,
		ResourceType: resourceType,
		Capacity:     capacity,
		Floor:        floor,
		IsActive:     true,
		CreatedBy:    f.userID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := repo.Create(ctx, res); err != nil {
		t.Fatalf("Create resource %s: %v", name, err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "resources", res.ID) })
	return res
}

func intPtr(i int) *int       { return &i }
func strPtr(s string) *string { return &s }

// ============================================================================
// List
// ============================================================================

func TestPostgresList_FiltersAndTenantScope(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fOwn := seedResourceFixture(t, pool, "List-Own")
	fOther := seedResourceFixture(t, pool, "List-Other")
	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), fOwn.tenantID)
	ctxOther := testutil.WithTenantCtx(context.Background(), fOther.tenantID)

	room := mustCreateResource(t, ctxOwn, repo, pool, fOwn, "Room-"+uuid.New().String()[:8], models.ResourceTypeRoom, intPtr(8), strPtr("2"))
	_ = mustCreateResource(t, ctxOwn, repo, pool, fOwn, "Equipment-"+uuid.New().String()[:8], models.ResourceTypeEquipment, intPtr(1), strPtr("1"))
	_ = mustCreateResource(t, ctxOther, repo, pool, fOther, "OtherTenantRoom-"+uuid.New().String()[:8], models.ResourceTypeRoom, intPtr(8), strPtr("2"))

	active := true

	t.Run("tenant scoping excludes other tenant's rows", func(t *testing.T) {
		list, err := repo.List(ctxOwn, ResourceFilters{TenantID: fOwn.tenantID, IsActive: &active})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(list) != 2 {
			t.Fatalf("expected 2 resources for own tenant, got %d", len(list))
		}
		for _, r := range list {
			if r.TenantID != fOwn.tenantID {
				t.Fatalf("List leaked a row from tenant %s into tenant %s's result", r.TenantID, fOwn.tenantID)
			}
		}
	})

	t.Run("type filter", func(t *testing.T) {
		roomType := models.ResourceTypeRoom
		list, err := repo.List(ctxOwn, ResourceFilters{TenantID: fOwn.tenantID, Type: &roomType, IsActive: &active})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(list) != 1 || list[0].ID != room.ID {
			t.Fatalf("expected exactly the room resource, got %d results", len(list))
		}
	})

	t.Run("min_capacity filter excludes the equipment resource", func(t *testing.T) {
		minCap := 5
		list, err := repo.List(ctxOwn, ResourceFilters{TenantID: fOwn.tenantID, MinCapacity: &minCap, IsActive: &active})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(list) != 1 || list[0].ID != room.ID {
			t.Fatalf("expected exactly the room resource (capacity 8 >= 5), got %d results", len(list))
		}
	})

	t.Run("floor filter", func(t *testing.T) {
		floor := "2"
		list, err := repo.List(ctxOwn, ResourceFilters{TenantID: fOwn.tenantID, Floor: &floor, IsActive: &active})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(list) != 1 || list[0].ID != room.ID {
			t.Fatalf("expected exactly the floor-2 resource, got %d results", len(list))
		}
	})
}

func TestPostgresList_TagsFilter(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	f := seedResourceFixture(t, pool, "List-Tags")
	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), f.tenantID)

	tagged := mustCreateResource(t, ctx, repo, pool, f, "Tagged-"+uuid.New().String()[:8], models.ResourceTypeRoom, nil, nil)
	if err := repo.SetTags(ctx, tagged.ID, []string{"projector", "whiteboard"}); err != nil {
		t.Fatalf("SetTags: %v", err)
	}
	_ = mustCreateResource(t, ctx, repo, pool, f, "Untagged-"+uuid.New().String()[:8], models.ResourceTypeRoom, nil, nil)

	active := true
	list, err := repo.List(ctx, ResourceFilters{TenantID: f.tenantID, Tags: []string{"projector"}, IsActive: &active})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 || list[0].ID != tagged.ID {
		t.Fatalf("expected exactly the tagged resource, got %d results", len(list))
	}
}

// ============================================================================
// SetTags
// ============================================================================

func TestPostgresSetTags_ReplacesExistingTags(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	f := seedResourceFixture(t, pool, "SetTags")
	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), f.tenantID)

	res := mustCreateResource(t, ctx, repo, pool, f, "Tagged-"+uuid.New().String()[:8], models.ResourceTypeRoom, nil, nil)

	if err := repo.SetTags(ctx, res.ID, []string{"a", "b"}); err != nil {
		t.Fatalf("SetTags (first): %v", err)
	}
	got, err := repo.GetByID(ctx, res.ID, f.tenantID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if len(got.Tags) != 2 {
		t.Fatalf("expected 2 tags after first SetTags, got %v", got.Tags)
	}

	if err := repo.SetTags(ctx, res.ID, []string{"c"}); err != nil {
		t.Fatalf("SetTags (replace): %v", err)
	}
	got, err = repo.GetByID(ctx, res.ID, f.tenantID)
	if err != nil {
		t.Fatalf("GetByID after replace: %v", err)
	}
	if len(got.Tags) != 1 || got.Tags[0] != "c" {
		t.Fatalf("expected tags to be replaced with just [c], got %v", got.Tags)
	}

	if err := repo.SetTags(ctx, res.ID, nil); err != nil {
		t.Fatalf("SetTags (clear): %v", err)
	}
	got, err = repo.GetByID(ctx, res.ID, f.tenantID)
	if err != nil {
		t.Fatalf("GetByID after clear: %v", err)
	}
	if len(got.Tags) != 0 {
		t.Fatalf("expected no tags after clearing, got %v", got.Tags)
	}
}

// ============================================================================
// CreateBooking / CancelBooking / GetBooking / ListBookings / ListBookingsByEvent
// ============================================================================

func TestPostgresCreateBooking_SuccessAndConflict(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	f := seedResourceFixture(t, pool, "Booking-Conflict")
	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), f.tenantID)

	res := mustCreateResource(t, ctx, repo, pool, f, "Bookable-"+uuid.New().String()[:8], models.ResourceTypeRoom, nil, nil)

	start := time.Now().UTC().Add(24 * time.Hour)
	end := start.Add(time.Hour)
	event1 := seedResourceEvent(t, pool, f, start, end)

	booking := &models.ResourceBooking{
		ID:         uuid.New(),
		TenantID:   f.tenantID,
		ResourceID: res.ID,
		EventID:    event1,
		BookedBy:   f.userID,
		StartTime:  start,
		EndTime:    end,
		CreatedAt:  time.Now().UTC(),
	}
	if err := repo.CreateBooking(ctx, booking); err != nil {
		t.Fatalf("CreateBooking: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "resource_bookings", booking.ID) })

	got, err := repo.GetBooking(ctx, booking.ID, f.tenantID)
	if err != nil {
		t.Fatalf("GetBooking: %v", err)
	}
	if got.ResourceID != res.ID {
		t.Fatalf("GetBooking returned resource_id %s, want %s", got.ResourceID, res.ID)
	}

	// Overlapping booking on the same resource must hit the EXCLUDE USING GIST
	// constraint and come back as ErrBookingConflict, not a raw pg error and
	// not a silent double-write -- this is the DB-level guarantee that makes
	// resource.Service.Book race-safe without an application-level
	// check-then-write (see route_calendar_resources_reminders_test.go header).
	overlapStart := start.Add(30 * time.Minute)
	overlapEnd := overlapStart.Add(time.Hour)
	event2 := seedResourceEvent(t, pool, f, overlapStart, overlapEnd)
	conflicting := &models.ResourceBooking{
		ID:         uuid.New(),
		TenantID:   f.tenantID,
		ResourceID: res.ID,
		EventID:    event2,
		BookedBy:   f.userID,
		StartTime:  overlapStart,
		EndTime:    overlapEnd,
		CreatedAt:  time.Now().UTC(),
	}
	err = repo.CreateBooking(ctx, conflicting)
	if err == nil {
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "resource_bookings", conflicting.ID) })
		t.Fatalf("CreateBooking: expected ErrBookingConflict for overlapping range, got nil")
	}
	if err != ErrBookingConflict {
		t.Fatalf("CreateBooking: expected ErrBookingConflict, got %v", err)
	}
}

func TestPostgresCancelBooking(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fOwn := seedResourceFixture(t, pool, "Cancel-Own")
	fOther := seedResourceFixture(t, pool, "Cancel-Other")
	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), fOwn.tenantID)

	res := mustCreateResource(t, ctx, repo, pool, fOwn, "Cancellable-"+uuid.New().String()[:8], models.ResourceTypeRoom, nil, nil)
	start := time.Now().UTC().Add(48 * time.Hour)
	end := start.Add(time.Hour)
	event := seedResourceEvent(t, pool, fOwn, start, end)
	booking := &models.ResourceBooking{
		ID: uuid.New(), TenantID: fOwn.tenantID, ResourceID: res.ID, EventID: event,
		BookedBy: fOwn.userID, StartTime: start, EndTime: end, CreatedAt: time.Now().UTC(),
	}
	if err := repo.CreateBooking(ctx, booking); err != nil {
		t.Fatalf("CreateBooking: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "resource_bookings", booking.ID) })

	// A cancel carrying the wrong tenant_id must not affect the row -- the
	// query's own tenant_id predicate rejects it before RLS even gets a say.
	if err := repo.CancelBooking(ctx, booking.ID, fOther.tenantID); err != ErrBookingNotFound {
		t.Fatalf("CancelBooking (foreign tenant): expected ErrBookingNotFound, got %v", err)
	}
	stillActive, err := repo.GetBooking(ctx, booking.ID, fOwn.tenantID)
	if err != nil {
		t.Fatalf("GetBooking after foreign cancel attempt: %v", err)
	}
	if stillActive.CancelledAt != nil {
		t.Fatalf("a foreign-tenant cancel reached the booking: cancelled_at=%v", stillActive.CancelledAt)
	}

	if err := repo.CancelBooking(ctx, booking.ID, fOwn.tenantID); err != nil {
		t.Fatalf("CancelBooking (own tenant): %v", err)
	}
	cancelled, err := repo.GetBooking(ctx, booking.ID, fOwn.tenantID)
	if err != nil {
		t.Fatalf("GetBooking after cancel: %v", err)
	}
	if cancelled.CancelledAt == nil {
		t.Fatalf("expected cancelled_at to be set after CancelBooking")
	}

	// A second cancel on an already-cancelled booking affects zero rows.
	if err := repo.CancelBooking(ctx, booking.ID, fOwn.tenantID); err != ErrBookingNotFound {
		t.Fatalf("CancelBooking (already cancelled): expected ErrBookingNotFound, got %v", err)
	}

	// ListBookings excludes cancelled bookings from the overlap window.
	inRange, err := repo.ListBookings(ctx, res.ID, start.Add(-time.Hour), end.Add(time.Hour))
	if err != nil {
		t.Fatalf("ListBookings: %v", err)
	}
	if len(inRange) != 0 {
		t.Fatalf("expected 0 active bookings after cancel, got %d", len(inRange))
	}
}

func TestPostgresListBookings_OverlapWindow(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	f := seedResourceFixture(t, pool, "ListBookings-Overlap")
	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), f.tenantID)

	res := mustCreateResource(t, ctx, repo, pool, f, "Overlap-"+uuid.New().String()[:8], models.ResourceTypeRoom, nil, nil)

	base := time.Now().UTC().Add(72 * time.Hour)
	// Booking well before the query window.
	before := base.Add(-4 * time.Hour)
	eventBefore := seedResourceEvent(t, pool, f, before, before.Add(time.Hour))
	bookingBefore := &models.ResourceBooking{ID: uuid.New(), TenantID: f.tenantID, ResourceID: res.ID, EventID: eventBefore, BookedBy: f.userID, StartTime: before, EndTime: before.Add(time.Hour), CreatedAt: time.Now().UTC()}
	if err := repo.CreateBooking(ctx, bookingBefore); err != nil {
		t.Fatalf("CreateBooking (before): %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "resource_bookings", bookingBefore.ID) })

	// Booking inside the query window.
	eventInside := seedResourceEvent(t, pool, f, base, base.Add(time.Hour))
	bookingInside := &models.ResourceBooking{ID: uuid.New(), TenantID: f.tenantID, ResourceID: res.ID, EventID: eventInside, BookedBy: f.userID, StartTime: base, EndTime: base.Add(time.Hour), CreatedAt: time.Now().UTC()}
	if err := repo.CreateBooking(ctx, bookingInside); err != nil {
		t.Fatalf("CreateBooking (inside): %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "resource_bookings", bookingInside.ID) })

	list, err := repo.ListBookings(ctx, res.ID, base.Add(-time.Minute), base.Add(2*time.Hour))
	if err != nil {
		t.Fatalf("ListBookings: %v", err)
	}
	if len(list) != 1 || list[0].ID != bookingInside.ID {
		t.Fatalf("expected exactly the in-window booking, got %d results", len(list))
	}

	byEvent, err := repo.ListBookingsByEvent(ctx, eventInside)
	if err != nil {
		t.Fatalf("ListBookingsByEvent: %v", err)
	}
	if len(byEvent) != 1 || byEvent[0].ID != bookingInside.ID {
		t.Fatalf("expected exactly one booking for eventInside, got %d results", len(byEvent))
	}
}

// ============================================================================
// FindAvailableResources / FindAlternatives
// ============================================================================

func TestPostgresFindAvailableResources_ExcludesBookedAndInactive(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	f := seedResourceFixture(t, pool, "FindAvailable")
	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), f.tenantID)

	free := mustCreateResource(t, ctx, repo, pool, f, "Free-"+uuid.New().String()[:8], models.ResourceTypeRoom, nil, nil)
	booked := mustCreateResource(t, ctx, repo, pool, f, "Booked-"+uuid.New().String()[:8], models.ResourceTypeRoom, nil, nil)
	inactive := mustCreateResource(t, ctx, repo, pool, f, "Inactive-"+uuid.New().String()[:8], models.ResourceTypeRoom, nil, nil)
	if err := repo.Delete(ctx, inactive.ID, f.tenantID); err != nil {
		t.Fatalf("Delete (deactivate): %v", err)
	}

	start := time.Now().UTC().Add(96 * time.Hour)
	end := start.Add(time.Hour)
	event := seedResourceEvent(t, pool, f, start, end)
	booking := &models.ResourceBooking{ID: uuid.New(), TenantID: f.tenantID, ResourceID: booked.ID, EventID: event, BookedBy: f.userID, StartTime: start, EndTime: end, CreatedAt: time.Now().UTC()}
	if err := repo.CreateBooking(ctx, booking); err != nil {
		t.Fatalf("CreateBooking: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "resource_bookings", booking.ID) })

	available, err := repo.FindAvailableResources(ctx, start, end, ResourceFilters{TenantID: f.tenantID, Type: strPtr(models.ResourceTypeRoom)})
	if err != nil {
		t.Fatalf("FindAvailableResources: %v", err)
	}
	ids := make(map[uuid.UUID]bool, len(available))
	for _, r := range available {
		ids[r.ID] = true
	}
	if !ids[free.ID] {
		t.Fatalf("expected the free resource to be available, results: %+v", available)
	}
	if ids[booked.ID] {
		t.Fatalf("booked resource must not appear as available")
	}
	if ids[inactive.ID] {
		t.Fatalf("inactive resource must not appear as available")
	}

	alternatives, err := repo.FindAlternatives(ctx, booked.ID, start, end, models.ResourceTypeRoom, f.tenantID)
	if err != nil {
		t.Fatalf("FindAlternatives: %v", err)
	}
	altIDs := make(map[uuid.UUID]bool, len(alternatives))
	for _, a := range alternatives {
		altIDs[a.ResourceID] = true
	}
	if !altIDs[free.ID] {
		t.Fatalf("expected the free resource among alternatives, got %+v", alternatives)
	}
	if altIDs[booked.ID] {
		t.Fatalf("FindAlternatives must exclude the excludeID resource itself")
	}
}
