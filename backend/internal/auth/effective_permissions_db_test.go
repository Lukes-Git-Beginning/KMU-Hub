package auth_test

import (
	"context"
	"slices"
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
			`INSERT INTO user_roles (user_id, role_id, tenant_id) VALUES ($1, $2, $3)`, userID, roleID, tenantID)
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
	// Sources carries role IDs, not names: the frontend resolves each entry
	// against the Roles slice of this same response to draw the provenance chip.
	assert.Equal(t, []string{got.Roles[0].ID.String()}, caps["work:task:comment"].Sources)
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
	roleIDs := make([]string, len(got.Roles))
	for i, r := range got.Roles {
		roleIDs[i] = r.ID.String()
	}
	for _, c := range got.Capabilities {
		assert.Equal(t, 2, strings.Count(c.Key, ":"),
			"key %q is not in module:subject:action form — coarse permission leaked through", c.Key)
		assert.Contains(t, []string{"own", "team", "all"}, c.Scope, "key %q", c.Key)
		assert.NotEmpty(t, c.Sources, "key %q has no contributing role", c.Key)
		// Every source must resolve against Roles — the frontend looks the chip
		// up there and would silently fall back to the raw string otherwise.
		for _, src := range c.Sources {
			assert.Containsf(t, roleIDs, src, "key %q cites an unknown role", c.Key)
		}
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

	wantSources := []string{bothGot.Roles[0].ID.String(), bothGot.Roles[1].ID.String()}
	slices.Sort(wantSources)

	caps := effPermCapMap(bothGot.Capabilities)
	// extern grants these at team/own, member at all — the wider grant wins,
	// and both roles are credited.
	for _, key := range []string{"documents:file:read", "work:task:read", "work:task:comment"} {
		assert.Equalf(t, "all", caps[key].Scope, "%s: extern must not narrow member's reach", key)
		assert.Equalf(t, wantSources, caps[key].Sources, "%s", key)
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

// TestEffectivePermissions_DB_ForeignCustomRoleStaysInvisible is the other
// half: an owner sees their own custom role and its grants, a stranger tenant
// sees nothing about that account at all — not even the preset it also wears.
// Before migration 000286 user_roles carried no RLS of its own, so a stranger
// resolving a foreign user_id still saw the preset assignment (roles' own
// policy lets tenant_id IS NULL rows through from anywhere); only the custom
// role was hidden. That was never reachable over HTTP even then —
// HandleGetUserPermissions resolves the target through GetUser first, which
// 404s a foreign id before this ever runs — so tightening it here to "nothing
// at all" only removes a repository-level nuance, not a product guarantee.
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
		`INSERT INTO user_roles (user_id, role_id, tenant_id) VALUES ($1, $2, $3)`, userID, customRole, effPermForeignTenant)
	require.NoError(t, err)

	// Resolved by its owner: the custom role and its key are there.
	ownerCtx := testutil.WithTenantCtx(context.Background(), effPermForeignTenant)
	owner, err := svc.GetEffectivePermissions(ownerCtx, userID)
	require.NoError(t, err)
	assert.Contains(t, effPermCapMap(owner.Capabilities), "fuhrpark:gps:read")
	assert.Contains(t, effPermRoleNames(owner.Roles), "eff-perm-custom")

	// Resolved under a different tenant: user_roles' own RLS (migration
	// 000286) hides the account's assignments entirely, presets included —
	// there is nothing left for roles' tenant_id IS NULL carve-out to widen.
	strangerCtx := testutil.WithTenantCtx(context.Background(), effPermTenant)
	stranger, err := svc.GetEffectivePermissions(strangerCtx, userID)
	require.NoError(t, err)
	assert.NotContains(t, effPermCapMap(stranger.Capabilities), "fuhrpark:gps:read")
	assert.NotContains(t, effPermRoleNames(stranger.Roles), "eff-perm-custom")
	assert.Empty(t, stranger.Roles, "a stranger tenant resolves nothing about a foreign account")
}

func effPermRoleNames(roles []auth.EffectiveRole) []string {
	names := make([]string, 0, len(roles))
	for _, r := range roles {
		names = append(names, r.Name)
	}
	return names
}

