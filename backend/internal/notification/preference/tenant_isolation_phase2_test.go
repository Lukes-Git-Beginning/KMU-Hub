package preference

// Cross-tenant isolation tests for notification tables (Migration 000124).
// notification_preferences, notification_mutes, notification_quiet_hours,
// and notifications all have tenant_id NOT NULL after backfill.
//
// These tables have FK user_id → users(id). We seed a minimal user row in
// system-context so the FK is satisfied without touching production user flows.

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_Notifications(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	// Seed a user for TenantA so FK on notification tables is satisfied.
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         fmt.Sprintf("notify-test-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	// notification_preferences.
	prefID := testutil.SeedRow(t, pool, "notification_preferences", map[string]any{
		"tenant_id": testutil.TenantA,
		"user_id":   userID,
	})
	defer testutil.CleanupRow(t, pool, "notification_preferences", prefID)

	// notification_mutes.
	muteID := testutil.SeedRow(t, pool, "notification_mutes", map[string]any{
		"tenant_id":   testutil.TenantA,
		"user_id":     userID,
		"module_id":   "crm",
		"resource_id": uuid.New().String(),
	})
	defer testutil.CleanupRow(t, pool, "notification_mutes", muteID)

	// notification_quiet_hours (UNIQUE on user_id — only one per user).
	quietID := testutil.SeedRow(t, pool, "notification_quiet_hours", map[string]any{
		"tenant_id": testutil.TenantA,
		"user_id":   userID,
	})
	defer testutil.CleanupRow(t, pool, "notification_quiet_hours", quietID)

	// notifications table (event_type_key, module_id, title are NOT NULL).
	notifID := testutil.SeedRow(t, pool, "notifications", map[string]any{
		"tenant_id":      testutil.TenantA,
		"user_id":        userID,
		"event_type_key": "test.event",
		"module_id":      "test",
		"title":          "Test notification",
	})
	defer testutil.CleanupRow(t, pool, "notifications", notifID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	tests := []struct {
		name  string
		table string
		id    uuid.UUID
	}{
		{"notification_preferences", "notification_preferences", prefID},
		{"notification_mutes", "notification_mutes", muteID},
		{"notification_quiet_hours", "notification_quiet_hours", quietID},
		{"notifications", "notifications", notifID},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			testutil.AssertRowCount(t, pool, ctxA, tc.table, tc.id, 1)
			testutil.AssertRowCount(t, pool, ctxB, tc.table, tc.id, 0)
		})
	}
}
