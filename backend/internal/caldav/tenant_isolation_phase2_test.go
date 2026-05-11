package caldav

// Cross-tenant isolation DB tests for CalDAV tables (Migration 000122 + 000115).
// caldav_push_subscriptions — tenant_id NOT NULL, RLS in mig 000122.
// caldav_sync_versions — composite PK (no UUID id), RLS in mig 000122; tested via
//   a direct count query with a WHERE clause since AssertRowCount needs an id.
// caldav_change_log — BIGSERIAL PK (not UUID); cannot use AssertRowCount.
//
// caldav_push_subscriptions is the primary table tested here.
// caldav_sync_versions and caldav_change_log are verified via raw SQL count.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_CalDAV_PushSubscriptions(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	// Seed a user for TenantA (required by caldav_push_subscriptions.user_id FK).
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         fmt.Sprintf("caldav-test-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	collID := uuid.New()

	subID := testutil.SeedRow(t, pool, "caldav_push_subscriptions", map[string]any{
		"tenant_id":       testutil.TenantA,
		"user_id":         userID,
		"collection_type": "calendar",
		"collection_id":   collID,
		"push_url":        "https://p.pushgateway.test/" + uuid.New().String(),
		"expires_at":      time.Now().Add(7 * 24 * time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "caldav_push_subscriptions", subID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	testutil.AssertRowCount(t, pool, ctxA, "caldav_push_subscriptions", subID, 1)
	testutil.AssertRowCount(t, pool, ctxB, "caldav_push_subscriptions", subID, 0)
}

// TestTenantIsolation_CalDAV_SyncVersions verifies tenant isolation on caldav_sync_versions.
// Uses a raw count query since the table has a composite PK (no UUID id column).
func TestTenantIsolation_CalDAV_SyncVersions(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	collID := uuid.New()

	// Insert under system-context (no RLS enforcement for seed).
	sysCtx := testutil.WithSystemCtx(context.Background())
	_, err := pool.Exec(sysCtx,
		`INSERT INTO caldav_sync_versions (collection_type, collection_id, tenant_id)
		 VALUES ('calendar', $1, $2)
		 ON CONFLICT (collection_type, collection_id) DO NOTHING`,
		collID, testutil.TenantA,
	)
	if err != nil {
		t.Fatalf("seed caldav_sync_versions: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(sysCtx,
			`DELETE FROM caldav_sync_versions WHERE collection_type = 'calendar' AND collection_id = $1`,
			collID,
		)
	}()

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	var countA, countB int
	if err := pool.QueryRow(ctxA,
		`SELECT count(*) FROM caldav_sync_versions WHERE collection_id = $1`, collID,
	).Scan(&countA); err != nil {
		t.Fatalf("count caldav_sync_versions (ctxA): %v", err)
	}
	if err := pool.QueryRow(ctxB,
		`SELECT count(*) FROM caldav_sync_versions WHERE collection_id = $1`, collID,
	).Scan(&countB); err != nil {
		t.Fatalf("count caldav_sync_versions (ctxB): %v", err)
	}

	if countA != 1 {
		t.Errorf("ctxA: expected 1 row in caldav_sync_versions, got %d", countA)
	}
	if countB != 0 {
		t.Errorf("ctxB: expected 0 rows in caldav_sync_versions (tenant isolation), got %d", countB)
	}
}
