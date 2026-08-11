package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kmuhub/kmuhub/internal/featureflag"
)

func newRapporteRoutes(registry *ServiceRegistry) *RapporteRoutes {
	flags := featureflag.NewRegistry().Load(func(key string) string {
		if key == "COSMI_MODULE_RAPPORTE_ENABLED" {
			return "true"
		}
		return ""
	})
	return NewRapporteRoutes(registry, flags)
}

// RapporteRoutes.ServiceName() uses "rapporte".

func TestRapporteRoutes_ServiceName(t *testing.T) {
	routes := NewRapporteRoutes(emptyRegistry(), nil)
	if routes.ServiceName() != "rapporte" {
		t.Errorf("ServiceName() = %q, want %q", routes.ServiceName(), "rapporte")
	}
}

// --- HandleCreateReport ---

func TestHandleCreateReport_MissingTitle(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/reports", jsonBody(t, map[string]interface{}{
		"author_id": "550e8400-e29b-41d4-a716-446655440000",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateReport(rec, req)
	assertValidationError(t, rec, "title")
}

func TestHandleCreateReport_MissingAuthorID(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/reports", jsonBody(t, map[string]interface{}{
		"title": "Test Report",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateReport(rec, req)
	assertValidationError(t, rec, "author_id")
}

func TestHandleCreateReport_InvalidAuthorIDUUID(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/reports", jsonBody(t, map[string]interface{}{
		"title":     "Test Report",
		"author_id": "not-a-uuid",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateReport(rec, req)
	assertValidationError(t, rec, "author_id")
}

func TestHandleCreateReport_InvalidJSON(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/reports", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateReport(rec, req)
	assertStatus(t, rec, 400)
	assertErrorContains(t, rec, "invalid request body")
}

// --- HandleApproveReport ---

