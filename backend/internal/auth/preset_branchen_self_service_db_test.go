package auth_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// Migration 000339 gives member and manager the Fuhrpark booking/license/damage:read keys at scope
// "own" (the gateway and the fuhrpark service confine them to the caller's records) and the
// produktion work-step / vermietung inspection read keys at "all". A scope that drifted to "all"
// would open every colleague's bookings and license checks, so the scope is pinned, not just the key.
func TestPresetRoles_DB_MemberAndManagerHoldBranchenSelfServiceKeys(t *testing.T) {
	pool, _ := effPermSetup(t)
	repo := auth.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), effPermTenant)

	own := []string{
		"fuhrpark:booking:read", "fuhrpark:booking:write",
		"fuhrpark:license:read", "fuhrpark:license:write",
		"fuhrpark:damage:read",
	}
	all := []string{"produktion:workstep:read", "vermietung:inspection:read"}

	for _, preset := range []string{"member", "manager"} {
		userID := effPermUser(t, pool, effPermTenant, "branchen-self-"+preset+"@test.local", preset)
		got, err := repo.GetUserPermissions(ctx, userID)
		require.NoError(t, err)
		for _, key := range append(append([]string{}, own...), all...) {
			assert.Contains(t, got, key, "%s preset must hold %s", preset, key)
		}

		for _, key := range append(append([]string{}, own...), all...) {
			want := "all"
			for _, o := range own {
				if o == key {
					want = "own"
				}
			}
			// 000360: member reads the damages of every vehicle; manager stays at own.
			if preset == "member" && key == "fuhrpark:damage:read" {
				want = "all"
			}
			var scope string
			err := pool.QueryRow(testutil.WithSystemCtx(context.Background()), `
				SELECT rp.scope FROM role_permissions rp
				JOIN roles r ON r.id = rp.role_id AND r.tenant_id IS NULL
				JOIN permissions p ON p.id = rp.permission_id
				WHERE r.name = $1 AND p.name = $2`, preset, key).Scan(&scope)
			require.NoError(t, err, "%s/%s", preset, key)
			assert.Equal(t, want, scope, "%s: scope of %s", preset, key)
		}
	}

	// Writing a damage (update/resolve) stays admin-only.
	member := effPermUser(t, pool, effPermTenant, "branchen-self-member2@test.local", "member")
	got, err := repo.GetUserPermissions(ctx, member)
	require.NoError(t, err)
	assert.NotContains(t, got, "fuhrpark:damage:write")
}
