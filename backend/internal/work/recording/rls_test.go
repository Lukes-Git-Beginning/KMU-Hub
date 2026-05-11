package recording_test

import (
	"context"
	"testing"
	"time"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// --- recordings ---

func TestRLS_Recordings_TenantBSeesNoTenantARows(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "tenant-a")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "tenant-b")

	// Need a user (organizer) and a meeting to satisfy FKs.
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         "rec-b-sees-none-u@rls-test.invalid",
		"password_hash": "x",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	meetingID := testutil.SeedRow(t, pool, "meetings", map[string]any{
		"tenant_id":       testutil.TenantA,
		"title":           "rls-test",
		"organizer_id":    userID,
		"status":          "scheduled",
		"scheduled_start": time.Now().UTC(),
		"scheduled_end":   time.Now().UTC().Add(time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "meetings", meetingID)

	id := testutil.SeedRow(t, pool, "recordings", map[string]any{
		"tenant_id":  testutil.TenantA,
		"meeting_id": meetingID,
		"status":     "active",
	})
	defer testutil.CleanupRow(t, pool, "recordings", id)

	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)
	testutil.AssertRowCount(t, pool, ctxB, "recordings", id, 0)
}

func TestRLS_Recordings_TenantASeesOwnRow(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "tenant-a")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         "rec-a-sees-own-u@rls-test.invalid",
		"password_hash": "x",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	meetingID := testutil.SeedRow(t, pool, "meetings", map[string]any{
		"tenant_id":       testutil.TenantA,
		"title":           "rls-test",
		"organizer_id":    userID,
		"status":          "scheduled",
		"scheduled_start": time.Now().UTC(),
		"scheduled_end":   time.Now().UTC().Add(time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "meetings", meetingID)

	id := testutil.SeedRow(t, pool, "recordings", map[string]any{
		"tenant_id":  testutil.TenantA,
		"meeting_id": meetingID,
		"status":     "active",
	})
	defer testutil.CleanupRow(t, pool, "recordings", id)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	testutil.AssertRowCount(t, pool, ctxA, "recordings", id, 1)
}

func TestRLS_Recordings_SystemContextSeesAll(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "tenant-a")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "tenant-b")

	userA := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         "rec-sys-ua@rls-test.invalid",
		"password_hash": "x",
	})
	defer testutil.CleanupRow(t, pool, "users", userA)

	userB := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantB,
		"email":         "rec-sys-ub@rls-test.invalid",
		"password_hash": "x",
	})
	defer testutil.CleanupRow(t, pool, "users", userB)

	now := time.Now().UTC()
	end := now.Add(time.Hour)

	meetA := testutil.SeedRow(t, pool, "meetings", map[string]any{
		"tenant_id": testutil.TenantA, "title": "rls-test",
		"organizer_id": userA, "status": "scheduled",
		"scheduled_start": now, "scheduled_end": end,
	})
	defer testutil.CleanupRow(t, pool, "meetings", meetA)

	meetB := testutil.SeedRow(t, pool, "meetings", map[string]any{
		"tenant_id": testutil.TenantB, "title": "rls-test",
		"organizer_id": userB, "status": "scheduled",
		"scheduled_start": now, "scheduled_end": end,
	})
	defer testutil.CleanupRow(t, pool, "meetings", meetB)

	idA := testutil.SeedRow(t, pool, "recordings", map[string]any{
		"tenant_id": testutil.TenantA, "meeting_id": meetA, "status": "active",
	})
	defer testutil.CleanupRow(t, pool, "recordings", idA)

	idB := testutil.SeedRow(t, pool, "recordings", map[string]any{
		"tenant_id": testutil.TenantB, "meeting_id": meetB, "status": "active",
	})
	defer testutil.CleanupRow(t, pool, "recordings", idB)

	sysCtx := testutil.WithSystemCtx(context.Background())
	testutil.AssertRowCount(t, pool, sysCtx, "recordings", idA, 1)
	testutil.AssertRowCount(t, pool, sysCtx, "recordings", idB, 1)
}

func TestRLS_Recordings_CrossTenantUpdateAffectsZeroRows(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "tenant-a")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "tenant-b")

	userA := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         "rec-cross-upd-u@rls-test.invalid",
		"password_hash": "x",
	})
	defer testutil.CleanupRow(t, pool, "users", userA)

	meetA := testutil.SeedRow(t, pool, "meetings", map[string]any{
		"tenant_id": testutil.TenantA, "title": "rls-test",
		"organizer_id":    userA,
		"status":          "scheduled",
		"scheduled_start": time.Now().UTC(),
		"scheduled_end":   time.Now().UTC().Add(time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "meetings", meetA)

	id := testutil.SeedRow(t, pool, "recordings", map[string]any{
		"tenant_id": testutil.TenantA, "meeting_id": meetA, "status": "active",
	})
	defer testutil.CleanupRow(t, pool, "recordings", id)

	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)
	testutil.AssertUpdateRowsAffected(t, pool, ctxB,
		"UPDATE recordings SET status='failed' WHERE id = $1", 0, id)
}

// --- recording_consents ---

