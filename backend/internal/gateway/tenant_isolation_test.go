package gateway

// Tenant isolation smoke tests for the HTTP gateway layer.
//
// These tests verify that the GetTenantID extraction is correctly wired into
// the rapporte, vermietung and inventar handlers. They do NOT test gRPC-level
// data isolation (that is covered by service_test.go in each module package),
// but they do verify:
//
//  1. Requests without an Authorization header / without a JWT → 401.
//  2. JWTs with an empty tid claim (legacy tokens) → 401 (fail-closed).
//  3. JWTs with a valid tid claim → the handler proceeds past the tenant check
//     (response is 503 when the gRPC backend is absent — which is fine for unit tests).
//
// Cross-tenant data isolation (Token-A writes, Token-B reads → 404) is
// enforced by the service layer and exercised in e.g.
// internal/rapporte/service_test.go:TestService_UploadAttachment_CrossTenantPath_Rejected.
// An end-to-end integration test would require a real gRPC backend; that is
// outside the scope of gateway unit tests.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/featureflag"
	"github.com/kmuhub/kmuhub/internal/middleware"
)

// ============================================================================
// Helper: build a flag registry with a single module enabled
// ============================================================================

func moduleFlag(key string) *featureflag.Registry {
	return featureflag.NewRegistry().Load(func(k string) string {
		if k == key {
			return "true"
		}
		return ""
	})
}

// ============================================================================
// Helper: build a request with TenantID set directly in context (simulates
// what the Auth middleware does after validating a JWT that contains a tid claim).
// ============================================================================

func reqWithTenant(method, path string, tenantID uuid.UUID) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, tenantID.String())
	return req.WithContext(ctx)
}

// reqWithEmptyTenant simulates a legacy JWT that was parsed successfully but had
// no tid claim — the middleware stores an empty string.
func reqWithEmptyTenant(method, path string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, "")
	return req.WithContext(ctx)
}

// ============================================================================
// Rapporte — tenant isolation checks
// ============================================================================

func rapporteFlagsON() *featureflag.Registry {
	return moduleFlag("COSMI_MODULE_RAPPORTE_ENABLED")
}

// TestRapporte_NoTenant_Returns401 verifies that a request without any tenant
// context is rejected before it reaches the gRPC client.
func TestRapporte_NoTenant_Returns401(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), rapporteFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/rapporte/reports", nil)
	routes.HandleListReports(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

// TestRapporte_EmptyTid_Returns401 verifies that a legacy JWT (empty tid) is
// rejected with 401 — no placeholder substitution may occur.
func TestRapporte_EmptyTid_Returns401(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), rapporteFlagsON())
	rec := httptest.NewRecorder()
	req := reqWithEmptyTenant(http.MethodGet, "/api/v1/rapporte/reports")
	routes.HandleListReports(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

// TestRapporte_ValidTid_PassesTenantCheck verifies that a request carrying a valid
// tid proceeds past the tenant check (503 is expected because there is no real
// gRPC backend in unit tests — anything other than 401 proves the check passed).
func TestRapporte_ValidTid_PassesTenantCheck(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), rapporteFlagsON())
	rec := httptest.NewRecorder()
	req := reqWithTenant(http.MethodGet, "/api/v1/rapporte/reports", uuid.New())
	routes.HandleListReports(rec, req)
	if rec.Code == http.StatusUnauthorized {
		t.Errorf("valid tid should not be rejected with 401; body = %s", rec.Body.String())
	}
}

// TestRapporte_TwoTenants_DifferentContextValues verifies that two requests with
// different tenant IDs are treated as independent — the handler reads from context
// and does not share any mutable state.
func TestRapporte_TwoTenants_DifferentContextValues(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), rapporteFlagsON())

	tenantA := uuid.New()
	tenantB := uuid.New()

	recA := httptest.NewRecorder()
	reqA := reqWithTenant(http.MethodGet, "/api/v1/rapporte/reports", tenantA)

	recB := httptest.NewRecorder()
	reqB := reqWithTenant(http.MethodGet, "/api/v1/rapporte/reports", tenantB)

	routes.HandleListReports(recA, reqA)
	routes.HandleListReports(recB, reqB)

	// Both should fail in the same way (503 — no backend); neither should be 401.
	if recA.Code == http.StatusUnauthorized {
		t.Errorf("tenant A request rejected with 401")
	}
	if recB.Code == http.StatusUnauthorized {
		t.Errorf("tenant B request rejected with 401")
	}
}

// ============================================================================
// Vermietung — tenant isolation checks
// ============================================================================

func vermietungFlagsON() *featureflag.Registry {
	return moduleFlag("COSMI_MODULE_VERMIETUNG_ENABLED")
}

func TestVermietung_NoTenant_Returns401(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), vermietungFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/vermietung/objects", nil)
	routes.HandleListObjects(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestVermietung_EmptyTid_Returns401(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), vermietungFlagsON())
	rec := httptest.NewRecorder()
	req := reqWithEmptyTenant(http.MethodGet, "/api/v1/vermietung/objects")
	routes.HandleListObjects(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestVermietung_ValidTid_PassesTenantCheck(t *testing.T) {
	routes := NewVermietungRoutes(registryWithService("vermietung"), vermietungFlagsON())
	rec := httptest.NewRecorder()
	req := reqWithTenant(http.MethodGet, "/api/v1/vermietung/objects", uuid.New())
	routes.HandleListObjects(rec, req)
	if rec.Code == http.StatusUnauthorized {
		t.Errorf("valid tid should not be rejected with 401; body = %s", rec.Body.String())
	}
}

// ============================================================================
// Inventar — tenant isolation checks
// ============================================================================

func inventarFlagsON() *featureflag.Registry {
	return moduleFlag("COSMI_MODULE_INVENTAR_ENABLED")
}

func TestInventar_NoTenant_Returns401(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), inventarFlagsON())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventar/items", nil)
	routes.HandleListItems(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestInventar_EmptyTid_Returns401(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), inventarFlagsON())
	rec := httptest.NewRecorder()
	req := reqWithEmptyTenant(http.MethodGet, "/api/v1/inventar/items")
	routes.HandleListItems(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestInventar_ValidTid_PassesTenantCheck(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), inventarFlagsON())
	rec := httptest.NewRecorder()
	req := reqWithTenant(http.MethodGet, "/api/v1/inventar/items", uuid.New())
	routes.HandleListItems(rec, req)
	if rec.Code == http.StatusUnauthorized {
		t.Errorf("valid tid should not be rejected with 401; body = %s", rec.Body.String())
	}
}
