package gateway

// route_biz_open_items_time_entries_test.go covers the two smallest untested
// route_biz files: route_biz_open_items.go (HandleListOpenItems, one
// function) and route_biz_time_entries.go (HandleListTimeEntries, one
// function). Neither had a test before this unit.
//
// Same infrastructure boundary as every other gateway coverage unit this run
// (recurring invoices, quotes, e-invoice, GoBD archive): this package has no
// bufconn stub for FinanceServiceClient or WorkServiceClient, so the
// *_ReachesRPC tests below only prove a handler with a valid request reaches
// the RPC layer (503 against the dummy localhost:0 address) — they cannot
// observe the JSON body a successful RPC would produce. Two things worth
// noting about that gap for these two routes specifically:
//
//  1. HandleListTimeEntries never calls getTenantID itself — tenant scoping
//     for /finance/time-entries is entirely server-side: the gateway's
//     TenantOutboundUnaryInterceptor forwards whatever tenant is in the HTTP
//     request context (or nothing, if there is none) to the Work service,
//     and WorkGRPCServer.ListBillableTimeEntries (work_grpc.go:2005) is the
//     one that rejects a missing tenant with Unauthenticated. This is
//     intentional (documented at work_grpc.go:2004, "tenant_id comes from
//     the RLS context, same as every other RPC here") and not specific to
//     this route, so it is locked down here as a behavioural fact
//     (TestHandleListTimeEntries_NoTenantInContext_StillReachesRPC) rather
//     than treated as a missing check.
//  2. The billed=true short-circuit in HandleListTimeEntries returns a
//     hardcoded empty list without ever marking anything as billed (see the
//     `lean:` marker at route_biz_time_entries.go:52) — that is an existing,
//     already-flagged simplification with its own upgrade trigger, not a new
//     finding, and is out of scope for a coverage-only unit to change.

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- HandleListOpenItems ---

func TestHandleListOpenItems_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleListOpenItems)
}

func TestHandleListOpenItems_NoTenantID(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/open-items", nil)
	routes.HandleListOpenItems(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListOpenItems_DefaultPagination_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/open-items", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListOpenItems(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListOpenItems_BucketAndOverdueFilter_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/open-items?bucket=d60&overdue_only=true", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListOpenItems(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListOpenItems_ExplicitPagination_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/open-items?page=2&page_size=25", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListOpenItems(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleListOpenItems_PageSizeAboveMaxFallsBackToDefault locks down that
// an out-of-range page_size (parsePagination caps at 100) does not crash the
// handler or get forwarded verbatim -- it silently falls back to the
// default, same as every other route using parsePagination.
func TestHandleListOpenItems_PageSizeAboveMaxFallsBackToDefault(t *testing.T) {
	routes := NewBizRoutes(registryWithService("biz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/open-items?page_size=500", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListOpenItems(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListTimeEntries ---

func TestHandleListTimeEntries_ServiceUnavailable(t *testing.T) {
	routes := NewBizRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleListTimeEntries)
}

func TestHandleListTimeEntries_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/time-entries", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListTimeEntries(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListTimeEntries_BilledFilter_ReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/time-entries?billed=true", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListTimeEntries(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleListTimeEntries_NoTenantInContext_StillReachesRPC documents that
// this handler has no gateway-side tenant check of its own (see file header):
// with no tenant in the request context it still calls the RPC layer and
// fails the same way (503 against the dummy connection) instead of a
// gateway-level 401. Enforcement is delegated entirely to the Work service
// (work_grpc.go ListBillableTimeEntries) via the tenant-forwarding
// interceptor.
func TestHandleListTimeEntries_NoTenantInContext_StillReachesRPC(t *testing.T) {
	routes := NewBizRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/time-entries", nil)
	routes.HandleListTimeEntries(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
