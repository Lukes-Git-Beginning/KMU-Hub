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

// The three writing roster routes. Everything worth testing about them is a
// database property — RLS deciding which roles an invitation may name, the
// guardrails counting live administrators, a token being replaced rather than
// added — so this runs against the real database, with its own tenants rather
// than the shared testutil.TenantA/B (a per-tenant unique index turns shared
// tenants into flaky duplicate keys, which is how a wave-1 test died in CI).
var (
	adminWriteTenant        = uuid.MustParse("40123b01-0000-4000-8000-000000000001")
	adminWriteForeignTenant = uuid.MustParse("40123b01-0000-4000-8000-000000000002")
	// The last-admin case needs a tenant whose entire administrator population
	// this test controls — see TestUpdateAdminUser_DB_LastAdminCannotBeDeactivated.
	adminWriteLastAdminTenant = uuid.MustParse("40123b01-0000-4000-8000-000000000003")
)

func adminWriteSetup(t *testing.T) (*pgxpool.Pool, *auth.Service) {
	t.Helper()
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	testutil.EnsureTenant(t, pool, adminWriteTenant, "AdminWriteTenant")
	testutil.EnsureTenant(t, pool, adminWriteForeignTenant, "AdminWriteForeignTenant")

	return pool, auth.NewService(
		auth.NewPostgresRepository(pool),
		auth.NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour),
	)
}

// seedAdminWriteUser creates an account and, optionally, gives it roles.
func seedAdminWriteUser(t *testing.T, pool *pgxpool.Pool, tenant uuid.UUID, email string, active bool, roles ...uuid.UUID) uuid.UUID {
	t.Helper()
	id := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     tenant,
		"email":         email,
		"password_hash": "x",
		"first_name":    "Write",
		"last_name":     "Test",
		"is_active":     active,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", id) })

	for _, roleID := range roles {
		// user_roles has a composite PK and no surrogate id, so SeedRow's
		// RETURNING id does not apply. Deleted by the users cascade.
		_, err := pool.Exec(testutil.WithSystemCtx(context.Background()),
			`INSERT INTO user_roles (user_id, role_id, tenant_id) VALUES ($1, $2, $3)`, id, roleID, tenant)
		require.NoError(t, err)
	}
	return id
}

// seedAdminWriteCustomRole creates a tenant-owned role carrying one capability.
func seedAdminWriteCustomRole(t *testing.T, pool *pgxpool.Pool, tenant uuid.UUID, name, permission string) uuid.UUID {
	t.Helper()
	sys := testutil.WithSystemCtx(context.Background())

	id := testutil.SeedRow(t, pool, "roles", map[string]any{
		"id":        uuid.New(),
		"tenant_id": tenant,
		"name":      name,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "roles", id) })

	if permission != "" {
		_, err := pool.Exec(sys,
			`INSERT INTO role_permissions (role_id, permission_id)
			 SELECT $1, p.id FROM permissions p WHERE p.name = $2`, id, permission)
		require.NoError(t, err)
	}
	return id
}

// TestInviteAdminUser_DB_StoresNameAndRoleIDs is the core of the invite route:
// the roster row it answers with carries the name the admin typed and the role
// ids they picked — including a CUSTOM role, which the legacy invitations.role
// name column could never express.
func TestInviteAdminUser_DB_StoresNameAndRoleIDs(t *testing.T) {
	pool, svc := adminWriteSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), adminWriteTenant)

	actorID := seedAdminWriteUser(t, pool, adminWriteTenant, "invite-actor@test.local", true)
	memberPreset := adminRosterPresetID(t, pool, "member")
	customID := seedAdminWriteCustomRole(t, pool, adminWriteTenant, "AdminWriteBuchhaltung", "")

	row, token, err := svc.InviteAdminUser(ctx, adminWriteTenant, actorID,
		"invite-target@test.local", "Neue", "Person", []uuid.UUID{memberPreset, customID})
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "invitations", row.ID) })

	assert.NotEmpty(t, token, "the accept token has to reach the caller — nothing mails it")
	assert.Equal(t, auth.AdminUserStatusInvited, row.Status)
	assert.Equal(t, "Neue", row.FirstName)
	assert.Equal(t, "Person", row.LastName)
	assert.ElementsMatch(t, []string{memberPreset.String(), customID.String()}, row.RoleIDs,
		"a custom role must survive the invitation — the whole point of role_ids")

	// And it shows up in the roster the surface reads.
	users, err := svc.ListAdminUsers(ctx)
	require.NoError(t, err)
	require.NotNil(t, findAdminUser(users, row.ID), "the new invitation must be in the roster")
}