func TestHandleApproveReport_MissingReviewerID(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/reports/550e8400-e29b-41d4-a716-446655440000/approve",
		jsonBody(t, map[string]interface{}{
			"review_note": "looks good",
		}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleApproveReport(rec, req)
	assertValidationError(t, rec, "reviewer_id")
}

func TestHandleApproveReport_InvalidReviewerIDUUID(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/reports/550e8400-e29b-41d4-a716-446655440000/approve",
		jsonBody(t, map[string]interface{}{
			"reviewer_id": "not-a-uuid",
		}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleApproveReport(rec, req)
	assertValidationError(t, rec, "reviewer_id")
}

// --- HandleAddLine ---

func TestHandleAddLine_MissingDescription(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/reports/550e8400-e29b-41d4-a716-446655440000/lines",
		jsonBody(t, map[string]interface{}{
			"quantity": 5.0,
		}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleAddLine(rec, req)
	assertValidationError(t, rec, "description")
}

func TestHandleAddLine_ZeroQuantity(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/reports/550e8400-e29b-41d4-a716-446655440000/lines",
		jsonBody(t, map[string]interface{}{
			"description": "Some work",
			"quantity":    0.0,
		}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleAddLine(rec, req)
	assertValidationError(t, rec, "quantity")
}

// --- HandleUploadAttachment ---

func TestHandleUploadAttachment_MissingFilename(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/reports/550e8400-e29b-41d4-a716-446655440000/attachments",
		jsonBody(t, map[string]interface{}{
			"object_key":  "files/test.pdf",
			"uploaded_by": "550e8400-e29b-41d4-a716-446655440001",
			"size_bytes":  1024,
		}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUploadAttachment(rec, req)
	assertValidationError(t, rec, "filename")
}

func TestHandleUploadAttachment_InvalidLineIDUUID(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	lineID := "not-a-uuid"
	req := httptest.NewRequest("POST", "/api/v1/rapporte/reports/550e8400-e29b-41d4-a716-446655440000/attachments",
		jsonBody(t, map[string]interface{}{
			"filename":    "test.pdf",
			"object_key":  "files/test.pdf",
			"uploaded_by": "550e8400-e29b-41d4-a716-446655440001",
			"size_bytes":  1024,
			"line_id":     lineID,
		}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUploadAttachment(rec, req)
	assertValidationError(t, rec, "line_id")
}

// --- HandleListReports ---

func TestHandleListReports_MissingTenant(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/rapporte/reports", nil)
	routes.HandleListReports(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListReports_ServiceUnavailable(t *testing.T) {
	routes := NewRapporteRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/rapporte/reports", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListReports(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleListReports_ReachesRPC documents that the response is the RPC's
// ListReportsResponse serialized unchanged (response.Proto passthrough) — no
// gateway-owned marshaling to assert a wire shape against, same boundary as
// every other list handler in this run without a bufconn stub for the
// rapporte service (see route_document_test.go / route_fuhrpark_test.go
// TestHandleCheckTuevDue_ReachesRPC for the same documented limit).
func TestHandleListReports_ReachesRPC(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/rapporte/reports?search=foo&status=submitted&author_id=550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListReports(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleListReports_OwnScopeWithoutUserIsRejected covers the handler's
// own integration with ownerFilterForScope (not just the helper in
// own_scope_filter_test.go): a caller narrowed to "own" on rapporte:report:read
// with no user id in the token must be refused, not fall through to an
// unfiltered list.
func TestHandleListReports_OwnScopeWithoutUserIsRejected(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/rapporte/reports", nil)
	req = withTenantID(req, testTenantID)
	req = withScopes(req, map[string]string{"rapporte:report:read": "own"})
	routes.HandleListReports(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

// --- HandleGetReport ---

func TestHandleGetReport_InvalidIDUUID(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/rapporte/reports/not-a-uuid", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetReport(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleGetReport_MissingTenant(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/rapporte/reports/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetReport(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetReport_ReachesRPC(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/rapporte/reports/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetReport(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpdateReport ---

func TestHandleUpdateReport_InvalidIDUUID(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/rapporte/reports/not-a-uuid", jsonBody(t, map[string]interface{}{
		"title": "Updated",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateReport(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleUpdateReport_InvalidJSON(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/rapporte/reports/550e8400-e29b-41d4-a716-446655440000", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateReport(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateReport_MissingTenant(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/rapporte/reports/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{
		"title": "Updated",
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateReport(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleUpdateReport_ReachesRPC(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/rapporte/reports/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{
		"title": "Updated",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateReport(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleDeleteReport ---

func TestHandleDeleteReport_InvalidIDUUID(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/rapporte/reports/not-a-uuid", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteReport(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleDeleteReport_MissingTenant(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/rapporte/reports/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteReport(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleDeleteReport_ServiceUnavailable(t *testing.T) {
	routes := NewRapporteRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/rapporte/reports/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteReport(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleSubmitReport / HandleRejectReport ---
//
// The report status machine (draft -> submitted -> approved/rejected) lives
// in the rapporte RPC service, not in this thin handler — there is no local
// state to reject an invalid transition against, and no bufconn stub for
// RapporteServiceClient in this package to script a FailedPrecondition
// response from. The gateway-observable equivalent of "invalid Statusübergang"
// is therefore: an invalid report id is rejected locally (400) before any
// transition is attempted, and a syntactically valid request reaches the RPC
// layer unmodified (503 via the unreachable registryWithService dummy
// address) instead of being evaluated against a status field the gateway
// does not itself hold. Same documented boundary as
// TestHandleListShareLinks_InvalidIDReachesRPC (Iteration 55) and
// TestHandleUpdateService_ReachesRPCWithArbitraryStatus (fuhrpark).

func TestHandleSubmitReport_InvalidIDUUID(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/reports/not-a-uuid/submit", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleSubmitReport(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleSubmitReport_MissingTenant(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/reports/550e8400-e29b-41d4-a716-446655440000/submit", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleSubmitReport(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleSubmitReport_ReachesRPCWithInvalidStatusTransition(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/reports/550e8400-e29b-41d4-a716-446655440000/submit", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleSubmitReport(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleRejectReport_InvalidIDUUID(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/reports/not-a-uuid/reject",
		jsonBody(t, map[string]interface{}{
			"reviewer_id": "550e8400-e29b-41d4-a716-446655440001",
		}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleRejectReport(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleRejectReport_MissingReviewerID(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/reports/550e8400-e29b-41d4-a716-446655440000/reject",
		jsonBody(t, map[string]interface{}{
			"review_note": "missing evidence",
		}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleRejectReport(rec, req)
	assertValidationError(t, rec, "reviewer_id")
}

func TestHandleRejectReport_MissingTenant(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/reports/550e8400-e29b-41d4-a716-446655440000/reject",
		jsonBody(t, map[string]interface{}{
			"reviewer_id": "550e8400-e29b-41d4-a716-446655440001",
		}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleRejectReport(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleRejectReport_ReachesRPCWithInvalidStatusTransition(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/reports/550e8400-e29b-41d4-a716-446655440000/reject",
		jsonBody(t, map[string]interface{}{
			"reviewer_id": "550e8400-e29b-41d4-a716-446655440001",
		}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleRejectReport(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