// ── Per-user overrides folded into the resolution (RBAC R-6) ───────────────
//
// The fixtures below write to user_permission_overrides directly instead of
// going through SetUserOverrides: that path carries four guardrails of its own,
// and a resolver test must not depend on being allowed to write what it
// resolves. The rows disappear with the user (ON DELETE CASCADE).
//
// The keys come out of the catalogue rather than being hardcoded — a preset
// losing a grant should fail this suite honestly instead of leaving it green
// against a key nobody holds any more.

// effPermGrantedKey returns one fine-grained key the preset grants, with the
// scope it grants it at.
func effPermGrantedKey(t *testing.T, pool *pgxpool.Pool, preset string) (string, string) {
	t.Helper()
	var key, scope string
	err := pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT p.name, rp.scope
		 FROM roles r
		 JOIN role_permissions rp ON rp.role_id = r.id
		 JOIN permissions p ON p.id = rp.permission_id
		 WHERE r.name = $1 AND r.tenant_id IS NULL AND p.resource LIKE '%:%'
		 ORDER BY p.name LIMIT 1`, preset).Scan(&key, &scope)
	require.NoErrorf(t, err, "preset %q has no fine-grained grant — migration 000256 not applied?", preset)
	return key, scope
}

// effPermUngrantedKey returns a catalogue key the preset does NOT grant — the
// allow case that has to produce a capability out of nothing.
func effPermUngrantedKey(t *testing.T, pool *pgxpool.Pool, preset string) string {
	t.Helper()
	var key string
	err := pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT p.name FROM permissions p
		 WHERE p.resource LIKE '%:%'
		   AND NOT EXISTS (
		       SELECT 1 FROM role_permissions rp
		       JOIN roles r ON r.id = rp.role_id
		       WHERE rp.permission_id = p.id AND r.name = $1 AND r.tenant_id IS NULL)
		 ORDER BY p.name LIMIT 1`, preset).Scan(&key)
	require.NoErrorf(t, err, "preset %q already grants the whole catalogue", preset)
	return key
}

func effPermSeedOverride(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID, key, mode, scope string) {
	t.Helper()
	_, err := pool.Exec(testutil.WithSystemCtx(context.Background()),
		`INSERT INTO user_permission_overrides (tenant_id, user_id, permission_key, mode, scope)
		 VALUES ($1, $2, $3, $4, $5)`, effPermTenant, userID, key, mode, scope)
	require.NoError(t, err)
}

// TestEffectivePermissions_DB_AccountWithoutOverridesIsUnchanged is the most
// important test of the override seam. Overrides are opt-in and rare; an
// account without any has to come out of the resolver exactly as it did before
// the resolver knew overrides existed — same keys, same scopes, same
// provenance, and neither override field set.
func TestEffectivePermissions_DB_AccountWithoutOverridesIsUnchanged(t *testing.T) {
	pool, svc := effPermSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), effPermTenant)
	userID := effPermUser(t, pool, effPermTenant, "eff-ovr-plain@test.local", "member")

	union, err := svc.GetRoleUnion(ctx, userID)
	require.NoError(t, err)
	effective, err := svc.GetEffectivePermissions(ctx, userID)
	require.NoError(t, err)

	require.NotEmpty(t, effective.Capabilities, "member grants keys — an empty set would prove nothing")
	assert.Equal(t, union, effective, "without overrides the effective set IS the role union")
	assert.False(t, effective.HasOverrides)
	assert.Empty(t, effective.DeniedByOverride)
}

