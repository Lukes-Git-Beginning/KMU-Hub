package gdpr

// Coverage for GuestSessionRetentionHandler (feat-retention-handler-guest-sessions,
// Lauf 12). The cases that matter: a session whose last_activity_at is still
// within the 90-day cutoff is never Due regardless of is_active, an older
// session is Due regardless of is_active, delete is idempotent, and a second
// tenant's expired session never leaks into the first tenant's plan.

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

func TestGuestSessionRetentionHandler_SupportsAction(t *testing.T) {
	t.Parallel()

	handler := NewGuestSessionRetentionHandler(nil)
	assert.True(t, handler.SupportsAction(models.RetentionActionDelete))
	assert.False(t, handler.SupportsAction(models.RetentionActionAnonymize), "a guest session without display_name/email/ip/user_agent has nothing left to anonymize")
	assert.False(t, handler.SupportsAction("retain"))
	assert.Equal(t, "guest_sessions", handler.ResourceType())
	assert.Equal(t, "guest_sessions", handler.Table())
}

func TestGuestSessionRetentionHandler_PlanUsesLastActivityRegardlessOfIsActiveAndIsTenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Guest Session Retention Plan Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	userID := seedExportUser(t, pool, tenantID, "guest-session-retention-plan-user")
	defer testutil.CleanupRow(t, pool, "users", userID)
	channelID := seedRetentionChannel(t, pool, tenantID, userID)
	defer testutil.CleanupRow(t, pool, "channels", channelID)

	otherTenantID := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenantID, "Guest Session Retention Plan Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", otherTenantID)
	otherUserID := seedExportUser(t, pool, otherTenantID, "guest-session-retention-plan-other-user")
	defer testutil.CleanupRow(t, pool, "users", otherUserID)
	otherChannelID := seedRetentionChannel(t, pool, otherTenantID, otherUserID)
	defer testutil.CleanupRow(t, pool, "channels", otherChannelID)

	old := time.Now().UTC().AddDate(0, 0, -120)
	fresh := time.Now().UTC().AddDate(0, 0, -5)

	oldActive := seedGuestSession(t, pool, tenantID, channelID, "old-active", old, true)
	defer testutil.CleanupRow(t, pool, "guest_sessions", oldActive)

	oldInactive := seedGuestSession(t, pool, tenantID, channelID, "old-inactive", old, false)
	defer testutil.CleanupRow(t, pool, "guest_sessions", oldInactive)

	freshActive := seedGuestSession(t, pool, tenantID, channelID, "fresh-active", fresh, true)
	defer testutil.CleanupRow(t, pool, "guest_sessions", freshActive)

	// Another tenant's expired session must never leak into this tenant's plan.
	otherOld := seedGuestSession(t, pool, otherTenantID, otherChannelID, "other-old", old, true)
	defer testutil.CleanupRow(t, pool, "guest_sessions", otherOld)

	handler := NewGuestSessionRetentionHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	cutoff := time.Now().UTC().AddDate(0, 0, -90)

	plan, err := handler.Plan(ctx, tenantID, cutoff, models.RetentionActionDelete)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{oldActive, oldInactive}, plan.Due, "is_active must not gate retention — an active-but-stale session is just as due as an inactive one")
	assert.NotContains(t, plan.Due, freshActive, "a session whose last_activity_at is within the cutoff must not be due")
	assert.NotContains(t, plan.Due, otherOld, "a second tenant's expired session must never appear in this tenant's plan")
}

func TestGuestSessionRetentionHandler_ApplyDeleteIsIdempotent(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Guest Session Retention Apply Delete Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	userID := seedExportUser(t, pool, tenantID, "guest-session-retention-apply-del-user")
	defer testutil.CleanupRow(t, pool, "users", userID)
	channelID := seedRetentionChannel(t, pool, tenantID, userID)
	defer testutil.CleanupRow(t, pool, "channels", channelID)

	old := time.Now().UTC().AddDate(0, 0, -120)
	sessionID := seedGuestSession(t, pool, tenantID, channelID, "apply-delete", old, true)

	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	handler := NewGuestSessionRetentionHandler(pool)

	affected, err := handler.Apply(ctx, tenantID, []uuid.UUID{sessionID}, models.RetentionActionDelete, "")
	require.NoError(t, err)
	assert.Equal(t, 1, affected)

	var count int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM guest_sessions WHERE id = $1`, sessionID).Scan(&count))
	assert.Zero(t, count, "deleted guest session must be gone")

	// A second run over the same id finds nothing left to delete.
	againAffected, err := handler.Apply(ctx, tenantID, []uuid.UUID{sessionID}, models.RetentionActionDelete, "")
	require.NoError(t, err)
	assert.Zero(t, againAffected, "second run must not report affected rows for an already-deleted session")

	cutoff := time.Now().UTC().AddDate(0, 0, -90)
	plan, err := handler.Plan(ctx, tenantID, cutoff, models.RetentionActionDelete)
	require.NoError(t, err)
	assert.NotContains(t, plan.Due, sessionID)
}

func TestGuestSessionRetentionHandler_ApplyUnsupportedAction(t *testing.T) {
	t.Parallel()

	handler := NewGuestSessionRetentionHandler(nil)
	_, err := handler.Apply(context.Background(), uuid.New(), []uuid.UUID{uuid.New()}, models.RetentionActionAnonymize, "")
	require.Error(t, err)
}

// seedGuestSession seeds a minimal guest_sessions row with a given
// last_activity_at and is_active state.
func seedGuestSession(t *testing.T, pool *pgxpool.Pool, tenantID, channelID uuid.UUID, prefix string, lastActivityAt time.Time, isActive bool) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "guest_sessions", map[string]any{
		"tenant_id":        tenantID,
		"channel_id":       channelID,
		"token_hash":       uuid.New().String(),
		"display_name":     prefix + "-guest",
		"email":            prefix + "@retention-test.example",
		"last_activity_at": lastActivityAt,
		"is_active":        isActive,
		"expires_at":       lastActivityAt.AddDate(0, 0, 7),
	})
}
