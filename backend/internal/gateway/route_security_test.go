package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// SecurityRoutes registers under the "auth" gRPC service connection.

// --- ServiceName ---

func TestSecurityRoutes_ServiceName(t *testing.T) {
	routes := NewSecurityRoutes(emptyRegistry())
	if routes.ServiceName() != "auth" {
		t.Errorf("ServiceName() = %q, want %q", routes.ServiceName(), "auth")
	}
}

// --- HandleSetVaultSecret ---

func TestHandleSetVaultSecret_ServiceUnavailable(t *testing.T) {
	routes := NewSecurityRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleSetVaultSecret)
}

func TestHandleSetVaultSecret_InvalidJSON(t *testing.T) {
	routes := NewSecurityRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/security/vault", invalidJSON())
	req = withUserID(req, "admin-123")
	routes.HandleSetVaultSecret(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleSetVaultSecret_MissingKeyName(t *testing.T) {
	routes := NewSecurityRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/security/vault", jsonBody(t, map[string]interface{}{
		"plaintext_value": "supersecret",
	}))
	req = withUserID(req, "admin-123")
	routes.HandleSetVaultSecret(rec, req)
	assertValidationError(t, rec, "key_name")
}

func TestHandleSetVaultSecret_MissingPlaintextValue(t *testing.T) {
	routes := NewSecurityRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/security/vault", jsonBody(t, map[string]interface{}{
		"key_name": "my-secret",
	}))
	req = withUserID(req, "admin-123")
	routes.HandleSetVaultSecret(rec, req)
	assertValidationError(t, rec, "plaintext_value")
}

// --- HandlePreviewErasure ---

func TestHandlePreviewErasure_ServiceUnavailable(t *testing.T) {
	routes := NewSecurityRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandlePreviewErasure)
}

func TestHandlePreviewErasure_InvalidJSON(t *testing.T) {
	routes := NewSecurityRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/security/gdpr/erasure/preview", invalidJSON())
	routes.HandlePreviewErasure(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandlePreviewErasure_MissingUserID(t *testing.T) {
	routes := NewSecurityRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/security/gdpr/erasure/preview", jsonBody(t, map[string]interface{}{}))
	routes.HandlePreviewErasure(rec, req)
	assertValidationError(t, rec, "user_id")
}

func TestHandlePreviewErasure_InvalidUserIDFormat(t *testing.T) {
	routes := NewSecurityRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/security/gdpr/erasure/preview", jsonBody(t, map[string]interface{}{
		"user_id": "not-a-uuid",
	}))
	routes.HandlePreviewErasure(rec, req)
	assertValidationError(t, rec, "user_id")
}

// --- HandleExecuteErasure ---

func TestHandleExecuteErasure_ServiceUnavailable(t *testing.T) {
	routes := NewSecurityRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleExecuteErasure)
}

func TestHandleExecuteErasure_MissingUserID(t *testing.T) {
	routes := NewSecurityRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/security/gdpr/erasure/execute", jsonBody(t, map[string]interface{}{
		"admin_password": "adminpass",
	}))
	req = withUserID(req, "admin-123")
	routes.HandleExecuteErasure(rec, req)
	assertValidationError(t, rec, "user_id")
}

func TestHandleExecuteErasure_MissingAdminPassword(t *testing.T) {
	routes := NewSecurityRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/security/gdpr/erasure/execute", jsonBody(t, map[string]interface{}{
		"user_id": "550e8400-e29b-41d4-a716-446655440000",
	}))
	req = withUserID(req, "admin-123")
	routes.HandleExecuteErasure(rec, req)
	assertValidationError(t, rec, "admin_password")
}

// --- HandleValidatePassword ---

func TestHandleValidatePassword_ServiceUnavailable(t *testing.T) {
	routes := NewSecurityRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleValidatePassword)
}

func TestHandleValidatePassword_InvalidJSON(t *testing.T) {
	routes := NewSecurityRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/security/password/validate", invalidJSON())
	req = withUserID(req, "user-123")
	routes.HandleValidatePassword(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleValidatePassword_MissingPassword(t *testing.T) {
	routes := NewSecurityRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/security/password/validate", jsonBody(t, map[string]interface{}{}))
	req = withUserID(req, "user-123")
	routes.HandleValidatePassword(rec, req)
	assertValidationError(t, rec, "password")
}

// --- HandleCreateIPRule ---

func TestHandleCreateIPRule_ServiceUnavailable(t *testing.T) {
	routes := NewSecurityRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleCreateIPRule)
}

func TestHandleCreateIPRule_InvalidJSON(t *testing.T) {
	routes := NewSecurityRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/security/ip-rules", invalidJSON())
	req = withUserID(req, "admin-123")
	routes.HandleCreateIPRule(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleCreateIPRule_MissingIPCIDR(t *testing.T) {
	routes := NewSecurityRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/security/ip-rules", jsonBody(t, map[string]interface{}{
		"rule_type": "allow",
	}))
	req = withUserID(req, "admin-123")
	routes.HandleCreateIPRule(rec, req)
	assertValidationError(t, rec, "ip_cidr")
}

