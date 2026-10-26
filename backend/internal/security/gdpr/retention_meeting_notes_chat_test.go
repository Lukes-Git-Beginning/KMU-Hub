package gdpr

// Coverage for MeetingNoteRetentionHandler and MeetingChatRetentionHandler
// (feat-retention-meeting-notes-and-chat-2y, Lauf 17). The cases that matter:
// the clock is the END of the meeting (actual_end wins over scheduled_end),
// younger meetings are never due, a dry-run Plan deletes nothing, delete is
// idempotent and tenant-scoped, and the meeting row itself survives.

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

func TestMeetingRetentionHandlers_Metadata(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		handler RetentionHandler
		name    string
	}{
		{NewMeetingNoteRetentionHandler(nil), "meeting_notes"},
		{NewMeetingChatRetentionHandler(nil), "meeting_chat"},
	} {
		assert.Equal(t, tc.name, tc.handler.ResourceType())
		assert.Equal(t, tc.name, tc.handler.Table())
		assert.NotEmpty(t, tc.handler.DateColumn())
		assert.True(t, tc.handler.SupportsAction(models.RetentionActionDelete))
		assert.False(t, tc.handler.SupportsAction(models.RetentionActionAnonymize))

		_, err := tc.handler.Apply(context.Background(), uuid.New(), []uuid.UUID{uuid.New()}, models.RetentionActionAnonymize, "x")
		require.Error(t, err, "%s must refuse an action it cannot perform instead of deleting", tc.name)
	}
}

type meetingRetentionFixture struct {
	tenantID uuid.UUID
	// Rows by case, notes and chat side by side.
	oldNote, oldScheduledOnlyNote, freshNote, longRunningNote, otherTenantNote uuid.UUID
	oldChat, oldScheduledOnlyChat, freshChat, longRunningChat, otherTenantChat uuid.UUID
	oldMeeting                                                                 uuid.UUID
}

func seedMeetingRetentionFixture(t *testing.T, pool *pgxpool.Pool) meetingRetentionFixture {
	t.Helper()

	f := meetingRetentionFixture{tenantID: uuid.New()}
	testutil.EnsureTenant(t, pool, f.tenantID, "Meeting Retention Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", f.tenantID) })
	userID := seedExportUser(t, pool, f.tenantID, "meeting-retention-user")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })

	otherTenant := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenant, "Meeting Retention Other Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", otherTenant) })
	otherUser := seedExportUser(t, pool, otherTenant, "meeting-retention-other-user")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", otherUser) })

	now := time.Now().UTC()
	threeYears := now.AddDate(-3, 0, 0)
	oneMonth := now.AddDate(0, -1, 0)

	newMeeting := func(tenantID, organizer uuid.UUID, scheduledEnd time.Time, actualEnd *time.Time) uuid.UUID {
		cols := map[string]any{
			"tenant_id": tenantID, "title": "Retention " + uuid.NewString(), "organizer_id": organizer,
			"scheduled_start": scheduledEnd.Add(-time.Hour), "scheduled_end": scheduledEnd,
		}
		if actualEnd != nil {
			cols["actual_end"] = *actualEnd
		}
		id := testutil.SeedRow(t, pool, "meetings", cols)
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "meetings", id) })
		return id
	}
	note := func(tenantID, meetingID, author uuid.UUID) uuid.UUID {
		id := testutil.SeedRow(t, pool, "meeting_notes", map[string]any{
			"tenant_id": tenantID, "meeting_id": meetingID, "author_id": author, "content": "Notiz " + uuid.NewString(),
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "meeting_notes", id) })
		return id
	}
	chat := func(tenantID, meetingID, sender uuid.UUID) uuid.UUID {
		id := testutil.SeedRow(t, pool, "meeting_chat", map[string]any{
			"tenant_id": tenantID, "meeting_id": meetingID, "sender_id": sender, "sender_name": "Anna", "message": "Hallo " + uuid.NewString(),
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "meeting_chat", id) })
		return id
	}

	// Ended three years ago, actual_end recorded.
	oldEnd := threeYears
	f.oldMeeting = newMeeting(f.tenantID, userID, threeYears, &oldEnd)
	// Never started, scheduled_end three years ago.
	oldScheduledOnly := newMeeting(f.tenantID, userID, threeYears, nil)
	// Ended a month ago.
	freshEnd := oneMonth
	fresh := newMeeting(f.tenantID, userID, oneMonth, &freshEnd)
	// Scheduled three years ago but actually ran until a month ago: actual_end must win.
	longEnd := oneMonth
	longRunning := newMeeting(f.tenantID, userID, threeYears, &longEnd)
	// Another tenant's old meeting.
	otherEnd := threeYears
	other := newMeeting(otherTenant, otherUser, threeYears, &otherEnd)

	f.oldNote, f.oldChat = note(f.tenantID, f.oldMeeting, userID), chat(f.tenantID, f.oldMeeting, userID)
	f.oldScheduledOnlyNote, f.oldScheduledOnlyChat = note(f.tenantID, oldScheduledOnly, userID), chat(f.tenantID, oldScheduledOnly, userID)
	f.freshNote, f.freshChat = note(f.tenantID, fresh, userID), chat(f.tenantID, fresh, userID)
	f.longRunningNote, f.longRunningChat = note(f.tenantID, longRunning, userID), chat(f.tenantID, longRunning, userID)
	f.otherTenantNote, f.otherTenantChat = note(otherTenant, other, otherUser), chat(otherTenant, other, otherUser)
	return f
}

