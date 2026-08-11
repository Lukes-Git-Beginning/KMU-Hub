package caldav

// DB-backed coverage for checkCompanyContactPermission. It used to select a
// nonexistent users.role column and fail-closed to 403 for every user; the
// fix routes through the real RBAC tables (user_roles/role_permissions/
// permissions) via auth.PostgresRepository.UserHasPermission. These tests
// prove the intended admin/manager-can-write, member-cannot-write behaviour
// against the real schema, not just that the query no longer errors.

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// seedCardDAVUserWithSystemRole creates a tenant + user and assigns the given
// system role (tenant_id IS NULL, e.g. "admin"/"manager"/"member") to it.
func seedCardDAVUserWithSystemRole(t *testing.T, pool *pgxpool.Pool, roleName string) uuid.UUID {
	t.Helper()
	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "CardDAV Permission Test Tenant")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("carddav-perm-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })

	systemCtx := testutil.WithSystemCtx(context.Background())

	var roleID uuid.UUID
	err := pool.QueryRow(systemCtx,
		`SELECT id FROM roles WHERE name = $1 AND tenant_id IS NULL`, roleName,
	).Scan(&roleID)
	require.NoError(t, err, "system role %q must exist (migration 000002)", roleName)

	_, err = pool.Exec(systemCtx,
		`INSERT INTO user_roles (user_id, role_id, tenant_id) VALUES ($1, $2, $3)`, userID, roleID, tenantID)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(systemCtx,
			`DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2`, userID, roleID)
	})

	return userID
}

func TestCheckCompanyContactPermission_AdminAllowed(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	backend := &CardDAVBackend{pool: pool}
	userID := seedCardDAVUserWithSystemRole(t, pool, "admin")

	err := backend.checkCompanyContactPermission(context.Background(), userID)

	assert.NoError(t, err)
}

func TestCheckCompanyContactPermission_ManagerAllowed(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	backend := &CardDAVBackend{pool: pool}
	userID := seedCardDAVUserWithSystemRole(t, pool, "manager")

	err := backend.checkCompanyContactPermission(context.Background(), userID)

	assert.NoError(t, err)
}

func TestCheckCompanyContactPermission_MemberForbidden(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	backend := &CardDAVBackend{pool: pool}
	userID := seedCardDAVUserWithSystemRole(t, pool, "member")

	err := backend.checkCompanyContactPermission(context.Background(), userID)

	assert.Equal(t, 403, webdavStatusCode(t, err))
}

func TestCheckCompanyContactPermission_NoRoleForbidden(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	backend := &CardDAVBackend{pool: pool}
	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "CardDAV Permission Test Tenant No Role")
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("carddav-perm-norole-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })

	err := backend.checkCompanyContactPermission(context.Background(), userID)

	assert.Equal(t, 403, webdavStatusCode(t, err))
}
