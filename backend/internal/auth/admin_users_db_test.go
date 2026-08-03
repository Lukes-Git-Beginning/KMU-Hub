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

// The roster merges two RLS-scoped sources (users, invitations) plus a
// tenant-unscoped join (user_roles, user_sessions) — every failure mode here
// is a database property (cross-tenant leak, a JOIN losing the GROUP BY
// functional dependency), so this runs against the real database with its
// own tenants rather than the shared testutil.TenantA/B.
var (
	adminRosterTenant        = uuid.MustParse("40123b00-0000-4000-8000-000000000001")
	adminRosterForeignTenant = uuid.MustParse("40123b00-0000-4000-8000-000000000002")
)

func adminRosterSetup(t *testing.T) (*pgxpool.Pool, *auth.Service) {
	t.Helper()
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	testutil.EnsureTenant(t, pool, adminRosterTenant, "AdminRosterTenant")
	testutil.EnsureTenant(t, pool, adminRosterForeignTenant, "AdminRosterForeignTenant")

	svc := auth.NewService(
		auth.NewPostgresRepository(pool),
		auth.NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour),
	)
	return pool, svc
}

func adminRosterPresetID(t *testing.T, pool *pgxpool.Pool, name string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT id FROM roles WHERE name = $1 AND tenant_id IS NULL`, name).Scan(&id)
	require.NoErrorf(t, err, "preset %q missing — migration 000256 not applied?", name)
	return id
}

func findAdminUser(users []auth.AdminUser, id uuid.UUID) *auth.AdminUser {
	for i := range users {
		if users[i].ID == id {
			return &users[i]
		}
	}
	return nil
}

// TestListAdminUsers_DB_MergesAccountsAndInvitations covers the roster's
// whole point: an active account, a deactivated one, and a still-open
// invitation all show up as one list, each with the right status and role.
func TestListAdminUsers_DB_MergesAccountsAndInvitations(t *testing.T) {
	pool, svc := adminRosterSetup(t)
	ctx := testutil.WithTenantCtx(context.Background(), adminRosterTenant)
	adminPreset := adminRosterPresetID(t, pool, "admin")
	memberPreset := adminRosterPresetID(t, pool, "member")

	activeID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     adminRosterTenant,
		"email":         "roster-active@test.local",
		"password_hash": "x",
		"first_name":    "Active",
		"last_name":     "One",
		"is_active":     true,
	})
	defer testutil.CleanupRow(t, pool, "users", activeID)
	// user_roles has no surrogate id (composite PK user_id+role_id), so
	// SeedRow's `RETURNING id` does not apply — plain insert under system
	// context, same as roleAdminSetup's seedRoleAdminActor.
	_, err := pool.Exec(testutil.WithSystemCtx(context.Background()),
		`INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)`, activeID, adminPreset)
	require.NoError(t, err)

	loginAt := time.Now().Add(-2 * time.Hour).Truncate(time.Second)
	sessionID := testutil.SeedRow(t, pool, "user_sessions", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  adminRosterTenant,
		"user_id":    activeID,
		"created_at": loginAt,
	})
	defer testutil.CleanupRow(t, pool, "user_sessions", sessionID)

	deactivatedID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     adminRosterTenant,
		"email":         "roster-deactivated@test.local",
		"password_hash": "x",
		"first_name":    "Deactivated",
		"last_name":     "Two",
		"is_active":     false,
	})
	defer testutil.CleanupRow(t, pool, "users", deactivatedID)

	inviterID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     adminRosterTenant,
		"email":         "roster-inviter@test.local",
		"password_hash": "x",
		"first_name":    "Inviter",
		"last_name":     "Three",
	})
	defer testutil.CleanupRow(t, pool, "users", inviterID)

	invID := testutil.SeedRow(t, pool, "invitations", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  adminRosterTenant,
		"email":      "roster-invited@test.local",
		"role":       "member",
		"token_hash": uuid.NewString(),
		"created_by": inviterID,
		"expires_at": time.Now().Add(24 * time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "invitations", invID)

	expiredInvID := testutil.SeedRow(t, pool, "invitations", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  adminRosterTenant,
		"email":      "roster-expired@test.local",
		"role":       "member",
		"token_hash": uuid.NewString(),
		"created_by": inviterID,
		"expires_at": time.Now().Add(-24 * time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "invitations", expiredInvID)

	users, err := svc.ListAdminUsers(ctx)
	require.NoError(t, err)

	active := findAdminUser(users, activeID)
	require.NotNil(t, active, "active account must be in the roster")
	assert.Equal(t, auth.AdminUserStatusActive, active.Status)
	assert.Contains(t, active.RoleIDs, adminPreset.String())
	require.NotNil(t, active.LastLoginAt, "active account has a session, lastLoginAt must not be nil")
	assert.WithinDuration(t, loginAt, *active.LastLoginAt, time.Second)
	assert.Nil(t, active.InvitedAt)

	deactivated := findAdminUser(users, deactivatedID)
	require.NotNil(t, deactivated, "deactivated account must be in the roster")
	assert.Equal(t, auth.AdminUserStatusDeactivated, deactivated.Status)
	assert.Nil(t, deactivated.LastLoginAt, "no session was seeded for this account")

	invited := findAdminUser(users, invID)
	require.NotNil(t, invited, "open invitation must be in the roster")
	assert.Equal(t, auth.AdminUserStatusInvited, invited.Status)
	assert.Empty(t, invited.FirstName, "invitations carry no name")
	assert.Empty(t, invited.LastName, "invitations carry no name")
	assert.Equal(t, []string{memberPreset.String()}, invited.RoleIDs)
	require.NotNil(t, invited.InvitedAt)
	assert.Nil(t, invited.LastLoginAt)

	assert.Nil(t, findAdminUser(users, expiredInvID), "an expired invitation is not an open account")
}

// TestListAdminUsers_DB_TenantIsolation is the RLS half: a foreign tenant's
// accounts and invitations must not leak into the roster.
func TestListAdminUsers_DB_TenantIsolation(t *testing.T) {
	pool, svc := adminRosterSetup(t)

	foreignUserID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     adminRosterForeignTenant,
		"email":         "roster-foreign-user@test.local",
		"password_hash": "x",
		"first_name":    "Foreign",
		"last_name":     "User",
	})
	defer testutil.CleanupRow(t, pool, "users", foreignUserID)

	foreignInviterID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     adminRosterForeignTenant,
		"email":         "roster-foreign-inviter@test.local",
		"password_hash": "x",
		"first_name":    "Foreign",
		"last_name":     "Inviter",
	})
	defer testutil.CleanupRow(t, pool, "users", foreignInviterID)

	foreignInvID := testutil.SeedRow(t, pool, "invitations", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  adminRosterForeignTenant,
		"email":      "roster-foreign-invited@test.local",
		"role":       "member",
		"token_hash": uuid.NewString(),
		"created_by": foreignInviterID,
		"expires_at": time.Now().Add(24 * time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "invitations", foreignInvID)

	ownID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     adminRosterTenant,
		"email":         "roster-own-user@test.local",
		"password_hash": "x",
		"first_name":    "Own",
		"last_name":     "User",
	})
	defer testutil.CleanupRow(t, pool, "users", ownID)

	ctx := testutil.WithTenantCtx(context.Background(), adminRosterTenant)
	users, err := svc.ListAdminUsers(ctx)
	require.NoError(t, err)

	assert.NotNil(t, findAdminUser(users, ownID), "own tenant's account must be visible")
	assert.Nil(t, findAdminUser(users, foreignUserID), "foreign tenant's account must not leak")
	assert.Nil(t, findAdminUser(users, foreignInvID), "foreign tenant's invitation must not leak")
}
