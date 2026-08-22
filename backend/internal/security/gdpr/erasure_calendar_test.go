package gdpr

// Integration coverage for CalendarErasureHandler.ExecuteErasure — before this,
// only the ModuleName registration was tested. Fixes fix-calendar-erasure-
// incomplete-and-doc-mismatch (Lauf 10): the handler's doc comment claimed
// "Deletes personal calendars and owned events" while the code deleted
// neither, only counted calendar_events for reporting, and never touched
// calendars, calendar_members or caldav_push_subscriptions at all (their
// CASCADE FKs on users(id) never fire because erasure anonymizes the users
// row in place instead of deleting it).

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// seedCalendarMember inserts into calendar_members, which has a composite
// primary key (calendar_id, user_id) and no id column.
func seedCalendarMember(t *testing.T, pool *pgxpool.Pool, tenantID, calendarID, userID uuid.UUID) {
	t.Helper()
	ctx := testutil.WithSystemCtx(context.Background())
	if _, err := pool.Exec(ctx,
		`INSERT INTO calendar_members (calendar_id, user_id, tenant_id) VALUES ($1, $2, $3)`,
		calendarID, userID, tenantID,
	); err != nil {
		t.Fatalf("seed calendar_members: %v", err)
	}
}

// seedEventAttendee inserts into event_attendees, which has a composite
// primary key (event_id, user_id) and no id column.
func seedEventAttendee(t *testing.T, pool *pgxpool.Pool, tenantID, eventID, userID uuid.UUID) {
	t.Helper()
	ctx := testutil.WithSystemCtx(context.Background())
	if _, err := pool.Exec(ctx,
		`INSERT INTO event_attendees (event_id, user_id, tenant_id) VALUES ($1, $2, $3)`,
		eventID, userID, tenantID,
	); err != nil {
		t.Fatalf("seed event_attendees: %v", err)
	}
}

