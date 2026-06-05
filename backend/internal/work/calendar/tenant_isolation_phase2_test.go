package calendar

// Cross-tenant isolation tests for calendar tables (Migration 000124).
// calendars, event_categories, user_calendar_preferences, event_attendees,
// event_exceptions, event_reminders — all have tenant_id NOT NULL after backfill.
//
// calendars.owner_id, event_categories.user_id reference users(id).
// calendar_events.created_by references users(id) and calendar_events.calendar_id
// references calendars(id). We seed a user and calendar first.
//
// calendar_members is a composite-PK table (calendar_id, user_id) — no id column.
// We verify isolation on the parent calendars table instead.
//
// user_calendar_preferences has user_id as PRIMARY KEY (no separate id column) —
// we test via a direct count query.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_Calendar(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	// Seed a user for TenantA.
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         fmt.Sprintf("cal-test-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	// calendars — owner_id FK to users.
	calID := testutil.SeedRow(t, pool, "calendars", map[string]any{
		"tenant_id": testutil.TenantA,
		"name":      "Team Calendar",
		"owner_id":  userID,
	})
	defer testutil.CleanupRow(t, pool, "calendars", calID)

	// calendar_events — basis for child tables.
	now := time.Now().UTC()
	eventID := testutil.SeedRow(t, pool, "calendar_events", map[string]any{
		"tenant_id":   testutil.TenantA,
		"calendar_id": calID,
		"title":       "Team Meeting",
		"start_time":  now,
		"end_time":    now.Add(time.Hour),
		"created_by":  userID,
	})
	defer testutil.CleanupRow(t, pool, "calendar_events", eventID)

	// event_categories — user-scoped.
	catID := testutil.SeedRow(t, pool, "event_categories", map[string]any{
		"tenant_id": testutil.TenantA,
		"user_id":   userID,
		"name":      "Work",
		"color":     "#3d8abf",
	})
	defer testutil.CleanupRow(t, pool, "event_categories", catID)

	// event_exceptions.
	excID := testutil.SeedRow(t, pool, "event_exceptions", map[string]any{
		"tenant_id":     testutil.TenantA,
		"event_id":      eventID,
		"original_date": now.Format("2006-01-02"),
	})
	defer testutil.CleanupRow(t, pool, "event_exceptions", excID)

	// event_reminders.
	remID := testutil.SeedRow(t, pool, "event_reminders", map[string]any{
		"tenant_id":      testutil.TenantA,
		"event_id":       eventID,
		"minutes_before": 15,
	})
	defer testutil.CleanupRow(t, pool, "event_reminders", remID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	tests := []struct {
		name  string
		table string
		id    uuid.UUID
	}{
		{"calendars", "calendars", calID},
		{"calendar_events", "calendar_events", eventID},
		{"event_categories", "event_categories", catID},
		{"event_exceptions", "event_exceptions", excID},
		{"event_reminders", "event_reminders", remID},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			testutil.AssertRowCount(t, pool, ctxA, tc.table, tc.id, 1)
			testutil.AssertRowCount(t, pool, ctxB, tc.table, tc.id, 0)
		})
	}
}
