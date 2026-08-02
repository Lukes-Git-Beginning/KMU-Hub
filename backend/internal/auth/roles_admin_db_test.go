package auth_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// Role administration runs against the real database because everything that
// can go wrong here is a database property: the clone has to copy grants
// through the role_permissions write policy, the name check has to see the
// presets a tenant does not own, and the expression unique index only exists
// in the schema. A mock repository would prove none of it.
//
// Own tenants, never the shared testutil.TenantA/B — these tests write roles
// whose names are unique per tenant and run beside the rest of the package.
var (
	roleAdminTenant        = uuid.MustParse("40123a00-0000-4000-8000-000000000001")
	roleAdminForeignTenant = uuid.MustParse("40123a00-0000-4000-8000-000000000002")
)

// TestRoleAdminErrorsCarryFrontendCodes pins the sentinel messages. They are
// not prose: mapError passes err.Error() through as the gRPC status message,
// the gateway writes it as {"error": "<message>"}, and rbac-format.ts looks
// that string up in RBAC_ERROR_CODES. Rewording one of these to something
// readable would turn a precise message in the builder into "Unbekannter
// Fehler" without breaking a single compile.
func TestRoleAdminErrorsCarryFrontendCodes(t *testing.T) {
	assert.Equal(t, "role_limit_reached", auth.ErrRoleLimitReached.Error())
	assert.Equal(t, "role_name_exists", auth.ErrRoleNameExists.Error())
	assert.Equal(t, "not_found", auth.ErrBaseRoleNotFound.Error())
	assert.Equal(t, "unknown_capability_key", auth.ErrCapabilityKeyUnknown.Error())
}

func roleAdminSetup(t *testing.T) (*pgxpool.Pool, *auth.Service) {
	t.Helper()
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	testutil.EnsureTenant(t, pool, roleAdminTenant, "RoleAdminTenant")
	testutil.EnsureTenant(t, pool, roleAdminForeignTenant, "RoleAdminForeignTenant")
	roleAdminCleanupRoles(t, pool)

	svc := auth.NewService(
		auth.NewPostgresRepository(pool),
		auth.NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour),
	)
	return pool, svc
}

// roleAdminCleanupRoles drops everything the previous run of these tests left
// in the two tenants, both up front and afterwards: a failed assertion must
// not poison the next run through the 20-role budget.
func roleAdminCleanupRoles(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	drop := func() {
		_, err := pool.Exec(testutil.WithSystemCtx(context.Background()),
			`DELETE FROM roles WHERE tenant_id = ANY($1)`,
			[]uuid.UUID{roleAdminTenant, roleAdminForeignTenant})
		require.NoError(t, err)
	}
	drop()
	t.Cleanup(drop)
}

