package preference

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

// TestPostgresRepository_EmailSMSRoundTrip proves the email/sms columns added
// in migration 000304 actually persist through Upsert and come back through
// every read path -- not just that the SELECT list mentions them.
func TestPostgresRepository_EmailSMSRoundTrip(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         fmt.Sprintf("pref-roundtrip-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	eventKey := "chat.mention"

	now := time.Now()
	pref := &models.NotificationPreference{
		ID:           uuid.New(),
		TenantID:     testutil.TenantA,
		UserID:       userID,
		EventTypeKey: &eventKey,
		InApp:        true,
		DesktopPush:  false,
		Email:        false,
		SMS:          true,
		Sound:        "subtle",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	require.NoError(t, repo.UpsertPreference(ctx, pref))
	defer testutil.CleanupRow(t, pool, "notification_preferences", pref.ID)

	got, err := repo.GetEventTypePreference(ctx, testutil.TenantA, userID, eventKey)
	require.NoError(t, err)
	require.False(t, got.Email, "email must round-trip as false, not the column default")
	require.True(t, got.SMS, "sms must round-trip as true")

	// Flip both on the update path (ON CONFLICT branch) and confirm the new
	// values replace the old ones, not just the insert path.
	pref.Email = true
	pref.SMS = false
	require.NoError(t, repo.UpsertPreference(ctx, pref))

	got, err = repo.GetEventTypePreference(ctx, testutil.TenantA, userID, eventKey)
	require.NoError(t, err)
	require.True(t, got.Email, "email must round-trip as true after update")
	require.False(t, got.SMS, "sms must round-trip as false after update")

	list, err := repo.ListPreferences(ctx, testutil.TenantA, userID, nil)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.True(t, list[0].Email)
	require.False(t, list[0].SMS)
}
