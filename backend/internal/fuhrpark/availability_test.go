package fuhrpark

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestService_GetVehicleAvailabilityValidatesWindowAndVehicle(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	tenantID, otherTenant := uuid.New(), uuid.New()
	vehicleID := uuid.New()
	repo.vehicles[vehicleID] = &Vehicle{ID: vehicleID, TenantID: tenantID}
	from := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)

	for name, tc := range map[string]struct {
		tenant uuid.UUID
		to     time.Time
		want   error
	}{
		"to equal to from":          {tenantID, from, ErrInvalidInput},
		"to before from":            {tenantID, from.Add(-time.Hour), ErrInvalidInput},
		"window longer than limit":  {tenantID, from.Add(MaxAvailabilityWindow + time.Second), ErrInvalidInput},
		"vehicle of another tenant": {otherTenant, from.Add(24 * time.Hour), ErrVehicleNotFound},
		"window exactly at limit":   {tenantID, from.Add(MaxAvailabilityWindow), nil},
	} {
		t.Run(name, func(t *testing.T) {
			slots, err := svc.GetVehicleAvailability(context.Background(), tc.tenant, vehicleID, from, tc.to)
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
			if tc.want == nil && slots == nil {
				t.Error("slots = nil, want an empty list")
			}
		})
	}
}

// ListBusySlots is the only thing standing between a member and the bookings of
// colleagues, so its filter is proven against SQL: active bookings only, the
// half-open window, one vehicle, one tenant.
func TestListBusySlots_ActiveOnlyHalfOpenTenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Availability Tenant A")
	testutil.EnsureTenant(t, pool, tenantB, "Availability Tenant B")

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)
	vehicle := seedBookingVehicle(t, repo, ctxA, pool, tenantA)
	otherVehicle := seedBookingVehicle(t, repo, ctxA, pool, tenantA)
	user := seedBookingUser(t, pool, tenantA, "availability")

	day := time.Now().UTC().Truncate(24 * time.Hour).Add(48 * time.Hour)
	at := func(h int) time.Time { return day.Add(time.Duration(h) * time.Hour) }
	seed := func(vehicleID uuid.UUID, start, end int, st BookingStatus) {
		t.Helper()
		b := &VehicleBooking{
			ID: uuid.New(), TenantID: tenantA, VehicleID: vehicleID, UserID: user,
			StartsAt: at(start), EndsAt: at(end), Purpose: "Kundentermin", Status: st,
			CreatedAt: at(0), UpdatedAt: at(0),
		}
		if conflict, err := repo.CreateBookingWithLock(ctxA, b); err != nil || conflict != nil {
			t.Fatalf("seed booking %d-%d: conflict %v, err %v", start, end, conflict, err)
		}
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "vehicle_bookings", b.ID) })
	}
	seed(vehicle, 6, 8, BookingStatusBooked)        // ends exactly at the window start: outside
	seed(vehicle, 8, 10, BookingStatusInUse)        // 08-10 running into the window
	seed(vehicle, 12, 13, BookingStatusBooked)      // inside
	seed(vehicle, 14, 15, BookingStatusCancelled)   // released
	seed(vehicle, 15, 16, BookingStatusCompleted)   // released
	seed(vehicle, 17, 18, BookingStatusBooked)      // starts exactly at the window end: outside
	seed(otherVehicle, 12, 13, BookingStatusBooked) // another vehicle

	slots, err := repo.ListBusySlots(ctxA, tenantA, vehicle, at(9), at(17))
	if err != nil {
		t.Fatalf("ListBusySlots: %v", err)
	}
	want := [][2]time.Time{{at(8), at(10)}, {at(12), at(13)}}
	if len(slots) != len(want) {
		t.Fatalf("slots = %v, want %v", slots, want)
	}
	for i, w := range want {
		if !slots[i].StartsAt.Equal(w[0]) || !slots[i].EndsAt.Equal(w[1]) {
			t.Errorf("slot %d = %v-%v, want %v-%v", i, slots[i].StartsAt, slots[i].EndsAt, w[0], w[1])
		}
	}

	foreign, err := repo.ListBusySlots(ctxB, tenantB, vehicle, at(0), at(24))
	if err != nil {
		t.Fatalf("ListBusySlots from tenant B: %v", err)
	}
	if len(foreign) != 0 {
		t.Errorf("tenant B sees %d slots of tenant A's vehicle, want 0", len(foreign))
	}
}