func TestHandleCreateIPRule_MissingRuleType(t *testing.T) {
	routes := NewSecurityRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/security/ip-rules", jsonBody(t, map[string]interface{}{
		"ip_cidr": "192.168.1.0/24",
	}))
	req = withUserID(req, "admin-123")
	routes.HandleCreateIPRule(rec, req)
	assertValidationError(t, rec, "rule_type")
}

func TestHandleCreateIPRule_InvalidRuleType(t *testing.T) {
	routes := NewSecurityRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/security/ip-rules", jsonBody(t, map[string]interface{}{
		"ip_cidr":   "192.168.1.0/24",
		"rule_type": "deny", // not "allow" or "block"
	}))
	req = withUserID(req, "admin-123")
	routes.HandleCreateIPRule(rec, req)
	assertValidationError(t, rec, "rule_type")
}

// --- HandleApproveDataExport ---

func TestHandleApproveDataExport_ServiceUnavailable(t *testing.T) {
	routes := NewSecurityRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	req = withUserID(req, "admin-123")
	routes.HandleApproveDataExport(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleApproveDataExport_InvalidJSON(t *testing.T) {
	routes := NewSecurityRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", invalidJSON())
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	req = withUserID(req, "admin-123")
	routes.HandleApproveDataExport(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

// --- GDPR export route paths (fix-security-gdpr-paths) ---
//
// The desktop client (security-client.ts) calls a singular "/gdpr/export/..."
// tree; the gateway used to serve a mismatched mix of "/gdpr/export",
// "/gdpr/exports/{id}/..." and "/gdpr/download/{token}". These are
// router-level tests on purpose: a handler unit test calling
// HandleApproveDataExport directly can never catch a mistyped path, only a
// real chi.ServeHTTP round trip can.

// TestGDPRExportRoutes_MatchFrontendPaths locks in the four paths the
// frontend actually calls. The registry is empty, so a request that resolves
// to a handler dies at the gRPC client with 503; 404 would mean the path
// itself is unmatched, which is exactly the defect this test guards against.
func TestGDPRExportRoutes_MatchFrontendPaths(t *testing.T) {
	r := chi.NewRouter()
	NewSecurityRoutes(emptyRegistry()).RegisterRoutes(r, guardTestAuth)

	cases := []struct {
		name   string
		method string
		path   string
		admin  bool
	}{
		{"request", http.MethodPost, "/api/v1/security/gdpr/export/request", false},
		{"approve", http.MethodPost, "/api/v1/security/gdpr/export/550e8400-e29b-41d4-a716-446655440000/approve", true},
		{"deny", http.MethodPost, "/api/v1/security/gdpr/export/550e8400-e29b-41d4-a716-446655440000/deny", true},
		{"download", http.MethodGet, "/api/v1/security/gdpr/export/one-time-token-value/download", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.admin {
				req = withRoles(req, "admin")
			}
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
			assertStatus(t, rec, http.StatusServiceUnavailable)
		})
	}
}

// TestGDPRExportRoutes_ApproveDenyRequireAdmin verifies the path move did not
// drop the RequireRole("admin") guard on approve/deny.
func TestGDPRExportRoutes_ApproveDenyRequireAdmin(t *testing.T) {
	r := chi.NewRouter()
	NewSecurityRoutes(emptyRegistry()).RegisterRoutes(r, guardTestAuth)

	for _, path := range []string{
		"/api/v1/security/gdpr/export/550e8400-e29b-41d4-a716-446655440000/approve",
		"/api/v1/security/gdpr/export/550e8400-e29b-41d4-a716-446655440000/deny",
	} {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assertStatus(t, rec, http.StatusForbidden)
	}
}

// TestGDPRExportRoutes_DownloadUsesTokenNotID verifies HandleGetExportDownload
// reads the one-time download token out of the "id" chi param (the shared
// wildcard name forced by "/export/{id}/approve" already claiming that tree
// position — see the comment in HandleGetExportDownload) rather than "token".
func TestGDPRExportRoutes_DownloadUsesTokenNotID(t *testing.T) {
	routes := NewSecurityRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = withChiURLParam(req, "id", "")
	routes.HandleGetExportDownload(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "download token is required")
}

// --- HandleUpdatePasswordPolicy ---

func TestHandleUpdatePasswordPolicy_ServiceUnavailable(t *testing.T) {
	routes := NewSecurityRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleUpdatePasswordPolicy)
}

func TestHandleUpdatePasswordPolicy_InvalidJSON(t *testing.T) {
	routes := NewSecurityRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/security/password/policy", invalidJSON())
	req = withUserID(req, "admin-123")
	routes.HandleUpdatePasswordPolicy(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

// HandleUpdatePasswordPolicy has no required fields (all bool/int with zero values valid)
// so we just confirm valid empty body is accepted at decode level.
func TestHandleUpdatePasswordPolicy_ValidEmptyBody(t *testing.T) {
	routes := NewSecurityRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/security/password/policy", jsonBody(t, map[string]interface{}{}))
	req = withUserID(req, "admin-123")
	routes.HandleUpdatePasswordPolicy(rec, req)
	// Will fail at gRPC level (localhost:0), not at validation
	if rec.Code == http.StatusBadRequest {
		t.Errorf("expected non-400 for valid body, got %d: %s", rec.Code, rec.Body.String())
	}
}
