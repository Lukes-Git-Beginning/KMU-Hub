package gdpr

// Coverage for NotificationRetentionHandler (Lauf 10, eighth handler on the
// retention engine from A10). The cases that matter: only READ notifications
// past the cutoff are Due (an unread one is never due, no matter how old,
// because deleting it unseen is a feature loss, not clean-up), the handler
// only supports delete (anonymize is rejected outright), delete is
// idempotent, and a second tenant's old read notification never appears in
// the first tenant's plan.

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

func TestNotificationRetentionHandler_SupportsAction(t *testing.T) {
	t.Parallel()

	handler := NewNotificationRetentionHandler(nil)
	assert.True(t, handler.SupportsAction(models.RetentionActionDelete))
	assert.False(t, handler.SupportsAction(models.RetentionActionAnonymize), "anonymizing a notification is worthless to its recipient")
	assert.False(t, handler.SupportsAction("retain"))
	assert.Equal(t, "notifications", handler.ResourceType())
	assert.Equal(t, "notifications", handler.Table())
}

func TestNotificationRetentionHandler_PlanOnlyMatchesReadPastCutoffAndIsTenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Notification Retention Plan Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	userID := seedExportUser(t, pool, tenantID, "notification-retention-plan-user")
	defer testutil.CleanupRow(t, pool, "users", userID)

	otherTenantID := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenantID, "Notification Retention Plan Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", otherTenantID)
	otherUserID := seedExportUser(t, pool, otherTenantID, "notification-retention-plan-other-user")
	defer testutil.CleanupRow(t, pool, "users", otherUserID)

	old := time.Now().UTC().AddDate(0, 0, -400)
	fresh := time.Now().UTC().AddDate(0, 0, -2)

	readOld := seedNotification(t, pool, tenantID, userID, "Alt, gelesen", old, true)
	defer testutil.CleanupRow(t, pool, "notifications", readOld)

	readFresh := seedNotification(t, pool, tenantID, userID, "Neu, gelesen", fresh, true)
	defer testutil.CleanupRow(t, pool, "notifications", readFresh)

	// Another tenant's old, read notification must never leak into this
	// tenant's plan.
	otherReadOld := seedNotification(t, pool, otherTenantID, otherUserID, "Fremd, gelesen", old, true)
	defer testutil.CleanupRow(t, pool, "notifications", otherReadOld)

	handler := NewNotificationRetentionHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	cutoff := time.Now().UTC().AddDate(0, 0, -90)

	plan, err := handler.Plan(ctx, tenantID, cutoff, models.RetentionActionDelete)
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{readOld}, plan.Due)
}

func TestNotificationRetentionHandler_PlanExcludesUnreadRegardlessOfAge(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Notification Retention Unread Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	userID := seedExportUser(t, pool, tenantID, "notification-retention-unread-user")
	defer testutil.CleanupRow(t, pool, "users", userID)

	old := time.Now().UTC().AddDate(0, 0, -400)
	unreadOld := seedNotification(t, pool, tenantID, userID, "Alt, ungelesen", old, false)
	defer testutil.CleanupRow(t, pool, "notifications", unreadOld)

	handler := NewNotificationRetentionHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	cutoff := time.Now().UTC().AddDate(0, 0, -90)

	plan, err := handler.Plan(ctx, tenantID, cutoff, models.RetentionActionDelete)
	require.NoError(t, err)
	assert.NotContains(t, plan.Due, unreadOld, "an unread notification must never be due, regardless of its age")
}

func TestNotificationRetentionHandler_ApplyDeleteIsIdempotent(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Notification Retention Apply Delete Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	userID := seedExportUser(t, pool, tenantID, "notification-retention-apply-del-user")
	defer testutil.CleanupRow(t, pool, "users", userID)

	old := time.Now().UTC().AddDate(0, 0, -400)
	notificationID := seedNotification(t, pool, tenantID, userID, "Alt, gelesen", old, true)

	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	handler := NewNotificationRetentionHandler(pool)

	affected, err := handler.Apply(ctx, tenantID, []uuid.UUID{notificationID}, models.RetentionActionDelete, "")
	require.NoError(t, err)
	assert.Equal(t, 1, affected)

	var count int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE id = $1`, notificationID).Scan(&count))
	assert.Zero(t, count, "deleted notification must be gone")

	// A second run over the same id finds nothing left to delete.
	againAffected, err := handler.Apply(ctx, tenantID, []uuid.UUID{notificationID}, models.RetentionActionDelete, "")
	require.NoError(t, err)
	assert.Zero(t, againAffected, "second run must not report affected rows for an already-deleted notification")

	cutoff := time.Now().UTC().AddDate(0, 0, -90)
	plan, err := handler.Plan(ctx, tenantID, cutoff, models.RetentionActionDelete)
	require.NoError(t, err)
	assert.NotContains(t, plan.Due, notificationID)
}

func TestNotificationRetentionHandler_ApplyUnsupportedAction(t *testing.T) {
	t.Parallel()

	handler := NewNotificationRetentionHandler(nil)
	_, err := handler.Apply(context.Background(), uuid.New(), []uuid.UUID{uuid.New()}, models.RetentionActionAnonymize, "")
	require.Error(t, err)
}

// seedNotification seeds a minimal notification with an explicit read state.
func seedNotification(t *testing.T, pool *pgxpool.Pool, tenantID, userID uuid.UUID, title string, createdAt time.Time, isRead bool) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "notifications", map[string]any{
		"tenant_id":      tenantID,
		"user_id":        userID,
		"event_type_key": "retention.test",
		"module_id":      "security",
		"title":          title,
		"created_at":     createdAt,
		"is_read":        isRead,
	})
}
