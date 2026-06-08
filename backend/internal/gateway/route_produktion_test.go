package gateway

import (
	"net/http/httptest"
	"testing"
)

// --- HandleCreateOrder ---

func TestHandleCreateOrder_MissingOrderNumber(t *testing.T) {
	routes := NewProduktionRoutes(registryWithService("produktion"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/produktion/orders", jsonBody(t, map[string]interface{}{
		"product_name":  "Widget",
		"quantity":      5,
		"planned_start": "2026-06-10T08:00:00Z",
		"planned_end":   "2026-06-10T18:00:00Z",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateOrder(rec, req)
	assertValidationError(t, rec, "order_number")
}

func TestHandleCreateOrder_MissingProductName(t *testing.T) {
	routes := NewProduktionRoutes(registryWithService("produktion"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/produktion/orders", jsonBody(t, map[string]interface{}{
		"order_number":  "ORD-001",
		"quantity":      5,
		"planned_start": "2026-06-10T08:00:00Z",
		"planned_end":   "2026-06-10T18:00:00Z",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateOrder(rec, req)
	assertValidationError(t, rec, "product_name")
}

func TestHandleCreateOrder_MissingPlannedStart(t *testing.T) {
	routes := NewProduktionRoutes(registryWithService("produktion"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/produktion/orders", jsonBody(t, map[string]interface{}{
		"order_number": "ORD-001",
		"product_name": "Widget",
		"quantity":     5,
		"planned_end":  "2026-06-10T18:00:00Z",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateOrder(rec, req)
	assertValidationError(t, rec, "planned_start")
}

// --- HandleCreateMachineBooking ---

func TestHandleCreateMachineBooking_MissingMachineID(t *testing.T) {
	routes := NewProduktionRoutes(registryWithService("produktion"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/produktion/bookings", jsonBody(t, map[string]interface{}{
		"production_order_id": "550e8400-e29b-41d4-a716-446655440000",
		"starts_at":           "2026-06-10T08:00:00Z",
		"ends_at":             "2026-06-10T18:00:00Z",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateMachineBooking(rec, req)
	assertValidationError(t, rec, "machine_id")
}

func TestHandleCreateMachineBooking_InvalidMachineIDUUID(t *testing.T) {
	routes := NewProduktionRoutes(registryWithService("produktion"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/produktion/bookings", jsonBody(t, map[string]interface{}{
		"machine_id":          "not-a-uuid",
		"production_order_id": "550e8400-e29b-41d4-a716-446655440000",
		"starts_at":           "2026-06-10T08:00:00Z",
		"ends_at":             "2026-06-10T18:00:00Z",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateMachineBooking(rec, req)
	assertValidationError(t, rec, "machine_id")
}

// --- HandleCreatePlan ---

func TestHandleCreatePlan_MissingName(t *testing.T) {
	routes := NewProduktionRoutes(registryWithService("produktion"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/produktion/plans", jsonBody(t, map[string]interface{}{
		"week_number":          24,
		"year":                 2026,
		"total_capacity_hours": 40.0,
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreatePlan(rec, req)
	assertValidationError(t, rec, "name")
}
