package gateway

import (
	"net/http/httptest"
	"testing"

	"github.com/kmuhub/kmuhub/internal/featureflag"
)

func newFuhrparkRoutes(registry *ServiceRegistry) *FuhrparkRoutes {
	flags := featureflag.NewRegistry().Load(func(key string) string {
		if key == "COSMI_MODULE_FUHRPARK_ENABLED" {
			return "true"
		}
		return ""
	})
	return NewFuhrparkRoutes(registry, flags)
}

// FuhrparkRoutes.ServiceName() uses "fuhrpark".

func TestFuhrparkRoutes_ServiceName(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	if routes.ServiceName() != "fuhrpark" {
		t.Errorf("ServiceName() = %q, want %q", routes.ServiceName(), "fuhrpark")
	}
}

// --- HandleCreateVehicle ---

func TestHandleCreateVehicle_MissingLicensePlate(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/fuhrpark/vehicles", jsonBody(t, map[string]interface{}{
		"make":  "VW",
		"model": "Transporter",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateVehicle(rec, req)
	assertValidationError(t, rec, "license_plate")
}

func TestHandleCreateVehicle_MissingMake(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/fuhrpark/vehicles", jsonBody(t, map[string]interface{}{
		"license_plate": "B-AB 1234",
		"model":         "Transporter",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateVehicle(rec, req)
	assertValidationError(t, rec, "make")
}

func TestHandleCreateVehicle_MissingModel(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/fuhrpark/vehicles", jsonBody(t, map[string]interface{}{
		"license_plate": "B-AB 1234",
		"make":          "VW",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateVehicle(rec, req)
	assertValidationError(t, rec, "model")
}

func TestHandleCreateVehicle_InvalidFuelType(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/fuhrpark/vehicles", jsonBody(t, map[string]interface{}{
		"license_plate": "B-AB 1234",
		"make":          "VW",
		"model":         "Transporter",
		"fuel_type":     "hydrogen", // not in allowed set
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateVehicle(rec, req)
	assertValidationError(t, rec, "fuel_type")
}

func TestHandleCreateVehicle_InvalidAssignedDriverIDUUID(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/fuhrpark/vehicles", jsonBody(t, map[string]interface{}{
		"license_plate":      "B-AB 1234",
		"make":               "VW",
		"model":              "Transporter",
		"assigned_driver_id": "not-a-uuid",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateVehicle(rec, req)
	assertValidationError(t, rec, "assigned_driver_id")
}

// --- HandleScheduleService ---

func TestHandleScheduleService_MissingServiceType(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/services",
		jsonBody(t, map[string]interface{}{
			"scheduled_at": "2026-07-01",
		}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleScheduleService(rec, req)
	assertValidationError(t, rec, "service_type")
}

func TestHandleScheduleService_MissingScheduledAt(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/services",
		jsonBody(t, map[string]interface{}{
			"service_type": "oil_change",
		}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleScheduleService(rec, req)
	assertValidationError(t, rec, "scheduled_at")
}

// --- HandleReportDamage ---

func TestHandleReportDamage_MissingDescription(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/damages",
		jsonBody(t, map[string]interface{}{
			"severity": "minor",
		}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleReportDamage(rec, req)
	assertValidationError(t, rec, "description")
}

func TestHandleReportDamage_InvalidSeverity(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/damages",
		jsonBody(t, map[string]interface{}{
			"description": "Scratch on door",
			"severity":    "critical", // not in allowed set (minor/moderate/major/totalled)
		}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleReportDamage(rec, req)
	assertValidationError(t, rec, "severity")
}
