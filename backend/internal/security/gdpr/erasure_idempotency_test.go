package gdpr

// Doppellauf-Festigkeit der Erasure-Handler (scan-erasure-retention-idempotency-double-run,
// Lauf 10). Jeder Handler wird zweimal hintereinander gegen denselben Nutzer ausgefuehrt, genau
// wie es in Produktion passieren wuerde, wenn Service.ExecuteErasure ein zweites Mal fuer
// bereits geloeschte Daten aufgerufen wird -- etwa nach einem abgebrochenen und neu gestarteten
// Lauf. Service.GetNextAnonymizedLabel (postgres_repository.go:197) liefert bei jedem Aufruf ein
// NEUES Label (Zeilenzahl in gdpr_erasure_log + 1), deshalb simulieren diese Tests den zweiten
// Lauf bewusst mit einem zweiten, unterschiedlichen Label statt mit dem gleichen.
//
// Retention-Handler (retention_contacts.go, retention_dialer_chat.go,
// retention_helpdesk_formulare.go, retention.go) haben ihre Doppellauf-Festigkeit bereits als
// Teil ihrer eigenen Units (A11-A13) mit Tests belegt -- hier nicht dupliziert.
//
// Ergebnis (siehe JOURNAL Iteration 47 fuer die volle Einordnung):
//   - Auth, CRM, Chat, Work sind NICHT doppellauf-fest: jeder Handler matcht seine Zeilen ueber
//     eine Spalte, die die Erasure selbst nicht aendert (users.id, activities/tasks/messages/
//     task_comments.created_by bzw. .author_id), und zaehlt sie deshalb bei jedem Lauf erneut als
//     "affected" -- ohne dass beim zweiten Lauf noch etwas Sinnvolles passiert waere. Bei Chat und
//     Work wird der Nachrichten-/Kommentarinhalt beim zweiten Lauf sogar mit dem NEUEN Label
//     ueberschrieben, wenn Service ein frisches Label zieht. Je ein Fix-Unit ist ans Backlog-Ende
//     angehaengt.
//   - Calendar ist TEILWEISE doppellauf-fest seit fix-calendar-erasure-incomplete-and-doc-mismatch
//     (Lauf 10): ein Termin auf der EIGENEN, jetzt geloeschten Personenkalender-Zeile verschwindet
//     mit ihr und zaehlt beim zweiten Lauf nicht mehr. Ein Termin, den der Nutzer auf einem
//     FREMDEN (geteilten) Kalender organisiert hat, wird weiterhin nur anonymisiert statt
//     geloescht und matcht `calendar_events.created_by` bei jedem erneuten Lauf erneut -- dieser
//     Rest-Fall bleibt Teil von fix-erasure-handlers-not-idempotent-on-second-run.
//   - Notification ist doppellauf-fest (reine DELETEs, zweiter Lauf meldet 0).
//   - Audit ist ein No-Op und damit trivial doppellauf-fest (bereits durch
//     TestAuditErasureHandler_NoOp abgedeckt, hier nicht dupliziert).

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestAuthErasureHandler_ExecuteErasure_SecondRunReAnonymizesWithNewLabel(t *testing.T) {
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
	// BUG: the UPDATE has no guard against an already-anonymized row (e.g. WHERE is_active),
	// so it matches WHERE id = $1 again and reports the same "1 record affected" as if this
	// were a fresh erasure. See fix-erasure-handlers-not-idempotent-on-second-run.
	assert.Equal(t, 1, n2, "BUG: second run still reports the user row as newly affected")

	var email string
	require.NoError(t, pool.QueryRow(ctx, `SELECT email FROM users WHERE id = $1`, userID).Scan(&email))
	assert.Contains(t, email, "101@deleted.invalid",
		"BUG: the second run's label silently overwrites the first run's anonymized email")
}

func TestCRMErasureHandler_ExecuteErasure_SecondRunRecountsCreatedByForever(t *testing.T) {
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
	assert.Equal(t, 2, n1, "first run: 1 activity + 1 contact")

	// Second run: the activity's description is already NULL and assigned_to already NULL,
	// but the UPDATE's WHERE clause matches on created_by, which never changes -- so it
	// matches again. contacts.created_by is likewise a permanent, non-erasable FK that gets
	// re-counted on every call.
	n2, err := h.ExecuteErasure(ctx, userID, erasureLabel, ErasureAnonymize)
	require.NoError(t, err)
	// BUG: reports the exact same count again, though nothing changed this round.
	// See fix-erasure-handlers-not-idempotent-on-second-run.
	assert.Equal(t, n1, n2, "BUG: second run recounts the same activity+contact as if freshly erased")
}

func TestChatErasureHandler_ExecuteErasure_SecondRunOverwritesAnonymizedContent(t *testing.T) {
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
	// already gone (0), but the message still matches WHERE created_by = $1 -- created_by is
	// never cleared -- so its content is silently overwritten with the new label.
	labelSecond := "Geloeschter Benutzer #201"
	n2, err := h.ExecuteErasure(ctx, userID, labelSecond, ErasureAnonymize)
	require.NoError(t, err)
	// BUG: reports 1 record affected again though the message was already anonymized.
	// See fix-erasure-handlers-not-idempotent-on-second-run.
	assert.Equal(t, 1, n2, "BUG: second run still reports the already-anonymized message as newly affected")

	require.NoError(t, pool.QueryRow(ctx, `SELECT content FROM messages WHERE id = $1`, messageID).Scan(&content))
	assert.Equal(t, "["+labelSecond+"]", content,
		"BUG: the first run's anonymized content is silently overwritten by the second run's label")
}

func TestWorkErasureHandler_ExecuteErasure_SecondRunOverwritesAnonymizedComment(t *testing.T) {
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

	// Second run with a fresh label: the task still matches WHERE created_by = $1 (permanent
	// FK), and the comment still matches WHERE author_id = $1 (also permanent) -- both get
	// re-touched, and the comment's already-anonymized content is overwritten.
	labelSecond := "Geloeschter Benutzer #301"
	n2, err := h.ExecuteErasure(ctx, userID, labelSecond, ErasureAnonymize)
	require.NoError(t, err)
	// BUG: reports the same 2 records again though nothing new happened.
	// See fix-erasure-handlers-not-idempotent-on-second-run.
	assert.Equal(t, n1, n2, "BUG: second run recounts the same task+comment as if freshly erased")

	var commentContent string
	require.NoError(t, pool.QueryRow(ctx, `SELECT content FROM task_comments WHERE id = $1`, commentID).Scan(&commentContent))
	assert.Equal(t, "["+labelSecond+"]", commentContent,
		"BUG: the first run's anonymized comment content is silently overwritten by the second run's label")
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
	// This specific scenario (an event on the subject's OWN personal calendar)
	// is now genuinely idempotent because the calendar-deletion path removes
	// the event with it. That does NOT generalize: an event the subject
	// organizes on a calendar it does NOT own (shared calendar, or its own
	// calendar retained because a booking_pages row depends on it) is only
	// anonymized, never deleted, and calendar_events.created_by keeps matching
	// it on every subsequent run -- that half of
	// fix-erasure-handlers-not-idempotent-on-second-run's Calendar finding is
	// unresolved and out of scope here.
	n2, err := h.ExecuteErasure(ctx, userID, erasureLabel, ErasureDelete)
	require.NoError(t, err)
	assert.Equal(t, 0, n2, "second run: everything from the first run is already gone, nothing left to match")
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