func TestRLS_RecordingConsents_TenantBSeesNoTenantARows(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "tenant-a")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "tenant-b")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         "rc-b-sees-none-u@rls-test.invalid",
		"password_hash": "x",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	meetID := testutil.SeedRow(t, pool, "meetings", map[string]any{
		"tenant_id": testutil.TenantA, "title": "rls-test",
		"organizer_id":    userID,
		"status":          "scheduled",
		"scheduled_start": time.Now().UTC(),
		"scheduled_end":   time.Now().UTC().Add(time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "meetings", meetID)

	recID := testutil.SeedRow(t, pool, "recordings", map[string]any{
		"tenant_id": testutil.TenantA, "meeting_id": meetID, "status": "active",
	})
	defer testutil.CleanupRow(t, pool, "recordings", recID)

	sysCtx := testutil.WithSystemCtx(context.Background())
	if _, err := pool.Exec(sysCtx,
		`INSERT INTO recording_consents (recording_id, user_id, consented, tenant_id)
		 VALUES ($1, $2, true, $3)`,
		recID, userID, testutil.TenantA,
	); err != nil {
		t.Fatalf("seed recording_consents: %v", err)
	}
	defer pool.Exec(sysCtx,
		"DELETE FROM recording_consents WHERE recording_id=$1 AND user_id=$2", recID, userID)

	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)
	var count int
	if err := pool.QueryRow(ctxB,
		"SELECT count(*) FROM recording_consents WHERE recording_id=$1", recID,
	).Scan(&count); err != nil {
		t.Fatalf("count recording_consents: %v", err)
	}
	if count != 0 {
		t.Fatalf("TenantB must not see TenantA recording_consents: got %d", count)
	}
}

func TestRLS_RecordingConsents_TenantASeesOwnRow(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "tenant-a")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         "rc-a-sees-own-u@rls-test.invalid",
		"password_hash": "x",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	meetID := testutil.SeedRow(t, pool, "meetings", map[string]any{
		"tenant_id": testutil.TenantA, "title": "rls-test",
		"organizer_id":    userID,
		"status":          "scheduled",
		"scheduled_start": time.Now().UTC(),
		"scheduled_end":   time.Now().UTC().Add(time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "meetings", meetID)

	recID := testutil.SeedRow(t, pool, "recordings", map[string]any{
		"tenant_id": testutil.TenantA, "meeting_id": meetID, "status": "active",
	})
	defer testutil.CleanupRow(t, pool, "recordings", recID)

	sysCtx := testutil.WithSystemCtx(context.Background())
	if _, err := pool.Exec(sysCtx,
		`INSERT INTO recording_consents (recording_id, user_id, consented, tenant_id)
		 VALUES ($1, $2, true, $3)`,
		recID, userID, testutil.TenantA,
	); err != nil {
		t.Fatalf("seed recording_consents: %v", err)
	}
	defer pool.Exec(sysCtx,
		"DELETE FROM recording_consents WHERE recording_id=$1 AND user_id=$2", recID, userID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	var count int
	if err := pool.QueryRow(ctxA,
		"SELECT count(*) FROM recording_consents WHERE recording_id=$1", recID,
	).Scan(&count); err != nil {
		t.Fatalf("count recording_consents: %v", err)
	}
	if count != 1 {
		t.Fatalf("TenantA must see own recording_consents: got %d", count)
	}
}

func TestRLS_RecordingConsents_SystemContextSeesAll(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "tenant-a")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "tenant-b")

	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()
	end := now.Add(time.Hour)

	userA := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         "rc-sys-ua@rls-test.invalid",
		"password_hash": "x",
	})
	defer testutil.CleanupRow(t, pool, "users", userA)

	userB := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantB,
		"email":         "rc-sys-ub@rls-test.invalid",
		"password_hash": "x",
	})
	defer testutil.CleanupRow(t, pool, "users", userB)

	meetA := testutil.SeedRow(t, pool, "meetings", map[string]any{
		"tenant_id": testutil.TenantA, "title": "rls-test",
		"organizer_id": userA, "status": "scheduled",
		"scheduled_start": now, "scheduled_end": end,
	})
	defer testutil.CleanupRow(t, pool, "meetings", meetA)

	meetB := testutil.SeedRow(t, pool, "meetings", map[string]any{
		"tenant_id": testutil.TenantB, "title": "rls-test",
		"organizer_id": userB, "status": "scheduled",
		"scheduled_start": now, "scheduled_end": end,
	})
	defer testutil.CleanupRow(t, pool, "meetings", meetB)

	recA := testutil.SeedRow(t, pool, "recordings", map[string]any{
		"tenant_id": testutil.TenantA, "meeting_id": meetA, "status": "active",
	})
	defer testutil.CleanupRow(t, pool, "recordings", recA)

	recB := testutil.SeedRow(t, pool, "recordings", map[string]any{
		"tenant_id": testutil.TenantB, "meeting_id": meetB, "status": "active",
	})
	defer testutil.CleanupRow(t, pool, "recordings", recB)

	pool.Exec(sysCtx, `INSERT INTO recording_consents (recording_id, user_id, consented, tenant_id) VALUES ($1,$2,true,$3)`, recA, userA, testutil.TenantA)
	pool.Exec(sysCtx, `INSERT INTO recording_consents (recording_id, user_id, consented, tenant_id) VALUES ($1,$2,true,$3)`, recB, userB, testutil.TenantB)
	defer pool.Exec(sysCtx, "DELETE FROM recording_consents WHERE recording_id=$1", recA)
	defer pool.Exec(sysCtx, "DELETE FROM recording_consents WHERE recording_id=$1", recB)

	var count int
	if err := pool.QueryRow(sysCtx,
		"SELECT count(*) FROM recording_consents WHERE recording_id IN ($1,$2)", recA, recB,
	).Scan(&count); err != nil {
		t.Fatalf("system count: %v", err)
	}
	if count != 2 {
		t.Fatalf("system context must see both tenants' consents: got %d", count)
	}
}
