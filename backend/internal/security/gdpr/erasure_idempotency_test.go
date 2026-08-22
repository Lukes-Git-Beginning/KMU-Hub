package gdpr

// Doppellauf-Festigkeit der Erasure-Handler (scan-erasure-retention-idempotency-double-run,
// Lauf 10; korrigiert durch fix-erasure-handlers-not-idempotent-on-second-run, Lauf 11).
// Jeder Handler wird zweimal hintereinander gegen denselben Nutzer ausgefuehrt, genau wie es in
// Produktion passieren wuerde, wenn Service.ExecuteErasure ein zweites Mal fuer bereits
// geloeschte Daten aufgerufen wird -- etwa nach einem abgebrochenen und neu gestarteten Lauf.
// Service.GetNextAnonymizedLabel (postgres_repository.go:197) liefert bei jedem Aufruf ein NEUES
// Label (Zeilenzahl in gdpr_erasure_log + 1), deshalb fuehren diese Tests den zweiten Lauf
// bewusst mit einem zweiten, unterschiedlichen Label aus statt mit dem gleichen.
//
// Retention-Handler (retention_contacts.go, retention_dialer_chat.go,
// retention_helpdesk_formulare.go, retention.go) haben ihre Doppellauf-Festigkeit bereits als
// Teil ihrer eigenen Units (A11-A13) mit Tests belegt -- hier nicht dupliziert.
//
// Erwartetes Verhalten seit dem Fix: der zweite Lauf meldet 0 veraenderte Datensaetze und laesst
// den anonymisierten Inhalt des ersten Laufs unangetastet. Zwei Guard-Muster tragen das:
//   - Zustands-Guard, wo der erste Lauf einen abfragbaren Zustand hinterlaesst
//     (users.last_name = 'ANONYMIZED'; activities.description IS NULL).
//   - Label-Muster-Guard (anonymizedContentPattern) fuer Freitext, dessen Label pro Lauf wechselt
//     (messages.content, task_comments.content, calendar_events.title).
// Wo gar nichts geschrieben wird, faellt die Zeile stattdessen aus dem Count (tasks, die der
// Nutzer nur erstellt hat; Termine, die der Kalender-Cascade ohnehin entfernt).
//
// Audit ist ein No-Op und damit trivial doppellauf-fest (bereits durch TestAuditErasureHandler_NoOp
// abgedeckt, hier nicht dupliziert).

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestAuthErasureHandler_ExecuteErasure_SecondRunIsIdempotent(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         "auth-erasure-double-" + uuid.New().String()[:8] + "@tenanta.local",
		"password_hash": "$argon2id$v=19$test",
		"first_name":    "Erase",
		"last_name":     "Twice",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	ctx := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	h := NewAuthErasureHandler(pool)

	n1, err := h.ExecuteErasure(ctx, userID, "Geloeschter Benutzer #100", ErasureAnonymize)
	require.NoError(t, err)
	assert.Equal(t, 1, n1, "first run anonymizes the one seeded user row")

	// Second run with a DIFFERENT label -- reflects GetNextAnonymizedLabel incrementing on
	// every real call, not a test artifact.
	n2, err := h.ExecuteErasure(ctx, userID, "Geloeschter Benutzer #101", ErasureAnonymize)
	require.NoError(t, err)
	assert.Equal(t, 0, n2, "second run: the row is already anonymized, nothing left to change")

	var email string
	require.NoError(t, pool.QueryRow(ctx, `SELECT email FROM users WHERE id = $1`, userID).Scan(&email))
	assert.Contains(t, email, "100@deleted.invalid",
		"the first run's anonymized email must survive the second run untouched")
}

