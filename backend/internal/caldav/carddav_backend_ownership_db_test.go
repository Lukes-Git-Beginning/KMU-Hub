package caldav

// DB-backed coverage for checkPersonalContactOwnership, the guard
// DeleteAddressObject runs before deleting a "personal" contact. Unlike
// checkCompanyContactPermission (already covered in
// carddav_backend_permission_db_test.go), this reads the `contacts` table
// directly by ID -- no RBAC tables involved -- so the sysctx.With(ctx) bypass
// documented in its own comment is sufficient and does not carry the same
// cross-tenant risk callers of checkCompanyContactPermission would (the
// caller already knows the contact ID from the request path).

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestCheckPersonalContactOwnership_Owner_Allowed(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "CardDAV Ownership Test Tenant")
	ownerID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("carddav-own-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", ownerID) })
	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantID,
		"first_name": "Own", "last_name": "Contact",
		"owner_id": ownerID, "visibility": "personal", "created_by": ownerID,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "contacts", contactID) })

	backend := &CardDAVBackend{pool: pool}
	path := "/carddav/principals/" + ownerID.String() + "/addressbooks/personal/" + contactID.String() + ".vcf"

	err := backend.checkPersonalContactOwnership(context.Background(), ownerID, path)

	assert.NoError(t, err)
}

func TestCheckPersonalContactOwnership_DifferentUser_Forbidden(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "CardDAV Ownership Test Tenant")
	ownerID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("carddav-own-owner-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", ownerID) })
	strangerID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("carddav-own-stranger-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", strangerID) })
	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantID,
		"first_name": "Own", "last_name": "Contact",
		"owner_id": ownerID, "visibility": "personal", "created_by": ownerID,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "contacts", contactID) })

	backend := &CardDAVBackend{pool: pool}
	path := "/carddav/principals/" + strangerID.String() + "/addressbooks/personal/" + contactID.String() + ".vcf"

	err := backend.checkPersonalContactOwnership(context.Background(), strangerID, path)

	assert.Equal(t, 403, webdavStatusCode(t, err))
}

func TestCheckPersonalContactOwnership_ContactNotFound_Returns404(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	userID := uuid.New()
	nonexistentContactID := uuid.New()
	backend := &CardDAVBackend{pool: pool}
	path := "/carddav/principals/" + userID.String() + "/addressbooks/personal/" + nonexistentContactID.String() + ".vcf"

	err := backend.checkPersonalContactOwnership(context.Background(), userID, path)

	assert.Equal(t, 404, webdavStatusCode(t, err))
}

func TestCheckPersonalContactOwnership_InvalidPath_Returns400(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	backend := &CardDAVBackend{pool: pool}

	err := backend.checkPersonalContactOwnership(context.Background(), uuid.New(), "/carddav/principals/u1/addressbooks/personal/not-a-uuid.vcf")

	assert.Equal(t, 400, webdavStatusCode(t, err))
}