func TestMeetingRetentionHandlers_PlanUsesMeetingEndAndIsTenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)
	f := seedMeetingRetentionFixture(t, pool)

	ctx := testutil.WithTenantCtx(context.Background(), f.tenantID)
	cutoff := time.Now().UTC().AddDate(-2, 0, 0)

	notePlan, err := NewMeetingNoteRetentionHandler(pool).Plan(ctx, f.tenantID, cutoff, models.RetentionActionDelete)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{f.oldNote, f.oldScheduledOnlyNote}, notePlan.Due)
	assert.NotContains(t, notePlan.Due, f.freshNote, "a meeting that ended a month ago is not due")
	assert.NotContains(t, notePlan.Due, f.longRunningNote, "actual_end wins over an old scheduled_end")
	assert.NotContains(t, notePlan.Due, f.otherTenantNote, "another tenant's old note must not leak into the plan")

	chatPlan, err := NewMeetingChatRetentionHandler(pool).Plan(ctx, f.tenantID, cutoff, models.RetentionActionDelete)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{f.oldChat, f.oldScheduledOnlyChat}, chatPlan.Due)
	assert.NotContains(t, chatPlan.Due, f.freshChat)
	assert.NotContains(t, chatPlan.Due, f.longRunningChat)
	assert.NotContains(t, chatPlan.Due, f.otherTenantChat)

	// RLS already hides the other tenant above. Under the system context it
	// sees every tenant, so only the handler's own tenant filter keeps the
	// other tenant's old rows out of the plan.
	sys := testutil.WithSystemCtx(context.Background())
	sysNotePlan, err := NewMeetingNoteRetentionHandler(pool).Plan(sys, f.tenantID, cutoff, models.RetentionActionDelete)
	require.NoError(t, err)
	assert.NotContains(t, sysNotePlan.Due, f.otherTenantNote)
	sysChatPlan, err := NewMeetingChatRetentionHandler(pool).Plan(sys, f.tenantID, cutoff, models.RetentionActionDelete)
	require.NoError(t, err)
	assert.NotContains(t, sysChatPlan.Due, f.otherTenantChat)

	// Plan is the dry-run path and must not modify anything.
	var notes, chats int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM meeting_notes WHERE tenant_id = $1`, f.tenantID).Scan(&notes))
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM meeting_chat WHERE tenant_id = $1`, f.tenantID).Scan(&chats))
	assert.Equal(t, 4, notes, "Plan must not delete")
	assert.Equal(t, 4, chats, "Plan must not delete")
}

func TestMeetingRetentionHandlers_ApplyDeletesDueRowsIdempotentlyAndKeepsTheMeeting(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)
	f := seedMeetingRetentionFixture(t, pool)

	ctx := testutil.WithTenantCtx(context.Background(), f.tenantID)
	cutoff := time.Now().UTC().AddDate(-2, 0, 0)
	noteHandler := NewMeetingNoteRetentionHandler(pool)
	chatHandler := NewMeetingChatRetentionHandler(pool)

	// An id from another tenant handed to this tenant's Apply is never deleted,
	// even where RLS would let the statement see it (system context).
	sys := testutil.WithSystemCtx(context.Background())
	foreign, err := noteHandler.Apply(sys, f.tenantID, []uuid.UUID{f.otherTenantNote}, models.RetentionActionDelete, "")
	require.NoError(t, err)
	assert.Zero(t, foreign)
	foreign, err = chatHandler.Apply(sys, f.tenantID, []uuid.UUID{f.otherTenantChat}, models.RetentionActionDelete, "")
	require.NoError(t, err)
	assert.Zero(t, foreign)

	notePlan, err := noteHandler.Plan(ctx, f.tenantID, cutoff, models.RetentionActionDelete)
	require.NoError(t, err)
	chatPlan, err := chatHandler.Plan(ctx, f.tenantID, cutoff, models.RetentionActionDelete)
	require.NoError(t, err)

	affected, err := noteHandler.Apply(ctx, f.tenantID, notePlan.Due, models.RetentionActionDelete, "")
	require.NoError(t, err)
	assert.Equal(t, 2, affected)
	affected, err = chatHandler.Apply(ctx, f.tenantID, chatPlan.Due, models.RetentionActionDelete, "")
	require.NoError(t, err)
	assert.Equal(t, 2, affected)

	exists := func(table string, id uuid.UUID) bool {
		var n int
		sys := testutil.WithSystemCtx(context.Background())
		require.NoError(t, pool.QueryRow(sys, `SELECT COUNT(*) FROM `+table+` WHERE id = $1`, id).Scan(&n))
		return n > 0
	}
	assert.False(t, exists("meeting_notes", f.oldNote))
	assert.False(t, exists("meeting_notes", f.oldScheduledOnlyNote))
	assert.False(t, exists("meeting_chat", f.oldChat))
	assert.False(t, exists("meeting_chat", f.oldScheduledOnlyChat))
	assert.True(t, exists("meeting_notes", f.freshNote))
	assert.True(t, exists("meeting_notes", f.longRunningNote))
	assert.True(t, exists("meeting_notes", f.otherTenantNote), "the other tenant's note must survive")
	assert.True(t, exists("meeting_chat", f.freshChat))
	assert.True(t, exists("meeting_chat", f.otherTenantChat), "the other tenant's chat line must survive")
	assert.True(t, exists("meetings", f.oldMeeting), "the meeting row is the booking record and stays")

	// A second run over the same ids finds nothing left.
	again, err := noteHandler.Apply(ctx, f.tenantID, notePlan.Due, models.RetentionActionDelete, "")
	require.NoError(t, err)
	assert.Zero(t, again)
	again, err = chatHandler.Apply(ctx, f.tenantID, chatPlan.Due, models.RetentionActionDelete, "")
	require.NoError(t, err)
	assert.Zero(t, again)
}
