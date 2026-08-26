package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kmuhub/kmuhub/internal/featureflag"
	"github.com/kmuhub/kmuhub/internal/server/response"
	inventarv1 "github.com/kmuhub/kmuhub/proto/inventar/v1"
)

func newInventarRoutes(registry *ServiceRegistry) *InventarRoutes {
	flags := featureflag.NewRegistry().Load(func(key string) string {
		if key == "COSMI_MODULE_INVENTAR_ENABLED" {
			return "true"
		}
		return ""
	})
	return NewInventarRoutes(registry, flags)
}

// --- HandleCreateItem ---

func TestHandleCreateItem_MissingName(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/inventar/items", jsonBody(t, map[string]interface{}{
		"sku":      "SKU-001",
		"quantity": 10,
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateItem(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleCreateItem_MissingSKU(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/inventar/items", jsonBody(t, map[string]interface{}{
		"name":     "Widget A",
		"quantity": 10,
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateItem(rec, req)
	assertValidationError(t, rec, "sku")
}

func TestHandleCreateItem_InvalidJSON(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/inventar/items", invalidJSON())
	req = withTenantID(req, testTenantID)
	routes.HandleCreateItem(rec, req)
	assertStatus(t, rec, 400)
	assertErrorContains(t, rec, "invalid request body")
}

// --- HandleCreateWarning ---

func TestHandleCreateWarning_MissingItemID(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/inventar/warnings", jsonBody(t, map[string]interface{}{
		"threshold":        5,
		"current_quantity": 2,
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateWarning(rec, req)
	assertValidationError(t, rec, "item_id")
}

func TestHandleCreateWarning_InvalidItemIDUUID(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/inventar/warnings", jsonBody(t, map[string]interface{}{
		"item_id":          "not-a-uuid",
		"threshold":        5,
		"current_quantity": 2,
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateWarning(rec, req)
	assertValidationError(t, rec, "item_id")
}

// --- HandleTransferStock ---

func TestHandleTransferStock_MissingFromItemID(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/inventar/transfer", jsonBody(t, map[string]interface{}{
		"to_item_id": "550e8400-e29b-41d4-a716-446655440000",
		"quantity":   5,
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleTransferStock(rec, req)
	assertValidationError(t, rec, "from_item_id")
}

func TestHandleTransferStock_MissingToItemID(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/inventar/transfer", jsonBody(t, map[string]interface{}{
		"from_item_id": "550e8400-e29b-41d4-a716-446655440000",
		"quantity":     5,
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleTransferStock(rec, req)
	assertValidationError(t, rec, "to_item_id")
}

// --- HandleRecordMovement ---

func TestHandleRecordMovement_InvalidMovementType(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/inventar/items/550e8400-e29b-41d4-a716-446655440000/movements",
		jsonBody(t, map[string]interface{}{
			"movement_type": "invalid",
			"quantity":      5,
		}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleRecordMovement(rec, req)
	assertValidationError(t, rec, "movement_type")
}

// --- HandleListItems ---

func TestHandleListItems_ServiceUnavailable(t *testing.T) {
	routes := NewInventarRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/inventar/items", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListItems(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListItems_MissingTenant(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/inventar/items", nil)
	routes.HandleListItems(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListItems_ReachesRPC(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/inventar/items?search=widget&low_stock=true&location=Lager1", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListItems(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetItem ---

func TestHandleGetItem_InvalidIDUUID(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/inventar/items/not-a-uuid", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetItem(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleGetItem_ServiceUnavailable(t *testing.T) {
	routes := NewInventarRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("GET", "/api/v1/inventar/items/"+id, nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleGetItem(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetItem_ReachesRPC(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("GET", "/api/v1/inventar/items/"+id, nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleGetItem(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpdateItem ---

func TestHandleUpdateItem_InvalidIDUUID(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/inventar/items/not-a-uuid", jsonBody(t, map[string]interface{}{
		"name": "Widget B",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateItem(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateItem_InvalidJSON(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("PATCH", "/api/v1/inventar/items/"+id, invalidJSON())
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateItem(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateItem_ServiceUnavailable(t *testing.T) {
	routes := NewInventarRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("PATCH", "/api/v1/inventar/items/"+id, jsonBody(t, map[string]interface{}{
		"name": "Widget B",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateItem(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateItem_ReachesRPC(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("PATCH", "/api/v1/inventar/items/"+id, jsonBody(t, map[string]interface{}{
		"name":         "Widget B",
		"min_quantity": 5,
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateItem(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleDeleteItem ---

func TestHandleDeleteItem_InvalidIDUUID(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/inventar/items/not-a-uuid", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteItem(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleDeleteItem_ServiceUnavailable(t *testing.T) {
	routes := NewInventarRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("DELETE", "/api/v1/inventar/items/"+id, nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleDeleteItem(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteItem_ReachesRPC(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("DELETE", "/api/v1/inventar/items/"+id, nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleDeleteItem(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleAdjustStock ---

func TestHandleAdjustStock_InvalidIDUUID(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/inventar/items/not-a-uuid/adjust", jsonBody(t, map[string]interface{}{
		"delta":  5,
		"reason": "recount",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleAdjustStock(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleAdjustStock_InvalidPerformedByUUID(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("POST", "/api/v1/inventar/items/"+id+"/adjust", jsonBody(t, map[string]interface{}{
		"delta":        5,
		"performed_by": "not-a-uuid",
		"reason":       "recount",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleAdjustStock(rec, req)
	assertValidationError(t, rec, "performed_by")
}

// TestHandleAdjustStock_InvalidDeltaType covers the one locally-checked error
// path for an unusable quantity change: a non-numeric delta fails JSON
// decoding before validation ever runs.
func TestHandleAdjustStock_InvalidDeltaType(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("POST", "/api/v1/inventar/items/"+id+"/adjust", jsonBody(t, map[string]interface{}{
		"delta":  "not-a-number",
		"reason": "recount",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleAdjustStock(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

// TestHandleAdjustStock_MissingDelta_ReachesRPC documents a real gap:
// adjustStockRequest.Delta (route_inventar.go) carries no validate tag, so an
// omitted delta decodes to its zero value and is treated as a valid request
// instead of being rejected locally as a missing quantity change -- the
// request reaches the RPC layer unchanged. Coverage units don't change
// behaviour; see this iteration's journal entry for the finding.
func TestHandleAdjustStock_MissingDelta_ReachesRPC(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("POST", "/api/v1/inventar/items/"+id+"/adjust", jsonBody(t, map[string]interface{}{
		"reason": "recount",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleAdjustStock(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleAdjustStock_ServiceUnavailable(t *testing.T) {
	routes := NewInventarRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("POST", "/api/v1/inventar/items/"+id+"/adjust", jsonBody(t, map[string]interface{}{
		"delta":  5,
		"reason": "recount",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleAdjustStock(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleAdjustStock_ReachesRPC(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("POST", "/api/v1/inventar/items/"+id+"/adjust", jsonBody(t, map[string]interface{}{
		"delta":  -3,
		"reason": "breakage",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleAdjustStock(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListMovements ---

func TestHandleListMovements_InvalidIDUUID(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/inventar/items/not-a-uuid/movements", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleListMovements(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleListMovements_ServiceUnavailable(t *testing.T) {
	routes := NewInventarRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("GET", "/api/v1/inventar/items/"+id+"/movements", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleListMovements(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListMovements_ReachesRPC(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("GET", "/api/v1/inventar/items/"+id+"/movements?page=2&page_size=10", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleListMovements(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetStockHistory ---

func TestHandleGetStockHistory_InvalidIDUUID(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/inventar/items/not-a-uuid/history", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetStockHistory(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleGetStockHistory_ServiceUnavailable(t *testing.T) {
	routes := NewInventarRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("GET", "/api/v1/inventar/items/"+id+"/history", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleGetStockHistory(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetStockHistory_ReachesRPC(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("GET", "/api/v1/inventar/items/"+id+"/history", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleGetStockHistory(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestProtoListMovements_WireShape locks the shape HandleListMovements and
// HandleGetStockHistory (both response.Proto a *inventarv1.ListMovementsResponse,
// route_inventar.go) send to the frontend for an empty result. EmitUnpopulated
// is false (internal/server/response/response.go), so an empty repeated field
// is omitted from the body entirely rather than serialised as [] -- the same
// tolerant contract already locked in elsewhere in this package (see
// TestProtoListRecordings_WireShape in route_video_recording_test.go). The one
// shape that must never happen is an explicit "movements":null, which would
// force the frontend to null-check instead of just mapping over [].
func TestProtoListMovements_WireShape(t *testing.T) {
	rec := httptest.NewRecorder()
	response.Proto(rec, http.StatusOK, &inventarv1.ListMovementsResponse{
		Movements: []*inventarv1.Movement{},
		Total:     0,
	})
	var sink map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &sink); err != nil {
		t.Fatalf("empty response is not valid JSON: %v", err)
	}
	if movements, ok := sink["movements"]; ok && movements == nil {
		t.Errorf("movements serialised as null, want [] or omitted; body: %s", rec.Body.String())
	}
}

// --- HandleCreateInventurSession ---

func TestHandleCreateInventurSession_MissingName(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/inventar/inventur/sessions", jsonBody(t, map[string]interface{}{
		"item_ids": []string{"550e8400-e29b-41d4-a716-446655440000"},
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateInventurSession(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleCreateInventurSession_InvalidLocationIDUUID(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/inventar/inventur/sessions", jsonBody(t, map[string]interface{}{
		"name":        "Q3 Inventur",
		"location_id": "not-a-uuid",
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateInventurSession(rec, req)
	assertValidationError(t, rec, "location_id")
}

func TestHandleCreateInventurSession_InvalidDateFormat(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/inventar/inventur/sessions", jsonBody(t, map[string]interface{}{
		"name": "Q3 Inventur",
		"date": "not-a-date",
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateInventurSession(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid date format")
}

func TestHandleCreateInventurSession_ServiceUnavailable(t *testing.T) {
	routes := NewInventarRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/inventar/inventur/sessions", jsonBody(t, map[string]interface{}{
		"name": "Q3 Inventur",
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateInventurSession(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateInventurSession_ReachesRPC(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/inventar/inventur/sessions", jsonBody(t, map[string]interface{}{
		"name":        "Q3 Inventur",
		"date":        "2026-09-01",
		"location_id": "550e8400-e29b-41d4-a716-446655440000",
		"item_ids":    []string{"550e8400-e29b-41d4-a716-446655440001"},
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateInventurSession(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpdateInventurSessionStatus ---

func TestHandleUpdateInventurSessionStatus_InvalidIDUUID(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/inventar/inventur/sessions/not-a-uuid/status", jsonBody(t, map[string]interface{}{
		"status": "counting",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateInventurSessionStatus(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

// TestHandleUpdateInventurSessionStatus_InvalidStatusValue covers the one
// locally-checked error path for the session status transition: the
// updateInventurSessionStatusRequest.Status field only validates membership in
// the fixed oneof list (open/counting/review/completed, route_inventar.go).
// It does not check that the transition FROM the session's current status is
// legal -- that state-machine logic, if any, lives server-side.
func TestHandleUpdateInventurSessionStatus_InvalidStatusValue(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("PATCH", "/api/v1/inventar/inventur/sessions/"+id+"/status", jsonBody(t, map[string]interface{}{
		"status": "cancelled",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateInventurSessionStatus(rec, req)
	assertValidationError(t, rec, "status")
}

func TestHandleUpdateInventurSessionStatus_ServiceUnavailable(t *testing.T) {
	routes := NewInventarRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("PATCH", "/api/v1/inventar/inventur/sessions/"+id+"/status", jsonBody(t, map[string]interface{}{
		"status": "counting",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateInventurSessionStatus(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateInventurSessionStatus_ReachesRPC(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("PATCH", "/api/v1/inventar/inventur/sessions/"+id+"/status", jsonBody(t, map[string]interface{}{
		"status": "completed",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateInventurSessionStatus(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpsertInventurCount ---

func TestHandleUpsertInventurCount_InvalidIDUUID(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/inventar/inventur/sessions/not-a-uuid/counts", jsonBody(t, map[string]interface{}{
		"item_id": "550e8400-e29b-41d4-a716-446655440000",
		"counted": 3,
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpsertInventurCount(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpsertInventurCount_InvalidJSON(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("POST", "/api/v1/inventar/inventur/sessions/"+id+"/counts", invalidJSON())
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpsertInventurCount(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpsertInventurCount_MissingItemID(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("POST", "/api/v1/inventar/inventur/sessions/"+id+"/counts", jsonBody(t, map[string]interface{}{
		"counted": 3,
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpsertInventurCount(rec, req)
	assertValidationError(t, rec, "item_id")
}

func TestHandleUpsertInventurCount_ServiceUnavailable(t *testing.T) {
	routes := NewInventarRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("POST", "/api/v1/inventar/inventur/sessions/"+id+"/counts", jsonBody(t, map[string]interface{}{
		"item_id": "550e8400-e29b-41d4-a716-446655440001",
		"counted": 3,
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpsertInventurCount(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpsertInventurCount_ReachesRPC(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("POST", "/api/v1/inventar/inventur/sessions/"+id+"/counts", jsonBody(t, map[string]interface{}{
		"item_id": "550e8400-e29b-41d4-a716-446655440001",
		"counted": 0,
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpsertInventurCount(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleBookInventurDifferences ---

// TestHandleBookInventurDifferences_InvalidIDUUID covers the done_when
// requirement that an invalid/missing session ID is rejected before the
// booking RPC is ever attempted.
func TestHandleBookInventurDifferences_InvalidIDUUID(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/inventar/inventur/sessions/not-a-uuid/book", jsonBody(t, map[string]interface{}{}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleBookInventurDifferences(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleBookInventurDifferences_InvalidBookedByUUID(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("POST", "/api/v1/inventar/inventur/sessions/"+id+"/book", jsonBody(t, map[string]interface{}{
		"booked_by": "not-a-uuid",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleBookInventurDifferences(rec, req)
	assertValidationError(t, rec, "booked_by")
}

func TestHandleBookInventurDifferences_ServiceUnavailable(t *testing.T) {
	routes := NewInventarRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("POST", "/api/v1/inventar/inventur/sessions/"+id+"/book", jsonBody(t, map[string]interface{}{}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleBookInventurDifferences(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleBookInventurDifferences_ReachesRPC(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("POST", "/api/v1/inventar/inventur/sessions/"+id+"/book", jsonBody(t, map[string]interface{}{
		"booked_by": "550e8400-e29b-41d4-a716-446655440001",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleBookInventurDifferences(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListWarnings ---

func TestHandleListWarnings_ServiceUnavailable(t *testing.T) {
	routes := NewInventarRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/inventar/warnings", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListWarnings(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListWarnings_MissingTenant(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/inventar/warnings", nil)
	routes.HandleListWarnings(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListWarnings_ReachesRPC(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/inventar/warnings?status=open&page=2&page_size=10", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListWarnings(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpdateWarning ---

func TestHandleUpdateWarning_InvalidIDUUID(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/inventar/warnings/not-a-uuid", jsonBody(t, map[string]interface{}{
		"status": "resolved",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateWarning(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

// TestHandleUpdateWarning_InvalidJSON documents that updateWarningRequest.Status
// (route_inventar.go) carries no validate tag -- malformed JSON is the only
// error path decodeAndValidate can reject locally for this handler; any string
// value for status reaches the RPC layer unchecked.
func TestHandleUpdateWarning_InvalidJSON(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("PATCH", "/api/v1/inventar/warnings/"+id, invalidJSON())
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateWarning(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateWarning_ServiceUnavailable(t *testing.T) {
	routes := NewInventarRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("PATCH", "/api/v1/inventar/warnings/"+id, jsonBody(t, map[string]interface{}{
		"status": "resolved",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateWarning(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateWarning_ReachesRPC(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("PATCH", "/api/v1/inventar/warnings/"+id, jsonBody(t, map[string]interface{}{
		"status": "resolved",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateWarning(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleAcknowledgeWarning ---

func TestHandleAcknowledgeWarning_InvalidIDUUID(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/inventar/warnings/not-a-uuid/acknowledge", jsonBody(t, map[string]interface{}{}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleAcknowledgeWarning(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleAcknowledgeWarning_InvalidJSON(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("POST", "/api/v1/inventar/warnings/"+id+"/acknowledge", invalidJSON())
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleAcknowledgeWarning(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleAcknowledgeWarning_InvalidAcknowledgedByUUID(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("POST", "/api/v1/inventar/warnings/"+id+"/acknowledge", jsonBody(t, map[string]interface{}{
		"acknowledged_by": "not-a-uuid",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleAcknowledgeWarning(rec, req)
	assertValidationError(t, rec, "acknowledged_by")
}

func TestHandleAcknowledgeWarning_ServiceUnavailable(t *testing.T) {
	routes := NewInventarRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("POST", "/api/v1/inventar/warnings/"+id+"/acknowledge", jsonBody(t, map[string]interface{}{}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleAcknowledgeWarning(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleAcknowledgeWarning_ReachesRPC_FallsBackToAuthenticatedUser covers
// the omitted-acknowledged_by branch (route_inventar.go): the handler falls
// back to middleware.GetUserID(ctx) rather than leaving the field empty.
func TestHandleAcknowledgeWarning_ReachesRPC_FallsBackToAuthenticatedUser(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("POST", "/api/v1/inventar/warnings/"+id+"/acknowledge", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-1", testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleAcknowledgeWarning(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListLocations ---

func TestHandleListLocations_ServiceUnavailable(t *testing.T) {
	routes := NewInventarRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/inventar/locations", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListLocations(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListLocations_MissingTenant(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/inventar/locations", nil)
	routes.HandleListLocations(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListLocations_ReachesRPC(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/inventar/locations?page=2&page_size=10", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListLocations(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleCreateLocation ---

func TestHandleCreateLocation_MissingName(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/inventar/locations", jsonBody(t, map[string]interface{}{
		"address": "Weg 1",
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateLocation(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleCreateLocation_InvalidType(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/inventar/locations", jsonBody(t, map[string]interface{}{
		"name": "Lager A",
		"type": "spaceship",
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateLocation(rec, req)
	assertValidationError(t, rec, "type")
}

func TestHandleCreateLocation_InvalidJSON(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/inventar/locations", invalidJSON())
	req = withTenantID(req, testTenantID)
	routes.HandleCreateLocation(rec, req)
	assertStatus(t, rec, 400)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleCreateLocation_ServiceUnavailable(t *testing.T) {
	routes := NewInventarRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/inventar/locations", jsonBody(t, map[string]interface{}{
		"name": "Lager A",
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateLocation(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateLocation_ReachesRPC(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/inventar/locations", jsonBody(t, map[string]interface{}{
		"name": "Lager A",
		"type": "warehouse",
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateLocation(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetLocation ---

func TestHandleGetLocation_InvalidIDUUID(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/inventar/locations/not-a-uuid", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetLocation(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleGetLocation_ServiceUnavailable(t *testing.T) {
	routes := NewInventarRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("GET", "/api/v1/inventar/locations/"+id, nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleGetLocation(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetLocation_ReachesRPC(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("GET", "/api/v1/inventar/locations/"+id, nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleGetLocation(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpdateLocation ---

func TestHandleUpdateLocation_InvalidIDUUID(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/inventar/locations/not-a-uuid", jsonBody(t, map[string]interface{}{
		"name": "Lager B",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateLocation(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateLocation_InvalidJSON(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("PATCH", "/api/v1/inventar/locations/"+id, invalidJSON())
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateLocation(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateLocation_InvalidType(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("PATCH", "/api/v1/inventar/locations/"+id, jsonBody(t, map[string]interface{}{
		"type": "spaceship",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateLocation(rec, req)
	assertValidationError(t, rec, "type")
}

func TestHandleUpdateLocation_ServiceUnavailable(t *testing.T) {
	routes := NewInventarRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("PATCH", "/api/v1/inventar/locations/"+id, jsonBody(t, map[string]interface{}{
		"name": "Lager B",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateLocation(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateLocation_ReachesRPC(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("PATCH", "/api/v1/inventar/locations/"+id, jsonBody(t, map[string]interface{}{
		"name": "Lager B",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateLocation(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleDeleteLocation ---

func TestHandleDeleteLocation_InvalidIDUUID(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/inventar/locations/not-a-uuid", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteLocation(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleDeleteLocation_ServiceUnavailable(t *testing.T) {
	routes := NewInventarRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("DELETE", "/api/v1/inventar/locations/"+id, nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleDeleteLocation(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteLocation_ReachesRPC(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("DELETE", "/api/v1/inventar/locations/"+id, nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleDeleteLocation(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListPickingLists ---

func TestHandleListPickingLists_ServiceUnavailable(t *testing.T) {
	routes := NewInventarRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/inventar/picking", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListPickingLists(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListPickingLists_MissingTenant(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/inventar/picking", nil)
	routes.HandleListPickingLists(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListPickingLists_InvalidStatusFilter(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/inventar/picking?status=bogus", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListPickingLists(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid status filter")
}

func TestHandleListPickingLists_ReachesRPC(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/inventar/picking?status=open&page=2&page_size=10", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListPickingLists(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleCreatePickingList ---

func TestHandleCreatePickingList_MissingReference(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/inventar/picking", jsonBody(t, map[string]interface{}{}))
	req = withTenantID(req, testTenantID)
	routes.HandleCreatePickingList(rec, req)
	assertValidationError(t, rec, "reference")
}

func TestHandleCreatePickingList_InvalidItemQuantity(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/inventar/picking", jsonBody(t, map[string]interface{}{
		"reference": "PICK-1",
		"items": []map[string]interface{}{
			{"item_id": "550e8400-e29b-41d4-a716-446655440000", "quantity_requested": 0},
		},
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleCreatePickingList(rec, req)
	assertValidationError(t, rec, "quantity_requested")
}

func TestHandleCreatePickingList_InvalidJSON(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/inventar/picking", invalidJSON())
	req = withTenantID(req, testTenantID)
	routes.HandleCreatePickingList(rec, req)
	assertStatus(t, rec, 400)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleCreatePickingList_ServiceUnavailable(t *testing.T) {
	routes := NewInventarRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/inventar/picking", jsonBody(t, map[string]interface{}{
		"reference": "PICK-1",
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleCreatePickingList(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleCreatePickingList_ReachesRPC_WithItemsAndCreatedBy covers the two
// loop/fallback branches route_inventar.go's HandleCreatePickingList carries
// (items -> CreatePickingListItemInput, CreatedBy from the authenticated user)
// that a bare ServiceUnavailable-only call would never execute.
func TestHandleCreatePickingList_ReachesRPC_WithItemsAndCreatedBy(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/inventar/picking", jsonBody(t, map[string]interface{}{
		"reference": "PICK-1",
		"items": []map[string]interface{}{
			{"item_id": "550e8400-e29b-41d4-a716-446655440000", "quantity_requested": 5},
		},
	}))
	req = withAuth(req, "user-1", testTenantID)
	routes.HandleCreatePickingList(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetPickingList ---

func TestHandleGetPickingList_InvalidIDUUID(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/inventar/picking/not-a-uuid", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetPickingList(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleGetPickingList_ServiceUnavailable(t *testing.T) {
	routes := NewInventarRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("GET", "/api/v1/inventar/picking/"+id, nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleGetPickingList(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetPickingList_ReachesRPC(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("GET", "/api/v1/inventar/picking/"+id, nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleGetPickingList(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpdatePickingList ---

func TestHandleUpdatePickingList_InvalidIDUUID(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/inventar/picking/not-a-uuid", jsonBody(t, map[string]interface{}{
		"reference": "PICK-2",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdatePickingList(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdatePickingList_InvalidJSON(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("PATCH", "/api/v1/inventar/picking/"+id, invalidJSON())
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdatePickingList(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

// TestHandleUpdatePickingList_StatusCompletedRejected documents that
// route_inventar.go's updatePickingListRequest validate tag
// (oneof=open picking completed) accepts "completed" at the JSON layer —
// the actual rejection (completion is reached only by booking) happens
// service-side and surfaces as an RPC error, not a local 400. This handler
// test proves the request reaches the RPC layer instead of being rejected
// locally, matching internal/inventar/service_test.go's
// TestUpdatePickingList_CompletedIsBookingOnly at the service layer.
func TestHandleUpdatePickingList_StatusCompletedRejected(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("PATCH", "/api/v1/inventar/picking/"+id, jsonBody(t, map[string]interface{}{
		"status": "completed",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdatePickingList(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdatePickingList_InvalidStatusValue(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("PATCH", "/api/v1/inventar/picking/"+id, jsonBody(t, map[string]interface{}{
		"status": "bogus",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdatePickingList(rec, req)
	assertValidationError(t, rec, "status")
}

func TestHandleUpdatePickingList_ServiceUnavailable(t *testing.T) {
	routes := NewInventarRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("PATCH", "/api/v1/inventar/picking/"+id, jsonBody(t, map[string]interface{}{
		"reference": "PICK-2",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdatePickingList(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdatePickingList_ReachesRPC(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("PATCH", "/api/v1/inventar/picking/"+id, jsonBody(t, map[string]interface{}{
		"reference": "PICK-2",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdatePickingList(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleDeletePickingList ---

func TestHandleDeletePickingList_InvalidIDUUID(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/inventar/picking/not-a-uuid", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeletePickingList(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleDeletePickingList_ServiceUnavailable(t *testing.T) {
	routes := NewInventarRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("DELETE", "/api/v1/inventar/picking/"+id, nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleDeletePickingList(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeletePickingList_ReachesRPC(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("DELETE", "/api/v1/inventar/picking/"+id, nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleDeletePickingList(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpsertPickingListItem ---

func TestHandleUpsertPickingListItem_InvalidIDUUID(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/inventar/picking/not-a-uuid/items", jsonBody(t, map[string]interface{}{
		"item_id":            "550e8400-e29b-41d4-a716-446655440000",
		"quantity_requested": 5,
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpsertPickingListItem(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpsertPickingListItem_InvalidJSON(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("POST", "/api/v1/inventar/picking/"+id+"/items", invalidJSON())
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpsertPickingListItem(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpsertPickingListItem_MissingItemID(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("POST", "/api/v1/inventar/picking/"+id+"/items", jsonBody(t, map[string]interface{}{
		"quantity_requested": 5,
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpsertPickingListItem(rec, req)
	assertValidationError(t, rec, "item_id")
}

func TestHandleUpsertPickingListItem_InvalidQuantityRequested(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("POST", "/api/v1/inventar/picking/"+id+"/items", jsonBody(t, map[string]interface{}{
		"item_id":            "550e8400-e29b-41d4-a716-446655440001",
		"quantity_requested": 0,
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpsertPickingListItem(rec, req)
	assertValidationError(t, rec, "quantity_requested")
}

func TestHandleUpsertPickingListItem_ServiceUnavailable(t *testing.T) {
	routes := NewInventarRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("POST", "/api/v1/inventar/picking/"+id+"/items", jsonBody(t, map[string]interface{}{
		"item_id":            "550e8400-e29b-41d4-a716-446655440001",
		"quantity_requested": 5,
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpsertPickingListItem(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpsertPickingListItem_ReachesRPC(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("POST", "/api/v1/inventar/picking/"+id+"/items", jsonBody(t, map[string]interface{}{
		"item_id":            "550e8400-e29b-41d4-a716-446655440001",
		"quantity_requested": 5,
		"quantity_picked":    5,
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpsertPickingListItem(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleDeletePickingListItem ---

func TestHandleDeletePickingListItem_InvalidIDUUID(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/inventar/picking-items/not-a-uuid", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeletePickingListItem(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleDeletePickingListItem_ServiceUnavailable(t *testing.T) {
	routes := NewInventarRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("DELETE", "/api/v1/inventar/picking-items/"+id, nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleDeletePickingListItem(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeletePickingListItem_ReachesRPC(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("DELETE", "/api/v1/inventar/picking-items/"+id, nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleDeletePickingListItem(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleBookPickingList ---

// TestHandleBookPickingList_InvalidIDUUID through _ReachesRPC cover the HTTP
// layer only. Idempotency (a second booking must not move stock twice) and
// atomicity (a partial failure mid-list must leave no partial booking) are
// proven below the handler, where the real behaviour lives:
// service.BookPickingList delegates the claim to
// PostgresRepository.BookPickingListTx, a single SQL transaction with a
// conditional UPDATE ... WHERE status != 'completed' claim predicate.
//   - idempotency: internal/inventar/picking_service_test.go
//     TestBookPickingList_SecondBookingDoesNotMoveStockAgain (mock repo) and
//     internal/inventar/picking_booking_tx_test.go
//     TestBookPickingListTx_UpsertConflictAndClaimAgainstRealSchema's
//     "claim predicate makes a second booking a no-op" subtest (real Postgres,
//     asserts stock quantity unchanged after a second BookPickingListTx call).
//   - atomicity: internal/inventar/picking_service_test.go
//     TestBookPickingList_InsufficientStockLeavesEverythingUntouched (mock) and
//     internal/inventar/picking_booking_tx_test.go
//     TestBookPickingListTx_PartialFailureRollsBackClaimAndStock (real
//     Postgres: one bookable + one short position, asserts the bookable
//     position's stock is untouched and zero movement rows exist after the
//     batch fails on the second position).
func TestHandleBookPickingList_InvalidIDUUID(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/inventar/picking/not-a-uuid/book", jsonBody(t, map[string]interface{}{}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleBookPickingList(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleBookPickingList_InvalidJSON(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("POST", "/api/v1/inventar/picking/"+id+"/book", invalidJSON())
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleBookPickingList(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleBookPickingList_InvalidBookedByUUID(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("POST", "/api/v1/inventar/picking/"+id+"/book", jsonBody(t, map[string]interface{}{
		"booked_by": "not-a-uuid",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleBookPickingList(rec, req)
	assertValidationError(t, rec, "booked_by")
}

func TestHandleBookPickingList_ServiceUnavailable(t *testing.T) {
	routes := NewInventarRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("POST", "/api/v1/inventar/picking/"+id+"/book", jsonBody(t, map[string]interface{}{}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleBookPickingList(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleBookPickingList_ReachesRPC_FallsBackToAuthenticatedUser covers
// the omitted-booked_by branch (route_inventar.go): the handler falls back to
// middleware.GetUserID(ctx), mirroring HandleAcknowledgeWarning's fallback.
func TestHandleBookPickingList_ReachesRPC_FallsBackToAuthenticatedUser(t *testing.T) {
	routes := NewInventarRoutes(registryWithService("inventar"), nil)
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("POST", "/api/v1/inventar/picking/"+id+"/book", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-1", testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleBookPickingList(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
