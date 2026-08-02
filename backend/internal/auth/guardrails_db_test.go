package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// The four guardrails of PHASE-1-RBAC-PLAN §4 all resolve the caller's actual
// rights out of the database — the JWT carries the coarse legacy keys, not the
// fine-grained ones, so none of this can be proven against a mock. What is
// under test here is a property of role_permissions and of the RLS policies
// around it, not of a Go function.
//
// Own tenant, never testutil.TenantA/B: these tests count the administrators
// OF A TENANT, so anything another test seeds beside them would change the
// answer.
var (
	guardTenant        = uuid.MustParse("9a4d0000-0000-4000-8000-000000000001")
	guardForeignTenant = uuid.MustParse("9a4d0000-0000-4000-8000-000000000002")
)

func guardSetup(t *testing.T) (*pgxpool.Pool, *auth.Service) {
	t.Helper()
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	testutil.EnsureTenant(t, pool, guardTenant, "GuardrailTenant")
	testutil.EnsureTenant(t, pool, guardForeignTenant, "GuardrailForeignTenant")
	drop := func() {
		_, err := pool.Exec(testutil.WithSystemCtx(context.Background()),
			`DELETE FROM roles WHERE tenant_id = ANY($1)`,
			[]uuid.UUID{guardTenant, guardForeignTenant})
		require.NoError(t, err)
	}
	drop()
	t.Cleanup(drop)

	svc := auth.NewService(
		auth.NewPostgresRepository(pool),
		auth.NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour),
	)
	return pool, svc
}

// guardUser seeds an account of the guardrail tenant, optionally wearing one
// preset. The role goes in through SQL rather than through AssignUserRole
// because the assignment path is itself under test here — the fixture must not
// depend on the behaviour it sets up.
func guardUser(t *testing.T, pool *pgxpool.Pool, email string, presets ...string) uuid.UUID {
	t.Helper()
	return guardUserIn(t, pool, guardTenant, email, presets...)
}

func guardUserIn(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, email string, presets ...string) uuid.UUID {
	t.Helper()
	ctx := testutil.WithSystemCtx(context.Background())
	userID := uuid.New()
	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, tenant_id, email, password_hash, first_name, last_name)
		 VALUES ($1, $2, $3, 'x', 'Guard', 'User')`, userID, tenantID, email)
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })

	for _, preset := range presets {
		_, err := pool.Exec(ctx,
			`INSERT INTO user_roles (user_id, role_id)
			 SELECT $1, id FROM roles WHERE name = $2 AND tenant_id IS NULL`, userID, preset)
		require.NoErrorf(t, err, "preset %q missing — migration 000256 not applied?", preset)
	}
	return userID
}

// guardRole builds a custom role carrying exactly the given grants, as an
// administrator would through the builder, and hands it to the accounts named.
func guardRole(t *testing.T, pool *pgxpool.Pool, svc *auth.Service, admin uuid.UUID, name string, grants []auth.RoleGrant, wearers ...uuid.UUID) uuid.UUID {
	t.Helper()
	ctx := testutil.WithTenantCtx(context.Background(), guardTenant)

	role, err := svc.CreateRole(ctx, admin, guardTenant, auth.CreateRoleInput{
		Name: name, BasedOn: presetID(t, pool, "extern"),
	})
	require.NoError(t, err)
	_, err = svc.SetRolePermissions(ctx, admin, role.ID, grants)
	require.NoError(t, err)

	for _, wearer := range wearers {
		_, err := pool.Exec(testutil.WithSystemCtx(context.Background()),
			`INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)`, wearer, role.ID)
		require.NoError(t, err)
	}
	return role.ID
}

// TestGuardrail_DB_LastAdminCannotLoseTheRole is (a): a tenant that loses its
// last administrator cannot repair itself — nobody left can hand the right
// back, so this is the one revoke that has to be refused outright.
func TestGuardrail_DB_LastAdminCannotLoseTheRole(t *testing.T) {
	pool, svc := guardSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), guardTenant)
	admin := guardUser(t, pool, "guard-last-admin@test.local", "admin")
	// Another tenant's administrator must not count. user_roles has no RLS of
	// its own, so the count is only tenant-scoped through its users join —
	// without that join this revoke would look perfectly safe.
	guardUserIn(t, pool, guardForeignTenant, "guard-foreign-admin@test.local", "admin")

	_, err := svc.RevokeUserRole(ctx, admin, admin, presetID(t, pool, "admin"))
	assert.ErrorIs(t, err, auth.ErrLastAdmin)
	assert.Equal(t, "last_admin", auth.ErrLastAdmin.Error(),
		"the message is the frontend's error code, not prose")

	var held int
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT COUNT(*) FROM user_roles WHERE user_id = $1`, admin).Scan(&held))
	assert.Equal(t, 1, held, "the refused revoke must not have touched the row")
}