// presetID resolves a preset by name. The tests clone from presets because
// that is what the builder does.
func presetID(t *testing.T, pool *pgxpool.Pool, name string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT id FROM roles WHERE name = $1 AND tenant_id IS NULL`, name).Scan(&id)
	require.NoErrorf(t, err, "preset %q missing — migration 000256 not applied?", name)
	return id
}

func grantsOf(t *testing.T, pool *pgxpool.Pool, roleID uuid.UUID) map[string]string {
	t.Helper()
	rows, err := pool.Query(testutil.WithSystemCtx(context.Background()),
		`SELECT p.resource || ':' || p.action, rp.scope
		 FROM role_permissions rp JOIN permissions p ON p.id = rp.permission_id
		 WHERE rp.role_id = $1`, roleID)
	require.NoError(t, err)
	defer rows.Close()

	out := map[string]string{}
	for rows.Next() {
		var key, scope string
		require.NoError(t, rows.Scan(&key, &scope))
		out[key] = scope
	}
	require.NoError(t, rows.Err())
	return out
}

// TestCreateRole_DB_ClonesEveryGrantWithScope is the core of the unit: the new
// role must carry the source's grants byte for byte, scope included. Copying
// the keys but dropping the scope would silently upgrade every own/team grant
// to all — the exact privilege escalation wave 1b's guardrails exist to stop.
func TestCreateRole_DB_ClonesEveryGrantWithScope(t *testing.T) {
	pool, svc := roleAdminSetup(t)
	source := presetID(t, pool, "extern")
	ctx := testutil.WithTenantCtx(context.Background(), roleAdminTenant)

	role, err := svc.CreateRole(ctx, roleAdminTenant, auth.CreateRoleInput{
		Name:        "  Lagerhilfe  ",
		Description: "  Aushilfe im Lager  ",
		Color:       "hsl(217 91% 60%)",
		BasedOn:     source,
	})
	require.NoError(t, err)

	assert.Equal(t, "Lagerhilfe", role.Name, "surrounding whitespace is trimmed before it reaches the unique index")
	assert.Equal(t, "Aushilfe im Lager", role.Description)
	require.NotNil(t, role.TenantID)
	assert.Equal(t, roleAdminTenant, *role.TenantID)
	require.NotNil(t, role.BasedOn)
	assert.Equal(t, source, *role.BasedOn)
	assert.False(t, role.IsSystem)

	want := grantsOf(t, pool, source)
	got := grantsOf(t, pool, role.ID)
	assert.Equal(t, want, got, "the clone must carry the source's grants including scope")
	assert.Equal(t, len(want), role.CapabilityCount, "capabilityCount is what the list view shows")
	assert.NotEmpty(t, want, "the extern preset carries grants — otherwise this test proves nothing")
}

// TestCreateRole_DB_ListRolesSeesTheClone closes the loop with the wave's
// first unit: a freshly created role has to show up in GET /admin/roles with
// zero members and the cloned grant count.
func TestCreateRole_DB_ListRolesSeesTheClone(t *testing.T) {
	pool, svc := roleAdminSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), roleAdminTenant)

	created, err := svc.CreateRole(ctx, roleAdminTenant, auth.CreateRoleInput{
		Name: "Sichtbar", BasedOn: presetID(t, pool, "extern"),
	})
	require.NoError(t, err)

	roles, err := svc.ListRoles(ctx)
	require.NoError(t, err)

	var found *auth.Role
	for i := range roles {
		if roles[i].ID == created.ID {
			found = &roles[i]
		}
	}
	require.NotNil(t, found, "the new role must be listed for its own tenant")
	assert.Equal(t, 0, found.MemberCount)
	assert.Equal(t, created.CapabilityCount, found.CapabilityCount)
}

// TestCreateRole_DB_NameCollision covers both halves of the frontend
// contract: the comparison is case-insensitive, and it also fires against a
// preset the tenant does not own. Neither is expressible in the unique index
// (it compares case-sensitively and puts presets in their own COALESCE
// bucket), so without the explicit check the builder would happily create a
// second "Admin" beside the preset.
func TestCreateRole_DB_NameCollision(t *testing.T) {
	pool, svc := roleAdminSetup(t)
	source := presetID(t, pool, "extern")
	ctx := testutil.WithTenantCtx(context.Background(), roleAdminTenant)

	_, err := svc.CreateRole(ctx, roleAdminTenant, auth.CreateRoleInput{Name: "Buchhaltung", BasedOn: source})
	require.NoError(t, err)

	_, err = svc.CreateRole(ctx, roleAdminTenant, auth.CreateRoleInput{Name: "buchhaltung", BasedOn: source})
	assert.ErrorIs(t, err, auth.ErrRoleNameExists, "the check is case-insensitive")

	_, err = svc.CreateRole(ctx, roleAdminTenant, auth.CreateRoleInput{Name: "Admin", BasedOn: source})
	assert.ErrorIs(t, err, auth.ErrRoleNameExists, "a preset name is taken too")
}

// TestCreateRole_DB_NameIsFreeInAnotherTenant is the counterpart: role names
// are unique per tenant, not globally. If this ever fails, the check lost its
// RLS scoping and one tenant's naming would block every other one.
func TestCreateRole_DB_NameIsFreeInAnotherTenant(t *testing.T) {
	pool, svc := roleAdminSetup(t)
	source := presetID(t, pool, "extern")

	_, err := svc.CreateRole(testutil.WithTenantCtx(context.Background(), roleAdminTenant),
		roleAdminTenant, auth.CreateRoleInput{Name: "Doppelname", BasedOn: source})
	require.NoError(t, err)

	_, err = svc.CreateRole(testutil.WithTenantCtx(context.Background(), roleAdminForeignTenant),
		roleAdminForeignTenant, auth.CreateRoleInput{Name: "Doppelname", BasedOn: source})
	assert.NoError(t, err, "two tenants may both name a role Doppelname")
}

// TestCreateRole_DB_BasedOnNotVisible proves the clone source is resolved
// through the RLS read policy: another tenant's role is not a 500 and not a
// silent empty clone, it is the 404 the builder shows as "not found".
func TestCreateRole_DB_BasedOnNotVisible(t *testing.T) {
	pool, svc := roleAdminSetup(t)
	foreign, err := svc.CreateRole(testutil.WithTenantCtx(context.Background(), roleAdminForeignTenant),
		roleAdminForeignTenant, auth.CreateRoleInput{Name: "Fremdrolle", BasedOn: presetID(t, pool, "extern")})
	require.NoError(t, err)

	ctx := testutil.WithTenantCtx(context.Background(), roleAdminTenant)
	_, err = svc.CreateRole(ctx, roleAdminTenant, auth.CreateRoleInput{Name: "Klau", BasedOn: foreign.ID})
	assert.ErrorIs(t, err, auth.ErrBaseRoleNotFound, "another tenant's role is not a valid clone source")

	_, err = svc.CreateRole(ctx, roleAdminTenant, auth.CreateRoleInput{Name: "Klau", BasedOn: uuid.New()})
	assert.ErrorIs(t, err, auth.ErrBaseRoleNotFound, "an unknown id is a 404 as well")

	// Nothing may survive the rejected clone.
	roles, err := svc.ListRoles(ctx)
	require.NoError(t, err)
	for _, r := range roles {
		assert.NotEqual(t, "Klau", r.Name, "a rejected clone must leave no role behind")
	}
}

// TestCreateRole_DB_LimitCountsOnlyCustomRoles fills the budget to the brim.
// The presets must not count against it — if they did, the eight seeded roles
// would eat nearly half of every tenant's allowance.
func TestCreateRole_DB_LimitCountsOnlyCustomRoles(t *testing.T) {
	pool, svc := roleAdminSetup(t)
	source := presetID(t, pool, "extern")
	ctx := testutil.WithTenantCtx(context.Background(), roleAdminTenant)

	for i := range auth.CustomRoleLimit {
		_, err := svc.CreateRole(ctx, roleAdminTenant, auth.CreateRoleInput{
			Name: fmt.Sprintf("Limit %02d", i), BasedOn: source,
		})
		require.NoErrorf(t, err, "role %d of %d must still fit — presets are counted against the budget",
			i+1, auth.CustomRoleLimit)
	}

	_, err := svc.CreateRole(ctx, roleAdminTenant, auth.CreateRoleInput{Name: "Einer zuviel", BasedOn: source})
	assert.ErrorIs(t, err, auth.ErrRoleLimitReached)

	// The other tenant keeps its own budget.
	_, err = svc.CreateRole(testutil.WithTenantCtx(context.Background(), roleAdminForeignTenant),
		roleAdminForeignTenant, auth.CreateRoleInput{Name: "Erste", BasedOn: source})
	assert.NoError(t, err, "the budget is per tenant")
}

func ptr(s string) *string { return &s }

// roleAdminUser inserts a minimal active user into tenantID via the real
// repository, mirroring tenant_write_test.go's fixture shape.
func roleAdminUser(t *testing.T, pool *pgxpool.Pool, repo *auth.PostgresRepository, tenantID uuid.UUID) *models.User {
	t.Helper()
	now := time.Now().UTC()
	user := &models.User{
		ID:           uuid.New(),
		TenantID:     tenantID,
		Email:        "role-admin-" + uuid.New().String() + "@test.local",
		PasswordHash: "x",
		FirstName:    "Role",
		LastName:     "Holder",
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	require.NoError(t, repo.CreateUser(testutil.WithTenantCtx(context.Background(), tenantID), user))
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", user.ID) })
	return user
}

// TestUpdateRole_DB_AppliesOnlyProvidedFields is the partial-PATCH contract:
// an absent field must survive untouched, not be reset to empty.
func TestUpdateRole_DB_AppliesOnlyProvidedFields(t *testing.T) {
	pool, svc := roleAdminSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), roleAdminTenant)

	created, err := svc.CreateRole(ctx, roleAdminTenant, auth.CreateRoleInput{
		Name: "Werkstatt", Description: "Urspruengliche Beschreibung", Color: "hsl(0 0% 50%)",
		BasedOn: presetID(t, pool, "extern"),
	})
	require.NoError(t, err)

	updated, err := svc.UpdateRole(ctx, created.ID, auth.UpdateRoleInput{Color: ptr("hsl(200 80% 50%)")})
	require.NoError(t, err)

	assert.Equal(t, "Werkstatt", updated.Name, "name was not part of the patch and must survive")
	assert.Equal(t, "Urspruengliche Beschreibung", updated.Description, "description was not part of the patch")
	assert.Equal(t, "hsl(200 80% 50%)", updated.Color)
	assert.Equal(t, created.CapabilityCount, updated.CapabilityCount, "the update must not touch the cloned grants")
}

// TestUpdateRole_DB_NameCollision mirrors the create-side test: the check is
// case-insensitive, covers the presets too, but must not fire against the
// role's own current name (exceptID) — a no-op rename is not a collision.
func TestUpdateRole_DB_NameCollision(t *testing.T) {
	pool, svc := roleAdminSetup(t)
	source := presetID(t, pool, "extern")
	ctx := testutil.WithTenantCtx(context.Background(), roleAdminTenant)

	taken, err := svc.CreateRole(ctx, roleAdminTenant, auth.CreateRoleInput{Name: "Belegt", BasedOn: source})
	require.NoError(t, err)
	mine, err := svc.CreateRole(ctx, roleAdminTenant, auth.CreateRoleInput{Name: "Eigene", BasedOn: source})
	require.NoError(t, err)

	_, err = svc.UpdateRole(ctx, mine.ID, auth.UpdateRoleInput{Name: ptr("belegt")})
	assert.ErrorIs(t, err, auth.ErrRoleNameExists, "case-insensitive collision against another custom role")

	_, err = svc.UpdateRole(ctx, mine.ID, auth.UpdateRoleInput{Name: ptr("Admin")})
	assert.ErrorIs(t, err, auth.ErrRoleNameExists, "a preset name is taken too")

	_, err = svc.UpdateRole(ctx, taken.ID, auth.UpdateRoleInput{Name: ptr("Belegt")})
	assert.NoError(t, err, "renaming a role onto its own current name is not a collision")
}

// TestUpdateRole_DB_PresetIsImmutable proves the explicit 403: the write
// policy alone would just touch zero rows on a preset id, which would look
// like a silent no-op success to the builder instead of an error.
func TestUpdateRole_DB_PresetIsImmutable(t *testing.T) {
	pool, svc := roleAdminSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), roleAdminTenant)

	_, err := svc.UpdateRole(ctx, presetID(t, pool, "extern"), auth.UpdateRoleInput{Name: ptr("Gehackt")})
	assert.ErrorIs(t, err, auth.ErrRolePresetImmutable)
}

// TestUpdateRole_DB_ForeignRoleIsNotFound proves the role is resolved through
// the RLS read policy: another tenant's custom role must not be reachable —
// not even to learn that it exists.
func TestUpdateRole_DB_ForeignRoleIsNotFound(t *testing.T) {
	pool, svc := roleAdminSetup(t)
	foreign, err := svc.CreateRole(testutil.WithTenantCtx(context.Background(), roleAdminForeignTenant),
		roleAdminForeignTenant, auth.CreateRoleInput{Name: "Fremd", BasedOn: presetID(t, pool, "extern")})
	require.NoError(t, err)

	ctx := testutil.WithTenantCtx(context.Background(), roleAdminTenant)
	_, err = svc.UpdateRole(ctx, foreign.ID, auth.UpdateRoleInput{Name: ptr("Geklaut")})
	assert.ErrorIs(t, err, auth.ErrBaseRoleNotFound)

	_, err = svc.UpdateRole(ctx, uuid.New(), auth.UpdateRoleInput{Name: ptr("Unbekannt")})
	assert.ErrorIs(t, err, auth.ErrBaseRoleNotFound, "an unknown id is a 404 as well")
}

// TestDeleteRole_DB_PresetIsImmutable is the delete-side counterpart of the
// update guard.
func TestDeleteRole_DB_PresetIsImmutable(t *testing.T) {
	pool, svc := roleAdminSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), roleAdminTenant)

	err := svc.DeleteRole(ctx, presetID(t, pool, "extern"))
	assert.ErrorIs(t, err, auth.ErrRolePresetImmutable)
}

// TestDeleteRole_DB_RoleHasMembers blocks deleting a role out from under its
// holder; once the holder is removed the same role deletes cleanly.
func TestDeleteRole_DB_RoleHasMembers(t *testing.T) {
	pool, svc := roleAdminSetup(t)
	repo := auth.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), roleAdminTenant)

	role, err := svc.CreateRole(ctx, roleAdminTenant, auth.CreateRoleInput{
		Name: "Besetzt", BasedOn: presetID(t, pool, "extern"),
	})
	require.NoError(t, err)

	user := roleAdminUser(t, pool, repo, roleAdminTenant)
	require.NoError(t, repo.AssignRole(ctx, user.ID, role.Name))

	err = svc.DeleteRole(ctx, role.ID)
	assert.ErrorIs(t, err, auth.ErrRoleHasMembers)

	require.NoError(t, repo.RemoveRole(ctx, user.ID, role.Name))
	assert.NoError(t, svc.DeleteRole(ctx, role.ID), "once the last holder is gone the role must be deletable")
}

// TestDeleteRole_DB_RemovesRoleAndGrants closes the loop: a deleted role must
// disappear from ListRoles, and role_permissions must cascade with it rather
// than leaving orphaned grants behind.
func TestDeleteRole_DB_RemovesRoleAndGrants(t *testing.T) {
	pool, svc := roleAdminSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), roleAdminTenant)

	role, err := svc.CreateRole(ctx, roleAdminTenant, auth.CreateRoleInput{
		Name: "Vergaenglich", BasedOn: presetID(t, pool, "extern"),
	})
	require.NoError(t, err)
	require.NotEmpty(t, grantsOf(t, pool, role.ID), "the clone must carry grants — otherwise the cascade proves nothing")

	require.NoError(t, svc.DeleteRole(ctx, role.ID))

	roles, err := svc.ListRoles(ctx)
	require.NoError(t, err)
	for _, r := range roles {
		assert.NotEqual(t, role.ID, r.ID, "the deleted role must not be listed anymore")
	}
	assert.Empty(t, grantsOf(t, pool, role.ID), "role_permissions must cascade on role_id")
}

// TestDeleteRole_DB_NotFound covers an unknown id and a foreign tenant's role
// alike — both must answer ErrBaseRoleNotFound, never a 500 or a silent no-op.
func TestDeleteRole_DB_NotFound(t *testing.T) {
	pool, svc := roleAdminSetup(t)
	foreign, err := svc.CreateRole(testutil.WithTenantCtx(context.Background(), roleAdminForeignTenant),
		roleAdminForeignTenant, auth.CreateRoleInput{Name: "Unerreichbar", BasedOn: presetID(t, pool, "extern")})
	require.NoError(t, err)

	ctx := testutil.WithTenantCtx(context.Background(), roleAdminTenant)
	assert.ErrorIs(t, svc.DeleteRole(ctx, foreign.ID), auth.ErrBaseRoleNotFound)
	assert.ErrorIs(t, svc.DeleteRole(ctx, uuid.New()), auth.ErrBaseRoleNotFound)
}

// grantMap turns []auth.RoleGrant into the key→scope shape grantsOf returns,
// so the two can be compared directly regardless of row order.
func grantMap(grants []auth.RoleGrant) map[string]string {
	out := make(map[string]string, len(grants))
	for _, g := range grants {
		out[g.Key] = g.Scope
	}
	return out
}

// TestGetRolePermissions_DB_MatchesClonedGrants closes the read side against
// the same fixture TestCreateRole_DB_ClonesEveryGrantWithScope uses: what the
// clone wrote to role_permissions is exactly what the builder reads back.
func TestGetRolePermissions_DB_MatchesClonedGrants(t *testing.T) {
	pool, svc := roleAdminSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), roleAdminTenant)

	role, err := svc.CreateRole(ctx, roleAdminTenant, auth.CreateRoleInput{
		Name: "Leser", BasedOn: presetID(t, pool, "extern"),
	})
	require.NoError(t, err)

	grants, err := svc.GetRolePermissions(ctx, role.ID)
	require.NoError(t, err)
	assert.Equal(t, grantsOf(t, pool, role.ID), grantMap(grants))
	assert.NotEmpty(t, grants, "the extern clone carries grants — otherwise this test proves nothing")
}

// TestGetRolePermissions_DB_ReadsPresets proves presets are readable: the
// builder shows a preset's grants while the tenant is still choosing what to
// clone, before any custom role exists. Only the write side is restricted.
func TestGetRolePermissions_DB_ReadsPresets(t *testing.T) {
	pool, svc := roleAdminSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), roleAdminTenant)

	grants, err := svc.GetRolePermissions(ctx, presetID(t, pool, "extern"))
	require.NoError(t, err)
	assert.NotEmpty(t, grants)
}

// TestGetRolePermissions_DB_NotFound mirrors the other GetRoleByID-backed
// reads: an unknown id and a foreign tenant's role both answer
// ErrBaseRoleNotFound, never a 500 and never an empty grant map masquerading
// as "role exists but has nothing granted".
func TestGetRolePermissions_DB_NotFound(t *testing.T) {
	pool, svc := roleAdminSetup(t)
	foreign, err := svc.CreateRole(testutil.WithTenantCtx(context.Background(), roleAdminForeignTenant),
		roleAdminForeignTenant, auth.CreateRoleInput{Name: "Verborgen", BasedOn: presetID(t, pool, "extern")})
	require.NoError(t, err)

	ctx := testutil.WithTenantCtx(context.Background(), roleAdminTenant)
	_, err = svc.GetRolePermissions(ctx, foreign.ID)
	assert.ErrorIs(t, err, auth.ErrBaseRoleNotFound)

	_, err = svc.GetRolePermissions(ctx, uuid.New())
	assert.ErrorIs(t, err, auth.ErrBaseRoleNotFound)
}

// TestSetRolePermissions_DB_FullReplace is the PUT contract end to end: a key
// missing from the new set is revoked, a key present with a changed scope is
// updated, and a key not granted before is added — all in the single call.
func TestSetRolePermissions_DB_FullReplace(t *testing.T) {
	pool, svc := roleAdminSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), roleAdminTenant)

	role, err := svc.CreateRole(ctx, roleAdminTenant, auth.CreateRoleInput{
		Name: "Umbau", BasedOn: presetID(t, pool, "extern"),
	})
	require.NoError(t, err)
	before := grantsOf(t, pool, role.ID)
	require.Equal(t, "team", before["work:task:read"], "fixture assumption from migration 000256's extern seed")
	require.NotContains(t, before, "crm:contact:read", "fixture assumption: extern does not carry this key")

	updated, err := svc.SetRolePermissions(ctx, role.ID, []auth.RoleGrant{
		{Key: "dashboard:module:view", Scope: "all"}, // unchanged
		{Key: "work:task:read", Scope: "all"},        // scope raised team -> all
		{Key: "crm:contact:read", Scope: "own"},      // newly granted
	})
	require.NoError(t, err)

	want := map[string]string{
		"dashboard:module:view": "all",
		"work:task:read":        "all",
		"crm:contact:read":      "own",
	}
	assert.Equal(t, want, grantMap(updated), "the response must already reflect the replacement")
	assert.Equal(t, want, grantsOf(t, pool, role.ID), "everything else from the clone must be revoked")
}

// TestSetRolePermissions_DB_EmptyRevokesEverything is the edge of full
// replacement: an empty grant map is a valid PUT body and must strip every
// capability the role held, not be mistaken for "nothing to do".
func TestSetRolePermissions_DB_EmptyRevokesEverything(t *testing.T) {
	pool, svc := roleAdminSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), roleAdminTenant)

	role, err := svc.CreateRole(ctx, roleAdminTenant, auth.CreateRoleInput{
		Name: "Entleert", BasedOn: presetID(t, pool, "extern"),
	})
	require.NoError(t, err)

	updated, err := svc.SetRolePermissions(ctx, role.ID, []auth.RoleGrant{})
	require.NoError(t, err)
	assert.Empty(t, updated)
	assert.Empty(t, grantsOf(t, pool, role.ID))
}

// TestSetRolePermissions_DB_PresetIsImmutable is the write-side counterpart
// of TestGetRolePermissions_DB_ReadsPresets: reading a preset's grants is
// fine, replacing them is not.
func TestSetRolePermissions_DB_PresetIsImmutable(t *testing.T) {
	pool, svc := roleAdminSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), roleAdminTenant)

	_, err := svc.SetRolePermissions(ctx, presetID(t, pool, "extern"), []auth.RoleGrant{
		{Key: "dashboard:module:view", Scope: "all"},
	})
	assert.ErrorIs(t, err, auth.ErrRolePresetImmutable)
}

// TestSetRolePermissions_DB_UnknownKeyRejected proves an unrecognized
// capability key fails the whole write instead of being silently dropped —
// the done_when this unit exists for. Every other key in the same call must
// also not apply: a partial write here would let the builder believe it
// saved the valid keys while the invalid one vanished unremarked.
func TestSetRolePermissions_DB_UnknownKeyRejected(t *testing.T) {
	pool, svc := roleAdminSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), roleAdminTenant)

	role, err := svc.CreateRole(ctx, roleAdminTenant, auth.CreateRoleInput{
		Name: "Geschuetzt", BasedOn: presetID(t, pool, "extern"),
	})
	require.NoError(t, err)
	before := grantsOf(t, pool, role.ID)

	_, err = svc.SetRolePermissions(ctx, role.ID, []auth.RoleGrant{
		{Key: "dashboard:module:view", Scope: "all"},
		{Key: "not:a:real:permission", Scope: "all"},
	})
	assert.ErrorIs(t, err, auth.ErrCapabilityKeyUnknown)
	assert.Equal(t, before, grantsOf(t, pool, role.ID), "a rejected write must leave the previous grants untouched")
}

// TestSetRolePermissions_DB_InvalidScopeRejected proves the service rejects
// an out-of-range scope itself rather than letting it reach the database: a
// value longer than the scope column's VARCHAR(8) would otherwise surface as
// a raw length-error 500 instead of the 422 an invalid grant deserves.
func TestSetRolePermissions_DB_InvalidScopeRejected(t *testing.T) {
	pool, svc := roleAdminSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), roleAdminTenant)

	role, err := svc.CreateRole(ctx, roleAdminTenant, auth.CreateRoleInput{
		Name: "Randfall", BasedOn: presetID(t, pool, "extern"),
	})
	require.NoError(t, err)
	before := grantsOf(t, pool, role.ID)

	_, err = svc.SetRolePermissions(ctx, role.ID, []auth.RoleGrant{
		{Key: "dashboard:module:view", Scope: "not_a_real_scope"},
	})
	assert.ErrorIs(t, err, auth.ErrCapabilityKeyUnknown)
	assert.Equal(t, before, grantsOf(t, pool, role.ID))
}

// TestSetRolePermissions_DB_FailureMidwayLeavesOldGrantsStanding is the
// transactionality proof the done_when calls for: it drives the failure
// through the repository directly so the DELETE half of delete-then-insert
// actually runs before the constraint on scope aborts the INSERT — a failure
// the service-level key check alone (which runs before any write) cannot
// exercise. role_permissions_scope_check (migration 000256) is the backstop
// that fires here.
func TestSetRolePermissions_DB_FailureMidwayLeavesOldGrantsStanding(t *testing.T) {
	pool, svc := roleAdminSetup(t)
	repo := auth.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), roleAdminTenant)

	role, err := svc.CreateRole(ctx, roleAdminTenant, auth.CreateRoleInput{
		Name: "Unversehrt", BasedOn: presetID(t, pool, "extern"),
	})
	require.NoError(t, err)
	before := grantsOf(t, pool, role.ID)
	require.NotEmpty(t, before)

	_, err = repo.SetRolePermissions(ctx, role.ID, []auth.RoleGrant{
		{Key: "dashboard:module:view", Scope: "invalid"}, // fits scope VARCHAR(8), fails the CHECK
	})
	assert.ErrorIs(t, err, auth.ErrCapabilityKeyUnknown)
	assert.Equal(t, before, grantsOf(t, pool, role.ID),
		"the DELETE that ran before the CHECK violation must have rolled back with it")
}
