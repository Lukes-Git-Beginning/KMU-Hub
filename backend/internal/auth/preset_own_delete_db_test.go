package auth_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// Migration 000336 gives member and manager the three delete keys the UI offers on their own
// events, calendars, chat messages and chat files. The services enforce ownership; this pins the
// seed itself, read through the same query the login uses to bake the JWT, so a missing grant
// stays a visible test failure instead of a 403 for every employee.
func TestPresetRoles_DB_MemberAndManagerHoldTheOwnDeleteKeys(t *testing.T) {
	pool, _ := effPermSetup(t)
	repo := auth.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), effPermTenant)

	keys := []string{"calendars:delete", "messages:delete", "files:delete"}

	for _, preset := range []string{"member", "manager", "admin"} {
		userID := effPermUser(t, pool, effPermTenant, "own-delete-"+preset+"@test.local", preset)
		got, err := repo.GetUserPermissions(ctx, userID)
		require.NoError(t, err)
		for _, key := range keys {
			assert.Contains(t, got, key, "%s preset must hold %s", preset, key)
		}
	}

	extern := effPermUser(t, pool, effPermTenant, "own-delete-extern@test.local", "extern")
	got, err := repo.GetUserPermissions(ctx, extern)
	require.NoError(t, err)
	for _, key := range keys {
		assert.NotContains(t, got, key, "extern must not hold %s", key)
	}
}
