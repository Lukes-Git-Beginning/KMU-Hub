package resource

// Cross-tenant isolation DB tests for resources and resource_bookings (Migration 000124).
// resources, resource_bookings have tenant_id NOT NULL after backfill, RLS in mig 000124.
// resource_tags has composite PK (resource_id, tag) — tested via resources isolation.
//
// resources.created_by FK → users. resource_bookings.booked_by FK → users.
// resource_bookings.event_id FK → calendar_events (requires calendar + user chain).

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_Resources(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	// Seed a user for TenantA (for created_by and booked_by FKs).
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         fmt.Sprintf("resource-test-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	// resources.
	resID := testutil.SeedRow(t, pool, "resources", map[string]any{
		"tenant_id":     testutil.TenantA,
		"name":          "Conference Room A",
		"resource_type": "room",
		"created_by":    userID,
	})
	defer testutil.CleanupRow(t, pool, "resources", resID)

	// resource_bookings requires event_id FK → calendar_events. We need a calendar first.
	calID := testutil.SeedRow(t, pool, "calendars", map[string]any{
		"tenant_id": testutil.TenantA,
		"name":      "Resource Calendar",
		"owner_id":  userID,
	})
	defer testutil.CleanupRow(t, pool, "calendars", calID)

	now := time.Now().UTC()
	eventID := testutil.SeedRow(t, pool, "calendar_events", map[string]any{
		"tenant_id":   testutil.TenantA,
		"calendar_id": calID,
		"title":       "Room Booking",
		"start_time":  now,
		"end_time":    now.Add(time.Hour),
		"created_by":  userID,
	})
	defer testutil.CleanupRow(t, pool, "calendar_events", eventID)

	// resource_bookings.
	bookingID := testutil.SeedRow(t, pool, "resource_bookings", map[string]any{
		"tenant_id":   testutil.TenantA,
		"resource_id": resID,
		"event_id":    eventID,
		"booked_by":   userID,
		"start_time":  now,
		"end_time":    now.Add(time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "resource_bookings", bookingID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	tests := []struct {
		name  string
		table string
		id    uuid.UUID
	}{
		{"resources", "resources", resID},
		{"resource_bookings", "resource_bookings", bookingID},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			testutil.AssertRowCount(t, pool, ctxA, tc.table, tc.id, 1)
			testutil.AssertRowCount(t, pool, ctxB, tc.table, tc.id, 0)
		})
	}
}
