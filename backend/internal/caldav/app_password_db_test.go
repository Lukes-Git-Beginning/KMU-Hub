package caldav

// DB-backed coverage for AppPasswordService.Validate (the Basic-Auth check every
// CalDAV/CardDAV request runs through), Revoke/List error paths, the org-wide
// enable toggle, and UserPreferenceRepository (per-user toggle, deprovisioning
// via RevokeAllUserPasswords). Complements app_password_test.go (pure mock-repo
// unit tests) and tenant_write_test.go (tenant isolation on the write path).
//
// Note: AppSpecificPassword has no expiry field anywhere in the schema
// (migrations 000049/000050) or the model (models/caldav.go) — app passwords
// are valid until explicitly revoked, the same model GitHub uses for personal
// access tokens without an expiry set. "Abgelaufenes Passwort" from the
// backlog scope has no corresponding state to test; revoked is the closest
// and is covered thoroughly below.

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

// seedCalDAVUser creates a tenant + user row and returns a tenant-scoped
// context alongside the tenant and user IDs. The user row is cleaned up via
// t.Cleanup.
func seedCalDAVUser(t *testing.T, pool *pgxpool.Pool) (context.Context, uuid.UUID, uuid.UUID) {
	t.Helper()
	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "CalDAV App Password Test Tenant")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("caldav-apppw-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })

	return testutil.WithTenantCtx(context.Background(), tenantID), tenantID, userID
}

// withOrgEnabledRestored snapshots the current caldav_settings org toggle and
// restores it via t.Cleanup. caldav_settings is a single global row (not
// tenant-scoped), so every test that flips it must restore it afterwards.
func withOrgEnabledRestored(t *testing.T, svc *AppPasswordService) {
	t.Helper()
	original, err := svc.IsOrgEnabled(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, svc.SetOrgEnabled(context.Background(), original))
	})
}

// TestAppPasswordService_Validate_DBBacked exercises the real Basic-Auth
// credential check end to end: org toggle, username parsing, wrong password,
// correct password (and its last_used_at side effect), and revocation.
// Deliberately not t.Parallel() — it mutates the shared caldav_settings row.
func TestAppPasswordService_Validate_DBBacked(t *testing.T) {
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() }) // registered first: closes last, after row/setting cleanups

	svc := NewAppPasswordService(NewPostgresAppPasswordRepository(pool), pool)
	withOrgEnabledRestored(t, svc)

	ctx, tenantID, userID := seedCalDAVUser(t, pool)

	t.Run("org disabled rejects before any credential check", func(t *testing.T) {
		require.NoError(t, svc.SetOrgEnabled(context.Background(), false))
		_, err := svc.Validate(context.Background(), userID.String(), "irrelevant")
		assert.ErrorIs(t, err, ErrCalDAVDisabled)
	})

	require.NoError(t, svc.SetOrgEnabled(context.Background(), true))

	t.Run("non-UUID username rejected", func(t *testing.T) {
		_, err := svc.Validate(context.Background(), "not-a-uuid", "irrelevant")
		assert.ErrorIs(t, err, ErrInvalidCredentials)
	})

	t.Run("unknown user with no passwords rejected", func(t *testing.T) {
		_, err := svc.Validate(context.Background(), uuid.New().String(), "irrelevant")
		assert.ErrorIs(t, err, ErrInvalidCredentials)
	})

	plaintext, pw, err := svc.Create(ctx, userID, tenantID, "Test MacOS Calendar")
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "app_specific_passwords", pw.ID) })

	t.Run("wrong password rejected", func(t *testing.T) {
		_, err := svc.Validate(context.Background(), userID.String(), plaintext+"-wrong")
		assert.ErrorIs(t, err, ErrInvalidCredentials)
	})

	t.Run("correct password accepted and updates last_used_at", func(t *testing.T) {
		got, err := svc.Validate(context.Background(), userID.String(), plaintext)
		require.NoError(t, err)
		assert.Equal(t, userID, got)

		list, err := svc.List(ctx, userID)
		require.NoError(t, err)
		require.Len(t, list, 1)
		assert.NotNil(t, list[0].LastUsedAt, "Validate must update last_used_at on a successful match")
	})

	t.Run("revoked password rejected", func(t *testing.T) {
		require.NoError(t, svc.Revoke(ctx, pw.ID, userID))
		_, err := svc.Validate(context.Background(), userID.String(), plaintext)
		assert.ErrorIs(t, err, ErrInvalidCredentials)
	})
}

// TestAppPasswordService_Revoke_List_ErrorPaths covers Revoke's not-found and
// wrong-owner cases and List's ordering/visibility of revoked entries.
func TestAppPasswordService_Revoke_List_ErrorPaths(t *testing.T) {
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() }) // registered first: closes last, after row/setting cleanups

	svc := NewAppPasswordService(NewPostgresAppPasswordRepository(pool), pool)
	ctx, tenantID, userID := seedCalDAVUser(t, pool)

	t.Run("revoke unknown password id", func(t *testing.T) {
		err := svc.Revoke(ctx, uuid.New(), userID)
		assert.ErrorIs(t, err, ErrPasswordNotFound)
	})

	t.Run("revoke with wrong owner user id", func(t *testing.T) {
		_, pw, err := svc.Create(ctx, userID, tenantID, "Owner Mismatch")
		require.NoError(t, err)
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "app_specific_passwords", pw.ID) })

		err = svc.Revoke(ctx, pw.ID, uuid.New())
		assert.ErrorIs(t, err, ErrPasswordNotFound)
	})

	t.Run("List orders by created_at DESC and keeps revoked entries visible", func(t *testing.T) {
		_, pwOld, err := svc.Create(ctx, userID, tenantID, "Old")
		require.NoError(t, err)
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "app_specific_passwords", pwOld.ID) })

		_, pwNew, err := svc.Create(ctx, userID, tenantID, "New")
		require.NoError(t, err)
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "app_specific_passwords", pwNew.ID) })

		require.NoError(t, svc.Revoke(ctx, pwOld.ID, userID))

		list, err := svc.List(ctx, userID)
		require.NoError(t, err)
		require.Len(t, list, 2)
		assert.Equal(t, pwNew.ID, list[0].ID, "List must order by created_at DESC (newest first)")
		assert.Equal(t, pwOld.ID, list[1].ID)
		assert.Nil(t, list[0].RevokedAt)
		assert.NotNil(t, list[1].RevokedAt)
	})
}

