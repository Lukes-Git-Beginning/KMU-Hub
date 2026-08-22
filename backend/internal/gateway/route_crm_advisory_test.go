package gateway

// route_crm_advisory_test.go covers route_crm_advisory.go (unit
// cov-gateway-crm-advisory-protocols), which had no test file. Advisory
// protocols are one of the two RESTRICT foreign keys on contacts (the other is
// dialer_campaign_contacts) -- see internal/crm/contact/postgres_repository.go
// IsInUse. All handlers here are thin pass-throughs to the CRM gRPC client
// (no tenant check of their own, unlike route_crm_activities.go -- the tenant
// is threaded through context to the gRPC call and checked server-side), so
// this file only proves the pre-RPC branches: invalid path params, invalid
// JSON, and validation.
//
// The two behavioural done_when items for this unit are proven elsewhere,
// deliberately not re-pinned here as a gateway-level fake:
//   - Cross-tenant read returns 404: internal/server/crm_grpc_advisory_test.go
//     TestGetAdvisoryProtocol/"wrong tenant is treated as not found" proves
//     this against advisoryprotocol.Service.GetByID + a fake repository.
//   - Deleting a contact with a hanging protocol returns 409 with a reason,
//     not 500: internal/crm/contact/postgres_repository_db_test.go
//     TestRepository_IsInUse_AdvisoryProtocolBlocksDeletion_DB proves this
//     against the real RESTRICT constraint (migration 000137) and
//     Service.Delete; internal/server/crm_grpc.go maps ErrContactInUse to
//     codes.FailedPrecondition, which internal/gateway/helpers.go maps to
//     HTTP 409.

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const advisoryTestID = "22222222-2222-2222-2222-222222222222"
const advisoryContactID = "33333333-3333-3333-3333-333333333333"

// ============================================================================
// ServiceUnavailable — every handler resolves the CRM client first.
// ============================================================================

func TestCRMAdvisoryRoutes_ServiceUnavailable(t *testing.T) {
	routes := NewCRMRoutes(emptyRegistry(), nil)

	handlers := map[string]http.HandlerFunc{
		"HandleCreateAdvisoryProtocol":   routes.HandleCreateAdvisoryProtocol,
		"HandleGetAdvisoryProtocol":      routes.HandleGetAdvisoryProtocol,
		"HandleListAdvisoryProtocols":    routes.HandleListAdvisoryProtocols,
		"HandleUpdateAdvisoryProtocol":   routes.HandleUpdateAdvisoryProtocol,
		"HandleDeleteAdvisoryProtocol":   routes.HandleDeleteAdvisoryProtocol,
		"HandleHandOverAdvisoryProtocol": routes.HandleHandOverAdvisoryProtocol,
		"HandleAdvisoryProtocolPDF":      routes.HandleAdvisoryProtocolPDF,
		"HandleReferralReport":           routes.HandleReferralReport,
	}

	for name, h := range handlers {
		t.Run(name, func(t *testing.T) {
			testServiceUnavailable(t, h)
		})
	}
}

// ============================================================================
// HandleCreateAdvisoryProtocol
// ============================================================================

func TestHandleCreateAdvisoryProtocol_InvalidContactID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contacts/bad/advisory-protocols", nil)
	req = withChiURLParam(req, "contactId", "not-a-uuid")
	routes.HandleCreateAdvisoryProtocol(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid contactId")
}

func TestHandleCreateAdvisoryProtocol_ReachesRPC(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contacts/"+advisoryContactID+"/advisory-protocols", nil)
	req = withChiURLParam(req, "contactId", advisoryContactID)
	routes.HandleCreateAdvisoryProtocol(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleGetAdvisoryProtocol
// ============================================================================

func TestHandleGetAdvisoryProtocol_InvalidID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/advisory-protocols/bad", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetAdvisoryProtocol(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleGetAdvisoryProtocol_ReachesRPC(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/advisory-protocols/"+advisoryTestID, nil)
	req = withChiURLParam(req, "id", advisoryTestID)
	routes.HandleGetAdvisoryProtocol(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListAdvisoryProtocols
// ============================================================================

func TestHandleListAdvisoryProtocols_InvalidContactID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/contacts/bad/advisory-protocols", nil)
	req = withChiURLParam(req, "contactId", "not-a-uuid")
	routes.HandleListAdvisoryProtocols(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid contactId")
}

func TestHandleListAdvisoryProtocols_ReachesRPC(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/contacts/"+advisoryContactID+"/advisory-protocols", nil)
	req = withChiURLParam(req, "contactId", advisoryContactID)
	routes.HandleListAdvisoryProtocols(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleUpdateAdvisoryProtocol — the UUID check runs before decode/validate.
// ============================================================================

func TestHandleUpdateAdvisoryProtocol_InvalidID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/advisory-protocols/bad", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateAdvisoryProtocol(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateAdvisoryProtocol_InvalidJSON(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/advisory-protocols/"+advisoryTestID, invalidJSON())
	req = withChiURLParam(req, "id", advisoryTestID)
	routes.HandleUpdateAdvisoryProtocol(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateAdvisoryProtocol_MissingAdvisor(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/advisory-protocols/"+advisoryTestID, jsonBody(t, map[string]interface{}{
		"risk_class": 3,
	}))
	req = withChiURLParam(req, "id", advisoryTestID)
	routes.HandleUpdateAdvisoryProtocol(rec, req)
	assertValidationError(t, rec, "advisor")
}

func TestHandleUpdateAdvisoryProtocol_InvalidRiskClass(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/advisory-protocols/"+advisoryTestID, jsonBody(t, map[string]interface{}{
		"advisor":    "Jane Advisor",
		"risk_class": 9,
	}))
	req = withChiURLParam(req, "id", advisoryTestID)
	routes.HandleUpdateAdvisoryProtocol(rec, req)
	assertValidationError(t, rec, "risk_class")
}

