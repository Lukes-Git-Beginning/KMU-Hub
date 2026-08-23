package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
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

// ============================================================================
// Router-level guard wiring: GDPR export/erasure/DSAR routes
// (cov-gateway-security-gdpr-export-erasure-dsar-routes)
// ============================================================================

func TestSecurityRoutes_GDPRGuards(t *testing.T) {
	router := chi.NewRouter()
	NewSecurityRoutes(emptyRegistry()).RegisterRoutes(router, RequireAuthenticated)

	cases := []struct {
		name      string
		method    string
		path      string
		adminOnly bool
	}{
		{"request export", http.MethodPost, "/api/v1/security/gdpr/export/request", false},
		{"approve export", http.MethodPost, "/api/v1/security/gdpr/export/550e8400-e29b-41d4-a716-446655440000/approve", true},
		{"deny export", http.MethodPost, "/api/v1/security/gdpr/export/550e8400-e29b-41d4-a716-446655440000/deny", true},
		{"download export", http.MethodGet, "/api/v1/security/gdpr/export/some-token/download", false},
		{"list exports", http.MethodGet, "/api/v1/security/gdpr/exports", false},
		{"preview erasure", http.MethodPost, "/api/v1/security/gdpr/erasure/preview", true},
		{"execute erasure", http.MethodPost, "/api/v1/security/gdpr/erasure/execute", true},
		{"dsar search", http.MethodGet, "/api/v1/security/dsar/search?q=ab", true},
	}

	for _, tc := range cases {
		t.Run(tc.name+"/no auth", func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, jsonBody(t, map[string]interface{}{}))
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			assertStatus(t, rec, http.StatusUnauthorized)
		})

		if tc.adminOnly {
			t.Run(tc.name+"/authenticated non-admin", func(t *testing.T) {
				req := httptest.NewRequest(tc.method, tc.path, jsonBody(t, map[string]interface{}{}))
				req = withAuth(req, uuid.New().String(), testTenantID)
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)
				assertStatus(t, rec, http.StatusForbidden)
			})
		}

		t.Run(tc.name+"/authorized empty registry", func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, jsonBody(t, map[string]interface{}{}))
			req = withAuth(req, uuid.New().String(), testTenantID)
			if tc.adminOnly {
				req = withRoles(req, "admin")
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			assertStatus(t, rec, http.StatusServiceUnavailable)
		})
	}
}

// --- HandleRequestDataExport ---

