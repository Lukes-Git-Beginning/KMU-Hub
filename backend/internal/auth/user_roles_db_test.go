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

// Role assignment runs against the real database for one reason above all
// others: user_roles carries neither tenant_id nor an RLS policy (Block B,
// g-user-roles-rls). Its tenant boundary is entirely made of joins against
// users and roles, and only the real policies can prove those joins hold. A
// mock repository would happily "assign" a foreign tenant's role and the test
// would pass.
//
// Own tenants throughout, never the shared testutil.TenantA/B — these tests
// seed users and roles and run beside the rest of the package.
var (
	userRoleTenant        = uuid.MustParse("c0117000-0000-4000-8000-000000000001")
	userRoleForeignTenant = uuid.MustParse("c0117000-0000-4000-8000-000000000002")
	// userRoleActor is the administrator these tests act as — never the account
	// under test, so the self-assignment cap of AssignUserRole stays out of the
	// way here (it has its own tests in guardrails_db_test.go). It wears the
	// admin preset, which also keeps the tenant's administrator count above
	// zero so the last-admin guardrail does not fire on unrelated revokes.
	userRoleActor = uuid.MustParse("c0117000-0000-4000-8000-0000000000a1")
)

func userRoleSetup(t *testing.T) (*pgxpool.Pool, *auth.Service) {
	t.Helper()
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	testutil.EnsureTenant(t, pool, userRoleTenant, "UserRoleTenant")
	testutil.EnsureTenant(t, pool, userRoleForeignTenant, "UserRoleForeignTenant")

	drop := func() {
		_, err := pool.Exec(testutil.WithSystemCtx(context.Background()),
			`DELETE FROM roles WHERE tenant_id = ANY($1)`,
			[]uuid.UUID{userRoleTenant, userRoleForeignTenant})
		require.NoError(t, err)
	}
	drop()
	t.Cleanup(drop)
	seedUserRoleActor(t, pool)

	svc := auth.NewService(
		auth.NewPostgresRepository(pool),
		auth.NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour),
	)
	return pool, svc
}

// seedUserRoleActor seeds the administrator whose id these tests pass as the
// caller, with the admin preset behind it. Plain SQL under system context
// because the id is fixed: the tests name it, and the guardrails read the
// caller's grants out of the database rather than trusting the parameter.
func seedUserRoleActor(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := testutil.WithSystemCtx(context.Background())
	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, tenant_id, email, password_hash, first_name, last_name)
		 VALUES ($1, $2, 'user-role-actor@test.local', 'x', 'User', 'Actor')
		 ON CONFLICT DO NOTHING`, userRoleActor, userRoleTenant)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO user_roles (user_id, role_id)
		 SELECT $1, id FROM roles WHERE name = 'admin' AND tenant_id IS NULL
		 ON CONFLICT DO NOTHING`, userRoleActor)
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userRoleActor) })
}

// userRoleUser seeds an account without any role — the assignment under test
// is what gives it one.
func userRoleUser(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, email string) uuid.UUID {
	t.Helper()
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     tenantID,
		"email":         email,
		"password_hash": "x",
		"first_name":    "User",
		"last_name":     "Roles",
	})
	// user_roles rows go away with the user (ON DELETE CASCADE, migration 000002).
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })
	return userID
}

// userRoleCustom clones a preset into a custom role of the given tenant — the
// kind of role the old oneof validator could never name.
func userRoleCustom(t *testing.T, svc *auth.Service, pool *pgxpool.Pool, tenantID uuid.UUID, name string) uuid.UUID {
	t.Helper()
	role, err := svc.CreateRole(testutil.WithTenantCtx(context.Background(), tenantID), userRoleActor, tenantID,
		auth.CreateRoleInput{Name: name, BasedOn: presetID(t, pool, "extern")})
	require.NoError(t, err)
	return role.ID
}

// TestAssignUserRole_DB_CustomRoleIsAssignable is the point of the unit. Until
// the validator was decoupled, assignRoleRequest only accepted the three seed
// preset names, which made every custom role of wave 1b unassignable — the
// role builder could create roles nobody could ever wear.
func TestAssignUserRole_DB_CustomRoleIsAssignable(t *testing.T) {
	pool, svc := userRoleSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), userRoleTenant)
	userID := userRoleUser(t, pool, userRoleTenant, "assign-custom@test.local")
	roleID := userRoleCustom(t, svc, pool, userRoleTenant, "Lagerleitung")

	roles, err := svc.AssignUserRole(ctx, userRoleActor, userID, roleID)
	require.NoError(t, err)
	assert.Equal(t, []string{roleID.String()}, roles, "the response is the account's new role list")

	again, err := svc.AssignUserRole(ctx, userRoleActor, userID, roleID)
	require.NoError(t, err, "assigning a role the account already holds is a no-op, not a conflict")
	assert.Equal(t, roles, again)
}

