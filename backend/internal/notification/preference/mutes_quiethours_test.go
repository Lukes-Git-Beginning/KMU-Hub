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

	// REAL BUG, documented not fixed (Coverage-Units aendern kein Verhalten,
	// siehe BACKLOG.yml Kopf): UpsertQuietHours targets
	// `ON CONFLICT (tenant_id, user_id)`, but notification_quiet_hours only
	// carries `UNIQUE(user_id)` (migration 000022) -- migration 000124 made
	// tenant_id NOT NULL everywhere but never widened this constraint, unlike
	// idx_notification_preferences_event_type, which got exactly this fix in
	// migration 000305. Postgres validates the ON CONFLICT target against a
	// real index at plan time regardless of whether a row actually conflicts,
	// so EVERY call fails -- not just the second one -- with 42P10 ("no unique
	// or exclusion constraint matching the ON CONFLICT specification"). Quiet
	// hours can never be configured through this repository at all. Journal
	// entry for this iteration carries the ready fix (widen the unique index
	// the same way 000305 did) for a future Block-A-style unit.
	err = repo.UpsertQuietHours(ctx, qh)
	require.Error(t, err, "documents the current broken state -- UpsertQuietHours must fail until the unique index is widened like migration 000305 did for notification_preferences")
	require.Contains(t, err.Error(), "42P10")

	// Confirms the break is total: GetQuietHours still finds nothing, because
	// the failed upsert above never wrote a row.
	_, err = repo.GetQuietHours(ctx, testutil.TenantA, userID)
	require.ErrorIs(t, err, ErrQuietHoursNotFound)
}
