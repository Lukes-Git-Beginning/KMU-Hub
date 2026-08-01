package auth_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// The resolver is only meaningful against the real seed from migration 000256.
// A mock can prove the union rule but not that the query joins the right four
// tables, that the fine/coarse filter matches the seeded key shape, or that the
// roles RLS policy lets a tenant read the tenant_id IS NULL presets at all —
// and that last one is the failure mode that would 403 every user in
// production while looking exactly like "no rights granted".
//
// Own tenants throughout, never the shared testutil.TenantA/B: these tests seed
// per-tenant-unique rows and run beside the rest of the package.
var (
	effPermTenant = uuid.MustParse("e4fec700-0000-4000-8000-000000000001")
	// Only 'extern' would fit a member of another tenant, so the second tenant
	// exists purely to prove preset visibility is not tenant-bound.
	effPermForeignTenant = uuid.MustParse("e4fec700-0000-4000-8000-000000000002")
)

func effPermSetup(t *testing.T) (*pgxpool.Pool, *auth.Service) {
	t.Helper()
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	testutil.EnsureTenant(t, pool, effPermTenant, "EffPermTenant")
	testutil.EnsureTenant(t, pool, effPermForeignTenant, "EffPermForeignTenant")

	svc := auth.NewService(
		auth.NewPostgresRepository(pool),
		auth.NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour),
	)
	return pool, svc
}

// effPermUser creates a user in the given tenant and assigns the named preset
// roles. AssignRole is not used on purpose: it resolves the role by name
// without a tenant condition, and pinning the preset IDs here keeps the test
// honest about which rows it means.
func effPermUser(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, email string, roleNames ...string) uuid.UUID {
	t.Helper()

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     tenantID,
		"email":         email,
		"password_hash": "x",
		"first_name":    "Eff",
		"last_name":     "Perm",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })

	sysCtx := testutil.WithSystemCtx(context.Background())
	for _, name := range roleNames {
		var roleID uuid.UUID
		err := pool.QueryRow(sysCtx,
			`SELECT id FROM roles WHERE name = $1 AND tenant_id IS NULL`, name).Scan(&roleID)
		require.NoErrorf(t, err, "preset role %q missing — migration 000256 not applied?", name)

		_, err = pool.Exec(sysCtx,
			`INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)`, userID, roleID)
		require.NoError(t, err)
	}
	// user_roles rows go away with the user (ON DELETE CASCADE).
	return userID
}

func effPermCapMap(caps []auth.EffectiveCapability) map[string]auth.EffectiveCapability {
	byKey := make(map[string]auth.EffectiveCapability, len(caps))
	for _, c := range caps {
		byKey[c.Key] = c
	}
	return byKey
}

// TestEffectivePermissions_DB_ExternIsExactlyTheCatalogueGrants pins the
// smallest preset: 'extern' carries eleven catalogue keys and nothing else.
func TestEffectivePermissions_DB_ExternIsExactlyTheCatalogueGrants(t *testing.T) {
	pool, svc := effPermSetup(t)
	userID := effPermUser(t, pool, effPermTenant, "eff-extern@test.local", "extern")

	ctx := testutil.WithTenantCtx(context.Background(), effPermTenant)
	got, err := svc.GetEffectivePermissions(ctx, userID)
	require.NoError(t, err)

	require.Len(t, got.Roles, 1)
	assert.Equal(t, "extern", got.Roles[0].Name)
	assert.True(t, got.Roles[0].IsSystem, "a preset carries tenant_id IS NULL")
	assert.NotEmpty(t, got.Roles[0].Color, "the frontend renders the role chip from this")

	assert.Len(t, got.Capabilities, 11, "extern holds exactly the eleven catalogue grants")

	caps := effPermCapMap(got.Capabilities)
	assert.Equal(t, "team", caps["documents:file:read"].Scope)
	assert.Equal(t, "own", caps["work:task:comment"].Scope)
	assert.Equal(t, []string{"extern"}, caps["work:task:comment"].Sources)
}

// TestEffectivePermissions_DB_OnlyCatalogueKeys guards the fine/coarse filter.
// The coarse legacy permissions still back the existing RequirePermission
// gates; leaking one into this response would show the frontend a capability
// its catalogue cannot render.
func TestEffectivePermissions_DB_OnlyCatalogueKeys(t *testing.T) {
	pool, svc := effPermSetup(t)
	userID := effPermUser(t, pool, effPermTenant, "eff-admin@test.local", "admin")

	ctx := testutil.WithTenantCtx(context.Background(), effPermTenant)
	got, err := svc.GetEffectivePermissions(ctx, userID)
	require.NoError(t, err)

	assert.Greater(t, len(got.Capabilities), 250, "admin holds the full catalogue")
	for _, c := range got.Capabilities {
		assert.Equal(t, 2, strings.Count(c.Key, ":"),
			"key %q is not in module:subject:action form — coarse permission leaked through", c.Key)
		assert.Contains(t, []string{"own", "team", "all"}, c.Scope, "key %q", c.Key)
		assert.NotEmpty(t, c.Sources, "key %q has no contributing role", c.Key)
	}
}

