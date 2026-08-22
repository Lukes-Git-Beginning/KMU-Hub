package gdpr

// Coverage for CalendarEventRetentionHandler (Lauf 10, seventh handler on the
// retention engine from A10). The cases that matter: a one-off event is due
// once its own end_time passes the cutoff, an open-ended recurring series
// (recurrence_end NULL) is never due no matter how old its first occurrence,
// an ended recurring series (recurrence_end past cutoff) is due, deleting an
// event cascades attendees/exceptions/reminders, anonymizing one clears the
// event's free text plus any exception override back to NULL, the anonymize
// path is idempotent via updated_at, and a second tenant's old event never
// appears in the first tenant's plan.

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

func TestCalendarEventRetentionHandler_SupportsAction(t *testing.T) {
	t.Parallel()

	handler := NewCalendarEventRetentionHandler(nil)
	assert.True(t, handler.SupportsAction(models.RetentionActionDelete))
	assert.True(t, handler.SupportsAction(models.RetentionActionAnonymize))
	assert.False(t, handler.SupportsAction("retain"))
	assert.Equal(t, "calendar_events", handler.ResourceType())
	assert.Equal(t, "calendar_events", handler.Table())
}

func TestCalendarEventRetentionHandler_PlanOnlyMatchesPastCutoffAndIsTenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Calendar Retention Plan Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	userID := seedExportUser(t, pool, tenantID, "calendar-retention-plan-user")
	defer testutil.CleanupRow(t, pool, "users", userID)
	calendarID := seedCalendar(t, pool, tenantID, userID)
	defer testutil.CleanupRow(t, pool, "calendars", calendarID)

	otherTenantID := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenantID, "Calendar Retention Plan Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", otherTenantID)
	otherUserID := seedExportUser(t, pool, otherTenantID, "calendar-retention-plan-other-user")
	defer testutil.CleanupRow(t, pool, "users", otherUserID)
	otherCalendarID := seedCalendar(t, pool, otherTenantID, otherUserID)
	defer testutil.CleanupRow(t, pool, "calendars", otherCalendarID)

	old := time.Now().UTC().AddDate(0, 0, -400)
	fresh := time.Now().UTC().AddDate(0, 0, -2)

	oldEvent := seedCalendarEvent(t, pool, tenantID, calendarID, old, old.Add(time.Hour), nil, nil)
	defer testutil.CleanupRow(t, pool, "calendar_events", oldEvent)

	freshEvent := seedCalendarEvent(t, pool, tenantID, calendarID, fresh, fresh.Add(time.Hour), nil, nil)
	defer testutil.CleanupRow(t, pool, "calendar_events", freshEvent)

	otherOldEvent := seedCalendarEvent(t, pool, otherTenantID, otherCalendarID, old, old.Add(time.Hour), nil, nil)
	defer testutil.CleanupRow(t, pool, "calendar_events", otherOldEvent)

	handler := NewCalendarEventRetentionHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	cutoff := time.Now().UTC().AddDate(0, 0, -90)

	plan, err := handler.Plan(ctx, tenantID, cutoff, models.RetentionActionDelete)
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{oldEvent}, plan.Due)
}

func TestCalendarEventRetentionHandler_PlanExcludesOpenEndedRecurringSeries(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Calendar Retention Open Series Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	userID := seedExportUser(t, pool, tenantID, "calendar-retention-open-series-user")
	defer testutil.CleanupRow(t, pool, "users", userID)
	calendarID := seedCalendar(t, pool, tenantID, userID)
	defer testutil.CleanupRow(t, pool, "calendars", calendarID)

	old := time.Now().UTC().AddDate(0, 0, -400)
	rrule := "FREQ=DAILY"
	// First occurrence is very old, but the series has no end -- it is still
	// happening today and must never be treated as due.
	openSeries := seedCalendarEvent(t, pool, tenantID, calendarID, old, old.Add(time.Hour), &rrule, nil)
	defer testutil.CleanupRow(t, pool, "calendar_events", openSeries)

	handler := NewCalendarEventRetentionHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	cutoff := time.Now().UTC().AddDate(0, 0, -90)

	plan, err := handler.Plan(ctx, tenantID, cutoff, models.RetentionActionDelete)
	require.NoError(t, err)
	assert.NotContains(t, plan.Due, openSeries, "an open-ended recurring series must never be due, regardless of its first occurrence's age")
}