func TestHandleUpdateAdvisoryProtocol_InvalidSelfAssessment(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/advisory-protocols/"+advisoryTestID, jsonBody(t, map[string]interface{}{
		"advisor":         "Jane Advisor",
		"risk_class":      3,
		"self_assessment": 6,
	}))
	req = withChiURLParam(req, "id", advisoryTestID)
	routes.HandleUpdateAdvisoryProtocol(rec, req)
	assertValidationError(t, rec, "self_assessment")
}

// TestHandleUpdateAdvisoryProtocol_ProductRiskClassNotValidated pins a real
// gap found while writing this test file: updateAdvisoryProtocolRequest.Products
// has no `dive` on its validate tag, so advisoryProduct's own `min=1,max=7` on
// RiskClass never runs (go-playground/validator only recurses into a slice of
// structs when the slice field itself carries `dive`; compare
// route_customization.go's `validate:"required,min=1,dive"` on a similar
// nested slice). A product with risk_class=0 or 99 reaches the RPC unvalidated
// instead of failing with 400. Not fixed here -- this is a coverage unit and
// the actual bug is pre-existing production code, not something introduced
// by this file. Filed as fix-gateway-advisory-product-riskclass-not-validated
// at the end of BACKLOG.yml.
func TestHandleUpdateAdvisoryProtocol_ProductRiskClassNotValidated(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/advisory-protocols/"+advisoryTestID, jsonBody(t, map[string]interface{}{
		"advisor":    "Jane Advisor",
		"risk_class": 3,
		"products": []map[string]interface{}{
			{"id": "p1", "name": "Fund A", "risk_class": 0, "recommended": true},
		},
	}))
	req = withChiURLParam(req, "id", advisoryTestID)
	routes.HandleUpdateAdvisoryProtocol(rec, req)
	// Documents the gap: an out-of-range product risk_class reaches the RPC
	// (503 here, since the client is unreachable) instead of failing with 400.
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateAdvisoryProtocol_ReachesRPC(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/advisory-protocols/"+advisoryTestID, jsonBody(t, map[string]interface{}{
		"advisor":    "Jane Advisor",
		"risk_class": 5,
		"products": []map[string]interface{}{
			{"id": "p1", "name": "Fund A", "risk_class": 4, "recommended": true},
		},
	}))
	req = withChiURLParam(req, "id", advisoryTestID)
	routes.HandleUpdateAdvisoryProtocol(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleDeleteAdvisoryProtocol
// ============================================================================

func TestHandleDeleteAdvisoryProtocol_InvalidID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/advisory-protocols/bad", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteAdvisoryProtocol(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleDeleteAdvisoryProtocol_ReachesRPC(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/advisory-protocols/"+advisoryTestID, nil)
	req = withChiURLParam(req, "id", advisoryTestID)
	routes.HandleDeleteAdvisoryProtocol(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleHandOverAdvisoryProtocol
// ============================================================================

func TestHandleHandOverAdvisoryProtocol_InvalidID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/advisory-protocols/bad/hand-over", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleHandOverAdvisoryProtocol(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleHandOverAdvisoryProtocol_ReachesRPC(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/advisory-protocols/"+advisoryTestID+"/hand-over", nil)
	req = withChiURLParam(req, "id", advisoryTestID)
	routes.HandleHandOverAdvisoryProtocol(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleAdvisoryProtocolPDF
// ============================================================================

func TestHandleAdvisoryProtocolPDF_InvalidID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/advisory-protocols/bad/pdf", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleAdvisoryProtocolPDF(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleAdvisoryProtocolPDF_ReachesRPC(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/advisory-protocols/"+advisoryTestID+"/pdf", nil)
	req = withChiURLParam(req, "id", advisoryTestID)
	routes.HandleAdvisoryProtocolPDF(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleReferralReport — no path param, tenant-wide.
// ============================================================================

func TestHandleReferralReport_ReachesRPC(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/contacts/referral-report", nil)
	routes.HandleReferralReport(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