// A deactivated account still carries its PII, so the idempotency guard must key on the
// anonymization sentinel, not on is_active -- otherwise erasure would silently skip every
// account somebody had disabled beforehand.
func TestAuthErasureHandler_ExecuteErasure_AnonymizesAlreadyDeactivatedAccount(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         "auth-erasure-inactive-" + uuid.New().String()[:8] + "@tenanta.local",
		"password_hash": "$argon2id$v=19$test",
		"first_name":    "Deaktiviert",
		"last_name":     "Mitarbeiter",
		"is_active":     false,
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	ctx := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	h := NewAuthErasureHandler(pool)

	n, err := h.ExecuteErasure(ctx, userID, "Geloeschter Benutzer #110", ErasureAnonymize)
	require.NoError(t, err)
	assert.Equal(t, 1, n, "an already deactivated but not yet anonymized account must still be erased")

	var email, lastName string
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT email, last_name FROM users WHERE id = $1`, userID,
	).Scan(&email, &lastName))
	assert.Contains(t, email, "110@deleted.invalid")
	assert.Equal(t, anonymizedLastName, lastName)
}

func TestCRMErasureHandler_ExecuteErasure_SecondRunIsIdempotent(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CRM Erasure Double-Run Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	userID := seedExportUser(t, pool, tenantOwn, "crm-erasure-double")
	defer testutil.CleanupRow(t, pool, "users", userID)

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantOwn,
		"first_name": "Erika",
		"last_name":  "Musterfrau",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "contacts", contactID)

	activityID := testutil.SeedRow(t, pool, "activities", map[string]any{
		"tenant_id":     tenantOwn,
		"activity_type": "note",
		"subject":       "Notiz",
		"description":   "Persoenliche Notiz",
		"assigned_to":   userID,
		"created_by":    userID,
	})
	defer testutil.CleanupRow(t, pool, "activities", activityID)

	h := NewCRMErasureHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantOwn)

	n1, err := h.ExecuteErasure(ctx, userID, erasureLabel, ErasureAnonymize)
	require.NoError(t, err)
	assert.Equal(t, 1, n1, "first run: 1 activity; the contact is retained and not counted")

	// Second run: the activity's description and assigned_to are already NULL, so the
	// UPDATE's WHERE clause no longer matches it even though created_by still points here.
	n2, err := h.ExecuteErasure(ctx, userID, erasureLabel, ErasureAnonymize)
	require.NoError(t, err)
	assert.Equal(t, 0, n2, "second run: nothing left to anonymize on the activity")
}

func TestChatErasureHandler_ExecuteErasure_SecondRunIsIdempotent(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Chat Erasure Double-Run Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	userID := seedExportUser(t, pool, tenantOwn, "chat-erasure-double")
	defer testutil.CleanupRow(t, pool, "users", userID)

	channelID := testutil.SeedRow(t, pool, "channels", map[string]any{
		"tenant_id":  tenantOwn,
		"name":       "double-run-channel",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "channels", channelID)

	messageID := testutil.SeedRow(t, pool, "messages", map[string]any{
		"tenant_id":  tenantOwn,
		"channel_id": channelID,
		"content":    "Urspruengliche Nachricht",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "messages", messageID)

	seedChannelMembership(t, pool, tenantOwn, channelID, userID, "member")

	h := NewChatErasureHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantOwn)

	labelFirst := "Geloeschter Benutzer #200"
	n1, err := h.ExecuteErasure(ctx, userID, labelFirst, ErasureAnonymize)
	require.NoError(t, err)
	assert.Equal(t, 2, n1, "first run: 1 message anonymized + 1 channel membership deleted")

	var content string
	require.NoError(t, pool.QueryRow(ctx, `SELECT content FROM messages WHERE id = $1`, messageID).Scan(&content))
	assert.Equal(t, "["+labelFirst+"]", content)

	// Second run with a fresh label (GetNextAnonymizedLabel semantics). The membership is
	// already gone, and the message content already carries a bracketed label, so the
	// pattern guard keeps the UPDATE from matching it again.
	labelSecond := "Geloeschter Benutzer #201"
	n2, err := h.ExecuteErasure(ctx, userID, labelSecond, ErasureAnonymize)
	require.NoError(t, err)
	assert.Equal(t, 0, n2, "second run: message already anonymized, membership already deleted")

	require.NoError(t, pool.QueryRow(ctx, `SELECT content FROM messages WHERE id = $1`, messageID).Scan(&content))
	assert.Equal(t, "["+labelFirst+"]", content,
		"the first run's anonymized content must survive the second run untouched")
}

// A message the user soft-deleted themselves keeps its content in the database
// (message.PostgresRepository.Delete only flips is_deleted), so erasure must still replace it.
// This pins that the idempotency guard keys on the label pattern, not on is_deleted.
func TestChatErasureHandler_ExecuteErasure_AnonymizesSoftDeletedMessage(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Chat Erasure Soft-Delete Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	userID := seedExportUser(t, pool, tenantOwn, "chat-erasure-softdel")
	defer testutil.CleanupRow(t, pool, "users", userID)

	channelID := testutil.SeedRow(t, pool, "channels", map[string]any{
		"tenant_id":  tenantOwn,
		"name":       "soft-delete-channel",
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "channels", channelID)

	messageID := testutil.SeedRow(t, pool, "messages", map[string]any{
		"tenant_id":  tenantOwn,
		"channel_id": channelID,
		"content":    "Selbst geloeschte, aber noch gespeicherte Nachricht",
		"created_by": userID,
		"is_deleted": true,
	})
	defer testutil.CleanupRow(t, pool, "messages", messageID)

	h := NewChatErasureHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantOwn)

	n, err := h.ExecuteErasure(ctx, userID, erasureLabel, ErasureAnonymize)
	require.NoError(t, err)
	assert.Equal(t, 1, n, "a soft-deleted message still holds personal data and must be anonymized")

	var content string
	require.NoError(t, pool.QueryRow(ctx, `SELECT content FROM messages WHERE id = $1`, messageID).Scan(&content))
	assert.Equal(t, "["+erasureLabel+"]", content)
}

func TestWorkErasureHandler_ExecuteErasure_SecondRunIsIdempotent(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Work Erasure Double-Run Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	userID := seedExportUser(t, pool, tenantOwn, "work-erasure-double")
	defer testutil.CleanupRow(t, pool, "users", userID)

	taskID := testutil.SeedRow(t, pool, "tasks", map[string]any{
		"tenant_id":   tenantOwn,
		"task_number": 1,
		"title":       "Doppellauf-Aufgabe",
		"assignee_id": userID,
		"created_by":  userID,
	})
	defer testutil.CleanupRow(t, pool, "tasks", taskID)

	commentID := testutil.SeedRow(t, pool, "task_comments", map[string]any{
		"tenant_id": tenantOwn,
		"task_id":   taskID,
		"author_id": userID,
		"content":   "Persoenliche Notiz",
	})
	defer testutil.CleanupRow(t, pool, "task_comments", commentID)

	h := NewWorkErasureHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantOwn)

	labelFirst := "Geloeschter Benutzer #300"
	n1, err := h.ExecuteErasure(ctx, userID, labelFirst, ErasureAnonymize)
	require.NoError(t, err)
	assert.Equal(t, 2, n1, "first run: 1 task unassigned + 1 comment anonymized (no time entries seeded)")

	// Second run with a fresh label: the task's assignee_id is already NULL and the comment
	// already carries a bracketed label, so neither statement matches again.
	labelSecond := "Geloeschter Benutzer #301"
	n2, err := h.ExecuteErasure(ctx, userID, labelSecond, ErasureAnonymize)
	require.NoError(t, err)
	assert.Equal(t, 0, n2, "second run: task already unassigned, comment already anonymized")

	var commentContent string
	require.NoError(t, pool.QueryRow(ctx, `SELECT content FROM task_comments WHERE id = $1`, commentID).Scan(&commentContent))
	assert.Equal(t, "["+labelFirst+"]", commentContent,
		"the first run's anonymized comment content must survive the second run untouched")
}

func TestCalendarErasureHandler_ExecuteErasure_SecondRunStillReportsCreatedEvents(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Calendar Erasure Double-Run Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	userID := seedExportUser(t, pool, tenantOwn, "calendar-erasure-double")
	defer testutil.CleanupRow(t, pool, "users", userID)

	calendarID := testutil.SeedRow(t, pool, "calendars", map[string]any{
		"tenant_id": tenantOwn,
		"name":      "Persoenlicher Kalender",
		"owner_id":  userID,
	})
	defer testutil.CleanupRow(t, pool, "calendars", calendarID)

	eventID := testutil.SeedRow(t, pool, "calendar_events", map[string]any{
		"tenant_id":   tenantOwn,
		"calendar_id": calendarID,
		"title":       "Termin",
		"start_time":  "2026-08-20T09:00:00Z",
		"end_time":    "2026-08-20T10:00:00Z",
		"created_by":  userID,
	})
	defer testutil.CleanupRow(t, pool, "calendar_events", eventID)

	// event_attendees has a composite primary key (event_id, user_id) and no id column --
	// testutil.SeedRow relies on RETURNING id, so insert directly (same pattern as
	// seedChannelMembership in export_crm_chat_test.go).
	_, err := pool.Exec(testutil.WithSystemCtx(context.Background()),
		`INSERT INTO event_attendees (event_id, user_id, tenant_id) VALUES ($1, $2, $3)`,
		eventID, userID, tenantOwn,
	)
	require.NoError(t, err)

	// user_calendar_preferences has user_id as its primary key and no id column --
	// testutil.SeedRow relies on RETURNING id, so insert directly.
	_, err = pool.Exec(testutil.WithSystemCtx(context.Background()),
		`INSERT INTO user_calendar_preferences (user_id, tenant_id) VALUES ($1, $2)`,
		userID, tenantOwn,
	)
	require.NoError(t, err)

	h := NewCalendarErasureHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantOwn)

	n1, err := h.ExecuteErasure(ctx, userID, erasureLabel, ErasureDelete)
	require.NoError(t, err)
	// This calendar is the subject's own personal calendar with no booking_pages
	// row attached, so fix-calendar-erasure-incomplete-and-doc-mismatch (Lauf 10)
	// now deletes it outright: 1 calendar + 1 event (cascaded, pre-counted) +
	// 1 attendee record + 1 preference row.
	assert.Equal(t, 4, n1, "first run: calendar and its event deleted, attendee and preference rows deleted")

	// Second run: the calendar, its event, the attendee row and the preference
	// row are all already gone from the first run, so nothing matches anymore.
	n2, err := h.ExecuteErasure(ctx, userID, erasureLabel, ErasureDelete)
	require.NoError(t, err)
	assert.Equal(t, 0, n2, "second run: everything from the first run is already gone, nothing left to match")
}

// The other half of the calendar case: an event the subject organizes on a calendar it does
// NOT own is only anonymized, never deleted, and calendar_events.created_by keeps pointing at
// the subject forever. Only the label-pattern guard stops the second run from recounting it.
func TestCalendarErasureHandler_ExecuteErasure_SecondRunSkipsAnonymizedForeignEvent(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Calendar Erasure Foreign-Cal Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	userID := seedExportUser(t, pool, tenantOwn, "calendar-erasure-foreign")
	defer testutil.CleanupRow(t, pool, "users", userID)

	colleagueID := seedExportUser(t, pool, tenantOwn, "calendar-erasure-colleague")
	defer testutil.CleanupRow(t, pool, "users", colleagueID)

	// A calendar owned by somebody else -- the erasure must not delete it.
	sharedCalID := testutil.SeedRow(t, pool, "calendars", map[string]any{
		"tenant_id": tenantOwn,
		"name":      "Team-Kalender",
		"owner_id":  colleagueID,
	})
	defer testutil.CleanupRow(t, pool, "calendars", sharedCalID)

	eventID := testutil.SeedRow(t, pool, "calendar_events", map[string]any{
		"tenant_id":   tenantOwn,
		"calendar_id": sharedCalID,
		"title":       "Quartalsgespraech mit Erika",
		"description": "Persoenliche Notizen",
		"start_time":  "2026-08-20T09:00:00Z",
		"end_time":    "2026-08-20T10:00:00Z",
		"created_by":  userID,
	})
	defer testutil.CleanupRow(t, pool, "calendar_events", eventID)

	h := NewCalendarErasureHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantOwn)

	labelFirst := "Geloeschter Benutzer #400"
	n1, err := h.ExecuteErasure(ctx, userID, labelFirst, ErasureDelete)
	require.NoError(t, err)
	assert.Equal(t, 1, n1, "first run: the event on the foreign calendar is anonymized")

	var title string
	var description *string
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT title, description FROM calendar_events WHERE id = $1`, eventID,
	).Scan(&title, &description))
	assert.Equal(t, "["+labelFirst+"]", title)
	assert.Nil(t, description)

	labelSecond := "Geloeschter Benutzer #401"
	n2, err := h.ExecuteErasure(ctx, userID, labelSecond, ErasureDelete)
	require.NoError(t, err)
	assert.Equal(t, 0, n2, "second run: the event is already anonymized, nothing left to change")

	require.NoError(t, pool.QueryRow(ctx,
		`SELECT title FROM calendar_events WHERE id = $1`, eventID,
	).Scan(&title))
	assert.Equal(t, "["+labelFirst+"]", title,
		"the first run's anonymized title must survive the second run untouched")

	// The colleague's calendar itself is untouched by the subject's erasure.
	testutil.AssertRowCount(t, pool, ctx, "calendars", sharedCalID, 1)
}