// TestEffectivePermissions_DB_DenyRemovesTheKey: a deny beats the roles however
// many of them grant the key — and the key does not vanish quietly. It moves to
// DeniedByOverride, because "taken from you" and "never had it" are different
// answers and the effective view shows the first one struck through.
func TestEffectivePermissions_DB_DenyRemovesTheKey(t *testing.T) {
	pool, svc := effPermSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), effPermTenant)
	userID := effPermUser(t, pool, effPermTenant, "eff-ovr-deny@test.local", "member")

	key, roleScope := effPermGrantedKey(t, pool, "member")
	before, err := svc.GetEffectivePermissions(ctx, userID)
	require.NoError(t, err)
	inherited, held := effPermCapMap(before.Capabilities)[key]
	require.True(t, held, "fixture key must come from the roles")

	effPermSeedOverride(t, pool, userID, key, "deny", "all")

	after, err := svc.GetEffectivePermissions(ctx, userID)
	require.NoError(t, err)

	assert.NotContains(t, effPermCapMap(after.Capabilities), key, "a denied key leaves the capability set")
	assert.Len(t, after.Capabilities, len(before.Capabilities)-1, "and exactly one key leaves it")
	assert.True(t, after.HasOverrides)
	require.Len(t, after.DeniedByOverride, 1)
	assert.Equal(t, key, after.DeniedByOverride[0].Key)
	assert.Equal(t, roleScope, after.DeniedByOverride[0].RoleScope, "the view keeps what the roles would have given")
	assert.Equal(t, inherited.Sources, after.DeniedByOverride[0].Sources)
}

// TestEffectivePermissions_DB_DenyOfAnUnheldKeyIsNotReported mirrors
// applyUserOverrides: DeniedByOverride lists what was TAKEN, so a deny on a key
// no role granted belongs nowhere — reporting it would show a user a right
// being revoked that they never had.
func TestEffectivePermissions_DB_DenyOfAnUnheldKeyIsNotReported(t *testing.T) {
	pool, svc := effPermSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), effPermTenant)
	userID := effPermUser(t, pool, effPermTenant, "eff-ovr-deny-unheld@test.local", "member")

	key := effPermUngrantedKey(t, pool, "member")
	effPermSeedOverride(t, pool, userID, key, "deny", "all")

	got, err := svc.GetEffectivePermissions(ctx, userID)
	require.NoError(t, err)
	assert.True(t, got.HasOverrides, "the account still carries an override")
	assert.Empty(t, got.DeniedByOverride, "but nothing was taken away")
	assert.NotContains(t, effPermCapMap(got.Capabilities), key)
}

// TestEffectivePermissions_DB_AllowGrantsAndTagsProvenance covers the allow
// half: a key no role gives appears, carrying the override sentinel. Without
// it the effective view would credit a hand-granted right to a role that never
// granted it.
func TestEffectivePermissions_DB_AllowGrantsAndTagsProvenance(t *testing.T) {
	pool, svc := effPermSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), effPermTenant)
	userID := effPermUser(t, pool, effPermTenant, "eff-ovr-allow@test.local", "member")

	key := effPermUngrantedKey(t, pool, "member")
	effPermSeedOverride(t, pool, userID, key, "allow", "team")

	got, err := svc.GetEffectivePermissions(ctx, userID)
	require.NoError(t, err)

	granted, ok := effPermCapMap(got.Capabilities)[key]
	require.True(t, ok, "an allow has to produce the capability")
	assert.Equal(t, "team", granted.Scope)
	assert.Equal(t, []string{auth.OverrideSource}, granted.Sources, "no role contributes, so the sentinel stands alone")
	assert.True(t, got.HasOverrides)
	assert.Empty(t, got.DeniedByOverride)
}

// TestEffectivePermissions_DB_AllowSetsTheScopeAndKeepsRoleProvenance: an allow
// on a key the roles already grant SETS the scope instead of widening it —
// narrowing one person without cloning a role is half of what the feature is
// for. The contributing roles keep their place in Sources, the sentinel goes
// last.
func TestEffectivePermissions_DB_AllowSetsTheScopeAndKeepsRoleProvenance(t *testing.T) {
	pool, svc := effPermSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), effPermTenant)
	userID := effPermUser(t, pool, effPermTenant, "eff-ovr-narrow@test.local", "admin")

	// admin grants at "all" throughout, which is what makes a narrowing
	// override provably a narrowing here.
	key, roleScope := effPermGrantedKey(t, pool, "admin")
	require.Equal(t, "all", roleScope, "fixture assumes the admin preset grants at all")

	before, err := svc.GetEffectivePermissions(ctx, userID)
	require.NoError(t, err)
	inherited := effPermCapMap(before.Capabilities)[key]
	require.NotEmpty(t, inherited.Sources)

	effPermSeedOverride(t, pool, userID, key, "allow", "own")

	after, err := svc.GetEffectivePermissions(ctx, userID)
	require.NoError(t, err)
	narrowed := effPermCapMap(after.Capabilities)[key]
	assert.Equal(t, "own", narrowed.Scope, "the override sets the scope, it does not lose to the wider role scope")
	assert.Equal(t, append(slices.Clone(inherited.Sources), auth.OverrideSource), narrowed.Sources,
		"the roles keep their provenance and the sentinel goes last")
}