// TestAppPasswordService_OrgEnabledToggle_Roundtrip covers IsOrgEnabled/
// SetOrgEnabled directly. Not t.Parallel() — shares the caldav_settings row
// with TestAppPasswordService_Validate_DBBacked.
func TestAppPasswordService_OrgEnabledToggle_Roundtrip(t *testing.T) {
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() }) // registered first: closes last, after row/setting cleanups

	svc := NewAppPasswordService(nil, pool) // repo unused by IsOrgEnabled/SetOrgEnabled
	withOrgEnabledRestored(t, svc)

	require.NoError(t, svc.SetOrgEnabled(context.Background(), true))
	enabled, err := svc.IsOrgEnabled(context.Background())
	require.NoError(t, err)
	assert.True(t, enabled)

	require.NoError(t, svc.SetOrgEnabled(context.Background(), false))
	enabled, err = svc.IsOrgEnabled(context.Background())
	require.NoError(t, err)
	assert.False(t, enabled)
}

// TestUserPreferenceRepository_GetSetCalDAVEnabled_Roundtrip covers the
// per-user toggle and its unknown-user error path.
func TestUserPreferenceRepository_GetSetCalDAVEnabled_Roundtrip(t *testing.T) {
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() }) // registered first: closes last, after row/setting cleanups

	repo := NewPostgresUserPreferenceRepository(pool)
	_, _, userID := seedCalDAVUser(t, pool)

	t.Run("unknown user returns error", func(t *testing.T) {
		_, err := repo.GetCalDAVEnabled(context.Background(), uuid.New())
		assert.Error(t, err)
	})

	enabled, err := repo.GetCalDAVEnabled(context.Background(), userID)
	require.NoError(t, err)
	assert.False(t, enabled, "caldav_enabled defaults to false")

	require.NoError(t, repo.SetCalDAVEnabled(context.Background(), userID, true))

	enabled, err = repo.GetCalDAVEnabled(context.Background(), userID)
	require.NoError(t, err)
	assert.True(t, enabled)

	t.Run("ListCalDAVUsers includes the newly enabled user", func(t *testing.T) {
		users, err := repo.ListCalDAVUsers(context.Background())
		require.NoError(t, err)

		found := false
		for _, u := range users {
			if u.ID == userID.String() {
				found = true
				assert.True(t, u.CalDAVEnabled)
			}
		}
		assert.True(t, found, "ListCalDAVUsers must include the newly enabled user")
	})

	require.NoError(t, repo.SetCalDAVEnabled(context.Background(), userID, false))
	enabled, err = repo.GetCalDAVEnabled(context.Background(), userID)
	require.NoError(t, err)
	assert.False(t, enabled)
}

// TestUserPreferenceRepository_RevokeAllUserPasswords_InvalidatesValidate is
// the deprovisioning path: after a user is disabled, none of their existing
// app passwords may still authenticate. Verified through Validate, not just
// a row count, per the backlog's explicit demand.
func TestUserPreferenceRepository_RevokeAllUserPasswords_InvalidatesValidate(t *testing.T) {
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() }) // registered first: closes last, after row/setting cleanups

	appSvc := NewAppPasswordService(NewPostgresAppPasswordRepository(pool), pool)
	prefRepo := NewPostgresUserPreferenceRepository(pool)
	withOrgEnabledRestored(t, appSvc)
	require.NoError(t, appSvc.SetOrgEnabled(context.Background(), true))

	ctx, tenantID, userID := seedCalDAVUser(t, pool)

	plaintextA, pwA, err := appSvc.Create(ctx, userID, tenantID, "Password A")
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "app_specific_passwords", pwA.ID) })

	plaintextB, pwB, err := appSvc.Create(ctx, userID, tenantID, "Password B")
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "app_specific_passwords", pwB.ID) })

	// Sanity: both work before revocation.
	_, err = appSvc.Validate(context.Background(), userID.String(), plaintextA)
	require.NoError(t, err)
	_, err = appSvc.Validate(context.Background(), userID.String(), plaintextB)
	require.NoError(t, err)

	affected, err := prefRepo.RevokeAllUserPasswords(context.Background(), userID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), affected)

	_, err = appSvc.Validate(context.Background(), userID.String(), plaintextA)
	assert.ErrorIs(t, err, ErrInvalidCredentials)
	_, err = appSvc.Validate(context.Background(), userID.String(), plaintextB)
	assert.ErrorIs(t, err, ErrInvalidCredentials)

	// Idempotent: nothing left to revoke.
	affected, err = prefRepo.RevokeAllUserPasswords(context.Background(), userID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), affected)
}