// TestGuardrail_DB_SecondAdminMakesTheRevokePossible is the counterpart: the
// rule is about the tenant keeping an administrator, not about the admin
// preset being unrevokable.
func TestGuardrail_DB_SecondAdminMakesTheRevokePossible(t *testing.T) {
	pool, svc := guardSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), guardTenant)
	keeper := guardUser(t, pool, "guard-keeper@test.local", "admin")
	leaving := guardUser(t, pool, "guard-leaving@test.local", "admin")

	roles, err := svc.RevokeUserRole(ctx, keeper, leaving, presetID(t, pool, "admin"))
	require.NoError(t, err)
	assert.Empty(t, roles)
}

// TestGuardrail_DB_AdminCannotRevokeTheirOwnAdminRole is (b). It is not
// covered by (a): here a second administrator exists, so the tenant survives —
// what does not survive is the caller's own access, and undoing it would take
// the other administrator.
func TestGuardrail_DB_AdminCannotRevokeTheirOwnAdminRole(t *testing.T) {
	pool, svc := guardSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), guardTenant)
	actor := guardUser(t, pool, "guard-self@test.local", "admin")
	guardUser(t, pool, "guard-other-admin@test.local", "admin")

	_, err := svc.RevokeUserRole(ctx, actor, actor, presetID(t, pool, "admin"))
	assert.ErrorIs(t, err, auth.ErrSelfLockout)
}

// TestGuardrail_DB_RevokeOfAnUnrelatedRoleIsUntouched keeps the guardrails
// narrow: they only look at roles that actually carry role administration.
// Running the admin count on every revoke would be worse than useless — a
// tenant whose administrators are all deactivated would count zero, and every
// unrelated revoke would start failing with last_admin.
func TestGuardrail_DB_RevokeOfAnUnrelatedRoleIsUntouched(t *testing.T) {
	pool, svc := guardSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), guardTenant)
	actor := guardUser(t, pool, "guard-plain-actor@test.local", "admin")
	member := guardUser(t, pool, "guard-plain-member@test.local", "member")

	roles, err := svc.RevokeUserRole(ctx, actor, member, presetID(t, pool, "member"))
	require.NoError(t, err)
	assert.Empty(t, roles)
}

// TestGuardrail_DB_GrantSetCannotExceedTheCaller is (c) on the path that
// defines what a role may do. it_admin holds no finance capability, so it
// cannot put one into a role either — the escalation would otherwise be one
// assignment away.
func TestGuardrail_DB_GrantSetCannotExceedTheCaller(t *testing.T) {
	pool, svc := guardSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), guardTenant)
	admin := guardUser(t, pool, "guard-grant-admin@test.local", "admin")
	itAdmin := guardUser(t, pool, "guard-it-admin@test.local", "it_admin")

	role, err := svc.CreateRole(ctx, admin, guardTenant, auth.CreateRoleInput{
		Name: "Zielrolle", BasedOn: presetID(t, pool, "extern"),
	})
	require.NoError(t, err)

	_, err = svc.SetRolePermissions(ctx, itAdmin, role.ID, []auth.RoleGrant{
		{Key: "finance:invoice:read", Scope: "all"},
	})
	assert.ErrorIs(t, err, auth.ErrPrivilegeEscalation,
		"it_admin has no finance right, so it cannot hand one out")

	_, err = svc.SetRolePermissions(ctx, itAdmin, role.ID, []auth.RoleGrant{
		{Key: "work:task:read", Scope: "all"},
	})
	assert.NoError(t, err, "a key it_admin does hold goes through")
}

// TestGuardrail_DB_ScopeCannotBeWidenedPastTheCaller is the scope half of (c).
// Key coverage alone is not enough: handing out "all" while holding "own"
// widens the reach of a right the caller only has a slice of.
func TestGuardrail_DB_ScopeCannotBeWidenedPastTheCaller(t *testing.T) {
	pool, svc := guardSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), guardTenant)
	admin := guardUser(t, pool, "guard-scope-admin@test.local", "admin")
	narrow := guardUser(t, pool, "guard-scope-narrow@test.local")

	guardRole(t, pool, svc, admin, "Nur eigene Aufgaben",
		[]auth.RoleGrant{{Key: "work:task:read", Scope: "own"}}, narrow)
	target := guardRole(t, pool, svc, admin, "Aufgabenrolle",
		[]auth.RoleGrant{{Key: "work:task:read", Scope: "own"}})

	_, err := svc.SetRolePermissions(ctx, narrow, target, []auth.RoleGrant{
		{Key: "work:task:read", Scope: "all"},
	})
	assert.ErrorIs(t, err, auth.ErrPrivilegeEscalation, "own may not become all")

	_, err = svc.SetRolePermissions(ctx, narrow, target, []auth.RoleGrant{
		{Key: "work:task:read", Scope: "own"},
	})
	assert.NoError(t, err, "the same scope is within reach")
}

