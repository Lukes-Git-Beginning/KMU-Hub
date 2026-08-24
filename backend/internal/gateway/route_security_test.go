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

// ============================================================================
// Router-level guard wiring: retention, IP rule and password routes
// (cov-gateway-security-retention-ip-password-routes)
//
// /password/policy (GET) and /password/validate (POST) are deliberately not
// admin-only -- every authenticated user needs to read the active policy and
// validate a candidate password against it before submitting a change. Both
// still require authentication (401 without it), and both sit behind the
// gateway's global per-IP rate limiter wired in cmd/gateway/main.go
// (rateLimiter.Middleware, applied to the whole router before route
// registration) -- there is no route-local gap to close here, and adding a
// second limiter would be redundant, not a fix.
// ============================================================================

func TestSecurityRoutes_RetentionIPRulePasswordGuards(t *testing.T) {
	router := chi.NewRouter()
	NewSecurityRoutes(emptyRegistry()).RegisterRoutes(router, RequireAuthenticated)

	const validID = "550e8400-e29b-41d4-a716-446655440000"

	cases := []struct {
		name      string
		method    string
		path      string
		body      map[string]interface{}
		adminOnly bool
	}{
		{"list ip rules", http.MethodGet, "/api/v1/security/ip-rules", nil, true},
		{"create ip rule", http.MethodPost, "/api/v1/security/ip-rules", map[string]interface{}{
			"ip_cidr": "10.0.0.0/8", "rule_type": "allow",
		}, true},
		{"delete ip rule", http.MethodDelete, "/api/v1/security/ip-rules/" + validID, nil, true},
		{"list retention policies", http.MethodGet, "/api/v1/security/retention-policies", nil, true},
		{"create retention policy", http.MethodPost, "/api/v1/security/retention-policies", map[string]interface{}{
			"resource_type": "audit_log", "retention_days": 30, "action": "delete",
		}, true},
		{"update retention policy", http.MethodPut, "/api/v1/security/retention-policies/" + validID, map[string]interface{}{
			"retention_days": 30, "action": "delete",
		}, true},
		{"delete retention policy", http.MethodDelete, "/api/v1/security/retention-policies/" + validID, nil, true},
		{"latest retention run", http.MethodGet, "/api/v1/security/retention-runs/latest", nil, true},
		{"get password policy", http.MethodGet, "/api/v1/security/password/policy", nil, false},
		{"update password policy", http.MethodPut, "/api/v1/security/password/policy", map[string]interface{}{}, true},
		{"validate password", http.MethodPost, "/api/v1/security/password/validate", map[string]interface{}{
			"password": "irrelevant",
		}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name+"/no auth", func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, jsonBody(t, tc.body))
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			assertStatus(t, rec, http.StatusUnauthorized)
		})

		if tc.adminOnly {
			t.Run(tc.name+"/authenticated non-admin", func(t *testing.T) {
				req := httptest.NewRequest(tc.method, tc.path, jsonBody(t, tc.body))
				req = withAuth(req, uuid.New().String(), testTenantID)
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)
				assertStatus(t, rec, http.StatusForbidden)
			})
		}

		t.Run(tc.name+"/authorized empty registry", func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, jsonBody(t, tc.body))
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

// --- HandleCreateIPRule: invalid CIDR ---
//
// The router-level test above proves valid bodies reach the (unreachable in
// this test) gRPC layer; CreateIPRule's actual CIDR-format rejection is a
// gRPC-layer concern (net.ParseCIDR, added alongside this unit) and is
// covered in security_grpc_test.go's TestSecurityGRPC_ValidationErrors --
// there is no HTTP-layer struct tag for CIDR shape, so this handler has
// nothing further to validate before the call.

// --- HandleUpdateRetentionPolicy / HandleCreateRetentionPolicy: retention days ---
//
// Both createRetentionPolicyRequest and updateRetentionPolicyRequest carry
// `validate:"required,min=1"` on retention_days, so 0 and negative values are
// already rejected by decodeAndValidate before the gRPC call.

func TestHandleCreateRetentionPolicy_ZeroRetentionDaysRejected(t *testing.T) {
	routes := NewSecurityRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/security/retention-policies", jsonBody(t, map[string]interface{}{
		"resource_type": "audit_log", "retention_days": 0, "action": "delete",
	}))
	req = withUserID(req, "admin-123")
	routes.HandleCreateRetentionPolicy(rec, req)
	assertValidationError(t, rec, "retention_days")
}

func TestHandleCreateRetentionPolicy_NegativeRetentionDaysRejected(t *testing.T) {
	routes := NewSecurityRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/security/retention-policies", jsonBody(t, map[string]interface{}{
		"resource_type": "audit_log", "retention_days": -5, "action": "delete",
	}))
	req = withUserID(req, "admin-123")
	routes.HandleCreateRetentionPolicy(rec, req)
	assertValidationError(t, rec, "retention_days")
}

func TestHandleUpdateRetentionPolicy_ZeroRetentionDaysRejected(t *testing.T) {
	routes := NewSecurityRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	policyID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/security/retention-policies/"+policyID, jsonBody(t, map[string]interface{}{
		"retention_days": 0, "action": "delete",
	}))
	req = withUserID(req, "admin-123")
	req = withChiURLParam(req, "id", policyID)
	routes.HandleUpdateRetentionPolicy(rec, req)
	assertValidationError(t, rec, "retention_days")
}