func TestNotificationErasureHandler_ExecuteErasure_SecondRunIsTrulyIdempotent(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Notification Erasure Double-Run Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	userID := seedExportUser(t, pool, tenantOwn, "notif-erasure-double")
	defer testutil.CleanupRow(t, pool, "users", userID)

	testutil.SeedRow(t, pool, "notifications", map[string]any{
		"tenant_id":      tenantOwn,
		"user_id":        userID,
		"event_type_key": "task.assigned",
		"module_id":      "work",
		"title":          "Aufgabe zugewiesen",
	})
	testutil.SeedRow(t, pool, "notification_preferences", map[string]any{
		"tenant_id": tenantOwn,
		"user_id":   userID,
		"module_id": "work",
	})
	testutil.SeedRow(t, pool, "notification_quiet_hours", map[string]any{
		"tenant_id": tenantOwn,
		"user_id":   userID,
	})
	testutil.SeedRow(t, pool, "notification_mutes", map[string]any{
		"tenant_id":   tenantOwn,
		"user_id":     userID,
		"module_id":   "work",
		"resource_id": "task-123",
	})

	h := NewNotificationErasureHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantOwn)

	n1, err := h.ExecuteErasure(ctx, userID, erasureLabel, ErasureDelete)
	require.NoError(t, err)
	assert.Equal(t, 4, n1, "first run deletes all four seeded rows")

	// Second run: every table is a plain DELETE keyed on user_id, and all rows are already
	// gone -- this handler is correctly idempotent, unlike Auth/CRM/Chat/Work/Calendar above.
	n2, err := h.ExecuteErasure(ctx, userID, erasureLabel, ErasureDelete)
	require.NoError(t, err)
	assert.Equal(t, 0, n2, "second run correctly reports 0 -- nothing left to delete")
}
