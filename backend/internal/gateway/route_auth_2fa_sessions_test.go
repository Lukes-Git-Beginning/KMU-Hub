package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- HandleSetup2FA ---

func TestHandleSetup2FA_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/2fa/setup", nil)
	req = withUserID(req, "user-123")
	routes.HandleSetup2FA(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleVerify2FA ---

func TestHandleVerify2FA_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleVerify2FA)
}

func TestHandleVerify2FA_InvalidJSON(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/2fa/verify", invalidJSON())
	req = withUserID(req, "user-123")
	routes.HandleVerify2FA(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleVerify2FA_MissingCode(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/2fa/verify", jsonBody(t, map[string]interface{}{}))
	req = withUserID(req, "user-123")
	routes.HandleVerify2FA(rec, req)
	assertValidationError(t, rec, "code")
}

func TestHandleVerify2FA_WrongCodeLength(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/2fa/verify", jsonBody(t, map[string]interface{}{
		"code": "12345",
	}))
	req = withUserID(req, "user-123")
	routes.HandleVerify2FA(rec, req)
	assertValidationError(t, rec, "code")
}

// --- HandleValidate2FALogin ---

func TestHandleValidate2FALogin_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleValidate2FALogin)
}

func TestHandleValidate2FALogin_InvalidJSON(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/2fa/validate", invalidJSON())
	routes.HandleValidate2FALogin(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleValidate2FALogin_MissingPendingToken(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/2fa/validate", jsonBody(t, map[string]interface{}{
		"code": "123456",
	}))
	routes.HandleValidate2FALogin(rec, req)
	assertValidationError(t, rec, "pending_token")
}

func TestHandleValidate2FALogin_MissingCode(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/2fa/validate", jsonBody(t, map[string]interface{}{
		"pending_token": "tok-123",
	}))
	routes.HandleValidate2FALogin(rec, req)
	assertValidationError(t, rec, "code")
}

// --- HandleDisable2FA ---

func TestHandleDisable2FA_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleDisable2FA)
}

func TestHandleDisable2FA_InvalidJSON(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/2fa/disable", invalidJSON())
	req = withUserID(req, "user-123")
	routes.HandleDisable2FA(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleDisable2FA_MissingCode(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/2fa/disable", jsonBody(t, map[string]interface{}{}))
	req = withUserID(req, "user-123")
	routes.HandleDisable2FA(rec, req)
	assertValidationError(t, rec, "code")
}

// --- HandleRegenerateRecoveryCodes ---

func TestHandleRegenerateRecoveryCodes_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleRegenerateRecoveryCodes)
}

func TestHandleRegenerateRecoveryCodes_InvalidJSON(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/2fa/regenerate-codes", invalidJSON())
	req = withUserID(req, "user-123")
	routes.HandleRegenerateRecoveryCodes(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleRegenerateRecoveryCodes_MissingCode(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/2fa/regenerate-codes", jsonBody(t, map[string]interface{}{}))
	req = withUserID(req, "user-123")
	routes.HandleRegenerateRecoveryCodes(rec, req)
	assertValidationError(t, rec, "code")
}

// --- HandleAdminReset2FA ---

func TestHandleAdminReset2FA_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/admin/2fa/admin-reset", jsonBody(t, map[string]interface{}{}))
	req = withUserID(req, "admin-1")
	routes.HandleAdminReset2FA(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleAdminReset2FA_NoAdminID(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/admin/2fa/admin-reset", jsonBody(t, map[string]interface{}{
		"user_id": "550e8400-e29b-41d4-a716-446655440000",
		"reason":  "lost device",
	}))
	// No withUserID: adminID must come from context, and its absence must be
	// rejected before the RPC is attempted (not left to the nil-dereference
	// that AdminId: "" would otherwise smuggle through to the auth service).
	routes.HandleAdminReset2FA(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleAdminReset2FA_InvalidJSON(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/admin/2fa/admin-reset", invalidJSON())
	req = withUserID(req, "admin-1")
	routes.HandleAdminReset2FA(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleAdminReset2FA_MissingFields(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))

	tests := []struct {
		name      string
		body      map[string]interface{}
		wantField string
	}{
		{"no_user_id", map[string]interface{}{"reason": "lost device"}, "user_id"},
		{"invalid_user_id", map[string]interface{}{"user_id": "not-a-uuid", "reason": "lost device"}, "user_id"},
		{"no_reason", map[string]interface{}{"user_id": "550e8400-e29b-41d4-a716-446655440000"}, "reason"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/api/v1/admin/2fa/admin-reset", jsonBody(t, tt.body))
			req = withUserID(req, "admin-1")
			routes.HandleAdminReset2FA(rec, req)
			assertValidationError(t, rec, tt.wantField)
		})
	}
}

// --- HandleGetTwoFactorPolicy ---

func TestHandleGetTwoFactorPolicy_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/admin/2fa/policy", nil)
	routes.HandleGetTwoFactorPolicy(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpdateTwoFactorPolicy ---

func TestHandleUpdateTwoFactorPolicy_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/admin/2fa/policy", jsonBody(t, map[string]interface{}{}))
	req = withUserID(req, "admin-1")
	routes.HandleUpdateTwoFactorPolicy(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateTwoFactorPolicy_NoAdminID(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/admin/2fa/policy", jsonBody(t, map[string]interface{}{
		"role_name": "admin",
	}))
	routes.HandleUpdateTwoFactorPolicy(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleUpdateTwoFactorPolicy_InvalidGracePeriod(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))

	tests := []struct {
		name            string
		gracePeriodDays interface{}
	}{
		{"negative", -1},
		{"over_max", 366},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("PUT", "/api/v1/admin/2fa/policy", jsonBody(t, map[string]interface{}{
				"role_name":         "admin",
				"grace_period_days": tt.gracePeriodDays,
			}))
			req = withUserID(req, "admin-1")
			routes.HandleUpdateTwoFactorPolicy(rec, req)
			assertValidationError(t, rec, "grace_period_days")
		})
	}
}

func TestHandleUpdateTwoFactorPolicy_InvalidRoleName(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/admin/2fa/policy", jsonBody(t, map[string]interface{}{
		"role_name": "owner",
	}))
	req = withUserID(req, "admin-1")
	routes.HandleUpdateTwoFactorPolicy(rec, req)
	assertValidationError(t, rec, "role_name")
}

// --- HandleListSessions ---

func TestHandleListSessions_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/auth/sessions", nil)
	req = withUserID(req, "user-123")
	routes.HandleListSessions(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListAllSessions ---

func TestHandleListAllSessions_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/admin/sessions/all?user_id=550e8400-e29b-41d4-a716-446655440000", nil)
	routes.HandleListAllSessions(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListAllSessions_MissingUserID(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/admin/sessions/all", nil)
	routes.HandleListAllSessions(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "user_id")
}

// --- HandleTerminateSession ---

func TestHandleTerminateSession_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/auth/sessions/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withUserID(req, "user-123")
	routes.HandleTerminateSession(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleTerminateSession_InvalidUUID(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/auth/sessions/not-a-uuid", nil)
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleTerminateSession(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

// --- HandleTerminateAllSessions ---

func TestHandleTerminateAllSessions_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/auth/sessions", nil)
	req = withUserID(req, "user-123")
	routes.HandleTerminateAllSessions(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
