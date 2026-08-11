package preference

// Covers the repository methods around notification_mutes and
// notification_quiet_hours that TestPostgresRepository_EmailSMSRoundTrip does
// not touch: GetModuleDefault, IsResourceMuted, CreateMute, DeleteMute,
// ListMutes, GetQuietHours, UpsertQuietHours. Each gets a happy path plus at
// least one error path (not-found sentinel or a rejected duplicate).

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestPostgresRepository_GetModuleDefault(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         fmt.Sprintf("pref-moduledefault-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), testutil.TenantA)

	// Error path: no module default exists yet.
	_, err := repo.GetModuleDefault(ctx, testutil.TenantA, userID, "crm")
	require.ErrorIs(t, err, ErrPreferenceNotFound)

	now := time.Now()
	moduleID := "crm"
	pref := &models.NotificationPreference{
		ID:          uuid.New(),
		TenantID:    testutil.TenantA,
		UserID:      userID,
		ModuleID:    &moduleID,
		InApp:       true,
		DesktopPush: true,
		Sound:       "default",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	require.NoError(t, repo.UpsertPreference(ctx, pref))
	defer testutil.CleanupRow(t, pool, "notification_preferences", pref.ID)

	got, err := repo.GetModuleDefault(ctx, testutil.TenantA, userID, "crm")
	require.NoError(t, err)
	require.NotNil(t, got.ModuleID)
	require.Equal(t, "crm", *got.ModuleID)
	require.Nil(t, got.EventTypeKey, "module default row must have no event_type_key")

	// A module default for a different module still does not exist.
	_, err = repo.GetModuleDefault(ctx, testutil.TenantA, userID, "biz")
	require.ErrorIs(t, err, ErrPreferenceNotFound)
}

func TestPostgresRepository_MuteLifecycle(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         fmt.Sprintf("pref-mute-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	resourceID := uuid.New().String()

	// Error path: not muted before CreateMute ran.
	muted, err := repo.IsResourceMuted(ctx, testutil.TenantA, userID, "crm", resourceID)
	require.NoError(t, err)
	require.False(t, muted)

	mute := &models.NotificationMute{
		ID:         uuid.New(),
		TenantID:   testutil.TenantA,
		UserID:     userID,
		ModuleID:   "crm",
		ResourceID: resourceID,
		CreatedAt:  time.Now(),
	}
	require.NoError(t, repo.CreateMute(ctx, mute))

	muted, err = repo.IsResourceMuted(ctx, testutil.TenantA, userID, "crm", resourceID)
	require.NoError(t, err)
	require.True(t, muted)

	// A different module for the same resource id must not be muted.
	muted, err = repo.IsResourceMuted(ctx, testutil.TenantA, userID, "work", resourceID)
	require.NoError(t, err)
	require.False(t, muted)

	// Error path: the unique index (user_id, module_id, resource_id) rejects a duplicate.
	dup := &models.NotificationMute{
		ID:         uuid.New(),
		TenantID:   testutil.TenantA,
		UserID:     userID,
		ModuleID:   "crm",
		ResourceID: resourceID,
		CreatedAt:  time.Now(),
	}
	err = repo.CreateMute(ctx, dup)
	require.Error(t, err, "a duplicate (user_id, module_id, resource_id) must be rejected by the unique index")

	list, total, err := repo.ListMutes(ctx, testutil.TenantA, userID, nil, 0, 10)
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, list, 1)
	require.Equal(t, resourceID, list[0].ResourceID)

	// Module filter that matches nothing must not error and must return an empty page.
	otherModule := "work"
	list, total, err = repo.ListMutes(ctx, testutil.TenantA, userID, &otherModule, 0, 10)
	require.NoError(t, err)
	require.Equal(t, 0, total)
	require.Empty(t, list)

	require.NoError(t, repo.DeleteMute(ctx, testutil.TenantA, userID, mute.ID))

	muted, err = repo.IsResourceMuted(ctx, testutil.TenantA, userID, "crm", resourceID)
	require.NoError(t, err)
	require.False(t, muted)

	// Error path: deleting an already-deleted mute returns the not-found sentinel.
	err = repo.DeleteMute(ctx, testutil.TenantA, userID, mute.ID)
	require.ErrorIs(t, err, ErrMuteNotFound)
}

func TestPostgresRepository_QuietHoursRoundTrip(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         fmt.Sprintf("pref-quiethours-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), testutil.TenantA)

	// Error path: nothing configured yet.
	_, err := repo.GetQuietHours(ctx, testutil.TenantA, userID)
	require.ErrorIs(t, err, ErrQuietHoursNotFound)

	now := time.Now()
	qh := &models.QuietHours{
		ID:         uuid.New(),
		TenantID:   testutil.TenantA,
		UserID:     userID,
		StartTime:  "18:00",
		EndTime:    "08:00",
		Timezone:   "Europe/Berlin",
		DaysOfWeek: []int{1, 2, 3, 4, 5},
		Enabled:    true,
		ManualDND:  false,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// Regression pin for the bug fixed by migration 000312: UpsertQuietHours
	// targets `ON CONFLICT (tenant_id, user_id)`, but notification_quiet_hours
	// only carried the inline `UNIQUE(user_id)` from migration 000022 --
	// migration 000124 made tenant_id NOT NULL everywhere but never widened
	// this constraint, unlike idx_notification_preferences_event_type, which
	// got exactly this fix in migration 000305. Postgres validates the ON
	// CONFLICT arbiter at plan time regardless of whether a row conflicts, so
	// EVERY call failed with 42P10 and quiet hours could not be configured at
	// all.
	require.NoError(t, repo.UpsertQuietHours(ctx, qh))
	defer testutil.CleanupRow(t, pool, "notification_quiet_hours", qh.ID)

	got, err := repo.GetQuietHours(ctx, testutil.TenantA, userID)
	require.NoError(t, err)
	require.Equal(t, "18:00", got.StartTime[:5])
	require.True(t, got.Enabled)
	require.Equal(t, []int{1, 2, 3, 4, 5}, got.DaysOfWeek)

	// Second upsert for the same (tenant_id, user_id) must UPDATE, not insert.
	qh.Enabled = false
	qh.StartTime = "22:00"
	qh.DaysOfWeek = []int{6, 7}
	qh.UpdatedAt = time.Now()
	require.NoError(t, repo.UpsertQuietHours(ctx, qh))

	got, err = repo.GetQuietHours(ctx, testutil.TenantA, userID)
	require.NoError(t, err)
	require.Equal(t, "22:00", got.StartTime[:5])
	require.False(t, got.Enabled)
	require.Equal(t, []int{6, 7}, got.DaysOfWeek)

	var rowCount int
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		"SELECT COUNT(*) FROM notification_quiet_hours WHERE user_id = $1", userID).Scan(&rowCount))
	require.Equal(t, 1, rowCount, "second upsert must update the existing row, not insert a second one")
}

// The point of widening the unique index in migration 000312: the old
// UNIQUE(user_id) let the first tenant that configured quiet hours for a user
// block every other tenant from configuring the same user.
func TestPostgresRepository_QuietHoursPerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         fmt.Sprintf("pref-qh-pertenant-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	repo := NewPostgresRepository(pool)
	now := time.Now()

	newQH := func(tenantID uuid.UUID, startTime string, enabled bool) *models.QuietHours {
		return &models.QuietHours{
			ID:         uuid.New(),
			TenantID:   tenantID,
			UserID:     userID,
			StartTime:  startTime,
			EndTime:    "08:00",
			Timezone:   "Europe/Berlin",
			DaysOfWeek: []int{1, 2, 3, 4, 5},
			Enabled:    enabled,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
	}

	qhA := newQH(testutil.TenantA, "18:00", true)
	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	require.NoError(t, repo.UpsertQuietHours(ctxA, qhA))
	defer testutil.CleanupRow(t, pool, "notification_quiet_hours", qhA.ID)

	qhB := newQH(testutil.TenantB, "20:00", false)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)
	require.NoError(t, repo.UpsertQuietHours(ctxB, qhB),
		"a second tenant must be able to configure the same user independently")
	defer testutil.CleanupRow(t, pool, "notification_quiet_hours", qhB.ID)

	gotA, err := repo.GetQuietHours(ctxA, testutil.TenantA, userID)
	require.NoError(t, err)
	require.Equal(t, "18:00", gotA.StartTime[:5])
	require.True(t, gotA.Enabled)

	gotB, err := repo.GetQuietHours(ctxB, testutil.TenantB, userID)
	require.NoError(t, err)
	require.Equal(t, "20:00", gotB.StartTime[:5])
	require.False(t, gotB.Enabled)

	// Tenant isolation still holds: neither tenant sees the other's row.
	_, err = repo.GetQuietHours(ctxA, testutil.TenantB, userID)
	require.ErrorIs(t, err, ErrQuietHoursNotFound)
}

// Module defaults (event_type_key IS NULL) can only be arbitrated by
// idx_notification_preferences_module_default. Before migration 000312 that
// index was scoped (user_id, module_id) and UpsertPreference named the
// event-type arbiter for every row, so a second upsert of the same module
// default failed with 23505 instead of updating.
func TestPostgresRepository_UpsertModuleDefaultTwice(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         fmt.Sprintf("pref-moduleupsert-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	now := time.Now()
	moduleID := "crm"

	pref := &models.NotificationPreference{
		ID:          uuid.New(),
		TenantID:    testutil.TenantA,
		UserID:      userID,
		ModuleID:    &moduleID,
		InApp:       true,
		DesktopPush: true,
		Email:       true,
		Sound:       "default",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	require.NoError(t, repo.UpsertPreference(ctx, pref))
	defer testutil.CleanupRow(t, pool, "notification_preferences", pref.ID)

	// Same module default again, different ID (the service mints a fresh one
	// whenever the caller sends no ID) and different flags.
	second := *pref
	second.ID = uuid.New()
	second.InApp = false
	second.DesktopPush = false
	second.UpdatedAt = time.Now()
	require.NoError(t, repo.UpsertPreference(ctx, &second),
		"second upsert of the same module default must update, not violate the unique index")

	got, err := repo.GetModuleDefault(ctx, testutil.TenantA, userID, moduleID)
	require.NoError(t, err)
	require.Equal(t, pref.ID, got.ID, "the update must keep the original row")
	require.False(t, got.InApp)
	require.False(t, got.DesktopPush)

	var rowCount int
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM notification_preferences WHERE tenant_id = $1 AND user_id = $2 AND module_id = $3 AND event_type_key IS NULL",
		testutil.TenantA, userID, moduleID).Scan(&rowCount))
	require.Equal(t, 1, rowCount)
}
