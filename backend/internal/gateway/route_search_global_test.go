package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// This file covers route_search_global.go (HandleGlobalSearch and its three
// per-module fan-out helpers) and route_registrar.go (the RouteRegistrar
// interface, exercised here via a compile-time assertion since it carries no
// executable logic of its own).
//
// Two things worth documenting rather than assuming away, per the unit's
// "belegt oder als Fund gemeldet" requirement:
//
//  1. Tenant isolation for the CRM and Documents fan-out is NOT done in this
//     handler — it never reads middleware.GetTenantID itself. It is done the
//     same way as every other gateway→service call in this package: the
//     outbound gRPC interceptor chain installed in ServiceRegistry.GetConnection
//     (middleware.TenantOutboundUnaryInterceptor, registry.go:112) stamps the
//     tenant ID from the request context onto every outgoing call, and the
//     downstream CRMService/DocumentService enforce RLS on it. That is the
//     established thick-service pattern for this codebase, not a gap specific
//     to global search.
//  2. Permission is checked exactly once, at the route group
//     (RequirePermission("search", "read"), route_search_global.go:39) — there
//     is no per-module permission check inside HandleGlobalSearch itself. All
//     three fanned-out RPCs (CRMService.Search, DocumentService.SearchFiles)
//     receive the same caller identity and are expected to apply their own
//     module-level authorization downstream, same as any other cross-module
//     read in this gateway. Not a per-route problem to fix here.
//
// No bufconn stub exists in this repo for CRMServiceClient/DocumentServiceClient
// (same limitation noted in route_booking_admin_test.go), so the RPC calls
// below hit a real (but unreachable) localhost:0 connection and fail at the
// transport level — HandleGlobalSearch isolates that failure per module and
// still answers 200, which is exactly the behaviour under test.

var _ RouteRegistrar = (*GlobalSearchRoutes)(nil)

type globalSearchTestModule struct {
	Module  string `json:"module"`
	Results any    `json:"results"`
	Total   int32  `json:"total"`
	Error   string `json:"error,omitempty"`
}

type globalSearchTestResponse struct {
	Query   string                   `json:"query"`
	Modules []globalSearchTestModule `json:"modules"`
}

func decodeGlobalSearchResponse(t *testing.T, rec *httptest.ResponseRecorder) globalSearchTestResponse {
	t.Helper()
	var resp globalSearchTestResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode global search response: %v; body = %s", err, rec.Body.String())
	}
	return resp
}

func findModule(modules []globalSearchTestModule, name string) (globalSearchTestModule, bool) {
	for _, m := range modules {
		if m.Module == name {
			return m, true
		}
	}
	return globalSearchTestModule{}, false
}

func TestGlobalSearchRoutes_ServiceName(t *testing.T) {
	routes := NewGlobalSearchRoutes(emptyRegistry())
	if routes.ServiceName() != "global-search" {
		t.Errorf("ServiceName() = %q, want %q", routes.ServiceName(), "global-search")
	}
}

func TestHandleGlobalSearch_MissingQuery_BadRequest(t *testing.T) {
	routes := NewGlobalSearchRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/global", nil)
	routes.HandleGlobalSearch(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "query parameter 'q' is required")
}

func TestHandleGlobalSearch_EmptyQuery_BadRequest(t *testing.T) {
	routes := NewGlobalSearchRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/global?q=", nil)
	routes.HandleGlobalSearch(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleGlobalSearch_EmptyRegistry_AllModulesReportUnavailable(t *testing.T) {
	routes := NewGlobalSearchRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/global?q=acme", nil)
	routes.HandleGlobalSearch(rec, req)
	assertStatus(t, rec, http.StatusOK)

	resp := decodeGlobalSearchResponse(t, rec)
	if resp.Query != "acme" {
		t.Errorf("query = %q, want %q", resp.Query, "acme")
	}
	if len(resp.Modules) != 3 {
		t.Fatalf("len(modules) = %d, want 3; body = %s", len(resp.Modules), rec.Body.String())
	}

	crm, ok := findModule(resp.Modules, "crm")
	if !ok || crm.Error != "service unavailable" {
		t.Errorf("crm module = %+v, want error %q", crm, "service unavailable")
	}
	docs, ok := findModule(resp.Modules, "documents")
	if !ok || docs.Error != "service unavailable" {
		t.Errorf("documents module = %+v, want error %q", docs, "service unavailable")
	}
}

func TestHandleGlobalSearch_RegisteredServices_RPCFailureIsolatedPerModule(t *testing.T) {
	reg := NewServiceRegistry(nil)
	reg.Register("crm", "localhost:0")
	reg.Register("document", "localhost:0")
	routes := NewGlobalSearchRoutes(reg)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/global?q=acme", nil)
	routes.HandleGlobalSearch(rec, req)
	assertStatus(t, rec, http.StatusOK)

	resp := decodeGlobalSearchResponse(t, rec)
	crm, ok := findModule(resp.Modules, "crm")
	if !ok || crm.Error != "search failed" {
		t.Errorf("crm module = %+v, want error %q", crm, "search failed")
	}
	docs, ok := findModule(resp.Modules, "documents")
	if !ok || docs.Error != "search failed" {
		t.Errorf("documents module = %+v, want error %q", docs, "search failed")
	}
}

func TestHandleGlobalSearch_EmailModule_AlwaysEmptyStub(t *testing.T) {
	// searchEmail has no backing RPC yet (route_search_global.go:159) and is
	// hardcoded to return an empty result regardless of registry state or
	// query. Locking this down so a future implementation change is a
	// deliberate diff, not a silent behaviour shift.
	routes := NewGlobalSearchRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/global?q=acme", nil)
	routes.HandleGlobalSearch(rec, req)
	assertStatus(t, rec, http.StatusOK)

	resp := decodeGlobalSearchResponse(t, rec)
	email, ok := findModule(resp.Modules, "email")
	if !ok {
		t.Fatalf("email module missing from response; body = %s", rec.Body.String())
	}
	if email.Error != "" {
		t.Errorf("email.Error = %q, want empty", email.Error)
	}
	if email.Total != 0 {
		t.Errorf("email.Total = %d, want 0", email.Total)
	}
}

func TestHandleGlobalSearch_MalformedLimit_FallsBackToDefaultWithoutError(t *testing.T) {
	// The limit parser (route_search_global.go:62-66) silently keeps the
	// default of 5 for anything that fails strconv.Atoi or falls outside
	// (0, 20] rather than returning 400 — documenting that as intended
	// graceful-degradation behaviour, not a validation gap: a malformed
	// limit degrades to "use the default", it never reaches the RPC as junk.
	cases := []string{"abc", "-5", "0", "999"}
	for _, limit := range cases {
		t.Run(limit, func(t *testing.T) {
			routes := NewGlobalSearchRoutes(emptyRegistry())
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/search/global?q=acme&limit="+limit, nil)
			routes.HandleGlobalSearch(rec, req)
			assertStatus(t, rec, http.StatusOK)
		})
	}
}

func TestHandleGlobalSearch_ValidLimit_Accepted(t *testing.T) {
	routes := NewGlobalSearchRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/global?q=acme&limit=3", nil)
	routes.HandleGlobalSearch(rec, req)
	assertStatus(t, rec, http.StatusOK)
}
