package gdpr

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// ---------------------------------------------------------------------------
// Integration tests (require DATABASE_URL)
// These tests verify real SQL against the dev Postgres instance.
// ---------------------------------------------------------------------------

func TestAuthExportHandler_ExportUserData_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")

	// Seed a minimal user
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         "auth-export-test-" + uuid.New().String()[:8] + "@tenanta.local",
		"password_hash": "$argon2id$v=19$test",
		"first_name":    "Test",
		"last_name":     "User",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	// RLS: handlers run with the tenant-stamped context the gRPC interceptor provides in production.
	ctx := testutil.WithTenantCtx(context.Background(), testutil.TenantA)

	h := NewAuthExportHandler(pool)
	data, err := h.ExportUserData(ctx, userID)

	require.NoError(t, err)
	require.NotEmpty(t, data)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &result))

	profile, ok := result["profile"].(map[string]interface{})
	require.True(t, ok, "export should contain profile object")
	assert.Equal(t, userID.String(), profile["id"])
	assert.Equal(t, "Test", profile["first_name"])

	// Sessions/tokens should be present as arrays (empty for new user)
	sessions, ok := result["sessions"]
	assert.True(t, ok, "export should contain sessions key")
	_ = sessions
}

func TestAuthExportHandler_CrossTenant_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	// Seed a user in TenantA
	userA := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         "cross-tenant-export-" + uuid.New().String()[:8] + "@tenanta.local",
		"password_hash": "$argon2id$v=19$test",
		"first_name":    "TenantA",
		"last_name":     "User",
	})
	defer testutil.CleanupRow(t, pool, "users", userA)

	// Seed a session for userA
	sessionID := testutil.SeedRow(t, pool, "user_sessions", map[string]any{
		"tenant_id":   testutil.TenantA,
		"user_id":     userA,
		"device_name": "Test Device",
		"device_type": "desktop",
	})
	defer testutil.CleanupRow(t, pool, "user_sessions", sessionID)

	h := NewAuthExportHandler(pool)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	// Export data for userA in tenant A -- should succeed
	dataA, err := h.ExportUserData(ctxA, userA)
	require.NoError(t, err)
	require.NotEmpty(t, dataA)

	var resultA map[string]interface{}
	require.NoError(t, json.Unmarshal(dataA, &resultA))
	sessions := resultA["sessions"].([]interface{})
	assert.GreaterOrEqual(t, len(sessions), 1, "userA should see their own session")

	// Export userA under tenant B context -- RLS must hide the user entirely
	_, err = h.ExportUserData(ctxB, userA)
	assert.Error(t, err, "export across tenants must fail (RLS)")

	// Attempt to export a random user that doesn't exist -- should return error
	_, err = h.ExportUserData(ctxA, uuid.New())
	assert.Error(t, err, "export for non-existent user should return error")
}

