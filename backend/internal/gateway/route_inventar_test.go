package gateway

import (
	"net/http/httptest"
	"testing"
)

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
