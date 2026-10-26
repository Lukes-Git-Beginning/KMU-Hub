package gdpr

// Coverage for CallSessionRetentionHandler (feat-retention-call-sessions-24m,
// Lauf 17): older than the cutoff is due (clock = ended_at, else created_at),
// younger is not, a session with a recording is skipped with a reason and
// survives Apply, Plan (the dry-run path) deletes nothing, participants go
// with their session, Apply is tenant-scoped and idempotent.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestCallSessionRetentionHandler_Metadata(t *testing.T) {
	t.Parallel()

	h := NewCallSessionRetentionHandler(nil)
	assert.Equal(t, "call_sessions", h.ResourceType())
	assert.Equal(t, "call_sessions", h.Table())
	assert.NotEmpty(t, h.DateColumn())
	assert.True(t, h.SupportsAction(models.RetentionActionDelete))
	assert.False(t, h.SupportsAction(models.RetentionActionAnonymize))

	_, err := h.Apply(context.Background(), uuid.New(), []uuid.UUID{uuid.New()}, models.RetentionActionAnonymize, "x")
	require.Error(t, err, "must refuse an action it cannot perform instead of deleting")
}

func TestCallSessionRetentionHandler_PlanAndApply(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID, otherTenant := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Call Retention Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })
	testutil.EnsureTenant(t, pool, otherTenant, "Call Retention Other Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", otherTenant) })
	userID := seedExportUser(t, pool, tenantID, "call-retention-user")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })
	otherUser := seedExportUser(t, pool, otherTenant, "call-retention-other-user")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", otherUser) })

	sys := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()
	// call seeds a session plus one participant. endedAt nil = never ended.
	call := func(tenant, user uuid.UUID, createdAt time.Time, endedAt *time.Time) uuid.UUID {
		cols := map[string]any{
			"tenant_id": tenant, "call_type": "one_to_one", "status": "ended",
			"room_name": "ret-" + uuid.NewString(), "initiator_id": user, "created_at": createdAt,
		}
		if endedAt != nil {
			cols["ended_at"] = *endedAt
		}
		id := testutil.SeedRow(t, pool, "call_sessions", cols)
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "call_sessions", id) })
		_, err := pool.Exec(sys,
			`INSERT INTO call_participants (call_id, user_id, tenant_id) VALUES ($1, $2, $3)`,
			id, user, tenant)
		require.NoError(t, err)
		return id
	}
	ago := func(months int) time.Time { return now.AddDate(0, -months, 0) }
	endedOld, endedFresh, endedJustOver := ago(30), ago(6), now.AddDate(-2, 0, -1)

	old := call(tenantID, userID, ago(31), &endedOld)
	justOver := call(tenantID, userID, ago(25), &endedJustOver)
	fresh := call(tenantID, userID, ago(7), &endedFresh)
	// Ended recently although it started long ago: the clock is ended_at.
	longCall := call(tenantID, userID, ago(30), &endedFresh)
	neverEnded := call(tenantID, userID, ago(30), nil)
	neverEndedFresh := call(tenantID, userID, ago(2), nil)
	otherOld := call(otherTenant, otherUser, ago(31), &endedOld)

	withRecording := call(tenantID, userID, ago(31), &endedOld)
	recID := testutil.SeedRow(t, pool, "recordings", map[string]any{
		"tenant_id": tenantID, "call_id": withRecording,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "recordings", recID) })

	h := NewCallSessionRetentionHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	cutoff := now.AddDate(-2, 0, 0)

	plan, err := h.Plan(ctx, tenantID, cutoff, models.RetentionActionDelete)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{old, justOver, neverEnded}, plan.Due)
	require.Len(t, plan.Skipped, 1)
	assert.Equal(t, withRecording, plan.Skipped[0].RecordID)
	assert.NotEmpty(t, plan.Skipped[0].Reason)

	// RLS hides the other tenant above; under the system context only the
	// handler's own tenant filter keeps its old row out of the plan.
	sysPlan, err := h.Plan(sys, tenantID, cutoff, models.RetentionActionDelete)
	require.NoError(t, err)
	assert.NotContains(t, sysPlan.Due, otherOld)

	count := func(table, col string, id uuid.UUID) int {
		var n int
		require.NoError(t, pool.QueryRow(sys, `SELECT COUNT(*) FROM `+table+` WHERE `+col+` = $1`, id).Scan(&n))
		return n
	}
	for _, id := range []uuid.UUID{old, justOver, fresh, longCall, neverEnded, neverEndedFresh, otherOld, withRecording} {
		assert.Equal(t, 1, count("call_sessions", "id", id), "Plan is the dry-run path and must not delete")
	}

	// An id from another tenant handed to this tenant's Apply is never deleted,
	// even where RLS would let the statement see it (system context).
	foreign, err := h.Apply(sys, tenantID, []uuid.UUID{otherOld}, models.RetentionActionDelete, "")
	require.NoError(t, err)
	assert.Zero(t, foreign)

	// A recording appearing after Plan must not fail the batch on the FK.
	affected, err := h.Apply(ctx, tenantID, append(append([]uuid.UUID{}, plan.Due...), withRecording), models.RetentionActionDelete, "")
	require.NoError(t, err)
	assert.Equal(t, 3, affected)
	for _, id := range []uuid.UUID{old, justOver, neverEnded} {
		assert.Zero(t, count("call_sessions", "id", id))
		assert.Zero(t, count("call_participants", "call_id", id), "participants go with their session")
	}
	for _, id := range []uuid.UUID{fresh, longCall, neverEndedFresh, otherOld, withRecording} {
		assert.Equal(t, 1, count("call_sessions", "id", id))
		assert.Equal(t, 1, count("call_participants", "call_id", id))
	}

	again, err := h.Apply(ctx, tenantID, plan.Due, models.RetentionActionDelete, "")
	require.NoError(t, err)
	assert.Zero(t, again)
}