func TestHandleCreateRetentionPolicy_ServiceUnavailable(t *testing.T) {
	routes := NewSecurityRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleCreateRetentionPolicy)
}

func TestHandleUpdateRetentionPolicy_ServiceUnavailable(t *testing.T) {
	routes := NewSecurityRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleUpdateRetentionPolicy)
}

func TestHandleDeleteRetentionPolicy_ServiceUnavailable(t *testing.T) {
	routes := NewSecurityRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleDeleteRetentionPolicy)
}

func TestHandleListRetentionPolicies_ServiceUnavailable(t *testing.T) {
	routes := NewSecurityRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleListRetentionPolicies)
}

func TestHandleListIPRules_ServiceUnavailable(t *testing.T) {
	routes := NewSecurityRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleListIPRules)
}

func TestHandleDeleteIPRule_ServiceUnavailable(t *testing.T) {
	routes := NewSecurityRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	req = withChiURLParam(req, "id", uuid.New().String())
	routes.HandleDeleteIPRule(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- Vendor Access routes ---
//
// Unlike the admin-only routes above, /api/v1/vendor-access is guarded by
// middleware.RequirePermission("security:vendor_access", "manage") -- a
// fine-grained catalogue key baked into the JWT at login/refresh
// (rbac_phase1a seed, migrations/000256), not the coarse "admin" role. The
// "authenticated non-admin" case below therefore carries an unrelated
// permission rather than a non-admin role, to prove the guard checks the
// specific key and not just "is authenticated".

func TestSecurityRoutes_VendorAccessGuards(t *testing.T) {
	router := chi.NewRouter()
	NewSecurityRoutes(emptyRegistry()).RegisterRoutes(router, RequireAuthenticated)

	const validID = "550e8400-e29b-41d4-a716-446655440000"
	const requiredPerm = "security:vendor_access:manage"

	cases := []struct {
		name   string
		method string
		path   string
		body   map[string]interface{}
	}{
		{"list vendor access requests", http.MethodGet, "/api/v1/vendor-access/", nil},
		{"approve vendor access request", http.MethodPost, "/api/v1/vendor-access/" + validID + "/approve", map[string]interface{}{
			"sensitive_ack": false,
		}},
		{"decline vendor access request", http.MethodPost, "/api/v1/vendor-access/" + validID + "/decline", nil},
		{"counter-propose vendor access request", http.MethodPost, "/api/v1/vendor-access/" + validID + "/counter-propose", map[string]interface{}{
			"proposed_start": "2026-12-01",
		}},
		{"revoke vendor access request", http.MethodPost, "/api/v1/vendor-access/" + validID + "/revoke", nil},
	}

	for _, tc := range cases {
		t.Run(tc.name+"/no auth", func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, jsonBody(t, tc.body))
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			assertStatus(t, rec, http.StatusUnauthorized)
		})

		t.Run(tc.name+"/authenticated without the permission", func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, jsonBody(t, tc.body))
			req = withAuth(req, uuid.New().String(), testTenantID)
			req = withPermissions(req, "security:audit:read")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			assertStatus(t, rec, http.StatusForbidden)
		})

		t.Run(tc.name+"/authorized empty registry", func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, jsonBody(t, tc.body))
			req = withAuth(req, uuid.New().String(), testTenantID)
			req = withPermissions(req, requiredPerm)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			assertStatus(t, rec, http.StatusServiceUnavailable)
		})
	}
}

// --- HandleCounterProposeVendorAccessRequest: proposed_start required ---

func TestHandleCounterProposeVendorAccessRequest_MissingProposedStart(t *testing.T) {
	routes := NewSecurityRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	requestID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vendor-access/"+requestID+"/counter-propose", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", requestID)
	routes.HandleCounterProposeVendorAccessRequest(rec, req)
	assertValidationError(t, rec, "proposed_start")
}

// --- Double approval and post-revoke enforcement ---
//
// Both are proven, but not by a gateway-layer test:
//   - Double approval: approving an already-active request hits
//     vendoraccess.ErrInvalidStatus -> codes.FailedPrecondition
//     (security_grpc.go mapSecurityError) -> HTTP 409. The domain-error path
//     is covered end to end in
//     TestVendorAccessRPCs_HappyPathAndDomainErrors
//     (internal/server/security_grpc_test.go); the FailedPrecondition->409
//     mapping itself is proven generically in helpers_test.go. A third,
//     gateway-layer copy of the same assertion would not add coverage of a
//     new code path.
//   - Post-revoke enforcement: there is NO enforcement to test. A grep across
//     internal/ and cmd/ for VendorAccessStatusActive and
//     vendor_access_requests finds exactly the five files this unit already
//     touches (service.go, security_grpc.go, route_security.go, and their
//     tests) -- no middleware, session check, or other gate reads this
//     table. The "active" status is a consent/audit record for the
//     ein-Server-pro-Kunde delivery model, not an access-control mechanism:
//     revoking a request changes what a compliance report shows, not what
//     any vendor account can actually do. Documented at
//     vendoraccess.Service.Revoke.

func TestHandleListVendorAccessRequests_ServiceUnavailable(t *testing.T) {
	routes := NewSecurityRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleListVendorAccessRequests)
}

func TestHandleDeclineVendorAccessRequest_ServiceUnavailable(t *testing.T) {
	routes := NewSecurityRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = withChiURLParam(req, "id", uuid.New().String())
	routes.HandleDeclineVendorAccessRequest(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