// TestAssignUserRole_DB_PresetsStayAssignable is the counterpart: presets are
// immutable, not unassignable. "admin" and "member" are presets, so a preset
// check copied over from UpdateRole would break the most common assignment
// there is.
func TestAssignUserRole_DB_PresetsStayAssignable(t *testing.T) {
	pool, svc := userRoleSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), userRoleTenant)
	userID := userRoleUser(t, pool, userRoleTenant, "assign-preset@test.local")
	member := presetID(t, pool, "member")

	roles, err := svc.AssignUserRole(ctx, userRoleActor, userID, member)
	require.NoError(t, err)
	assert.Equal(t, []string{member.String()}, roles)
}

// TestAssignUserRole_DB_MultipleRolesAccumulate pins the n:m contract the
// frontend states explicitly ("existing roles stay"): assigning a second role
// adds to the first instead of replacing it.
func TestAssignUserRole_DB_MultipleRolesAccumulate(t *testing.T) {
	pool, svc := userRoleSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), userRoleTenant)
	userID := userRoleUser(t, pool, userRoleTenant, "assign-many@test.local")
	first := presetID(t, pool, "member")
	second := userRoleCustom(t, svc, pool, userRoleTenant, "Zweitrolle")

	_, err := svc.AssignUserRole(ctx, userRoleActor, userID, first)
	require.NoError(t, err)
	roles, err := svc.AssignUserRole(ctx, userRoleActor, userID, second)
	require.NoError(t, err)

	assert.ElementsMatch(t, []string{first.String(), second.String()}, roles)
}

// TestAssignUserRole_DB_ForeignTenantUserIsInvisible is one half of the tenant
// boundary this unit has to carry on its own. The target account is resolved
// through users, whose RLS read policy makes a foreign account invisible —
// "not visible" and "does not exist" must answer the same way.
func TestAssignUserRole_DB_ForeignTenantUserIsInvisible(t *testing.T) {
	pool, svc := userRoleSetup(t)
	foreignUser := userRoleUser(t, pool, userRoleForeignTenant, "assign-foreign-user@test.local")
	roleID := userRoleCustom(t, svc, pool, userRoleTenant, "Eigenrolle")

	_, err := svc.AssignUserRole(testutil.WithTenantCtx(context.Background(), userRoleTenant), userRoleActor, foreignUser, roleID)
	assert.ErrorIs(t, err, auth.ErrUserNotFound)

	var assigned int
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT COUNT(*) FROM user_roles WHERE user_id = $1`, foreignUser).Scan(&assigned))
	assert.Zero(t, assigned, "no row may reach the unprotected user_roles table")
}

// TestAssignUserRole_DB_ForeignTenantRoleIsInvisible is the other half: a role
// belonging to another tenant cannot be granted, even to one's own account.
func TestAssignUserRole_DB_ForeignTenantRoleIsInvisible(t *testing.T) {
	pool, svc := userRoleSetup(t)
	userID := userRoleUser(t, pool, userRoleTenant, "assign-foreign-role@test.local")
	foreignRole := userRoleCustom(t, svc, pool, userRoleForeignTenant, "Fremdrolle")

	_, err := svc.AssignUserRole(testutil.WithTenantCtx(context.Background(), userRoleTenant), userRoleActor, userID, foreignRole)
	assert.ErrorIs(t, err, auth.ErrBaseRoleNotFound)

	var assigned int
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT COUNT(*) FROM user_roles WHERE user_id = $1`, userID).Scan(&assigned))
	assert.Zero(t, assigned)
}