// TestInviteAdminUser_DB_RejectsForeignRole is the tenant boundary. A role
// belonging to another tenant is invisible under the roles read policy, so
// naming it answers not_found rather than quietly inviting into it.
func TestInviteAdminUser_DB_RejectsForeignRole(t *testing.T) {
	pool, svc := adminWriteSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), adminWriteTenant)

	actorID := seedAdminWriteUser(t, pool, adminWriteTenant, "invite-actor-foreign@test.local", true)
	foreignRole := seedAdminWriteCustomRole(t, pool, adminWriteForeignTenant, "AdminWriteForeignRole", "")

	_, _, err := svc.InviteAdminUser(ctx, adminWriteTenant, actorID,
		"invite-foreign-target@test.local", "", "", []uuid.UUID{foreignRole})
	require.ErrorIs(t, err, auth.ErrBaseRoleNotFound)

	// An invitation with no role at all is refused too — the account it would
	// create could not do anything and could not repair itself.
	_, _, err = svc.InviteAdminUser(ctx, adminWriteTenant, actorID,
		"invite-noroles@test.local", "", "", nil)
	require.ErrorIs(t, err, auth.ErrRoleNotFound)
}

// TestAcceptInvitation_DB_ResolvesRolesByID guards the defect migration 000280
// closed: the accept path used to match roles by NAME under system context,
// where RLS is off, so an invitation naming "admin" also granted any foreign
// tenant's custom role of that name.
func TestAcceptInvitation_DB_ResolvesRolesByID(t *testing.T) {
	pool, svc := adminWriteSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), adminWriteTenant)

	actorID := seedAdminWriteUser(t, pool, adminWriteTenant, "accept-actor@test.local", true)
	ownRole := seedAdminWriteCustomRole(t, pool, adminWriteTenant, "AdminWriteShared", "")
	// Same NAME, other tenant. Under the old name-based resolution this row
	// would have been granted as well.
	foreignSameName := seedAdminWriteCustomRole(t, pool, adminWriteForeignTenant, "AdminWriteShared", "")

	row, token, err := svc.InviteAdminUser(ctx, adminWriteTenant, actorID,
		"accept-target@test.local", "Accept", "Target", []uuid.UUID{ownRole})
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "invitations", row.ID) })

	user, _, err := svc.AcceptInvitation(context.Background(), token, "sufficiently-long-pw", "Accept", "Target")
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", user.ID) })

	assigned, err := auth.NewPostgresRepository(pool).GetUserRoleIDs(
		testutil.WithTenantCtx(context.Background(), adminWriteTenant), user.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{ownRole.String()}, assigned)
	assert.NotContains(t, assigned, foreignSameName.String(),
		"a foreign tenant's role of the same name must not be granted")
}

// TestResendAdminUserInvite_DB_InvalidatesOldToken: a resend replaces the token
// rather than adding one. Two live tokens would be two ways into one account.
func TestResendAdminUserInvite_DB_InvalidatesOldToken(t *testing.T) {
	pool, svc := adminWriteSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), adminWriteTenant)

	actorID := seedAdminWriteUser(t, pool, adminWriteTenant, "resend-actor@test.local", true)
	memberPreset := adminRosterPresetID(t, pool, "member")

	row, firstToken, err := svc.InviteAdminUser(ctx, adminWriteTenant, actorID,
		"resend-target@test.local", "Re", "Send", []uuid.UUID{memberPreset})
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "invitations", row.ID) })

	resent, secondToken, err := svc.ResendAdminUserInvite(ctx, adminWriteTenant, row.ID)
	require.NoError(t, err)
	assert.NotEqual(t, firstToken, secondToken, "a resend has to mint a new token")
	assert.Equal(t, row.ID, resent.ID)
	assert.Equal(t, "Re", resent.FirstName, "the name survives a resend")

	// The old token is gone: accepting with it fails, accepting with the new
	// one works.
	_, _, err = svc.AcceptInvitation(context.Background(), firstToken, "sufficiently-long-pw", "Re", "Send")
	require.ErrorIs(t, err, auth.ErrInvitationNotFound, "the previous link must stop working")

	user, _, err := svc.AcceptInvitation(context.Background(), secondToken, "sufficiently-long-pw", "Re", "Send")
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", user.ID) })
}