func TestCalendarEventRetentionHandler_PlanRecurringSeriesUsesRecurrenceEnd(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Calendar Retention Recurrence End Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	userID := seedExportUser(t, pool, tenantID, "calendar-retention-recend-user")
	defer testutil.CleanupRow(t, pool, "users", userID)
	calendarID := seedCalendar(t, pool, tenantID, userID)
	defer testutil.CleanupRow(t, pool, "calendars", calendarID)

	old := time.Now().UTC().AddDate(0, 0, -400)
	rrule := "FREQ=WEEKLY"

	endedRecurrence := old.AddDate(0, 0, 100) // ended long before the cutoff below
	endedSeries := seedCalendarEvent(t, pool, tenantID, calendarID, old, old.Add(time.Hour), &rrule, &endedRecurrence)
	defer testutil.CleanupRow(t, pool, "calendar_events", endedSeries)

	futureRecurrence := time.Now().UTC().AddDate(1, 0, 0) // still recurring into the future
	activeSeries := seedCalendarEvent(t, pool, tenantID, calendarID, old, old.Add(time.Hour), &rrule, &futureRecurrence)
	defer testutil.CleanupRow(t, pool, "calendar_events", activeSeries)

	handler := NewCalendarEventRetentionHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	cutoff := time.Now().UTC().AddDate(0, 0, -90)

	plan, err := handler.Plan(ctx, tenantID, cutoff, models.RetentionActionDelete)
	require.NoError(t, err)
	assert.Contains(t, plan.Due, endedSeries, "a series whose recurrence_end is past the cutoff has genuinely ended")
	assert.NotContains(t, plan.Due, activeSeries, "a series recurring into the future must not be due")
}

func TestCalendarEventRetentionHandler_ApplyDeleteCascadesAttendeesExceptionsReminders(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Calendar Retention Apply Delete Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	userID := seedExportUser(t, pool, tenantID, "calendar-retention-apply-del-user")
	defer testutil.CleanupRow(t, pool, "users", userID)
	calendarID := seedCalendar(t, pool, tenantID, userID)
	defer testutil.CleanupRow(t, pool, "calendars", calendarID)

	old := time.Now().UTC().AddDate(0, 0, -400)
	eventID := seedCalendarEvent(t, pool, tenantID, calendarID, old, old.Add(time.Hour), nil, nil)

	ctx := testutil.WithSystemCtx(context.Background())
	_, err := pool.Exec(ctx,
		`INSERT INTO event_attendees (event_id, user_id, tenant_id, rsvp_status) VALUES ($1, $2, $3, 'accepted')`,
		eventID, userID, tenantID,
	)
	require.NoError(t, err)

	exceptionID := testutil.SeedRow(t, pool, "event_exceptions", map[string]any{
		"tenant_id":     tenantID,
		"event_id":      eventID,
		"original_date": old.Format("2006-01-02"),
		"title":         "Verschoben",
	})

	reminderID := testutil.SeedRow(t, pool, "event_reminders", map[string]any{
		"tenant_id":      tenantID,
		"event_id":       eventID,
		"minutes_before": 15,
	})

	handler := NewCalendarEventRetentionHandler(pool)
	affected, err := handler.Apply(testutil.WithTenantCtx(context.Background(), tenantID), tenantID, []uuid.UUID{eventID}, models.RetentionActionDelete, "")
	require.NoError(t, err)
	assert.Equal(t, 1, affected)

	var eventCount, attendeeCount, exceptionCount, reminderCount int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM calendar_events WHERE id = $1`, eventID).Scan(&eventCount))
	assert.Zero(t, eventCount, "deleted event must be gone")
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM event_attendees WHERE event_id = $1`, eventID).Scan(&attendeeCount))
	assert.Zero(t, attendeeCount, "cascade must take attendees with it")
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM event_exceptions WHERE id = $1`, exceptionID).Scan(&exceptionCount))
	assert.Zero(t, exceptionCount, "cascade must take exceptions with it")
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM event_reminders WHERE id = $1`, reminderID).Scan(&reminderCount))
	assert.Zero(t, reminderCount, "cascade must take reminders with it")
}