// TestAssignUserRole_DB_RepositoryAloneRefusesForeignRole proves the write is
// safe even without the service's pre-checks. The repository is where a later
// caller (a guardrail path, a migration script) might land directly, and
// user_roles has no policy of its own to catch it there.
func TestAssignUserRole_DB_RepositoryAloneRefusesForeignRole(t *testing.T) {
	pool, svc := userRoleSetup(t)
	repo := auth.NewPostgresRepository(pool)
	userID := userRoleUser(t, pool, userRoleTenant, "assign-repo@test.local")
	foreignRole := userRoleCustom(t, svc, pool, userRoleForeignTenant, "Reporolle")

	ctx := testutil.WithTenantCtx(context.Background(), userRoleTenant)
	require.NoError(t, repo.AssignUserRole(ctx, userID, foreignRole),
		"the insert selects zero rows rather than failing — a foreign role is invisible, not rejected")

	var assigned int
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT COUNT(*) FROM user_roles WHERE user_id = $1`, userID).Scan(&assigned))
	assert.Zero(t, assigned, "the users/roles joins are the write's tenant boundary")
}

// TestRevokeUserRole_DB_RemovesOnlyTheNamedRole checks that a revoke leaves
// the rest of the account's roles standing and answers with what is left.
func TestRevokeUserRole_DB_RemovesOnlyTheNamedRole(t *testing.T) {
	pool, svc := userRoleSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), userRoleTenant)
	userID := userRoleUser(t, pool, userRoleTenant, "revoke-one@test.local")
	keep := presetID(t, pool, "member")
	drop := userRoleCustom(t, svc, pool, userRoleTenant, "Wegrolle")

	_, err := svc.AssignUserRole(ctx, userRoleActor, userID, keep)
	require.NoError(t, err)
	_, err = svc.AssignUserRole(ctx, userRoleActor, userID, drop)
	require.NoError(t, err)

	roles, err := svc.RevokeUserRole(ctx, userRoleActor, userID, drop)
	require.NoError(t, err)
	assert.Equal(t, []string{keep.String()}, roles)
}

// TestRevokeUserRole_DB_LastRoleLeavesEmptyList pins the empty case as a list,
// not nil: the gateway renders it as [] for a frontend that maps over it
// unconditionally.
func TestRevokeUserRole_DB_LastRoleLeavesEmptyList(t *testing.T) {
	pool, svc := userRoleSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), userRoleTenant)
	userID := userRoleUser(t, pool, userRoleTenant, "revoke-last@test.local")
	roleID := userRoleCustom(t, svc, pool, userRoleTenant, "Einzelrolle")

	_, err := svc.AssignUserRole(ctx, userRoleActor, userID, roleID)
	require.NoError(t, err)

	roles, err := svc.RevokeUserRole(ctx, userRoleActor, userID, roleID)
	require.NoError(t, err)
	assert.NotNil(t, roles, "an account without roles answers [], never null")
	assert.Empty(t, roles)
}

// TestRevokeUserRole_DB_ForeignTenantUserIsInvisible mirrors the assign side:
// an admin of one tenant cannot strip roles off another tenant's account.
func TestRevokeUserRole_DB_ForeignTenantUserIsInvisible(t *testing.T) {
	pool, svc := userRoleSetup(t)
	foreignUser := userRoleUser(t, pool, userRoleForeignTenant, "revoke-foreign@test.local")
	role := presetID(t, pool, "member")

	_, err := svc.AssignUserRole(testutil.WithTenantCtx(context.Background(), userRoleForeignTenant), userRoleActor, foreignUser, role)
	require.NoError(t, err)

	_, err = svc.RevokeUserRole(testutil.WithTenantCtx(context.Background(), userRoleTenant), userRoleActor, foreignUser, role)
	assert.ErrorIs(t, err, auth.ErrUserNotFound)

	var assigned int
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT COUNT(*) FROM user_roles WHERE user_id = $1`, foreignUser).Scan(&assigned))
	assert.Equal(t, 1, assigned, "the foreign account keeps its role")
}

// TestRevokeUserRole_DB_UnheldRoleIsNoOp: the caller asked for a state, not a
// transition. Answering 404 here would make the builder's "remove" button fail
// on a double click.
func TestRevokeUserRole_DB_UnheldRoleIsNoOp(t *testing.T) {
	pool, svc := userRoleSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), userRoleTenant)
	userID := userRoleUser(t, pool, userRoleTenant, "revoke-unheld@test.local")

	roles, err := svc.RevokeUserRole(ctx, userRoleActor, userID, presetID(t, pool, "member"))
	require.NoError(t, err)
	assert.Empty(t, roles)
}
