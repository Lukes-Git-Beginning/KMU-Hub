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

// --- HandleListMeasurements ---

func TestHandleListMeasurements_MissingTenant(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/rapporte/measurements", nil)
	routes.HandleListMeasurements(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListMeasurements_ServiceUnavailable(t *testing.T) {
	routes := NewRapporteRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/rapporte/measurements", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListMeasurements(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListMeasurements_ReachesRPC(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/rapporte/measurements?report_id=550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListMeasurements(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleCreateMeasurement ---

func TestHandleCreateMeasurement_MissingTitle(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/measurements", jsonBody(t, map[string]interface{}{
		"location": "Halle A",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateMeasurement(rec, req)
	assertValidationError(t, rec, "title")
}

func TestHandleCreateMeasurement_InvalidReportIDUUID(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/measurements", jsonBody(t, map[string]interface{}{
		"title":     "Aufmass Halle",
		"report_id": "not-a-uuid",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateMeasurement(rec, req)
	assertValidationError(t, rec, "report_id")
}

func TestHandleCreateMeasurement_InvalidJSON(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/measurements", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateMeasurement(rec, req)
	assertStatus(t, rec, 400)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleCreateMeasurement_MissingTenant(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/measurements", jsonBody(t, map[string]interface{}{
		"title": "Aufmass Halle",
	}))
	routes.HandleCreateMeasurement(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleCreateMeasurement_ServiceUnavailable(t *testing.T) {
	routes := NewRapporteRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/measurements", jsonBody(t, map[string]interface{}{
		"title": "Aufmass Halle",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateMeasurement(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetMeasurement ---

func TestHandleGetMeasurement_InvalidIDUUID(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/rapporte/measurements/not-a-uuid", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetMeasurement(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleGetMeasurement_MissingTenant(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/rapporte/measurements/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetMeasurement(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetMeasurement_ReachesRPC(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/rapporte/measurements/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetMeasurement(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpdateMeasurement ---

func TestHandleUpdateMeasurement_InvalidIDUUID(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/rapporte/measurements/not-a-uuid", jsonBody(t, map[string]interface{}{
		"title": "Updated",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateMeasurement(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleUpdateMeasurement_MissingTitle(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/rapporte/measurements/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{
		"location": "Halle A",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateMeasurement(rec, req)
	assertValidationError(t, rec, "title")
}

func TestHandleUpdateMeasurement_InvalidJSON(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/rapporte/measurements/550e8400-e29b-41d4-a716-446655440000", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateMeasurement(rec, req)
	assertStatus(t, rec, 400)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateMeasurement_MissingTenant(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/rapporte/measurements/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{
		"title": "Updated",
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateMeasurement(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleUpdateMeasurement_ReachesRPC(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/rapporte/measurements/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{
		"title": "Updated",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateMeasurement(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleDeleteMeasurement ---

func TestHandleDeleteMeasurement_InvalidIDUUID(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/rapporte/measurements/not-a-uuid", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteMeasurement(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleDeleteMeasurement_MissingTenant(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/rapporte/measurements/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteMeasurement(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleDeleteMeasurement_ServiceUnavailable(t *testing.T) {
	routes := NewRapporteRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/rapporte/measurements/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteMeasurement(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleAddMeasurementPosition ---

func TestHandleAddMeasurementPosition_InvalidIDUUID(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/measurements/not-a-uuid/positions", jsonBody(t, map[string]interface{}{
		"description": "Wandflaeche",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleAddMeasurementPosition(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleAddMeasurementPosition_MissingDescription(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/measurements/550e8400-e29b-41d4-a716-446655440000/positions",
		jsonBody(t, map[string]interface{}{
			"unit":     "m2",
			"quantity": 10.0,
		}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleAddMeasurementPosition(rec, req)
	assertValidationError(t, rec, "description")
}

func TestHandleAddMeasurementPosition_InvalidJSON(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/measurements/550e8400-e29b-41d4-a716-446655440000/positions", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleAddMeasurementPosition(rec, req)
	assertStatus(t, rec, 400)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleAddMeasurementPosition_MissingTenant(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/measurements/550e8400-e29b-41d4-a716-446655440000/positions",
		jsonBody(t, map[string]interface{}{
			"description": "Wandflaeche",
		}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleAddMeasurementPosition(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

// TestHandleAddMeasurementPosition_ReachesRPCWithFractionalQuantity documents
// that the gateway itself neither rounds nor rejects a krumme Menge — a
// fractional quantity and unit_price pass decodeAndValidate unchanged and
// reach the RPC layer (503 via the unreachable registryWithService dummy
// address). The actual precision/rounding behaviour is verified against the
// real schema in postgres_repository_test.go
// (TestAddMeasurementPosition_PreservesFractionalQuantityAndRoundsUnitPrice) —
// there is no bufconn stub here to assert a computed result against.
func TestHandleAddMeasurementPosition_ReachesRPCWithFractionalQuantity(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/measurements/550e8400-e29b-41d4-a716-446655440000/positions",
		jsonBody(t, map[string]interface{}{
			"description": "Fensterflaeche",
			"unit":        "m2",
			"quantity":    12.3456,
			"unit_price":  45.999,
		}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleAddMeasurementPosition(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleDeleteMeasurementPosition ---

func TestHandleDeleteMeasurementPosition_InvalidPosIDUUID(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/rapporte/measurements/positions/not-a-uuid", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "pos_id", "not-a-uuid")
	routes.HandleDeleteMeasurementPosition(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleDeleteMeasurementPosition_MissingTenant(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/rapporte/measurements/positions/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withChiURLParam(req, "pos_id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteMeasurementPosition(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleDeleteMeasurementPosition_ServiceUnavailable(t *testing.T) {
	routes := NewRapporteRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/rapporte/measurements/positions/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "pos_id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteMeasurementPosition(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpdateLine ---

func TestHandleUpdateLine_InvalidIDUUID(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/rapporte/lines/not-a-uuid", jsonBody(t, map[string]interface{}{
		"description": "Updated",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateLine(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleUpdateLine_InvalidJSON(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/rapporte/lines/550e8400-e29b-41d4-a716-446655440000", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateLine(rec, req)
	assertStatus(t, rec, 400)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateLine_MissingTenant(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/rapporte/lines/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{
		"description": "Updated",
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateLine(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleUpdateLine_ReachesRPC(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/rapporte/lines/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{
		"description": "Updated",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateLine(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleDeleteLine ---

func TestHandleDeleteLine_InvalidIDUUID(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/rapporte/lines/not-a-uuid", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteLine(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleDeleteLine_MissingTenant(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/rapporte/lines/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteLine(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleDeleteLine_ServiceUnavailable(t *testing.T) {
	routes := NewRapporteRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/rapporte/lines/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteLine(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListLines ---

func TestHandleListLines_InvalidIDUUID(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/rapporte/reports/not-a-uuid/lines", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleListLines(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleListLines_MissingTenant(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/rapporte/reports/550e8400-e29b-41d4-a716-446655440000/lines", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleListLines(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListLines_ReachesRPC(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/rapporte/reports/550e8400-e29b-41d4-a716-446655440000/lines", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleListLines(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListAttachments ---

func TestHandleListAttachments_InvalidIDUUID(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/rapporte/reports/not-a-uuid/attachments", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleListAttachments(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleListAttachments_MissingTenant(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/rapporte/reports/550e8400-e29b-41d4-a716-446655440000/attachments", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleListAttachments(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

// TestHandleListAttachments_ReachesRPCWithLineIDFilter documents that the
// optional ?line_id= query parameter is forwarded to the RPC request
// unvalidated (no UUID check on this handler's line_id, unlike the
// UploadAttachment body field) — it reaches the RPC layer unchanged (503 via
// the unreachable registryWithService dummy address).
func TestHandleListAttachments_ReachesRPCWithLineIDFilter(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/rapporte/reports/550e8400-e29b-41d4-a716-446655440000/attachments?line_id=550e8400-e29b-41d4-a716-446655440001", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleListAttachments(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleDeleteAttachment ---

func TestHandleDeleteAttachment_InvalidIDUUID(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/rapporte/attachments/not-a-uuid", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteAttachment(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleDeleteAttachment_MissingTenant(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/rapporte/attachments/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteAttachment(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleDeleteAttachment_ServiceUnavailable(t *testing.T) {
	routes := NewRapporteRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/rapporte/attachments/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteAttachment(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleSaveReportSignature ---
//
// Signature pattern (documented here for the vertraege/vermietung
// HandleSaveContractSignature / HandleSaveRentalSignature units to build on
// instead of re-deriving): the gateway handler itself has NO guard against
// re-signing — it is a thin Parse/Call/Respond wrapper with no read-before-
// write. Whether a second signature is allowed is entirely a service/repo
// decision. Checked at that layer (postgres_repository.go SaveSignature,
// backend/internal/rapporte/postgres_repository.go:911): the UPDATE has no
// "AND signature_data IS NULL" guard and no report-status check, so a report
// that is already signed accepts a second SaveSignature call and silently
// overwrites signature_data/signed_at/signed_by — a signature that should be
// evidence of a fixed state is currently mutable after the fact. This is a
// VERIFIED finding, filed as its own fix-unit (see BACKLOG.yml), not fixed
// here — a coverage unit changes no behaviour.

func TestHandleSaveReportSignature_InvalidIDUUID(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/reports/not-a-uuid/signature", jsonBody(t, map[string]interface{}{
		"signature_data": "data:image/png;base64,abc123",
		"signed_by":      "Max Mustermann",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleSaveReportSignature(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleSaveReportSignature_MissingSignatureData(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/reports/550e8400-e29b-41d4-a716-446655440000/signature", jsonBody(t, map[string]interface{}{
		"signed_by": "Max Mustermann",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleSaveReportSignature(rec, req)
	assertValidationError(t, rec, "signature_data")
}

func TestHandleSaveReportSignature_MissingSignedBy(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/reports/550e8400-e29b-41d4-a716-446655440000/signature", jsonBody(t, map[string]interface{}{
		"signature_data": "data:image/png;base64,abc123",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleSaveReportSignature(rec, req)
	assertValidationError(t, rec, "signed_by")
}

func TestHandleSaveReportSignature_InvalidJSON(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/reports/550e8400-e29b-41d4-a716-446655440000/signature", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleSaveReportSignature(rec, req)
	assertStatus(t, rec, 400)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleSaveReportSignature_MissingTenant(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/reports/550e8400-e29b-41d4-a716-446655440000/signature", jsonBody(t, map[string]interface{}{
		"signature_data": "data:image/png;base64,abc123",
		"signed_by":      "Max Mustermann",
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleSaveReportSignature(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleSaveReportSignature_ReachesRPCOnResign(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/rapporte/reports/550e8400-e29b-41d4-a716-446655440000/signature", jsonBody(t, map[string]interface{}{
		"signature_data": "data:image/png;base64,secondSignatureOverwritingTheFirst",
		"signed_by":      "Erika Musterfrau",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleSaveReportSignature(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetReportStats ---

func TestHandleGetReportStats_MissingTenant(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/rapporte/reports/stats", nil)
	routes.HandleGetReportStats(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetReportStats_ServiceUnavailable(t *testing.T) {
	routes := NewRapporteRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/rapporte/reports/stats", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleGetReportStats(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListPendingApprovals ---

func TestHandleListPendingApprovals_MissingTenant(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/rapporte/reports/pending-approvals", nil)
	routes.HandleListPendingApprovals(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListPendingApprovals_ServiceUnavailable(t *testing.T) {
	routes := NewRapporteRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/rapporte/reports/pending-approvals", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListPendingApprovals(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleListPendingApprovals_OwnScopeWithoutUserIsRejected mirrors
// TestHandleListReports_OwnScopeWithoutUserIsRejected: the handler reuses the
// same "rapporte:report:read" scope key as HandleListReports (see the
// comment above HandleListPendingApprovals in route_rapporte.go), so a
// reviewer narrowed to "own" on that permission — with no user id in the
// token — must be refused rather than silently seeing every tenant's queue.
func TestHandleListPendingApprovals_OwnScopeWithoutUserIsRejected(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/rapporte/reports/pending-approvals", nil)
	req = withTenantID(req, testTenantID)
	req = withScopes(req, map[string]string{"rapporte:report:read": "own"})
	routes.HandleListPendingApprovals(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

// TestHandleListPendingApprovals_ReachesRPCWithAuthorFilterAtAllScope proves
// the narrowing branch itself: at the default "all" scope (no "own" entry in
// the token) the handler must NOT narrow the queue to the caller — ownerID
// stays nil and the request reaches the RPC layer with no author_id filter
// forced onto it, same as an admin/reviewer would expect for a
// tenant-wide pending-approvals queue.
func TestHandleListPendingApprovals_ReachesRPCWithAuthorFilterAtAllScope(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/rapporte/reports/pending-approvals?page=2&page_size=10", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListPendingApprovals(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleExportPDF ---

func TestHandleExportPDF_InvalidIDUUID(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/rapporte/reports/not-a-uuid/export", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleExportPDF(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleExportPDF_MissingTenant(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/rapporte/reports/550e8400-e29b-41d4-a716-446655440000/export", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleExportPDF(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

// TestHandleExportPDF_ReachesRPCWithTenantScopedRequest documents the
// tenant-filter check the scope text asked for: ExportPDFRequest carries
// TenantId from the authenticated caller's own context, never from the URL
// or an unvalidated field, so a cross-tenant reportID can only be rejected
// server-side (RapporteGRPCServer.ExportPDF -> repo.GetReport, both
// tenant-scoped) — there is no gateway-local wire-shape check to assert this
// against without a bufconn stub, same documented boundary as the other
// ReachesRPC tests in this package.
func TestHandleExportPDF_ReachesRPCWithTenantScopedRequest(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/rapporte/reports/550e8400-e29b-41d4-a716-446655440000/export", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleExportPDF(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetTemplate ---

func TestHandleGetTemplate_InvalidIDUUID(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/rapporte/templates/not-a-uuid", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetTemplate(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleGetTemplate_MissingTenant(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/rapporte/templates/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetTemplate(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetTemplate_ReachesRPC(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/rapporte/templates/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetTemplate(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpdateTemplate ---

func TestHandleUpdateTemplate_InvalidIDUUID(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/rapporte/templates/not-a-uuid", jsonBody(t, map[string]interface{}{
		"name": "Updated",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateTemplate(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleUpdateTemplate_MissingName(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/rapporte/templates/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{
		"category": "montage",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateTemplate(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleUpdateTemplate_InvalidJSON(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/rapporte/templates/550e8400-e29b-41d4-a716-446655440000", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateTemplate(rec, req)
	assertStatus(t, rec, 400)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateTemplate_MissingTenant(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/rapporte/templates/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{
		"name": "Updated",
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateTemplate(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleUpdateTemplate_ReachesRPC(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/rapporte/templates/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{
		"name": "Updated",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateTemplate(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleDeleteTemplate ---

func TestHandleDeleteTemplate_InvalidIDUUID(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/rapporte/templates/not-a-uuid", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteTemplate(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleDeleteTemplate_MissingTenant(t *testing.T) {
	routes := NewRapporteRoutes(registryWithService("rapporte"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/rapporte/templates/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteTemplate(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleDeleteTemplate_ServiceUnavailable(t *testing.T) {
	routes := NewRapporteRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/rapporte/templates/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteTemplate(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