// TestGetRoleUnion_DB_IgnoresOverrides backs ?base=1: the override editor needs
// the inherited baseline to show what a deviation deviates from, so this path
// stays blind to the overrides even for an account that has them.
func TestGetRoleUnion_DB_IgnoresOverrides(t *testing.T) {
	pool, svc := effPermSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), effPermTenant)
	userID := effPermUser(t, pool, effPermTenant, "eff-ovr-base@test.local", "member")

	key, _ := effPermGrantedKey(t, pool, "member")
	effPermSeedOverride(t, pool, userID, key, "deny", "all")

	base, err := svc.GetRoleUnion(ctx, userID)
	require.NoError(t, err)
	assert.Contains(t, effPermCapMap(base.Capabilities), key, "the baseline still shows what the roles grant")
	assert.False(t, base.HasOverrides, "the baseline reports no override state at all")
	assert.Empty(t, base.DeniedByOverride)

	effective, err := svc.GetEffectivePermissions(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, effective.Capabilities, len(base.Capabilities)-1, "and the effective set differs from it")
}

// TestResolveTokenPermissions_DB_DenyNarrowsInsteadOfDropping guards the trap
// that decided where the override seam sits. The scopes claim cannot express
// "denied": a key missing from the map reads as "all". So a deny must not
// simply drop the key out of the map — it writes it in at the narrowest scope,
// while the permissions claim is what actually refuses the request.
func TestResolveTokenPermissions_DB_DenyNarrowsInsteadOfDropping(t *testing.T) {
	pool, svc := effPermSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), effPermTenant)
	userID := effPermUser(t, pool, effPermTenant, "eff-ovr-scopes@test.local", "member")

	// A key member holds at a narrowed scope — exactly the entry a deny could
	// wrongly turn into an unrestricted one.
	var key string
	err := pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT p.name
		 FROM roles r
		 JOIN role_permissions rp ON rp.role_id = r.id
		 JOIN permissions p ON p.id = rp.permission_id
		 WHERE r.name = 'member' AND r.tenant_id IS NULL
		   AND p.resource LIKE '%:%' AND rp.scope <> 'all'
		 ORDER BY p.name LIMIT 1`).Scan(&key)
	require.NoError(t, err, "member must carry at least one narrowed grant")

	before, err := svc.ResolveTokenPermissions(ctx, effPermTenant, userID)
	require.NoError(t, err)
	require.Contains(t, before.Scopes, key)
	require.Contains(t, before.Permissions, key)
	require.Empty(t, before.Denied)

	effPermSeedOverride(t, pool, userID, key, "deny", "all")

	after, err := svc.ResolveTokenPermissions(ctx, effPermTenant, userID)
	require.NoError(t, err)
	assert.NotContains(t, after.Permissions, key, "the allow set is what refuses the request")
	assert.Contains(t, after.Denied, key, "the deny list is what stops a coarse key standing in")
	assert.Contains(t, after.Scopes, key, "dropping it here would read as unrestricted, not as denied")
	assert.Equal(t, auth.ScopeOwn, after.Scopes[key])
}

// A deny takes one key away and leaves everything else alone — in particular
// the coarse legacy keys, which are the currency of most RequirePermission
// gates. Rebuilding the allow set from the fine catalogue alone would 403 every
// one of them the moment an account gets its first override.
func TestResolveTokenPermissions_DB_CoarseKeysSurviveAnOverride(t *testing.T) {
	pool, svc := effPermSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), effPermTenant)
	userID := effPermUser(t, pool, effPermTenant, "eff-ovr-coarse@test.local", "member")

	before, err := svc.ResolveTokenPermissions(ctx, effPermTenant, userID)
	require.NoError(t, err)

	coarse := make([]string, 0, len(before.Permissions))
	for _, p := range before.Permissions {
		if strings.Count(p, ":") == 1 {
			coarse = append(coarse, p)
		}
	}
	require.NotEmpty(t, coarse, "member must hold at least one coarse legacy key")

	// Deny a fine key, then check the coarse ones are untouched.
	var fine string
	err = pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT p.name
		 FROM roles r
		 JOIN role_permissions rp ON rp.role_id = r.id
		 JOIN permissions p ON p.id = rp.permission_id
		 WHERE r.name = 'member' AND r.tenant_id IS NULL AND p.resource LIKE '%:%'
		 ORDER BY p.name LIMIT 1`).Scan(&fine)
	require.NoError(t, err)
	effPermSeedOverride(t, pool, userID, fine, "deny", "all")

	after, err := svc.ResolveTokenPermissions(ctx, effPermTenant, userID)
	require.NoError(t, err)
	for _, key := range coarse {
		assert.Contains(t, after.Permissions, key, "coarse key %q must survive", key)
	}
	assert.NotContains(t, after.Permissions, fine)
}

