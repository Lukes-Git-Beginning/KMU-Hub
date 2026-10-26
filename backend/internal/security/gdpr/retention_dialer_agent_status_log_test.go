package gdpr

// Coverage for DialerAgentStatusLogRetentionHandler
// (feat-retention-dialer-agent-status-log-12m, Lauf 17): older than the
// cutoff is due, younger is not, Plan (the dry-run path) deletes nothing,
// Apply is tenant-scoped and idempotent.

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

func TestDialerAgentStatusLogRetentionHandler_Metadata(t *testing.T) {
	t.Parallel()

	h := NewDialerAgentStatusLogRetentionHandler(nil)
	assert.Equal(t, "dialer_agent_status_log", h.ResourceType())
	assert.Equal(t, "dialer_agent_status_log", h.Table())
	assert.Equal(t, "changed_at", h.DateColumn())
	assert.True(t, h.SupportsAction(models.RetentionActionDelete))
	assert.False(t, h.SupportsAction(models.RetentionActionAnonymize))

	_, err := h.Apply(context.Background(), uuid.New(), []uuid.UUID{uuid.New()}, models.RetentionActionAnonymize, "x")
	require.Error(t, err, "must refuse an action it cannot perform instead of deleting")
}

func TestDialerAgentStatusLogRetentionHandler_PlanAndApply(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID, otherTenant := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Status Log Retention Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })
	testutil.EnsureTenant(t, pool, otherTenant, "Status Log Retention Other Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", otherTenant) })
	userID := seedExportUser(t, pool, tenantID, "status-log-retention-user")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })
	otherUser := seedExportUser(t, pool, otherTenant, "status-log-retention-other-user")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", otherUser) })

	now := time.Now().UTC()
	entry := func(tenant, user uuid.UUID, changedAt time.Time) uuid.UUID {
		id := testutil.SeedRow(t, pool, "dialer_agent_status_log", map[string]any{
			"tenant_id": tenant, "user_id": user, "status": "available", "changed_at": changedAt,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "dialer_agent_status_log", id) })
		return id
	}
	old := entry(tenantID, userID, now.AddDate(-2, 0, 0))
	justOver := entry(tenantID, userID, now.AddDate(-1, 0, -1))
	fresh := entry(tenantID, userID, now.AddDate(0, -6, 0))
	otherOld := entry(otherTenant, otherUser, now.AddDate(-2, 0, 0))

	h := NewDialerAgentStatusLogRetentionHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	cutoff := now.AddDate(-1, 0, 0)

	plan, err := h.Plan(ctx, tenantID, cutoff, models.RetentionActionDelete)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{old, justOver}, plan.Due)

	// RLS hides the other tenant above; under the system context only the
	// handler's own tenant filter keeps its old row out of the plan.
	sys := testutil.WithSystemCtx(context.Background())
	sysPlan, err := h.Plan(sys, tenantID, cutoff, models.RetentionActionDelete)
	require.NoError(t, err)
	assert.NotContains(t, sysPlan.Due, otherOld)

	exists := func(id uuid.UUID) bool {
		var n int
		require.NoError(t, pool.QueryRow(sys, `SELECT COUNT(*) FROM dialer_agent_status_log WHERE id = $1`, id).Scan(&n))
		return n > 0
	}
	for _, id := range []uuid.UUID{old, justOver, fresh, otherOld} {
		assert.True(t, exists(id), "Plan is the dry-run path and must not delete")
	}

	// An id from another tenant handed to this tenant's Apply is never deleted,
	// even where RLS would let the statement see it (system context).
	foreign, err := h.Apply(sys, tenantID, []uuid.UUID{otherOld}, models.RetentionActionDelete, "")
	require.NoError(t, err)
	assert.Zero(t, foreign)

	affected, err := h.Apply(ctx, tenantID, plan.Due, models.RetentionActionDelete, "")
	require.NoError(t, err)
	assert.Equal(t, 2, affected)
	assert.False(t, exists(old))
	assert.False(t, exists(justOver))
	assert.True(t, exists(fresh))
	assert.True(t, exists(otherOld), "the other tenant's row must survive")

	again, err := h.Apply(ctx, tenantID, plan.Due, models.RetentionActionDelete, "")
	require.NoError(t, err)
	assert.Zero(t, again)
}