func TestAuthErasureHandler_ExecuteErasure_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         "auth-erasure-test-" + uuid.New().String()[:8] + "@tenanta.local",
		"password_hash": "$argon2id$v=19$test",
		"first_name":    "Erase",
		"last_name":     "Me",
	})
	// Note: no defer CleanupRow -- the user will be anonymized in place

	// Seed a session
	sessionID := testutil.SeedRow(t, pool, "user_sessions", map[string]any{
		"tenant_id":   testutil.TenantA,
		"user_id":     userID,
		"device_name": "Phone",
		"device_type": "mobile",
	})
	defer testutil.CleanupRow(t, pool, "user_sessions", sessionID)

	ctx := testutil.WithTenantCtx(context.Background(), testutil.TenantA)

	h := NewAuthErasureHandler(pool)
	n, err := h.ExecuteErasure(ctx, userID, "Geloeschter Benutzer #1", ErasureAnonymize)
	require.NoError(t, err)
	assert.Greater(t, n, 0, "should report at least 1 affected record")

	// Verify user is anonymized (email ends in @deleted.invalid, is_active = false)
	var email string
	var isActive bool
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT email, is_active FROM users WHERE id = $1`, userID,
	).Scan(&email, &isActive))

	assert.Contains(t, email, "@deleted.invalid")
	assert.False(t, isActive)

	// Cleanup anonymized user
	testutil.CleanupRow(t, pool, "users", userID)
}

func TestNotificationErasureHandler_ExecuteErasure_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         "notif-erasure-" + uuid.New().String()[:8] + "@tenanta.local",
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	// Seed a notification preference
	prefID := testutil.SeedRow(t, pool, "notification_preferences", map[string]any{
		"tenant_id":    testutil.TenantA,
		"user_id":      userID,
		"module_id":    "crm",
		"in_app":       true,
		"desktop_push": true,
	})
	defer testutil.CleanupRow(t, pool, "notification_preferences", prefID)

	ctx := testutil.WithTenantCtx(context.Background(), testutil.TenantA)

	h := NewNotificationErasureHandler(pool)
	n, err := h.ExecuteErasure(ctx, userID, "Geloeschter Benutzer #1", ErasureDelete)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, n, 1, "should delete at least 1 preference record")

	// Verify preference is gone
	var count int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM notification_preferences WHERE user_id = $1`, userID,
	).Scan(&count))
	assert.Equal(t, 0, count, "notification preferences should be deleted")
}

func TestWorkErasureHandler_Preview_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         "work-preview-" + uuid.New().String()[:8] + "@tenanta.local",
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	ctx := testutil.WithTenantCtx(context.Background(), testutil.TenantA)

	h := NewWorkErasureHandler(pool)
	preview, err := h.PreviewErasure(ctx, userID)

	require.NoError(t, err)
	assert.Equal(t, "work", preview.ModuleName)
	assert.Equal(t, "anonymize", preview.Action)
	assert.GreaterOrEqual(t, preview.RecordCount, 0)
}

// ---------------------------------------------------------------------------
// Unit tests: handler constructors and module names
// ---------------------------------------------------------------------------

func TestHandlerModuleNames(t *testing.T) {
	// Verify all handlers return correct module names (nil pool OK for name-only test)
	handlers := []DataExportHandler{
		NewAuthExportHandler(nil),
		NewCRMExportHandler(nil),
		NewChatExportHandler(nil),
		NewWorkExportHandler(nil),
		NewCalendarExportHandler(nil),
		NewSessionExportHandler(nil),
		NewNotificationExportHandler(nil),
	}

	expected := []string{"auth", "crm", "chat", "work", "calendar", "sessions", "notifications"}
	for i, h := range handlers {
		assert.Equal(t, expected[i], h.ModuleName(), "unexpected module name at index %d", i)
	}
}

func TestErasureHandlerModuleNames(t *testing.T) {
	handlers := []ErasureHandler{
		NewAuthErasureHandler(nil),
		NewCRMErasureHandler(nil),
		NewChatErasureHandler(nil),
		NewWorkErasureHandler(nil),
		NewCalendarErasureHandler(nil),
		NewNotificationErasureHandler(nil),
		&AuditErasureHandler{},
	}

	expected := []string{"auth", "crm", "chat", "work", "calendar", "notifications", "audit"}
	for i, h := range handlers {
		assert.Equal(t, expected[i], h.ModuleName(), "unexpected module name at index %d", i)
	}
}

func TestAuditErasureHandler_NoOp(t *testing.T) {
	h := &AuditErasureHandler{}
	userID := uuid.New()

	preview, err := h.PreviewErasure(context.Background(), userID)
	require.NoError(t, err)
	assert.Equal(t, "audit", preview.ModuleName)
	assert.Equal(t, "retain", preview.Action)
	assert.Equal(t, 0, preview.RecordCount)

	n, err := h.ExecuteErasure(context.Background(), userID, "label", ErasureRetain)
	require.NoError(t, err)
	assert.Equal(t, 0, n)
}