// TestEffectivePermissions_DB_UnionAcrossRoles walks the case the union rule
// exists for. extern is a strict subset of member with three keys deliberately
// narrower, so holding both must resolve to member's reach with both roles
// named as sources.
func TestEffectivePermissions_DB_UnionAcrossRoles(t *testing.T) {
	pool, svc := effPermSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), effPermTenant)

	memberOnly := effPermUser(t, pool, effPermTenant, "eff-member@test.local", "member")
	both := effPermUser(t, pool, effPermTenant, "eff-both@test.local", "member", "extern")

	memberGot, err := svc.GetEffectivePermissions(ctx, memberOnly)
	require.NoError(t, err)
	bothGot, err := svc.GetEffectivePermissions(ctx, both)
	require.NoError(t, err)

	require.NotEmpty(t, memberGot.Capabilities, "member resolves to a non-empty set")
	assert.Len(t, bothGot.Capabilities, len(memberGot.Capabilities),
		"extern adds no key member does not already have")

	require.Len(t, bothGot.Roles, 2)
	assert.Equal(t, "extern", bothGot.Roles[0].Name)
	assert.Equal(t, "member", bothGot.Roles[1].Name)

	caps := effPermCapMap(bothGot.Capabilities)
	// extern grants these at team/own, member at all — the wider grant wins,
	// and both roles are credited.
	for _, key := range []string{"documents:file:read", "work:task:read", "work:task:comment"} {
		assert.Equalf(t, "all", caps[key].Scope, "%s: extern must not narrow member's reach", key)
		assert.Equalf(t, []string{"extern", "member"}, caps[key].Sources, "%s", key)
	}
}

// TestEffectivePermissions_DB_PresetsVisibleUnderForeignTenant is the RLS half
// of the wave. Presets live with tenant_id IS NULL; the standard
// enable_tenant_rls policy would hide exactly those rows and resolve every user
// of every tenant to an empty set — indistinguishable from "no rights granted"
// and a 403 for everyone including admin.
func TestEffectivePermissions_DB_PresetsVisibleUnderForeignTenant(t *testing.T) {
	pool, svc := effPermSetup(t)
	userID := effPermUser(t, pool, effPermForeignTenant, "eff-foreign@test.local", "member")

	ctx := testutil.WithTenantCtx(context.Background(), effPermForeignTenant)
	got, err := svc.GetEffectivePermissions(ctx, userID)
	require.NoError(t, err)

	require.NotEmpty(t, got.Roles, "the member preset must be readable from any tenant")
	assert.NotEmpty(t, got.Capabilities,
		"a member of a second tenant resolves to a non-empty capability set")
}

// TestEffectivePermissions_DB_ForeignCustomRoleStaysInvisible is the other half:
// the read policy widens visibility for presets only. A tenant-owned role must
// not bleed into another tenant's resolved rights.
func TestEffectivePermissions_DB_ForeignCustomRoleStaysInvisible(t *testing.T) {
	pool, svc := effPermSetup(t)
	sysCtx := testutil.WithSystemCtx(context.Background())

	customRole := testutil.SeedRow(t, pool, "roles", map[string]any{
		"id":          uuid.New(),
		"tenant_id":   effPermForeignTenant,
		"name":        "eff-perm-custom",
		"description": "tenant-owned role used to prove cross-tenant invisibility",
		"color":       "hsl(0 0% 0%)",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "roles", customRole) })

	// Give the custom role a capability that no preset grants, so its presence
	// in a resolved set would be unmistakable.
	var permID uuid.UUID
	err := pool.QueryRow(sysCtx,
		`SELECT id FROM permissions WHERE name = 'fuhrpark:gps:read'`).Scan(&permID)
	require.NoError(t, err, "seeded catalogue key missing")
	_, err = pool.Exec(sysCtx,
		`INSERT INTO role_permissions (role_id, permission_id, scope) VALUES ($1, $2, 'all')`,
		customRole, permID)
	require.NoError(t, err)

	userID := effPermUser(t, pool, effPermForeignTenant, "eff-custom@test.local", "member")
	_, err = pool.Exec(sysCtx,
		`INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)`, userID, customRole)
	require.NoError(t, err)

	// Resolved by its owner: the custom role and its key are there.
	ownerCtx := testutil.WithTenantCtx(context.Background(), effPermForeignTenant)
	owner, err := svc.GetEffectivePermissions(ownerCtx, userID)
	require.NoError(t, err)
	assert.Contains(t, effPermCapMap(owner.Capabilities), "fuhrpark:gps:read")
	assert.Contains(t, effPermRoleNames(owner.Roles), "eff-perm-custom")

	// Resolved under a different tenant: presets remain, the custom role does
	// not — RLS on roles filters it out before the join ever sees it.
	strangerCtx := testutil.WithTenantCtx(context.Background(), effPermTenant)
	stranger, err := svc.GetEffectivePermissions(strangerCtx, userID)
	require.NoError(t, err)
	assert.NotContains(t, effPermCapMap(stranger.Capabilities), "fuhrpark:gps:read")
	assert.NotContains(t, effPermRoleNames(stranger.Roles), "eff-perm-custom")
	assert.NotEmpty(t, stranger.Roles, "the presets stay visible — two zeroes would be a broken test")
}

func effPermRoleNames(roles []auth.EffectiveRole) []string {
	names := make([]string, 0, len(roles))
	for _, r := range roles {
		names = append(names, r.Name)
	}
	return names
}