// An allow override puts a key into the token the roles do not grant, at the
// scope the override names rather than at "all".
func TestResolveTokenPermissions_DB_AllowGrantsAndScopes(t *testing.T) {
	pool, svc := effPermSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), effPermTenant)
	userID := effPermUser(t, pool, effPermTenant, "eff-ovr-allow@test.local", "extern")

	before, err := svc.ResolveTokenPermissions(ctx, effPermTenant, userID)
	require.NoError(t, err)

	// A catalogue key the account demonstrably does not hold.
	var key string
	err = pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT p.name FROM permissions p
		 WHERE p.resource LIKE '%:%' AND p.name <> ALL($1)
		 ORDER BY p.name LIMIT 1`, before.Permissions).Scan(&key)
	require.NoError(t, err)

	effPermSeedOverride(t, pool, userID, key, "allow", "team")

	after, err := svc.ResolveTokenPermissions(ctx, effPermTenant, userID)
	require.NoError(t, err)
	assert.Contains(t, after.Permissions, key)
	assert.Equal(t, auth.ScopeTeam, after.Scopes[key], "an allow sets the scope, it does not open it up to all")
	assert.Empty(t, after.Denied)
}

// The login path runs under sysctx.With(), where RLS does not filter. The
// override read therefore has to name its tenant itself — a row belonging to
// another tenant must not reach this account's token.
func TestResolveTokenPermissions_DB_ForeignTenantOverrideIsIgnored(t *testing.T) {
	pool, svc := effPermSetup(t)
	sysCtx := testutil.WithSystemCtx(context.Background())
	userID := effPermUser(t, pool, effPermTenant, "eff-ovr-sysctx@test.local", "member")

	var key string
	err := pool.QueryRow(sysCtx,
		`SELECT p.name
		 FROM roles r
		 JOIN role_permissions rp ON rp.role_id = r.id
		 JOIN permissions p ON p.id = rp.permission_id
		 WHERE r.name = 'member' AND r.tenant_id IS NULL AND p.resource LIKE '%:%'
		 ORDER BY p.name LIMIT 1`).Scan(&key)
	require.NoError(t, err)

	// The row names the foreign tenant while pointing at this account — the
	// exact shape a read without a tenant condition would pick up under sysctx.
	_, err = pool.Exec(sysCtx,
		`INSERT INTO user_permission_overrides (tenant_id, user_id, permission_key, mode, scope)
		 VALUES ($1, $2, $3, 'deny', 'all')`, effPermForeignTenant, userID, key)
	require.NoError(t, err)

	got, err := svc.ResolveTokenPermissions(sysCtx, effPermTenant, userID)
	require.NoError(t, err)
	assert.Empty(t, got.Denied, "an override of another tenant must not reach this token")
	assert.Contains(t, got.Permissions, key)
}
