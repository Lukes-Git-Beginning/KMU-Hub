package event

// Covers internal/work/event/postgres_repository.go against the real schema —
// previously only exercised through service_test.go's mock Repository. See
// BACKLOG.yml unit c-cov-work-event-repo (Lauf 7). Builds on
// c-cov-work-event-rrule (rrule.go's expansion correctness lives there, not
// here); this file's focus per the unit scope is the storage layer: an RRULE
// string surviving a save/read roundtrip unchanged, an EXDATE exception
// leaving the parent series untouched, and range queries across a month
// boundary.

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func seedEventUser(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("wp-event-%s@test.invalid", uuid.New()),
		"password_hash": "x",
		"first_name":    "Eva",
		"last_name":     "Event",
	})
}

func seedEventCalendar(t *testing.T, pool *pgxpool.Pool, tenantID, ownerID uuid.UUID) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "calendars", map[string]any{
		"tenant_id": tenantID,
		"name":      "Test Calendar",
		"owner_id":  ownerID,
	})
}

// seedEventFixtureTenant mints a fresh tenant with one user and one calendar —
// the minimal parent chain calendar_events.calendar_id needs.
func seedEventFixtureTenant(t *testing.T, pool *pgxpool.Pool, label string) (tenantID, userID, calendarID uuid.UUID) {
	t.Helper()
	tenantID = uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, label)
	userID = seedEventUser(t, pool, tenantID)
	calendarID = seedEventCalendar(t, pool, tenantID, userID)
	return tenantID, userID, calendarID
}

func newTestEvent(tenantID, calendarID, createdBy uuid.UUID, title string, start time.Time) *models.CalendarEvent {
	return &models.CalendarEvent{
		ID:         uuid.New(),
		TenantID:   tenantID,
		CalendarID: calendarID,
		Title:      title,
		StartTime:  start,
		EndTime:    start.Add(time.Hour),
		Timezone:   "Europe/Berlin",
		CreatedBy:  createdBy,
		CreatedAt:  start,
		UpdatedAt:  start,
	}
}