// TestProvisionTenant_DB_InvitationCarriesRoleIDs: the provisioning path
// writes its own INSERT into invitations, so it has to fill role_ids too.
// Leaving it to the column default would hand a brand-new tenant an invitation
// that grants nothing and cannot be accepted — the tenant would be stranded.
func TestProvisionTenant_DB_InvitationCarriesRoleIDs(t *testing.T) {
	pool, svc := adminWriteSetup(t)

	operatorID := seedAdminWriteUser(t, pool, adminWriteTenant, "provision-operator@test.local", true)
	adminPreset := adminRosterPresetID(t, pool, "admin")

	res, err := svc.ProvisionTenant(context.Background(), auth.ProvisionTenantInput{
		Name:        "AdminWriteProvisioned",
		PlanType:    "cosmi",
		AdminEmail:  "provision-admin@test.local",
		CreatedBy:   operatorID,
		SupportTier: "standard",
	})
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", res.Tenant.ID) })

	var roleIDs []uuid.UUID
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT role_ids FROM invitations WHERE id = $1`, res.Invitation.ID).Scan(&roleIDs))
	assert.Equal(t, []uuid.UUID{adminPreset}, roleIDs,
		"the first administrator's invitation must name the admin preset by id")
}

// TestUpdateAdminUser_DB_ReplacesRoles: roles are a full replacement, and the
// replacement runs through the guarded assign/revoke path.
func TestUpdateAdminUser_DB_ReplacesRoles(t *testing.T) {
	pool, svc := adminWriteSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), adminWriteTenant)

	// The actor holds role administration through a preset; the guardrails
	// need a second live administrator to let anything through.
	adminPreset := adminRosterPresetID(t, pool, "admin")
	actorID := seedAdminWriteUser(t, pool, adminWriteTenant, "update-actor@test.local", true, adminPreset)

	memberPreset := adminRosterPresetID(t, pool, "member")
	customID := seedAdminWriteCustomRole(t, pool, adminWriteTenant, "AdminWriteTarget", "")
	targetID := seedAdminWriteUser(t, pool, adminWriteTenant, "update-target@test.local", true, memberPreset)

	want := []uuid.UUID{customID}
	row, err := svc.UpdateAdminUser(ctx, actorID, targetID, &want, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{customID.String()}, row.RoleIDs,
		"the old role must be revoked, not kept alongside the new one")

	// An empty slice strips every role; nil would have left them alone.
	none := []uuid.UUID{}
	row, err = svc.UpdateAdminUser(ctx, actorID, targetID, &none, nil)
	require.NoError(t, err)
	assert.Empty(t, row.RoleIDs)
}

// TestUpdateAdminUser_DB_StatusGuardrails covers the three refusals on the
// status edit: no self-deactivation, no deactivating the last administrator,
// and "invited" is not a state an account can be put into.
func TestUpdateAdminUser_DB_StatusGuardrails(t *testing.T) {
	pool, svc := adminWriteSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), adminWriteTenant)

	adminPreset := adminRosterPresetID(t, pool, "admin")
	actorID := seedAdminWriteUser(t, pool, adminWriteTenant, "status-actor@test.local", true, adminPreset)
	targetID := seedAdminWriteUser(t, pool, adminWriteTenant, "status-target@test.local", true)

	deactivated := auth.AdminUserStatusDeactivated
	invited := auth.AdminUserStatusInvited

	_, err := svc.UpdateAdminUser(ctx, actorID, actorID, nil, &deactivated)
	require.ErrorIs(t, err, auth.ErrSelfDeactivation,
		"an admin switching off their own account only notices at the next login")

	_, err = svc.UpdateAdminUser(ctx, actorID, targetID, nil, &invited)
	require.ErrorIs(t, err, auth.ErrStatusNotAssignable)

	// A plain member deactivates fine, and reactivates again.
	row, err := svc.UpdateAdminUser(ctx, actorID, targetID, nil, &deactivated)
	require.NoError(t, err)
	assert.Equal(t, auth.AdminUserStatusDeactivated, row.Status)

	active := auth.AdminUserStatusActive
	row, err = svc.UpdateAdminUser(ctx, actorID, targetID, nil, &active)
	require.NoError(t, err)
	assert.Equal(t, auth.AdminUserStatusActive, row.Status)
}

// TestUpdateAdminUser_DB_LastAdminCannotBeDeactivated: the tenant keeps at
// least one live administrator. Deliberately driven by a SECOND admin so the
// self-deactivation guard is not what refuses it.
func TestUpdateAdminUser_DB_LastAdminCannotBeDeactivated(t *testing.T) {
	pool, svc := adminWriteSetup(t)

	// Its OWN tenant, not the shared adminWriteTenant: "last administrator" is
	// a property of the tenant's whole population, so a neighbouring test's
	// admin account would decide the outcome. Here the count is exactly one by
	// construction.
	testutil.EnsureTenant(t, pool, adminWriteLastAdminTenant, "AdminWriteLastAdminTenant")
	ctx := testutil.WithTenantCtx(context.Background(), adminWriteLastAdminTenant)

	adminPreset := adminRosterPresetID(t, pool, "admin")
	lastAdminID := seedAdminWriteUser(t, pool, adminWriteLastAdminTenant, "last-admin@test.local", true, adminPreset)
	// The actor deliberately holds no role: the route's guard decides who may
	// call, the service decides what the tenant survives. Giving the actor the
	// admin preset here would make them the second administrator and the
	// guardrail would (correctly) let the deactivation through.
	actorID := seedAdminWriteUser(t, pool, adminWriteLastAdminTenant, "last-admin-actor@test.local", true)

	deactivated := auth.AdminUserStatusDeactivated
	_, err := svc.UpdateAdminUser(ctx, actorID, lastAdminID, nil, &deactivated)
	require.ErrorIs(t, err, auth.ErrLastAdmin)

	// A second administrator makes the same deactivation legitimate.
	seedAdminWriteUser(t, pool, adminWriteLastAdminTenant, "second-admin@test.local", true, adminPreset)
	row, err := svc.UpdateAdminUser(ctx, actorID, lastAdminID, nil, &deactivated)
	require.NoError(t, err)
	assert.Equal(t, auth.AdminUserStatusDeactivated, row.Status)
}

// TestUpdateAdminUser_DB_TenantIsolation: an account of another tenant is
// invisible under the users read policy, so it answers not found rather than
// forbidden — and cannot be edited across the boundary.
func TestUpdateAdminUser_DB_TenantIsolation(t *testing.T) {
	pool, svc := adminWriteSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), adminWriteTenant)

	actorID := seedAdminWriteUser(t, pool, adminWriteTenant, "isolation-actor@test.local", true)
	foreignID := seedAdminWriteUser(t, pool, adminWriteForeignTenant, "isolation-foreign@test.local", true)

	deactivated := auth.AdminUserStatusDeactivated
	_, err := svc.UpdateAdminUser(ctx, actorID, foreignID, nil, &deactivated)
	require.ErrorIs(t, err, auth.ErrUserNotFound)

	// The foreign account is untouched.
	var stillActive bool
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT is_active FROM users WHERE id = $1`, foreignID).Scan(&stillActive))
	assert.True(t, stillActive)

	// Same for a resend against another tenant's invitation.
	foreignInv := testutil.SeedRow(t, pool, "invitations", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  adminWriteForeignTenant,
		"email":      "isolation-foreign-invite@test.local",
		"role":       "member",
		"token_hash": uuid.NewString(),
		"created_by": foreignID,
		"expires_at": time.Now().Add(24 * time.Hour),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "invitations", foreignInv) })

	_, _, err = svc.ResendAdminUserInvite(ctx, adminWriteTenant, foreignInv)
	require.ErrorIs(t, err, auth.ErrInvitationNotFound)
}