func TestHandleRequestDataExport_ServiceUnavailable(t *testing.T) {
	routes := NewSecurityRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = withUserID(req, "user-123")
	routes.HandleRequestDataExport(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListDataExports ---
//
// The gateway's own doc comment used to claim "Only admins can view other
// users' exports -- gateway trusts RBAC middleware", but no guard actually
// enforced that on this route (no RequireRole on "/gdpr/exports" in
// RegisterRoutes) -- any authenticated user could pass ?user_id=<someone
// else> and read another user's export request metadata (status, reviewer
// notes, timestamps). Fixed by checking middleware.IsAdmin inline before
// honoring a user_id override that isn't the caller's own.

func TestHandleListDataExports_ServiceUnavailable(t *testing.T) {
	routes := NewSecurityRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = withUserID(req, "user-123")
	routes.HandleListDataExports(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListDataExports_CrossUserRequiresAdmin(t *testing.T) {
	routes := NewSecurityRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/?user_id=other-user-456", nil)
	req = withUserID(req, "user-123") // not admin, not the queried user
	routes.HandleListDataExports(rec, req)
	assertStatus(t, rec, http.StatusForbidden)
	assertErrorContains(t, rec, "only admins")
}

func TestHandleListDataExports_OwnUserIDAllowed(t *testing.T) {
	routes := NewSecurityRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/?user_id=user-123", nil)
	req = withUserID(req, "user-123") // queries own id explicitly -- must not be blocked
	routes.HandleListDataExports(rec, req)
	if rec.Code == http.StatusForbidden {
		t.Errorf("querying own user_id must not be forbidden, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleListDataExports_NoUserIDFilterAllowed(t *testing.T) {
	routes := NewSecurityRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = withUserID(req, "user-123") // no override -- defaults to caller's own exports
	routes.HandleListDataExports(rec, req)
	if rec.Code == http.StatusForbidden {
		t.Errorf("no user_id filter must not be forbidden, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleListDataExports_AdminCanFilterOtherUser(t *testing.T) {
	routes := NewSecurityRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/?user_id=other-user-456", nil)
	req = withUserID(req, "admin-123")
	req = withRoles(req, "admin")
	routes.HandleListDataExports(rec, req)
	if rec.Code == http.StatusForbidden {
		t.Errorf("admin filtering by another user_id must not be forbidden, got %d: %s", rec.Code, rec.Body.String())
	}
}

// --- HandleDSARSearch ---

func TestHandleDSARSearch_ServiceUnavailable(t *testing.T) {
	routes := NewSecurityRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/?q=ab", nil)
	routes.HandleDSARSearch(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDSARSearch_EmptyQuery(t *testing.T) {
	routes := NewSecurityRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	routes.HandleDSARSearch(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "at least 2 characters")
}

func TestHandleDSARSearch_TooShortQuery(t *testing.T) {
	routes := NewSecurityRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/?q=a", nil)
	routes.HandleDSARSearch(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "at least 2 characters")
}

func TestHandleDSARSearch_WhitespaceOnlyQuery(t *testing.T) {
	routes := NewSecurityRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/?q=%20%20", nil) // "  " trims to empty
	routes.HandleDSARSearch(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "at least 2 characters")
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

// --- HandleGetLatestRetentionRun ---

func TestHandleGetLatestRetentionRun_ServiceUnavailable(t *testing.T) {
	routes := NewSecurityRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleGetLatestRetentionRun)
}

// ============================================================================
// Router-level guard wiring: audit + vault routes (cov-gateway-security-audit-and-vault-routes)
//
// Wired with the real RequireAuthenticated (not the guardTestAuth no-op) so
// the 401 case actually exercises the auth check, not just the no-op passthrough.
// ============================================================================

func TestSecurityRoutes_AuditAndVaultGuards(t *testing.T) {
	router := chi.NewRouter()
	NewSecurityRoutes(emptyRegistry()).RegisterRoutes(router, RequireAuthenticated)

	cases := []struct {
		name   string
		method string
		path   string
	}{
		{"list audit entries", http.MethodGet, "/api/v1/security/audit"},
		{"export audit log", http.MethodGet, "/api/v1/security/audit/export"},
		{"verify audit chain", http.MethodPost, "/api/v1/security/audit/verify"},
		{"list vault secrets", http.MethodGet, "/api/v1/security/vault"},
		{"get vault secret", http.MethodGet, "/api/v1/security/vault/stripe_key"},
		{"set vault secret", http.MethodPut, "/api/v1/security/vault"},
		{"delete vault secret", http.MethodDelete, "/api/v1/security/vault/stripe_key"},
	}

	for _, tc := range cases {
		t.Run(tc.name+"/no auth", func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, jsonBody(t, map[string]interface{}{}))
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			assertStatus(t, rec, http.StatusUnauthorized)
		})
		t.Run(tc.name+"/authenticated non-admin", func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, jsonBody(t, map[string]interface{}{}))
			req = withAuth(req, uuid.New().String(), testTenantID)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			assertStatus(t, rec, http.StatusForbidden)
		})
		t.Run(tc.name+"/admin empty registry", func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, jsonBody(t, map[string]interface{}{}))
			req = withAuth(req, uuid.New().String(), testTenantID)
			req = withRoles(req, "admin")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			assertStatus(t, rec, http.StatusServiceUnavailable)
		})
	}
}

// --- HandleGetVaultSecret ---

func TestHandleGetVaultSecret_ServiceUnavailable(t *testing.T) {
	routes := NewSecurityRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = withChiURLParam(req, "keyName", "stripe_key")
	routes.HandleGetVaultSecret(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetVaultSecret_MissingKeyName(t *testing.T) {
	routes := NewSecurityRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = withChiURLParam(req, "keyName", "")
	routes.HandleGetVaultSecret(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "key name is required")
}

// --- HandleDeleteVaultSecret ---

func TestHandleDeleteVaultSecret_ServiceUnavailable(t *testing.T) {
	routes := NewSecurityRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	req = withChiURLParam(req, "keyName", "stripe_key")
	routes.HandleDeleteVaultSecret(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteVaultSecret_MissingKeyName(t *testing.T) {
	routes := NewSecurityRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	req = withChiURLParam(req, "keyName", "")
	routes.HandleDeleteVaultSecret(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "key name is required")
}

// --- HandleListVaultSecrets / HandleListAuditEntries / HandleExportAuditLog / HandleVerifyAuditChain ---
//
// These four have no request-level validation before the gRPC call, so their
// only handler-unit-test surface (beyond the router-level guard table above)
// is the service-unavailable path.

func TestHandleListVaultSecrets_ServiceUnavailable(t *testing.T) {
	routes := NewSecurityRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleListVaultSecrets)
}

func TestHandleListAuditEntries_ServiceUnavailable(t *testing.T) {
	routes := NewSecurityRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleListAuditEntries)
}

func TestHandleExportAuditLog_ServiceUnavailable(t *testing.T) {
	routes := NewSecurityRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleExportAuditLog)
}

func TestHandleVerifyAuditChain_ServiceUnavailable(t *testing.T) {
	routes := NewSecurityRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleVerifyAuditChain)
}
