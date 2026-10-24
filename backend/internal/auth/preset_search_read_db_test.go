package auth_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// Migration 000338 gives member and manager search:read, the guard of the header search. The
// gateway narrows each search source to the caller's own read keys (route_search_global.go); this
// pins the seed through the query the login uses to bake the JWT, so a lost grant turns into a
// failing test, not a 403 on every employee's search bar.
func TestPresetRoles_DB_MemberAndManagerHoldSearchRead(t *testing.T) {
	pool, _ := effPermSetup(t)
	repo := auth.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), effPermTenant)

	for _, preset := range []string{"member", "manager", "admin"} {
		userID := effPermUser(t, pool, effPermTenant, "search-read-"+preset+"@test.local", preset)
		got, err := repo.GetUserPermissions(ctx, userID)
		require.NoError(t, err)
		assert.Contains(t, got, "search:read", "%s preset must hold search:read", preset)
	}

	extern := effPermUser(t, pool, effPermTenant, "search-read-extern@test.local", "extern")
	got, err := repo.GetUserPermissions(ctx, extern)
	require.NoError(t, err)
	assert.NotContains(t, got, "search:read", "extern must not hold search:read")
}