func TestCalendarErasureHandler_ExecuteErasure_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Calendar Erasure Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	userID := seedExportUser(t, pool, tenantOwn, "cal-erasure-subject")
	defer testutil.CleanupRow(t, pool, "users", userID)
	colleagueID := seedExportUser(t, pool, tenantOwn, "cal-erasure-colleague")
	defer testutil.CleanupRow(t, pool, "users", colleagueID)

	start := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)

	// Personal calendar with no booking page attached -- must be deleted
	// outright, cascading its own event with it.
	personalCalNoBooking := testutil.SeedRow(t, pool, "calendars", map[string]any{
		"tenant_id":     tenantOwn,
		"name":          "Privatkalender",
		"calendar_type": "personal",
		"owner_id":      userID,
	})
	defer testutil.CleanupRow(t, pool, "calendars", personalCalNoBooking)
	eventOnPersonalCal := testutil.SeedRow(t, pool, "calendar_events", map[string]any{
		"tenant_id":   tenantOwn,
		"calendar_id": personalCalNoBooking,
		"title":       "Arzttermin",
		"start_time":  start,
		"end_time":    end,
		"created_by":  userID,
	})

	// Personal calendar that a booking_pages row depends on -- must survive so
	// the booking page (and the public_bookings customer records attached to
	// it) are not collaterally destroyed. Its own event survives too, but gets
	// its organizer-authored content anonymized.
	personalCalWithBooking := testutil.SeedRow(t, pool, "calendars", map[string]any{
		"tenant_id":     tenantOwn,
		"name":          "Beratungskalender",
		"calendar_type": "personal",
		"owner_id":      userID,
	})
	defer testutil.CleanupRow(t, pool, "calendars", personalCalWithBooking)
	bookingSlug := "cal-erasure-" + strings.ReplaceAll(uuid.New().String(), "-", "")[:10]
	bookingPageID := testutil.SeedRow(t, pool, "booking_pages", map[string]any{
		"tenant_id":    tenantOwn,
		"calendar_id":  personalCalWithBooking,
		"slug":         bookingSlug,
		"company_name": "Zentria Beratung",
	})
	eventOnBookingCal := testutil.SeedRow(t, pool, "calendar_events", map[string]any{
		"tenant_id":   tenantOwn,
		"calendar_id": personalCalWithBooking,
		"title":       "Erstberatung Herr Mueller",
		"description": "Vertrauliche Notizen zum Kunden",
		"location":    "Buero 3",
		"start_time":  start,
		"end_time":    end,
		"created_by":  userID,
	})

	// Shared team calendar owned by the colleague. The subject is a member and
	// also organizes one event on it; a second event is organized by the
	// colleague with both as attendees.
	sharedCal := testutil.SeedRow(t, pool, "calendars", map[string]any{
		"tenant_id":     tenantOwn,
		"name":          "Team-Kalender",
		"calendar_type": "shared",
		"owner_id":      colleagueID,
	})
	defer testutil.CleanupRow(t, pool, "calendars", sharedCal)
	seedCalendarMember(t, pool, tenantOwn, sharedCal, userID)

	// Personal event category, referenced by the event below -- must be
	// deleted, and the reference on the surviving event must go to NULL
	// (ON DELETE SET NULL), not dangle.
	categoryID := testutil.SeedRow(t, pool, "event_categories", map[string]any{
		"tenant_id": tenantOwn,
		"user_id":   userID,
		"name":      "Privat",
		"color":     "#ff00aa",
	})

	eventOnSharedCal := testutil.SeedRow(t, pool, "calendar_events", map[string]any{
		"tenant_id":   tenantOwn,
		"calendar_id": sharedCal,
		"title":       "Teamrunde",
		"category_id": categoryID,
		"start_time":  start,
		"end_time":    end,
		"created_by":  userID,
	})

	// Event organized by the colleague with BOTH users as attendees -- only
	// the subject's own attendee row must disappear; the event and the
	// colleague's attendance must survive untouched.
	colleagueEventID := testutil.SeedRow(t, pool, "calendar_events", map[string]any{
		"tenant_id":   tenantOwn,
		"calendar_id": sharedCal,
		"title":       "Standup",
		"start_time":  start,
		"end_time":    end,
		"created_by":  colleagueID,
	})
	defer testutil.CleanupRow(t, pool, "calendar_events", colleagueEventID)
	seedEventAttendee(t, pool, tenantOwn, colleagueEventID, userID)
	seedEventAttendee(t, pool, tenantOwn, colleagueEventID, colleagueID)

	pushSubID := testutil.SeedRow(t, pool, "caldav_push_subscriptions", map[string]any{
		"tenant_id":       tenantOwn,
		"user_id":         userID,
		"collection_type": "calendar",
		"collection_id":   uuid.New(),
		"push_url":        "https://caldav-client.invalid/push",
		"expires_at":      start.Add(24 * time.Hour),
	})

	ctx := testutil.WithTenantCtx(context.Background(), tenantOwn)

	h := NewCalendarErasureHandler(pool)

	preview, err := h.PreviewErasure(ctx, userID)
	require.NoError(t, err)

	affected, err := h.ExecuteErasure(ctx, userID, erasureLabel, ErasureAnonymize)
	require.NoError(t, err)

	// 1 personal calendar deleted (the one without a booking page) + 3 events
	// created by the subject (personal-cal event cascade-deleted, booking-cal
	// event anonymized, shared-cal event anonymized) + 1 attendee row (the
	// subject's own participation in the colleague's event) + 1 calendar
	// membership + 1 push subscription + 1 event category + 0 preferences
	// (none seeded) = 8.
	const wantAffected = 8
	assert.Equal(t, wantAffected, affected, "affected count must match PreviewErasure exactly")
	assert.Equal(t, wantAffected, preview.RecordCount, "PreviewErasure must report the same total ExecuteErasure later affects")

	// The personal calendar without a booking page is gone, cascading its event.
	testutil.AssertRowCount(t, pool, ctx, "calendars", personalCalNoBooking, 0)
	testutil.AssertRowCount(t, pool, ctx, "calendar_events", eventOnPersonalCal, 0)

	// The personal calendar backing a booking page survives, and so does the
	// booking page.
	testutil.AssertRowCount(t, pool, ctx, "calendars", personalCalWithBooking, 1)
	testutil.AssertRowCount(t, pool, ctx, "booking_pages", bookingPageID, 1)

	// Its event survives too, but the organizer-authored content is anonymized.
	var title string
	var description, location *string
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT title, description, location FROM calendar_events WHERE id = $1`, eventOnBookingCal,
	).Scan(&title, &description, &location))
	assert.Equal(t, "["+erasureLabel+"]", title)
	assert.Nil(t, description, "description must be cleared, not just the title")
	assert.Nil(t, location, "location must be cleared, not just the title")

	// The shared-calendar event the subject organized survives, anonymized,
	// and its category reference is now NULL (event_categories row is gone).
	var sharedTitle string
	var sharedCategory *uuid.UUID
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT title, category_id FROM calendar_events WHERE id = $1`, eventOnSharedCal,
	).Scan(&sharedTitle, &sharedCategory))
	assert.Equal(t, "["+erasureLabel+"]", sharedTitle)
	assert.Nil(t, sharedCategory, "category_id must go to NULL, not dangle, once the category is deleted")

	// The event organized by the colleague survives untouched (not the
	// subject's content) ...
	var colleagueEventTitle string
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT title FROM calendar_events WHERE id = $1`, colleagueEventID,
	).Scan(&colleagueEventTitle))
	assert.Equal(t, "Standup", colleagueEventTitle, "an event the subject did not organize must not be anonymized")

	// ... the subject's own attendee row on it is gone ...
	var subjectAttendeeRows int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM event_attendees WHERE event_id = $1 AND user_id = $2`, colleagueEventID, userID,
	).Scan(&subjectAttendeeRows))
	assert.Equal(t, 0, subjectAttendeeRows, "the subject's own attendance must be removed")

	// ... but the colleague's attendance on the same event survives.
	var colleagueAttendeeRows int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM event_attendees WHERE event_id = $1 AND user_id = $2`, colleagueEventID, colleagueID,
	).Scan(&colleagueAttendeeRows))
	assert.Equal(t, 1, colleagueAttendeeRows, "an event with other attendees must only lose the erased subject's own participation")

	// The subject's membership in the colleague's shared calendar is gone.
	var memberRows int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM calendar_members WHERE calendar_id = $1 AND user_id = $2`, sharedCal, userID,
	).Scan(&memberRows))
	assert.Equal(t, 0, memberRows, "calendar membership must be removed on erasure")

	// The push subscription and the event category are gone.
	testutil.AssertRowCount(t, pool, ctx, "caldav_push_subscriptions", pushSubID, 0)
	testutil.AssertRowCount(t, pool, ctx, "event_categories", categoryID, 0)
}

func TestCalendarErasureHandler_ExecuteErasure_DeadPool(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	pool.Close()

	h := NewCalendarErasureHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	userID := uuid.New()

	affected, err := h.ExecuteErasure(ctx, userID, erasureLabel, ErasureAnonymize)
	require.Error(t, err, "a dead pool must surface as an error, not as a silent no-op erasure")
	assert.Equal(t, 0, affected)
}
