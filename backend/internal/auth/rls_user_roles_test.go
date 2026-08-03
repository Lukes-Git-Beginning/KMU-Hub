package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// user_roles has a composite PK (user_id, role_id) and no surrogate id, so
// none of testutil's id-based helpers (SeedRow, AssertRowCount) apply — every
// assertion here is a hand-written COUNT, mirroring rls_refresh_tokens_test.go.
//
// Own tenants for the name-collision test (a custom role literally named
// "member" would pollute a shared tenant for every other suite that also
// seeds the "member" preset there); the two plain isolation tests below reuse
// testutil.TenantA/B like refresh_tokens' equivalents since they seed no
// tenant-unique-named row.
var (
	userRolesCollisionTenant      = uuid.MustParse("00028600-0000-4000-8000-000000000001")
	userRolesCollisionOtherTenant = uuid.MustParse("00028600-0000-4000-8000-000000000002")
)

// TestRLS_UserRoles_TenantBSeesNoTenantARoles verifies the standard
// tenant_isolation policy added in migration 000286.
func TestRLS_UserRoles_TenantBSeesNoTenantARoles(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")
	sysCtx := testutil.WithSystemCtx(context.Background())

	ownerID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "ur-owner-a@test.local",
		"password_hash": "x",
		"first_name":    "UR",
		"last_name":     "OwnerA",
	})
	defer testutil.CleanupRow(t, pool, "users", ownerID)

	var memberPreset uuid.UUID
	require.NoError(t, pool.QueryRow(sysCtx,
		`SELECT id FROM roles WHERE name = 'member' AND tenant_id IS NULL`).Scan(&memberPreset))

	_, err := pool.Exec(sysCtx,
		`INSERT INTO user_roles (user_id, role_id, tenant_id) VALUES ($1, $2, $3)`,
		ownerID, memberPreset, testutil.TenantA)
	require.NoError(t, err)
	defer func() {
		_, _ = pool.Exec(sysCtx, `DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2`, ownerID, memberPreset)
	}()

	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)
	var count int
	require.NoError(t, pool.QueryRow(ctxB,
		`SELECT COUNT(*) FROM user_roles WHERE user_id = $1`, ownerID).Scan(&count))
	require.Zero(t, count, "TenantB must not see TenantA's role assignment")
}

// TestUserRolesCrossTenantStampedWrite_Rejected mirrors
// TestRefreshTokens_CrossTenantStampedWrite_Rejected: a row stamped for
// TenantB but written on TenantA's connection must fail the policy's WITH
// CHECK, the same guarantee AssignUserRole's users/roles joins already give —
// this is the backstop for a caller that reaches user_roles directly.
func TestUserRolesCrossTenantStampedWrite_Rejected(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")
	sysCtx := testutil.WithSystemCtx(context.Background())

	ownerID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "ur-owner-a2@test.local",
		"password_hash": "x",
		"first_name":    "UR2",
		"last_name":     "OwnerA",
	})
	defer testutil.CleanupRow(t, pool, "users", ownerID)

	var memberPreset uuid.UUID
	require.NoError(t, pool.QueryRow(sysCtx,
		`SELECT id FROM roles WHERE name = 'member' AND tenant_id IS NULL`).Scan(&memberPreset))

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	_, err := pool.Exec(ctxA,
		`INSERT INTO user_roles (user_id, role_id, tenant_id) VALUES ($1, $2, $3)`,
		ownerID, memberPreset, testutil.TenantB)
	if err == nil {
		_, _ = pool.Exec(sysCtx, `DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2`, ownerID, memberPreset)
		t.Fatal("insert with a foreign tenant stamp succeeded; the RLS policy is not in force")
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "42501" {
		t.Fatalf("expected SQLSTATE 42501, got: %v", err)
	}
}

// TestAssignRole_DB_PresetOnlyDespiteNameCollision is the actual bug this
// unit closes. Migration 000256 lets a tenant name a custom role identically
// to a preset ("member" collides with tenant_id IS NULL's "member" because
// idx_roles_tenant_name keys on COALESCE(tenant_id, sentinel), and a real
// tenant_id never equals the sentinel). Before this fix, AssignRole's
// `WHERE r.name = $2` matched both rows and silently enrolled every new
// registrant, of every tenant, into userRolesCollisionTenant's role too — a
// cross-tenant privilege grant reachable from nothing more than Register.
func TestAssignRole_DB_PresetOnlyDespiteNameCollision(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, userRolesCollisionTenant, "URCollisionTenant")
	testutil.EnsureTenant(t, pool, userRolesCollisionOtherTenant, "URCollisionOtherTenant")
	sysCtx := testutil.WithSystemCtx(context.Background())

	// A tenant names its own custom role "member" — legal since migration
	// 000256, and exactly the row AssignRole's old query could not tell apart
	// from the preset.
	collidingRole := testutil.SeedRow(t, pool, "roles", map[string]any{
		"id":          uuid.New(),
		"tenant_id":   userRolesCollisionTenant,
		"name":        "member",
		"description": "custom role deliberately named like the preset",
		"color":       "hsl(0 0% 0%)",
	})
	// t.Cleanup callbacks run after the test function returns — strictly
	// after this function's own `defer pool.Close()` already fired. Plain
	// defers instead, so cleanup runs against a live pool (LIFO, before the
	// close above).
	defer testutil.CleanupRow(t, pool, "roles", collidingRole)

	// The victim: a brand-new registrant in a THIRD, unrelated tenant — the
	// shape of Service.Register, which always assigns "member" by name.
	victim := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     userRolesCollisionOtherTenant,
		"email":         "ur-collision-victim@test.local",
		"password_hash": "x",
		"first_name":    "Collision",
		"last_name":     "Victim",
	})
	defer testutil.CleanupRow(t, pool, "users", victim)

	var memberPreset uuid.UUID
	require.NoError(t, pool.QueryRow(sysCtx,
		`SELECT id FROM roles WHERE name = 'member' AND tenant_id IS NULL`).Scan(&memberPreset))

	repo := auth.NewPostgresRepository(pool)
	// Register wraps the whole call in sysctx.With() — reproduced here since
	// that is exactly the context AssignRole runs under in production, and
	// the one context where user_roles' own RLS cannot help at all.
	require.NoError(t, repo.AssignRole(sysCtx, victim, "member"))
	defer func() {
		_, _ = pool.Exec(sysCtx, `DELETE FROM user_roles WHERE user_id = $1`, victim)
	}()

	rows, err := pool.Query(sysCtx,
		`SELECT role_id, tenant_id FROM user_roles WHERE user_id = $1`, victim)
	require.NoError(t, err)
	defer rows.Close()

	type assignment struct {
		roleID   uuid.UUID
		tenantID uuid.UUID
	}
	var got []assignment
	for rows.Next() {
		var a assignment
		require.NoError(t, rows.Scan(&a.roleID, &a.tenantID))
		got = append(got, a)
	}
	require.NoError(t, rows.Err())

	require.Len(t, got, 1, "the name collision must not fan out into two assignments")
	require.Equal(t, memberPreset, got[0].roleID, "only the system preset may be resolved by name")
	require.NotEqual(t, collidingRole, got[0].roleID, "the colliding tenant-owned role must never be reached")
	require.Equal(t, userRolesCollisionOtherTenant, got[0].tenantID, "the assignment belongs to the victim's own tenant")
}
