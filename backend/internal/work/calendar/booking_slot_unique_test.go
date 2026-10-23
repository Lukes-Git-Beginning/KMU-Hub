package calendar

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// TestPublicBookings_OneActiveBookingPerSlot verifies migration 000330: the partial
// unique index public_bookings_unique_active_slot lets only one active booking hold a
// slot of a booking page, and the repository reports the violation as
// ErrBookingSlotUnavailable. The service runs without a calendar client, which is the
// failure mode the index closes: no calendar event exists for the second request to
// collide with.
func TestPublicBookings_OneActiveBookingPerSlot(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	userA := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         fmt.Sprintf("slot-test-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userA)
	calA := testutil.SeedRow(t, pool, "calendars", map[string]any{
		"tenant_id": testutil.TenantA, "name": "Cal A", "owner_id": userA,
	})
	defer testutil.CleanupRow(t, pool, "calendars", calA)

	slug := "slot-" + uuid.New().String()[:8]
	newPage := func(t *testing.T) uuid.UUID {
		id := testutil.SeedRow(t, pool, "booking_pages", map[string]any{
			"tenant_id": testutil.TenantA, "calendar_id": calA,
			"slug": "slot-" + uuid.New().String()[:8], "company_name": "A GmbH",
			"services": `[{"id":"svc","name":"Beratung","duration_min":30}]`,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "booking_pages", id) })
		return id
	}
	page := testutil.SeedRow(t, pool, "booking_pages", map[string]any{
		"tenant_id": testutil.TenantA, "calendar_id": calA, "slug": slug, "company_name": "A GmbH",
		"services": `[{"id":"svc","name":"Beratung","duration_min":30}]`,
	})
	defer testutil.CleanupRow(t, pool, "booking_pages", page)

	// The public route runs under system context (calendar_grpc.go), the RLS policy admits it.
	ctx := testutil.WithSystemCtx(context.Background())
	repo := NewPostgresBookingRepository(pool)
	svc := NewBookingService(repo)
	date := time.Now().UTC().AddDate(0, 0, 30).Truncate(24 * time.Hour)

	book := func(slot, email string) error {
		_, err := svc.CreatePublicBooking(ctx, nil, nil, CreatePublicBookingInput{
			Slug: slug, ServiceID: "svc", Date: date, TimeSlot: slot,
			CustomerName: "Kunde", CustomerEmail: email,
		})
		return err
	}

	t.Run("second customer on the same slot is rejected", func(t *testing.T) {
		require.NoError(t, book("09:00", "erster@kunde.test"))
		err := book("09:00", "zweiter@kunde.test")
		assert.ErrorIs(t, err, ErrBookingSlotUnavailable)
	})

	t.Run("same customer submitting twice is rejected", func(t *testing.T) {
		require.NoError(t, book("10:00", "doppelt@kunde.test"))
		assert.ErrorIs(t, book("10:00", "doppelt@kunde.test"), ErrBookingSlotUnavailable)
	})

	t.Run("other slot and other page stay bookable", func(t *testing.T) {
		require.NoError(t, book("09:30", "zweiter@kunde.test"))
		other := newPage(t)
		require.NoError(t, repo.CreatePublicBooking(ctx, &models.PublicBooking{
			ID: uuid.New(), TenantID: testutil.TenantA, BookingPageID: other, ServiceID: "svc",
			CustomerName: "Kunde", CustomerEmail: "zweiter@kunde.test", Date: date, TimeSlot: "09:00",
			Status: models.PublicBookingStatusConfirmed, CreatedAt: time.Now(),
		}))
	})

	t.Run("cancelled booking frees the slot", func(t *testing.T) {
		require.NoError(t, book("11:00", "storno@kunde.test"))
		_, err := pool.Exec(ctx,
			`UPDATE public_bookings SET status = 'cancelled'
			  WHERE booking_page_id = $1 AND date = $2 AND time_slot = '11:00'`, page, date)
		require.NoError(t, err)
		require.NoError(t, book("11:00", "nachruecker@kunde.test"))
	})

	t.Run("concurrent customers on one slot leave exactly one winner", func(t *testing.T) {
		const racers = 8
		var wg sync.WaitGroup
		errs := make([]error, racers)
		for i := range racers {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				errs[i] = book("14:00", fmt.Sprintf("racer-%d@kunde.test", i))
			}(i)
		}
		wg.Wait()

		won := 0
		for _, err := range errs {
			switch {
			case err == nil:
				won++
			case errors.Is(err, ErrBookingSlotUnavailable):
			default:
				t.Fatalf("unexpected error: %v", err)
			}
		}
		assert.Equal(t, 1, won)

		var active int
		require.NoError(t, pool.QueryRow(ctx,
			`SELECT count(*) FROM public_bookings
			  WHERE booking_page_id = $1 AND date = $2 AND time_slot = '14:00'
			    AND status IN ('pending','confirmed')`, page, date).Scan(&active))
		assert.Equal(t, 1, active)
	})
}
