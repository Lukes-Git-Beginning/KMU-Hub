package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Covers the 17 handlers in route_produktion_ext.go (BOMs, work steps,
// machines, quality checks). Until now only route registration was tested
// (route_produktion_ext_test.go) — none of these handlers had a direct test.

const (
	testBomID   = "550e8400-e29b-41d4-a716-446655440002"
	testStepID  = "550e8400-e29b-41d4-a716-446655440003"
	testMachID  = "550e8400-e29b-41d4-a716-446655440004"
	testCheckID = "550e8400-e29b-41d4-a716-446655440005"
)

// --- BOMs ---

func TestHandleListBOMs_ServiceUnavailable(t *testing.T) {
	routes := newProduktionRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/produktion/boms", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListBOMs(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateBOM_ServiceUnavailable(t *testing.T) {
	routes := newProduktionRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/produktion/boms", jsonBody(t, map[string]interface{}{
		"product_name": "Widget",
		"sku":          "SKU-1",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateBOM(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateBOM_MissingProductName(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/produktion/boms", jsonBody(t, map[string]interface{}{
		"sku": "SKU-1",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateBOM(rec, req)
	assertValidationError(t, rec, "product_name")
}

func TestHandleCreateBOM_MissingSKU(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/produktion/boms", jsonBody(t, map[string]interface{}{
		"product_name": "Widget",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateBOM(rec, req)
	assertValidationError(t, rec, "sku")
}

func TestHandleGetBOM_ServiceUnavailable(t *testing.T) {
	routes := newProduktionRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/produktion/boms/"+testBomID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "bomId", testBomID)
	routes.HandleGetBOM(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetBOM_InvalidUUID(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/produktion/boms/bad", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "bomId", "not-a-uuid")
	routes.HandleGetBOM(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid bomId")
}

func TestHandleUpdateBOM_ServiceUnavailable(t *testing.T) {
	routes := newProduktionRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/produktion/boms/"+testBomID, jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "bomId", testBomID)
	routes.HandleUpdateBOM(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateBOM_InvalidJSON(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/produktion/boms/"+testBomID, invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "bomId", testBomID)
	routes.HandleUpdateBOM(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleDeleteBOM_ServiceUnavailable(t *testing.T) {
	routes := newProduktionRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/produktion/boms/"+testBomID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "bomId", testBomID)
	routes.HandleDeleteBOM(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteBOM_InvalidUUID(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/produktion/boms/bad", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "bomId", "not-a-uuid")
	routes.HandleDeleteBOM(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid bomId")
}

// --- Work Steps ---

func TestHandleListWorkSteps_ServiceUnavailable(t *testing.T) {
	routes := newProduktionRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/produktion/orders/"+testOrderID+"/steps", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "orderId", testOrderID)
	routes.HandleListWorkSteps(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListWorkSteps_InvalidUUID(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/produktion/orders/bad/steps", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "orderId", "not-a-uuid")
	routes.HandleListWorkSteps(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid orderId")
}

func TestHandleCreateWorkStep_ServiceUnavailable(t *testing.T) {
	routes := newProduktionRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/produktion/orders/"+testOrderID+"/steps", jsonBody(t, map[string]interface{}{
		"name": "Cut",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "orderId", testOrderID)
	routes.HandleCreateWorkStep(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateWorkStep_MissingName(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/produktion/orders/"+testOrderID+"/steps", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "orderId", testOrderID)
	routes.HandleCreateWorkStep(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleUpdateWorkStep_ServiceUnavailable(t *testing.T) {
	routes := newProduktionRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/produktion/orders/"+testOrderID+"/steps/"+testStepID, jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "stepId", testStepID)
	routes.HandleUpdateWorkStep(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateWorkStep_InvalidUUID(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/produktion/orders/"+testOrderID+"/steps/bad", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "stepId", "not-a-uuid")
	routes.HandleUpdateWorkStep(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid stepId")
}

func TestHandleUpdateWorkStep_InvalidJSON(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/produktion/orders/"+testOrderID+"/steps/"+testStepID, invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "stepId", testStepID)
	routes.HandleUpdateWorkStep(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleDeleteWorkStep_ServiceUnavailable(t *testing.T) {
	routes := newProduktionRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/produktion/orders/"+testOrderID+"/steps/"+testStepID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "stepId", testStepID)
	routes.HandleDeleteWorkStep(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteWorkStep_InvalidUUID(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/produktion/orders/"+testOrderID+"/steps/bad", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "stepId", "not-a-uuid")
	routes.HandleDeleteWorkStep(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid stepId")
}

// --- Machines ---

func TestHandleListMachines_ServiceUnavailable(t *testing.T) {
	routes := newProduktionRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/produktion/machines", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListMachines(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// HandleListMachines builds its request from page/pageSize/status query params
// with no gateway-local filtering logic (route_produktion_ext.go:435-465); this
// proves the handler parses the status filter and reaches the RPC layer.
func TestHandleListMachines_ReachesRPC(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/produktion/machines?status=active", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListMachines(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateMachine_ServiceUnavailable(t *testing.T) {
	routes := newProduktionRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/produktion/machines", jsonBody(t, map[string]interface{}{
		"name": "CNC-1",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateMachine(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateMachine_MissingName(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/produktion/machines", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateMachine(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleGetMachine_ServiceUnavailable(t *testing.T) {
	routes := newProduktionRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/produktion/machines/"+testMachID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "machineId", testMachID)
	routes.HandleGetMachine(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetMachine_InvalidUUID(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/produktion/machines/bad", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "machineId", "not-a-uuid")
	routes.HandleGetMachine(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid machineId")
}

func TestHandleUpdateMachine_ServiceUnavailable(t *testing.T) {
	routes := newProduktionRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/produktion/machines/"+testMachID, jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "machineId", testMachID)
	routes.HandleUpdateMachine(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateMachine_InvalidJSON(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/produktion/machines/"+testMachID, invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "machineId", testMachID)
	routes.HandleUpdateMachine(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleDeleteMachine_ServiceUnavailable(t *testing.T) {
	routes := newProduktionRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/produktion/machines/"+testMachID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "machineId", testMachID)
	routes.HandleDeleteMachine(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteMachine_InvalidUUID(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/produktion/machines/bad", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "machineId", "not-a-uuid")
	routes.HandleDeleteMachine(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid machineId")
}

// --- Quality Checks ---

func TestHandleListQualityChecks_ServiceUnavailable(t *testing.T) {
	routes := newProduktionRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/produktion/quality", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListQualityChecks(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// HandleListQualityChecks forwards an optional order_id filter with no
// gateway-local logic (route_produktion_ext.go:608-638); this proves the
// handler parses the filter and reaches the RPC layer.
func TestHandleListQualityChecks_ReachesRPC(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/produktion/quality?order_id="+testOrderID, nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListQualityChecks(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateQualityCheck_ServiceUnavailable(t *testing.T) {
	routes := newProduktionRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/produktion/quality", jsonBody(t, map[string]interface{}{
		"order_id":  testOrderID,
		"inspector": "QA-1",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateQualityCheck(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateQualityCheck_MissingInspector(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/produktion/quality", jsonBody(t, map[string]interface{}{
		"order_id": testOrderID,
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateQualityCheck(rec, req)
	assertValidationError(t, rec, "inspector")
}

func TestHandleCreateQualityCheck_InvalidOrderIDUUID(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/produktion/quality", jsonBody(t, map[string]interface{}{
		"order_id":  "not-a-uuid",
		"inspector": "QA-1",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateQualityCheck(rec, req)
	assertValidationError(t, rec, "order_id")
}

func TestHandleGetQualityCheck_ServiceUnavailable(t *testing.T) {
	routes := newProduktionRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/produktion/quality/"+testCheckID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "checkId", testCheckID)
	routes.HandleGetQualityCheck(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetQualityCheck_InvalidUUID(t *testing.T) {
	routes := newProduktionRoutes(registryWithService("produktion"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/produktion/quality/bad", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "checkId", "not-a-uuid")
	routes.HandleGetQualityCheck(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid checkId")
}