func TestCalendarEventRetentionHandler_ApplyAnonymizeClearsEventAndExceptionOverrideAndIsIdempotent(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Calendar Retention Apply Anonymize Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	userID := seedExportUser(t, pool, tenantID, "calendar-retention-apply-anon-user")
	defer testutil.CleanupRow(t, pool, "users", userID)
	calendarID := seedCalendar(t, pool, tenantID, userID)
	defer testutil.CleanupRow(t, pool, "calendars", calendarID)

	old := time.Now().UTC().AddDate(0, 0, -400)
	eventID := seedCalendarEvent(t, pool, tenantID, calendarID, old, old.Add(time.Hour), nil, nil)
	defer testutil.CleanupRow(t, pool, "calendar_events", eventID)

	exceptionID := testutil.SeedRow(t, pool, "event_exceptions", map[string]any{
		"tenant_id":     tenantID,
		"event_id":      eventID,
		"original_date": old.Format("2006-01-02"),
		"title":         "Verschoben wegen Arzttermin Frau Muster",
		"description":   "Bitte nicht stoeren.",
	})

	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	handler := NewCalendarEventRetentionHandler(pool)
	affected, err := handler.Apply(ctx, tenantID, []uuid.UUID{eventID}, models.RetentionActionAnonymize, "Geloeschter Termin 1")
	require.NoError(t, err)
	assert.Equal(t, 1, affected)

	sysCtx := testutil.WithSystemCtx(context.Background())
	var title, description, location string
	require.NoError(t, pool.QueryRow(sysCtx,
		`SELECT title, description, location FROM calendar_events WHERE id = $1`, eventID,
	).Scan(&title, &description, &location))
	assert.Equal(t, "[Geloeschter Termin 1]", title)
	assert.Equal(t, "[Geloeschter Termin 1]", description)
	assert.Equal(t, "[Geloeschter Termin 1]", location)

	var exceptionTitle, exceptionDescription *string
	require.NoError(t, pool.QueryRow(sysCtx,
		`SELECT title, description FROM event_exceptions WHERE id = $1`, exceptionID,
	).Scan(&exceptionTitle, &exceptionDescription))
	assert.Nil(t, exceptionTitle, "exception override must fall back to the (now anonymized) parent event")
	assert.Nil(t, exceptionDescription)

	cutoff := time.Now().UTC().AddDate(0, 0, -90)
	plan, err := handler.Plan(ctx, tenantID, cutoff, models.RetentionActionAnonymize)
	require.NoError(t, err)
	assert.NotContains(t, plan.Due, eventID, "anonymize must bump updated_at, taking the row out of the next Plan")
}

func TestCalendarEventRetentionHandler_ApplyUnsupportedAction(t *testing.T) {
	t.Parallel()

	handler := NewCalendarEventRetentionHandler(nil)
	_, err := handler.Apply(context.Background(), uuid.New(), []uuid.UUID{uuid.New()}, "retain", "")
	require.Error(t, err)
}

// seedCalendar seeds a minimal personal calendar for a user.
func seedCalendar(t *testing.T, pool *pgxpool.Pool, tenantID, ownerID uuid.UUID) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "calendars", map[string]any{
		"tenant_id": tenantID,
		"name":      "Testkalender " + uuid.New().String(),
		"owner_id":  ownerID,
	})
}

// seedCalendarEvent seeds a minimal calendar event. rrule nil means a
// one-off event; recurrenceEnd is only meaningful when rrule is set.
func seedCalendarEvent(t *testing.T, pool *pgxpool.Pool, tenantID, calendarID uuid.UUID, start, end time.Time, rrule *string, recurrenceEnd *time.Time) uuid.UUID {
	t.Helper()
	// created_by must reference an existing user; reuse the calendar's owner.
	var ownerID uuid.UUID
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT owner_id FROM calendars WHERE id = $1`, calendarID,
	).Scan(&ownerID))

	cols := map[string]any{
		"tenant_id":   tenantID,
		"calendar_id": calendarID,
		"title":       "Testtermin " + uuid.New().String(),
		"description": "Freitext mit Namensbezug.",
		"location":    "Buero Muster",
		"start_time":  start,
		"end_time":    end,
		"created_by":  ownerID,
	}
	if rrule != nil {
		cols["rrule"] = *rrule
	}
	if recurrenceEnd != nil {
		cols["recurrence_end"] = *recurrenceEnd
	}
	return testutil.SeedRow(t, pool, "calendar_events", cols)
}