// TestGuardrail_DB_CloneCannotExceedTheCaller closes the way around the grant
// set check: a clone copies its source's grants, so cloning a richer role
// would produce exactly the role the previous test refuses to configure.
//
// The consequence is deliberate and worth stating: only a caller whose own
// reach covers a preset can clone that preset. it_admin cannot even clone
// "extern", which carries documents:file:upload/download it does not hold —
// the builder's clone list is effectively "what you could have configured
// yourself", and in practice only the admin preset covers all seven.
func TestGuardrail_DB_CloneCannotExceedTheCaller(t *testing.T) {
	pool, svc := guardSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), guardTenant)
	admin := guardUser(t, pool, "guard-clone-admin@test.local", "admin")
	itAdmin := guardUser(t, pool, "guard-clone-it@test.local", "it_admin")

	_, err := svc.CreateRole(ctx, itAdmin, guardTenant, auth.CreateRoleInput{
		Name: "Adminklon", BasedOn: presetID(t, pool, "admin"),
	})
	assert.ErrorIs(t, err, auth.ErrPrivilegeEscalation)

	withinReach := guardRole(t, pool, svc, admin, "IT-Basis",
		[]auth.RoleGrant{{Key: "work:task:read", Scope: "all"}})
	_, err = svc.CreateRole(ctx, itAdmin, guardTenant, auth.CreateRoleInput{
		Name: "IT-Basisklon", BasedOn: withinReach,
	})
	assert.NoError(t, err, "a source within it_admin's own reach clones fine")
}

// TestGuardrail_DB_SelfAssignmentIsCappedButDelegationIsNot pins the deliberate
// asymmetry of AssignUserRole. Staffing a role richer than one's own is the
// point of hr_admin (admin:role:assign without admin:role:create/edit);
// TAKING it is the direct route from "may assign roles" to "holds everything".
func TestGuardrail_DB_SelfAssignmentIsCappedButDelegationIsNot(t *testing.T) {
	pool, svc := guardSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), guardTenant)
	hrAdmin := guardUser(t, pool, "guard-hr@test.local", "hr_admin")
	colleague := guardUser(t, pool, "guard-colleague@test.local")
	adminPreset := presetID(t, pool, "admin")

	_, err := svc.AssignUserRole(ctx, hrAdmin, hrAdmin, adminPreset)
	assert.ErrorIs(t, err, auth.ErrPrivilegeEscalation, "nobody promotes themselves")

	roles, err := svc.AssignUserRole(ctx, hrAdmin, colleague, adminPreset)
	require.NoError(t, err, "delegating an existing role stays allowed")
	assert.Equal(t, []string{adminPreset.String()}, roles)
}

// TestGuardrail_DB_CallerCannotEditTheirOwnAdminRightAway is (b) on the grant
// set path. Revoking is not the only way to lock yourself out: unticking the
// role administration capability of the role you administer through does the
// same thing, and in a builder full of checkboxes it is the easier mistake.
func TestGuardrail_DB_CallerCannotEditTheirOwnAdminRightAway(t *testing.T) {
	pool, svc := guardSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), guardTenant)
	admin := guardUser(t, pool, "guard-edit-admin@test.local", "admin")
	actor := guardUser(t, pool, "guard-edit-actor@test.local")

	role := guardRole(t, pool, svc, admin, "Rollenverwaltung", []auth.RoleGrant{
		{Key: "admin:role:assign", Scope: "all"},
		{Key: "work:task:read", Scope: "all"},
	}, actor)

	_, err := svc.SetRolePermissions(ctx, actor, role, []auth.RoleGrant{
		{Key: "work:task:read", Scope: "all"},
	})
	assert.ErrorIs(t, err, auth.ErrSelfLockout)

	_, err = svc.SetRolePermissions(ctx, actor, role, []auth.RoleGrant{
		{Key: "admin:role:assign", Scope: "all"},
	})
	assert.NoError(t, err, "dropping other capabilities is fine as long as the administration survives")
}

// TestGuardrail_DB_PresetsStayImmutableOnEveryWritePath is (d). The three write
// paths each have their own preset check rather than one shared one, so this
// walks all three — a preset that became editable through the permissions
// route would rewrite every tenant's rights at once, since presets are shared.
func TestGuardrail_DB_PresetsStayImmutableOnEveryWritePath(t *testing.T) {
	pool, svc := guardSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), guardTenant)
	admin := guardUser(t, pool, "guard-preset-admin@test.local", "admin")
	preset := presetID(t, pool, "member")
	name := "Mitglied"

	_, err := svc.UpdateRole(ctx, preset, auth.UpdateRoleInput{Name: &name})
	assert.ErrorIs(t, err, auth.ErrRolePresetImmutable)

	err = svc.DeleteRole(ctx, preset)
	assert.ErrorIs(t, err, auth.ErrRolePresetImmutable)

	_, err = svc.SetRolePermissions(ctx, admin, preset, []auth.RoleGrant{
		{Key: "work:task:read", Scope: "all"},
	})
	assert.ErrorIs(t, err, auth.ErrRolePresetImmutable,
		"even an admin whose reach covers the grant may not touch a preset")
}