// TestEvent_CRUD_RRuleRoundtripAndNotFoundPaths proves an RRULE string
// survives a Create/GetByID roundtrip character-for-character, that Update
// can change it in place, and that Update/Delete report ErrEventNotFound
// for a row that isn't there instead of silently succeeding.
func TestEvent_CRUD_RRuleRoundtripAndNotFoundPaths(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, userOwn, calOwn := seedEventFixtureTenant(t, pool, "Event CRUD Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	defer testutil.CleanupRow(t, pool, "calendars", calOwn)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	start := time.Date(2031, 1, 30, 10, 0, 0, 0, time.UTC)
	rrule := "FREQ=WEEKLY;BYDAY=MO,WE,FR;COUNT=10"
	ev := newTestEvent(tenantOwn, calOwn, userOwn, "Weekly Sync", start)
	ev.RRule = &rrule
	recEnd := start.Add(90 * 24 * time.Hour)
	ev.RecurrenceEnd = &recEnd

	if err := repo.Create(ctxOwn, ev); err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "calendar_events", ev.ID)

	got, err := repo.GetByID(ctxOwn, ev.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.RRule == nil || *got.RRule != rrule {
		t.Fatalf("GetByID: RRULE roundtrip mismatch, want %q, got %v", rrule, got.RRule)
	}
	if got.RecurrenceEnd == nil || !got.RecurrenceEnd.Equal(recEnd) {
		t.Fatalf("GetByID: RecurrenceEnd mismatch, want %v, got %v", recEnd, got.RecurrenceEnd)
	}
	if got.Title != "Weekly Sync" {
		t.Fatalf("GetByID: unexpected row %+v", got)
	}

	// GetByID under a foreign tenant ctx must not find the row (RLS).
	tenantOther := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOther, "Event CRUD Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)
	if _, err := repo.GetByID(testutil.WithTenantCtx(context.Background(), tenantOther), ev.ID, tenantOwn); !errors.Is(err, ErrEventNotFound) {
		t.Fatalf("GetByID (foreign ctx): expected ErrEventNotFound, got %v", err)
	}

	if _, err := repo.GetByID(ctxOwn, uuid.New(), tenantOwn); !errors.Is(err, ErrEventNotFound) {
		t.Fatalf("GetByID (missing): expected ErrEventNotFound, got %v", err)
	}

	updatedRRule := "FREQ=WEEKLY;BYDAY=MO,WE,FR;UNTIL=20310401T000000Z"
	ev.RRule = &updatedRRule
	ev.Title = "Weekly Sync — Renamed"
	ev.UpdatedAt = time.Now().UTC()
	if err := repo.Update(ctxOwn, ev); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, err = repo.GetByID(ctxOwn, ev.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID (after update): %v", err)
	}
	if got.Title != "Weekly Sync — Renamed" || got.RRule == nil || *got.RRule != updatedRRule {
		t.Fatalf("Update: expected new title/RRULE to persist, got %+v rrule=%v", got, got.RRule)
	}

	ghost := newTestEvent(tenantOwn, calOwn, userOwn, "ghost", start)
	if err := repo.Update(ctxOwn, ghost); !errors.Is(err, ErrEventNotFound) {
		t.Fatalf("Update (missing): expected ErrEventNotFound, got %v", err)
	}

	if err := repo.Delete(ctxOwn, ev.ID, tenantOwn); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := repo.GetByID(ctxOwn, ev.ID, tenantOwn); !errors.Is(err, ErrEventNotFound) {
		t.Fatalf("GetByID (after delete): expected ErrEventNotFound, got %v", err)
	}
	if err := repo.Delete(ctxOwn, ev.ID, tenantOwn); !errors.Is(err, ErrEventNotFound) {
		t.Fatalf("Delete (already deleted): expected ErrEventNotFound, got %v", err)
	}
}

// TestEvent_ExceptionsIsolateSeriesFromInstance proves that adding an EXDATE
// exception for one instance of a recurring event does not touch the parent
// series (RRULE/RecurrenceEnd stay exactly what they were), that a duplicate
// exception for the same date is rejected, and that
// DeleteExceptionsAfterDate only removes exceptions on/after the cutoff.
func TestEvent_ExceptionsIsolateSeriesFromInstance(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, userOwn, calOwn := seedEventFixtureTenant(t, pool, "Event Exceptions Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	defer testutil.CleanupRow(t, pool, "calendars", calOwn)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	start := time.Date(2031, 3, 3, 9, 0, 0, 0, time.UTC)
	rrule := "FREQ=WEEKLY;BYDAY=MO"
	recEnd := start.Add(180 * 24 * time.Hour)
	ev := newTestEvent(tenantOwn, calOwn, userOwn, "Recurring Standup", start)
	ev.RRule = &rrule
	ev.RecurrenceEnd = &recEnd
	if err := repo.Create(ctxOwn, ev); err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "calendar_events", ev.ID)

	firstException := time.Date(2031, 3, 10, 0, 0, 0, 0, time.UTC)
	exc := &models.EventException{
		ID:           uuid.New(),
		EventID:      ev.ID,
		OriginalDate: firstException,
		IsCancelled:  true,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	if err := repo.CreateException(ctxOwn, exc); err != nil {
		t.Fatalf("CreateException: %v", err)
	}

	// Duplicate exception for the same (event_id, original_date) must be rejected.
	dup := &models.EventException{
		ID:           uuid.New(),
		EventID:      ev.ID,
		OriginalDate: firstException,
		IsCancelled:  true,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	if err := repo.CreateException(ctxOwn, dup); !errors.Is(err, ErrExceptionAlreadyExists) {
		t.Fatalf("CreateException (duplicate date): expected ErrExceptionAlreadyExists, got %v", err)
	}

	// The series itself must be unaffected by the exception.
	series, err := repo.GetByID(ctxOwn, ev.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID (series after exception): %v", err)
	}
	if series.RRule == nil || *series.RRule != rrule || series.RecurrenceEnd == nil || !series.RecurrenceEnd.Equal(recEnd) {
		t.Fatalf("exception creation must not modify the parent series, got rrule=%v recurrenceEnd=%v", series.RRule, series.RecurrenceEnd)
	}

	secondException := time.Date(2031, 3, 17, 0, 0, 0, 0, time.UTC)
	exc2 := &models.EventException{
		ID:           uuid.New(),
		EventID:      ev.ID,
		OriginalDate: secondException,
		IsCancelled:  true,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	if err := repo.CreateException(ctxOwn, exc2); err != nil {
		t.Fatalf("CreateException (second): %v", err)
	}

	all, err := repo.ListExceptions(ctxOwn, ev.ID, start, start.Add(365*24*time.Hour))
	if err != nil {
		t.Fatalf("ListExceptions: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("ListExceptions: expected 2 exceptions, got %d (%+v)", len(all), all)
	}

	// DeleteExceptionsAfterDate(secondException) must remove only the second
	// exception, leaving the first (earlier) one in place.
	if err := repo.DeleteExceptionsAfterDate(ctxOwn, ev.ID, secondException); err != nil {
		t.Fatalf("DeleteExceptionsAfterDate: %v", err)
	}
	remaining, err := repo.ListExceptions(ctxOwn, ev.ID, start, start.Add(365*24*time.Hour))
	if err != nil {
		t.Fatalf("ListExceptions (after delete): %v", err)
	}
	if len(remaining) != 1 || !remaining[0].OriginalDate.Equal(firstException) {
		t.Fatalf("DeleteExceptionsAfterDate: expected only the pre-cutoff exception to remain, got %+v", remaining)
	}
}

// TestEvent_ListInRange_And_ListRecurringOverlapping_MonthBoundary covers the
// two range queries with a window that spans a month boundary (Jan->Feb) and
// proves ListInRange only returns non-recurring events (rrule IS NULL) while
// ListRecurringOverlapping only returns recurring ones (rrule IS NOT NULL),
// each respecting the recurrence_end / start_time boundary conditions and RLS.
func TestEvent_ListInRange_And_ListRecurringOverlapping_MonthBoundary(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, userOwn, calOwn := seedEventFixtureTenant(t, pool, "Event Range Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	defer testutil.CleanupRow(t, pool, "calendars", calOwn)
	tenantOther, userOther, calOther := seedEventFixtureTenant(t, pool, "Event Range Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)
	defer testutil.CleanupRow(t, pool, "users", userOther)
	defer testutil.CleanupRow(t, pool, "calendars", calOther)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	windowStart := time.Date(2031, 1, 25, 0, 0, 0, 0, time.UTC)
	windowEnd := time.Date(2031, 2, 5, 0, 0, 0, 0, time.UTC)

	// Non-recurring, in-window, on either side of the month boundary.
	beforeBoundary := newTestEvent(tenantOwn, calOwn, userOwn, "In-Window Jan", time.Date(2031, 1, 30, 10, 0, 0, 0, time.UTC))
	afterBoundary := newTestEvent(tenantOwn, calOwn, userOwn, "In-Window Feb", time.Date(2031, 2, 3, 10, 0, 0, 0, time.UTC))
	// Non-recurring, outside the window entirely.
	outside := newTestEvent(tenantOwn, calOwn, userOwn, "Outside Window", time.Date(2031, 1, 1, 10, 0, 0, 0, time.UTC))
	for _, e := range []*models.CalendarEvent{beforeBoundary, afterBoundary, outside} {
		if err := repo.Create(ctxOwn, e); err != nil {
			t.Fatalf("Create (%s): %v", e.Title, err)
		}
		defer testutil.CleanupRow(t, pool, "calendar_events", e.ID)
	}

	// Recurring events — must never surface via ListInRange. recOngoing's own
	// start/end row deliberately falls INSIDE the window (not just its
	// recurrence_end horizon), so a query that dropped the "e.rrule IS NULL"
	// filter would otherwise return it — proving the filter, not just the
	// time-range predicate, is what excludes recurring events from ListInRange.
	rrule := "FREQ=WEEKLY"
	recOngoing := newTestEvent(tenantOwn, calOwn, userOwn, "Recurring Ongoing", time.Date(2031, 1, 28, 9, 0, 0, 0, time.UTC))
	recOngoing.RRule = &rrule
	recEndAfterWindow := time.Date(2031, 2, 10, 0, 0, 0, 0, time.UTC)
	recOngoing.RecurrenceEnd = &recEndAfterWindow

	recEndedBeforeWindow := newTestEvent(tenantOwn, calOwn, userOwn, "Recurring Ended Before Window", time.Date(2031, 1, 1, 9, 0, 0, 0, time.UTC))
	recEndedBeforeWindow.RRule = &rrule
	endedBefore := time.Date(2031, 1, 20, 0, 0, 0, 0, time.UTC)
	recEndedBeforeWindow.RecurrenceEnd = &endedBefore

	recStartsAfterWindow := newTestEvent(tenantOwn, calOwn, userOwn, "Recurring Starts After Window", time.Date(2031, 2, 10, 9, 0, 0, 0, time.UTC))
	recStartsAfterWindow.RRule = &rrule

	for _, e := range []*models.CalendarEvent{recOngoing, recEndedBeforeWindow, recStartsAfterWindow} {
		if err := repo.Create(ctxOwn, e); err != nil {
			t.Fatalf("Create (%s): %v", e.Title, err)
		}
		defer testutil.CleanupRow(t, pool, "calendar_events", e.ID)
	}

	// A same-shaped foreign-tenant recurring event that would otherwise match
	// — must never leak into tenantOwn's results.
	foreignRec := newTestEvent(tenantOther, calOther, userOther, "Foreign Recurring", time.Date(2031, 1, 1, 9, 0, 0, 0, time.UTC))
	foreignRec.RRule = &rrule
	foreignRec.RecurrenceEnd = &recEndAfterWindow
	if err := repo.Create(ctxOther, foreignRec); err != nil {
		t.Fatalf("Create (foreign recurring): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "calendar_events", foreignRec.ID)

	inRange, err := repo.ListInRange(ctxOwn, []uuid.UUID{calOwn}, windowStart, windowEnd, userOwn, tenantOwn)
	if err != nil {
		t.Fatalf("ListInRange: %v", err)
	}
	gotIDs := map[uuid.UUID]bool{}
	for _, e := range inRange {
		gotIDs[e.ID] = true
	}
	if !gotIDs[beforeBoundary.ID] || !gotIDs[afterBoundary.ID] {
		t.Fatalf("ListInRange: expected both cross-boundary events present, got %d rows: %+v", len(inRange), inRange)
	}
	if gotIDs[outside.ID] {
		t.Fatalf("ListInRange: out-of-window event must not be returned")
	}
	if gotIDs[recOngoing.ID] {
		t.Fatalf("ListInRange: recurring event must not be returned (rrule IS NULL filter)")
	}
	if len(inRange) != 2 {
		t.Fatalf("ListInRange: expected exactly 2 rows, got %d: %+v", len(inRange), inRange)
	}

	overlapping, err := repo.ListRecurringOverlapping(ctxOwn, []uuid.UUID{calOwn}, windowStart, windowEnd, tenantOwn)
	if err != nil {
		t.Fatalf("ListRecurringOverlapping: %v", err)
	}
	overlapIDs := map[uuid.UUID]bool{}
	for _, e := range overlapping {
		overlapIDs[e.ID] = true
	}
	if !overlapIDs[recOngoing.ID] {
		t.Fatalf("ListRecurringOverlapping: expected the ongoing recurring event to match, got %+v", overlapping)
	}
	if overlapIDs[recEndedBeforeWindow.ID] {
		t.Fatalf("ListRecurringOverlapping: recurrence_end before window start must be excluded")
	}
	if overlapIDs[recStartsAfterWindow.ID] {
		t.Fatalf("ListRecurringOverlapping: start_time after window end must be excluded")
	}
	if overlapIDs[beforeBoundary.ID] {
		t.Fatalf("ListRecurringOverlapping: non-recurring event must not be returned (rrule IS NOT NULL filter)")
	}
	if len(overlapping) != 1 {
		t.Fatalf("ListRecurringOverlapping: expected exactly 1 row, got %d: %+v", len(overlapping), overlapping)
	}

	// RLS: a foreign session asking for tenantOwn's explicit calendar/tenant
	// filter must still see nothing.
	if leaked, err := repo.ListInRange(ctxOther, []uuid.UUID{calOwn}, windowStart, windowEnd, userOwn, tenantOwn); err != nil {
		t.Fatalf("ListInRange (foreign ctx, victim filter): %v", err)
	} else if len(leaked) != 0 {
		t.Fatalf("ListInRange (foreign ctx, victim filter): expected 0 rows under RLS, got %d", len(leaked))
	}
	if leaked, err := repo.ListRecurringOverlapping(ctxOther, []uuid.UUID{calOwn}, windowStart, windowEnd, tenantOwn); err != nil {
		t.Fatalf("ListRecurringOverlapping (foreign ctx, victim filter): %v", err)
	} else if len(leaked) != 0 {
		t.Fatalf("ListRecurringOverlapping (foreign ctx, victim filter): expected 0 rows under RLS, got %d", len(leaked))
	}
}

// TestEvent_Attendees_Lifecycle covers Add/Remove/UpdateRSVP/List plus
// ListAttendeeEventIDs, including the not-an-attendee and already-an-attendee
// error paths.
func TestEvent_Attendees_Lifecycle(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, userOwn, calOwn := seedEventFixtureTenant(t, pool, "Event Attendees Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	defer testutil.CleanupRow(t, pool, "calendars", calOwn)
	attendeeUser := seedEventUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", attendeeUser)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	start := time.Date(2031, 4, 1, 9, 0, 0, 0, time.UTC)
	ev := newTestEvent(tenantOwn, calOwn, userOwn, "Attendee-Test", start)
	if err := repo.Create(ctxOwn, ev); err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "calendar_events", ev.ID)

	// UpdateRSVP on a non-existent attendee row must return ErrNotAttendee.
	if err := repo.UpdateRSVP(ctxOwn, ev.ID, attendeeUser, "accepted"); !errors.Is(err, ErrNotAttendee) {
		t.Fatalf("UpdateRSVP (not invited): expected ErrNotAttendee, got %v", err)
	}
	if err := repo.RemoveAttendee(ctxOwn, ev.ID, attendeeUser); !errors.Is(err, ErrNotAttendee) {
		t.Fatalf("RemoveAttendee (not invited): expected ErrNotAttendee, got %v", err)
	}

	now := time.Now().UTC()
	att := &models.EventAttendee{EventID: ev.ID, UserID: attendeeUser, RSVPStatus: "pending", CreatedAt: now}
	if err := repo.AddAttendee(ctxOwn, att); err != nil {
		t.Fatalf("AddAttendee: %v", err)
	}
	if err := repo.AddAttendee(ctxOwn, att); !errors.Is(err, ErrAlreadyAttendee) {
		t.Fatalf("AddAttendee (duplicate): expected ErrAlreadyAttendee, got %v", err)
	}

	if err := repo.UpdateRSVP(ctxOwn, ev.ID, attendeeUser, "accepted"); err != nil {
		t.Fatalf("UpdateRSVP: %v", err)
	}

	attendees, err := repo.ListAttendees(ctxOwn, ev.ID)
	if err != nil {
		t.Fatalf("ListAttendees: %v", err)
	}
	if len(attendees) != 1 || attendees[0].UserID != attendeeUser || attendees[0].RSVPStatus != "accepted" {
		t.Fatalf("ListAttendees: unexpected result %+v", attendees)
	}
	if attendees[0].FirstName != "Eva" || attendees[0].LastName != "Event" {
		t.Fatalf("ListAttendees: expected denormalized user name to be joined, got first=%q last=%q", attendees[0].FirstName, attendees[0].LastName)
	}

	eventIDs, err := repo.ListAttendeeEventIDs(ctxOwn, attendeeUser, start.Add(-time.Hour), start.Add(2*time.Hour))
	if err != nil {
		t.Fatalf("ListAttendeeEventIDs: %v", err)
	}
	if len(eventIDs) != 1 || eventIDs[0] != ev.ID {
		t.Fatalf("ListAttendeeEventIDs: expected exactly the event, got %+v", eventIDs)
	}
	// A window that doesn't overlap the event must return nothing.
	eventIDs, err = repo.ListAttendeeEventIDs(ctxOwn, attendeeUser, start.Add(24*time.Hour), start.Add(48*time.Hour))
	if err != nil {
		t.Fatalf("ListAttendeeEventIDs (no overlap): %v", err)
	}
	if len(eventIDs) != 0 {
		t.Fatalf("ListAttendeeEventIDs (no overlap): expected empty, got %+v", eventIDs)
	}

	if err := repo.RemoveAttendee(ctxOwn, ev.ID, attendeeUser); err != nil {
		t.Fatalf("RemoveAttendee: %v", err)
	}
	attendees, err = repo.ListAttendees(ctxOwn, ev.ID)
	if err != nil {
		t.Fatalf("ListAttendees (after remove): %v", err)
	}
	if len(attendees) != 0 {
		t.Fatalf("ListAttendees (after remove): expected empty, got %+v", attendees)
	}
}

// TestEvent_Reminders_Lifecycle covers SetReminders' delete-then-insert
// replace semantics, ListReminders' ordering, and ListUpcomingReminders'
// window + rrule IS NULL filter (a recurring event's reminder must never
// surface there — recurring instances get reminders computed from expansion,
// not this direct join).
func TestEvent_Reminders_Lifecycle(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, userOwn, calOwn := seedEventFixtureTenant(t, pool, "Event Reminders Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	defer testutil.CleanupRow(t, pool, "calendars", calOwn)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	sysCtx := testutil.WithSystemCtx(context.Background())

	start := time.Date(2031, 5, 1, 12, 0, 0, 0, time.UTC)
	ev := newTestEvent(tenantOwn, calOwn, userOwn, "Reminder-Test", start)
	if err := repo.Create(ctxOwn, ev); err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "calendar_events", ev.ID)

	if err := repo.SetReminders(ctxOwn, ev.ID, []int{30, 10, 60}); err != nil {
		t.Fatalf("SetReminders: %v", err)
	}
	reminders, err := repo.ListReminders(ctxOwn, ev.ID)
	if err != nil {
		t.Fatalf("ListReminders: %v", err)
	}
	if len(reminders) != 3 || reminders[0].MinutesBefore != 10 || reminders[1].MinutesBefore != 30 || reminders[2].MinutesBefore != 60 {
		t.Fatalf("ListReminders: expected [10,30,60] ascending, got %+v", reminders)
	}

	// SetReminders again must replace, not append.
	if err := repo.SetReminders(ctxOwn, ev.ID, []int{15}); err != nil {
		t.Fatalf("SetReminders (replace): %v", err)
	}
	reminders, err = repo.ListReminders(ctxOwn, ev.ID)
	if err != nil {
		t.Fatalf("ListReminders (after replace): %v", err)
	}
	if len(reminders) != 1 || reminders[0].MinutesBefore != 15 {
		t.Fatalf("SetReminders: expected replace semantics, got %+v", reminders)
	}

	windowStart := start.Add(-30 * time.Minute)
	windowEnd := start.Add(30 * time.Minute)
	upcoming, err := repo.ListUpcomingReminders(sysCtx, windowStart, windowEnd)
	if err != nil {
		t.Fatalf("ListUpcomingReminders: %v", err)
	}
	found := false
	for _, rwe := range upcoming {
		if rwe.Event.ID == ev.ID {
			found = true
			if rwe.Reminder.MinutesBefore != 15 {
				t.Fatalf("ListUpcomingReminders: unexpected reminder %+v", rwe.Reminder)
			}
		}
	}
	if !found {
		t.Fatalf("ListUpcomingReminders: expected to find the non-recurring event's reminder in window, got %+v", upcoming)
	}

	// A window that doesn't reach start_time - minutes_before must not include it.
	farWindow, err := repo.ListUpcomingReminders(sysCtx, start.Add(24*time.Hour), start.Add(25*time.Hour))
	if err != nil {
		t.Fatalf("ListUpcomingReminders (far window): %v", err)
	}
	for _, rwe := range farWindow {
		if rwe.Event.ID == ev.ID {
			t.Fatalf("ListUpcomingReminders (far window): must not include a reminder outside the window")
		}
	}

	// A recurring event's reminder must never surface — even with a window
	// that matches its own start_time exactly (rrule IS NULL filter).
	rrule := "FREQ=DAILY"
	recStart := time.Date(2031, 5, 2, 12, 0, 0, 0, time.UTC)
	recEv := newTestEvent(tenantOwn, calOwn, userOwn, "Recurring Reminder-Test", recStart)
	recEv.RRule = &rrule
	if err := repo.Create(ctxOwn, recEv); err != nil {
		t.Fatalf("Create (recurring): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "calendar_events", recEv.ID)
	if err := repo.SetReminders(ctxOwn, recEv.ID, []int{15}); err != nil {
		t.Fatalf("SetReminders (recurring): %v", err)
	}
	recWindow, err := repo.ListUpcomingReminders(sysCtx, recStart.Add(-30*time.Minute), recStart.Add(30*time.Minute))
	if err != nil {
		t.Fatalf("ListUpcomingReminders (recurring window): %v", err)
	}
	for _, rwe := range recWindow {
		if rwe.Event.ID == recEv.ID {
			t.Fatalf("ListUpcomingReminders: recurring event's reminder must not be returned (rrule IS NULL filter)")
		}
	}
}
