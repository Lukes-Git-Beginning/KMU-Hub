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

// Per-user overrides run against the real database for the same reason the
// guardrails do: everything worth proving here is a schema property. The
// catalogue check reads the permissions table, the escalation guard resolves
// the caller's own fine-grained grants, the last-admin count asks which
// accounts of THIS tenant still administer roles, and the RLS policy is what
// keeps a foreign tenant's overrides invisible. A mock repository would prove
// none of it.
//
// Own tenants, never testutil.TenantA/B: these tests count the administrators
// of a tenant, so a row another test seeds beside them would change the
// answer.
var (
	ovrTenant        = uuid.MustParse("6b0e0000-0000-4000-8000-000000000001")
	ovrForeignTenant = uuid.MustParse("6b0e0000-0000-4000-8000-000000000002")
)

func ovrSetup(t *testing.T) (*pgxpool.Pool, *auth.Service) {
	t.Helper()
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	testutil.EnsureTenant(t, pool, ovrTenant, "OverrideTenant")
	testutil.EnsureTenant(t, pool, ovrForeignTenant, "OverrideForeignTenant")
	drop := func() {
		ctx := testutil.WithSystemCtx(context.Background())
		_, err := pool.Exec(ctx, `DELETE FROM user_permission_overrides WHERE tenant_id = ANY($1)`,
			[]uuid.UUID{ovrTenant, ovrForeignTenant})
		require.NoError(t, err)
		_, err = pool.Exec(ctx, `DELETE FROM roles WHERE tenant_id = ANY($1)`,
			[]uuid.UUID{ovrTenant, ovrForeignTenant})
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

// ovrUser seeds an account of the override tenant wearing the given presets.
// SQL rather than AssignUserRole: the fixture must not depend on the
// assignment path, which has guardrails of its own.
func ovrUser(t *testing.T, pool *pgxpool.Pool, email string, presets ...string) uuid.UUID {
	t.Helper()
	return ovrUserIn(t, pool, ovrTenant, email, presets...)
}

func ovrUserIn(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, email string, presets ...string) uuid.UUID {
	t.Helper()
	ctx := testutil.WithSystemCtx(context.Background())
	userID := uuid.New()
	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, tenant_id, email, password_hash, first_name, last_name)
		 VALUES ($1, $2, $3, 'x', 'Override', 'User')`, userID, tenantID, email)
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })

	for _, preset := range presets {
		_, err := pool.Exec(ctx,
			`INSERT INTO user_roles (user_id, role_id, tenant_id)
			 SELECT $1, id, $3 FROM roles WHERE name = $2 AND tenant_id IS NULL`, userID, preset, tenantID)
		require.NoErrorf(t, err, "preset %q missing — migration 000256 not applied?", preset)
	}
	return userID
}

func ovrMap(overrides []auth.CapabilityOverride) map[string]auth.CapabilityOverride {
	out := make(map[string]auth.CapabilityOverride, len(overrides))
	for _, o := range overrides {
		out[o.Key] = o
	}
	return out
}

// TestSetUserOverrides_DB_StoresAndReplacesTheWholeMap is the core of the
// unit. The PUT is a full replace, so the second call has to delete the key
// the first one wrote — a merge would leave a deviation the editor believes it
// removed, and nothing in the UI would show it again.
func TestSetUserOverrides_DB_StoresAndReplacesTheWholeMap(t *testing.T) {
	pool, svc := ovrSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), ovrTenant)
	admin := ovrUser(t, pool, "ovr-admin-replace@test.local", "admin")
	target := ovrUser(t, pool, "ovr-target-replace@test.local", "member")

	stored, changes, err := svc.SetUserOverrides(ctx, admin, ovrTenant, target, []auth.CapabilityOverride{
		{Key: "work:project:edit", Mode: "allow", Scope: "all"},
		{Key: "rapporte:report:create", Mode: "deny", Scope: "all"},
	})
	require.NoError(t, err)
	require.Len(t, stored, 2)
	assert.Len(t, changes, 2, "both keys are new, so both are audited")

	byKey := ovrMap(stored)
	assert.Equal(t, "allow", byKey["work:project:edit"].Mode)
	assert.Equal(t, "all", byKey["work:project:edit"].Scope)
	assert.Equal(t, "deny", byKey["rapporte:report:create"].Mode)

	// Second write names only one of the two keys — the other has to go.
	stored, changes, err = svc.SetUserOverrides(ctx, admin, ovrTenant, target, []auth.CapabilityOverride{
		{Key: "work:project:edit", Mode: "allow", Scope: "team"},
	})
	require.NoError(t, err)
	require.Len(t, stored, 1)
	assert.Equal(t, "team", stored[0].Scope, "the scope change has to land")
	assert.Len(t, changes, 2, "one key changed scope, one disappeared")

	// And an empty map clears everything.
	stored, changes, err = svc.SetUserOverrides(ctx, admin, ovrTenant, target, nil)
	require.NoError(t, err)
	assert.Empty(t, stored)
	assert.Len(t, changes, 1)
	assert.Nil(t, changes[0].After, "a dropped override reads as back-to-inherited")
}

// TestSetUserOverrides_DB_UnchangedMapProducesNoChanges keeps the audit log a
// history of decisions rather than of save clicks: re-sending the identical
// map is a legitimate no-op and must not write events.
func TestSetUserOverrides_DB_UnchangedMapProducesNoChanges(t *testing.T) {
	pool, svc := ovrSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), ovrTenant)
	admin := ovrUser(t, pool, "ovr-admin-noop@test.local", "admin")
	target := ovrUser(t, pool, "ovr-target-noop@test.local", "member")

	same := []auth.CapabilityOverride{{Key: "work:project:edit", Mode: "allow", Scope: "all"}}
	_, _, err := svc.SetUserOverrides(ctx, admin, ovrTenant, target, same)
	require.NoError(t, err)

	_, changes, err := svc.SetUserOverrides(ctx, admin, ovrTenant, target, same)
	require.NoError(t, err)
	assert.Empty(t, changes)
}

// TestSetUserOverrides_DB_UnknownKeyIs422 is the reason the catalogue check
// exists: without it the editor could save a right that nothing in the system
// ever checks for, and the admin would believe the person has it.
func TestSetUserOverrides_DB_UnknownKeyIs422(t *testing.T) {
	pool, svc := ovrSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), ovrTenant)
	admin := ovrUser(t, pool, "ovr-admin-unknown@test.local", "admin")
	target := ovrUser(t, pool, "ovr-target-unknown@test.local", "member")

	_, _, err := svc.SetUserOverrides(ctx, admin, ovrTenant, target, []auth.CapabilityOverride{
		{Key: "work:project:edit", Mode: "allow", Scope: "all"},
		{Key: "work:project:teleport", Mode: "allow", Scope: "all"},
	})
	assert.ErrorIs(t, err, auth.ErrCapabilityKeyUnknown)

	// The whole write aborts — the valid key beside the typo must not land.
	current, err := svc.GetUserOverrides(ctx, target)
	require.NoError(t, err)
	assert.Empty(t, current)
}

// TestSetUserOverrides_DB_BadModeOrScopeIs422 covers the two other spellings a
// caller can get wrong. Both answer the same code as an unknown key, which is
// the call SetRolePermissions already made for an out-of-range scope: a
// sentinel rbac-format.ts does not know renders as "Unbekannter Fehler".
func TestSetUserOverrides_DB_BadModeOrScopeIs422(t *testing.T) {
	pool, svc := ovrSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), ovrTenant)
	admin := ovrUser(t, pool, "ovr-admin-badmode@test.local", "admin")
	target := ovrUser(t, pool, "ovr-target-badmode@test.local", "member")

	_, _, err := svc.SetUserOverrides(ctx, admin, ovrTenant, target, []auth.CapabilityOverride{
		{Key: "work:project:edit", Mode: "maybe", Scope: "all"},
	})
	assert.ErrorIs(t, err, auth.ErrCapabilityKeyUnknown)

	_, _, err = svc.SetUserOverrides(ctx, admin, ovrTenant, target, []auth.CapabilityOverride{
		{Key: "work:project:edit", Mode: "allow", Scope: "everything"},
	})
	assert.ErrorIs(t, err, auth.ErrCapabilityKeyUnknown)

	// A deny carries no meaningful scope, so leaving it out is not a mistake —
	// it stores the column default instead of failing.
	stored, _, err := svc.SetUserOverrides(ctx, admin, ovrTenant, target, []auth.CapabilityOverride{
		{Key: "work:project:edit", Mode: "deny"},
	})
	require.NoError(t, err)
	require.Len(t, stored, 1)
	assert.Equal(t, "all", stored[0].Scope)
}

// TestSetUserOverrides_DB_EscalationIsBlocked is the guardrail that makes the
// feature safe to delegate at all. hr_admin holds no finance capability, so it
// must not be able to hand one to somebody through an override either — that
// would be the same escalation the role builder already refuses, through a
// different door.
func TestSetUserOverrides_DB_EscalationIsBlocked(t *testing.T) {
	pool, svc := ovrSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), ovrTenant)
	hrAdmin := ovrUser(t, pool, "ovr-hr-admin@test.local", "hr_admin")
	target := ovrUser(t, pool, "ovr-target-escalation@test.local", "member")

	_, _, err := svc.SetUserOverrides(ctx, hrAdmin, ovrTenant, target, []auth.CapabilityOverride{
		{Key: "finance:invoice:send", Mode: "allow", Scope: "all"},
	})
	assert.ErrorIs(t, err, auth.ErrPrivilegeEscalation)

	// Taking a right AWAY that the caller does not hold is not escalation:
	// it enlarges nobody, and an administrator has to be able to clean up
	// after a colleague with a different role.
	_, _, err = svc.SetUserOverrides(ctx, hrAdmin, ovrTenant, target, []auth.CapabilityOverride{
		{Key: "finance:invoice:send", Mode: "deny", Scope: "all"},
	})
	assert.NoError(t, err)
}

// TestSetUserOverrides_DB_ScopeCannotBeWidenedPastTheCaller is the second half
// of the escalation rule. manager holds team:employee:read at scope team, so
// it may hand that key on — but not at scope all.
func TestSetUserOverrides_DB_ScopeCannotBeWidenedPastTheCaller(t *testing.T) {
	pool, svc := ovrSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), ovrTenant)
	manager := ovrUser(t, pool, "ovr-manager@test.local", "manager")
	target := ovrUser(t, pool, "ovr-target-scope@test.local", "member")

	_, _, err := svc.SetUserOverrides(ctx, manager, ovrTenant, target, []auth.CapabilityOverride{
		{Key: "team:employee:read", Mode: "allow", Scope: "all"},
	})
	assert.ErrorIs(t, err, auth.ErrPrivilegeEscalation)

	stored, _, err := svc.SetUserOverrides(ctx, manager, ovrTenant, target, []auth.CapabilityOverride{
		{Key: "team:employee:read", Mode: "allow", Scope: "team"},
	})
	require.NoError(t, err)
	require.Len(t, stored, 1)
}

// TestSetUserOverrides_DB_SelfEditIsBlocked: the caller's own account is off
// limits, in both directions. An allow on oneself is the direct route from
// "may fine-tune others" to "has every right", and a deny is the classic
// self-lockout — R6 briefing §3 names both.
func TestSetUserOverrides_DB_SelfEditIsBlocked(t *testing.T) {
	pool, svc := ovrSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), ovrTenant)
	admin := ovrUser(t, pool, "ovr-admin-self@test.local", "admin")
	// A second administrator exists, so this is not last_admin — it is about
	// the caller's own account being untouchable by their own hand.
	ovrUser(t, pool, "ovr-admin-self-other@test.local", "admin")

	_, _, err := svc.SetUserOverrides(ctx, admin, ovrTenant, admin, []auth.CapabilityOverride{
		{Key: "work:project:edit", Mode: "allow", Scope: "all"},
	})
	assert.ErrorIs(t, err, auth.ErrSelfLockout)

	_, err = svc.ClearUserOverrides(ctx, admin, admin)
	assert.ErrorIs(t, err, auth.ErrSelfLockout)
}

// TestSetUserOverrides_DB_LastAdminCannotBeDenied is the guardrail whose
// failure is unrecoverable: a tenant that loses its last role administrator
// cannot hand the right back to itself.
func TestSetUserOverrides_DB_LastAdminCannotBeDenied(t *testing.T) {
	pool, svc := ovrSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), ovrTenant)
	// The caller wears `manager`, the widest preset that holds NEITHER
	// roles:manage nor admin:role:assign — admin, it_admin and hr_admin all
	// carry the latter (migration 000256), so each of them would keep the
	// count above zero and the guardrail could never be reached. A deny needs
	// no reach of its own, which is what lets a non-administrator trigger it.
	actor := ovrUser(t, pool, "ovr-actor-lastadmin@test.local", "manager")
	target := ovrUser(t, pool, "ovr-target-lastadmin@test.local", "admin")
	// A foreign tenant's administrator must not count: the guardrail's count
	// query still scopes through a users join on top of user_roles' own RLS
	// (migration 000286) — without that join this deny would look perfectly
	// safe.
	ovrUserIn(t, pool, ovrForeignTenant, "ovr-foreign-admin@test.local", "admin")

	// Both role-admin keys have to go: denying one would leave the target an
	// administrator through the other, and there would be nothing to protect.
	deny := []auth.CapabilityOverride{
		{Key: "roles:manage", Mode: "deny", Scope: "all"},
		{Key: "admin:role:assign", Mode: "deny", Scope: "all"},
	}
	_, _, err := svc.SetUserOverrides(ctx, actor, ovrTenant, target, deny)
	assert.ErrorIs(t, err, auth.ErrLastAdmin)

	current, err := svc.GetUserOverrides(ctx, target)
	require.NoError(t, err)
	assert.Empty(t, current, "the refused write must not have stored anything")

	// A second administrator makes the same deny safe — the rule is about the
	// tenant keeping one, not about an admin account being untouchable.
	ovrUser(t, pool, "ovr-second-admin@test.local", "admin")
	_, _, err = svc.SetUserOverrides(ctx, actor, ovrTenant, target, deny)
	assert.NoError(t, err)
}

// TestSetUserOverrides_DB_LastAdminCountSeesAllowOverrides is why the count is
// effective rather than role-only. Once overrides exist, an account can be an
// administrator purely through an allow — and a count that only looked at
// roles would refuse a deny that is perfectly safe.
func TestSetUserOverrides_DB_LastAdminCountSeesAllowOverrides(t *testing.T) {
	pool, svc := ovrSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), ovrTenant)
	actor := ovrUser(t, pool, "ovr-actor-allowcount@test.local", "admin")
	target := ovrUser(t, pool, "ovr-target-allowcount@test.local", "admin")
	// Wears no role that administers anything — an allow override is the only
	// thing making them an administrator.
	byOverride := ovrUser(t, pool, "ovr-byoverride@test.local", "member")

	_, _, err := svc.SetUserOverrides(ctx, actor, ovrTenant, byOverride, []auth.CapabilityOverride{
		{Key: "admin:role:assign", Mode: "allow", Scope: "all"},
	})
	require.NoError(t, err)

	// Deactivate the actor so the tenant's only role-wearing administrator
	// besides the target is gone. The deny below then only survives because
	// byOverride counts.
	_, err = pool.Exec(testutil.WithSystemCtx(context.Background()),
		`UPDATE users SET is_active = false WHERE id = $1`, actor)
	require.NoError(t, err)

	caller := ovrUser(t, pool, "ovr-caller-allowcount@test.local", "manager")
	_, _, err = svc.SetUserOverrides(ctx, caller, ovrTenant, target, []auth.CapabilityOverride{
		{Key: "roles:manage", Mode: "deny", Scope: "all"},
		{Key: "admin:role:assign", Mode: "deny", Scope: "all"},
	})
	assert.NoError(t, err, "byOverride is an administrator through their override and keeps the tenant alive")
}

// TestSetUserOverrides_DB_PartialAdminDenyIsAllowed keeps the last-admin guard
// narrow: denying one of the two role-admin keys leaves the account an
// administrator through the other, so there is nothing to protect.
func TestSetUserOverrides_DB_PartialAdminDenyIsAllowed(t *testing.T) {
	pool, svc := ovrSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), ovrTenant)
	actor := ovrUser(t, pool, "ovr-actor-partial@test.local", "admin")
	target := ovrUser(t, pool, "ovr-target-partial@test.local", "admin")

	_, _, err := svc.SetUserOverrides(ctx, actor, ovrTenant, target, []auth.CapabilityOverride{
		{Key: "roles:manage", Mode: "deny", Scope: "all"},
	})
	assert.NoError(t, err)
}

// TestGetUserOverrides_DB_ForeignAccountIs404 is the tenant boundary of the
// read: users carries an RLS read policy, so an account of another tenant is
// invisible rather than forbidden — and the caller must not learn from the
// error which of the two it hit.
func TestGetUserOverrides_DB_ForeignAccountIs404(t *testing.T) {
	pool, svc := ovrSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), ovrTenant)
	foreign := ovrUserIn(t, pool, ovrForeignTenant, "ovr-foreign-target@test.local", "member")
	admin := ovrUser(t, pool, "ovr-admin-foreign@test.local", "admin")

	_, err := svc.GetUserOverrides(ctx, foreign)
	assert.ErrorIs(t, err, auth.ErrUserNotFound)

	_, _, err = svc.SetUserOverrides(ctx, admin, ovrTenant, foreign, []auth.CapabilityOverride{
		{Key: "work:project:edit", Mode: "allow", Scope: "all"},
	})
	assert.ErrorIs(t, err, auth.ErrUserNotFound)

	_, err = svc.ClearUserOverrides(ctx, admin, foreign)
	assert.ErrorIs(t, err, auth.ErrUserNotFound)
}

// TestUserOverrides_DB_RLSHidesForeignRows is the read-side check the loop
// keeps rediscovering the hard way: a table can have tenant_id NOT NULL, an
// INSERT that sets it correctly, and still leak or hide rows if the SELECT is
// not scoped. Here the scoping comes from the RLS policy, and this test is
// what proves the policy is actually on.
func TestUserOverrides_DB_RLSHidesForeignRows(t *testing.T) {
	pool, svc := ovrSetup(t)
	ownCtx := testutil.WithTenantCtx(context.Background(), ovrTenant)
	admin := ovrUser(t, pool, "ovr-admin-rls@test.local", "admin")
	target := ovrUser(t, pool, "ovr-target-rls@test.local", "member")

	_, _, err := svc.SetUserOverrides(ownCtx, admin, ovrTenant, target, []auth.CapabilityOverride{
		{Key: "work:project:edit", Mode: "allow", Scope: "all"},
	})
	require.NoError(t, err)

	own, err := svc.GetUserOverrides(ownCtx, target)
	require.NoError(t, err)
	require.Len(t, own, 1, "the owning tenant sees its own row")

	// The same query under the other tenant's context: the users lookup
	// already fails, which is the 404 above. What this asserts is the row
	// itself being invisible, independent of the account resolution.
	foreignCtx := testutil.WithTenantCtx(context.Background(), ovrForeignTenant)
	var visible int
	require.NoError(t, pool.QueryRow(foreignCtx,
		`SELECT COUNT(*) FROM user_permission_overrides WHERE user_id = $1`, target).Scan(&visible))
	assert.Equal(t, 0, visible, "a foreign tenant must not see the override row")
}

// TestClearUserOverrides_DB_ReportsWhatItRemoved: the clear is what the
// editor's "auf Rollen-Stand zurücksetzen" calls, and every key it drops has
// to reach the audit log — a silent reset would erase the trail of who had
// what.
func TestClearUserOverrides_DB_ReportsWhatItRemoved(t *testing.T) {
	pool, svc := ovrSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), ovrTenant)
	admin := ovrUser(t, pool, "ovr-admin-clear@test.local", "admin")
	target := ovrUser(t, pool, "ovr-target-clear@test.local", "member")

	_, _, err := svc.SetUserOverrides(ctx, admin, ovrTenant, target, []auth.CapabilityOverride{
		{Key: "work:project:edit", Mode: "allow", Scope: "all"},
		{Key: "rapporte:report:create", Mode: "deny", Scope: "all"},
	})
	require.NoError(t, err)

	changes, err := svc.ClearUserOverrides(ctx, admin, target)
	require.NoError(t, err)
	require.Len(t, changes, 2)
	for _, c := range changes {
		assert.NotNil(t, c.Before, "each entry names what was there")
		assert.Nil(t, c.After, "and that it now follows the roles again")
	}

	current, err := svc.GetUserOverrides(ctx, target)
	require.NoError(t, err)
	assert.Empty(t, current)

	// Clearing again is a no-op with nothing to audit, not an error.
	changes, err = svc.ClearUserOverrides(ctx, admin, target)
	require.NoError(t, err)
	assert.Empty(t, changes)
}

// TestSetUserOverrides_DB_RecordsWhoSetIt pins created_by. The audit log is
// rotated; this column is what still answers "who gave them this" afterwards.
func TestSetUserOverrides_DB_RecordsWhoSetIt(t *testing.T) {
	pool, svc := ovrSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), ovrTenant)
	admin := ovrUser(t, pool, "ovr-admin-createdby@test.local", "admin")
	target := ovrUser(t, pool, "ovr-target-createdby@test.local", "member")

	_, _, err := svc.SetUserOverrides(ctx, admin, ovrTenant, target, []auth.CapabilityOverride{
		{Key: "work:project:edit", Mode: "allow", Scope: "all"},
	})
	require.NoError(t, err)

	var createdBy, tenantID uuid.UUID
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT created_by, tenant_id FROM user_permission_overrides WHERE user_id = $1`,
		target).Scan(&createdBy, &tenantID))
	assert.Equal(t, admin, createdBy)
	assert.Equal(t, ovrTenant, tenantID, "the row lands in the caller's tenant, not the target's by accident")
}
