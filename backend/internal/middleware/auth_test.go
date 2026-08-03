package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/auth"
)

func newTestTokenMaker() *auth.TokenMaker {
	return auth.NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour)
}

func newTestAuthService() *auth.Service {
	tm := newTestTokenMaker()
	return auth.NewService(nil, tm)
}

func TestAuthMiddleware(t *testing.T) {
	authService := newTestAuthService()
	tm := newTestTokenMaker()
	mw := Auth(authService)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := GetUserID(r.Context())
		assert.NotEmpty(t, userID)
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("valid token", func(t *testing.T) {
		token, _ := tm.CreateAccessToken(uuid.New(), uuid.New().String(), []string{"admin"}, &auth.TokenPermissions{Permissions: []string{"contacts:read"}})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("missing header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("invalid format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "InvalidFormat")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("expired token", func(t *testing.T) {
		expiredTM := auth.NewTokenMaker("test-secret-minimum-32-characters!", -1*time.Minute, 7*24*time.Hour)
		token, _ := expiredTM.CreateAccessToken(uuid.New(), uuid.New().String(), nil, &auth.TokenPermissions{})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("invalid token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer invalid-jwt-token")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

func TestGetUserContext(t *testing.T) {
	authService := newTestAuthService()
	tm := newTestTokenMaker()

	userID := uuid.New()
	tenantID := uuid.New()
	roles := []string{"admin", "manager"}
	perms := []string{"contacts:read", "deals:write"}

	token, _ := tm.CreateAccessToken(userID, tenantID.String(), roles, &auth.TokenPermissions{Permissions: perms})

	handler := Auth(authService)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, userID.String(), GetUserID(r.Context()))
		assert.Equal(t, roles, GetUserRoles(r.Context()))
		assert.Equal(t, perms, GetUserPermissions(r.Context()))

		tid, err := GetTenantID(r.Context())
		require.NoError(t, err)
		assert.Equal(t, tenantID, tid)

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestGetTenantID_WithValidTid verifies that GetTenantID returns a valid UUID when tid is set.
func TestGetTenantID_WithValidTid(t *testing.T) {
	authService := newTestAuthService()
	tm := newTestTokenMaker()
	tenantID := uuid.New()

	token, err := tm.CreateAccessToken(uuid.New(), tenantID.String(), nil, &auth.TokenPermissions{})
	require.NoError(t, err)

	var gotTenantID uuid.UUID
	var gotErr error

	handler := Auth(authService)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTenantID, gotErr = GetTenantID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.NoError(t, gotErr)
	assert.Equal(t, tenantID, gotTenantID)
}

// TestGetTenantID_WithEmptyTid verifies that GetTenantID returns an error for legacy tokens
// without a tid claim. The middleware must not substitute a default tenant.
func TestGetTenantID_WithEmptyTid(t *testing.T) {
	authService := newTestAuthService()
	tm := newTestTokenMaker()

	// Issue a token with an explicitly empty tid (simulates legacy / pre-migration token)
	token, err := tm.CreateAccessToken(uuid.New(), "", nil, &auth.TokenPermissions{})
	require.NoError(t, err)

	var gotErr error

	handler := Auth(authService)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, gotErr = GetTenantID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Error(t, gotErr, "GetTenantID must return an error for empty tid (fail-closed)")
}
