package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// ============================================================================
// HandleListDriverLicenses
// ============================================================================

func TestHandleListDriverLicenses_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/driver-licenses", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListDriverLicenses(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListDriverLicenses_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/driver-licenses", nil)
	req = withUserID(req, "user-123")
	routes.HandleListDriverLicenses(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListDriverLicenses_ReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/driver-licenses?driver_id=550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListDriverLicenses(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleCreateDriverLicense
// ============================================================================

func TestHandleCreateDriverLicense_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/driver-licenses", jsonBody(t, map[string]interface{}{
		"driver_id":           "550e8400-e29b-41d4-a716-446655440000",
		"license_classes":     []string{"B"},
		"next_check_due_date": "2027-01-01",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateDriverLicense(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateDriverLicense_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/driver-licenses", jsonBody(t, map[string]interface{}{
		"driver_id":           "550e8400-e29b-41d4-a716-446655440000",
		"license_classes":     []string{"B"},
		"next_check_due_date": "2027-01-01",
	}))
	req = withUserID(req, "user-123")
	routes.HandleCreateDriverLicense(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleCreateDriverLicense_MissingDriverID(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/driver-licenses", jsonBody(t, map[string]interface{}{
		"license_classes":     []string{"B"},
		"next_check_due_date": "2027-01-01",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateDriverLicense(rec, req)
	assertValidationError(t, rec, "driver_id")
}

func TestHandleCreateDriverLicense_InvalidDriverIDUUID(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/driver-licenses", jsonBody(t, map[string]interface{}{
		"driver_id":           "not-a-uuid",
		"license_classes":     []string{"B"},
		"next_check_due_date": "2027-01-01",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateDriverLicense(rec, req)
	assertValidationError(t, rec, "driver_id")
}

func TestHandleCreateDriverLicense_MissingLicenseClasses(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/driver-licenses", jsonBody(t, map[string]interface{}{
		"driver_id":           "550e8400-e29b-41d4-a716-446655440000",
		"next_check_due_date": "2027-01-01",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateDriverLicense(rec, req)
	assertValidationError(t, rec, "license_classes")
}

func TestHandleCreateDriverLicense_MissingNextCheckDueDate(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/driver-licenses", jsonBody(t, map[string]interface{}{
		"driver_id":       "550e8400-e29b-41d4-a716-446655440000",
		"license_classes": []string{"B"},
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateDriverLicense(rec, req)
	assertValidationError(t, rec, "next_check_due_date")
}

func TestHandleCreateDriverLicense_ReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/driver-licenses", jsonBody(t, map[string]interface{}{
		"driver_id":           "550e8400-e29b-41d4-a716-446655440000",
		"license_classes":     []string{"B", "BE"},
		"next_check_due_date": "2027-01-01",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateDriverLicense(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleUpdateDriverLicense
// ============================================================================

func TestHandleUpdateDriverLicense_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/fuhrpark/driver-licenses/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{
		"license_classes":     []string{"B"},
		"next_check_due_date": "2027-01-01",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateDriverLicense(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateDriverLicense_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/fuhrpark/driver-licenses/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{
		"license_classes":     []string{"B"},
		"next_check_due_date": "2027-01-01",
	}))
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateDriverLicense(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleUpdateDriverLicense_InvalidIDUUID(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/fuhrpark/driver-licenses/not-a-uuid", jsonBody(t, map[string]interface{}{
		"license_classes":     []string{"B"},
		"next_check_due_date": "2027-01-01",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateDriverLicense(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateDriverLicense_MissingLicenseClasses(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/fuhrpark/driver-licenses/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{
		"next_check_due_date": "2027-01-01",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateDriverLicense(rec, req)
	assertValidationError(t, rec, "license_classes")
}

func TestHandleUpdateDriverLicense_MissingNextCheckDueDate(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/fuhrpark/driver-licenses/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{
		"license_classes": []string{"B"},
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateDriverLicense(rec, req)
	assertValidationError(t, rec, "next_check_due_date")
}

func TestHandleUpdateDriverLicense_ReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/fuhrpark/driver-licenses/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{
		"license_classes":     []string{"B", "C"},
		"expiry_date":         "2030-01-01",
		"checked_at":          "2026-08-01",
		"next_check_due_date": "2027-01-01",
		"notes":               "renewed",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateDriverLicense(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleDeleteDriverLicense
//
// NOTE: the service layer (internal/fuhrpark/service.go DeleteDriverLicense)
// deletes any check row by id unconditionally -- there is no guard against
// deleting the most recent (= currently valid) check for a driver_id, even
// though driver_licenses is an append-only audit trail of Fuehrerschein-
// kontrollen and the last row per driver is the Halterhaftung proof (see
// route_fuhrpark.go HandleDeleteDriverLicense doc comment). Filed as
// fix-fuhrpark-delete-driver-license-no-last-check-guard rather than fixed
// here, since a coverage unit must not change behavior.
// ============================================================================

func TestHandleDeleteDriverLicense_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/fuhrpark/driver-licenses/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteDriverLicense(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteDriverLicense_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/fuhrpark/driver-licenses/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteDriverLicense(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleDeleteDriverLicense_InvalidIDUUID(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/fuhrpark/driver-licenses/not-a-uuid", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteDriverLicense(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleDeleteDriverLicense_ReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/fuhrpark/driver-licenses/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteDriverLicense(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListVehicleDocuments
// ============================================================================

func TestHandleListVehicleDocuments_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/documents", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleListVehicleDocuments(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListVehicleDocuments_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/documents", nil)
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleListVehicleDocuments(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListVehicleDocuments_InvalidVehicleIDUUID(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/vehicles/not-a-uuid/documents", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleListVehicleDocuments(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleListVehicleDocuments_ReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/documents", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleListVehicleDocuments(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleCreateVehicleDocument
// ============================================================================

func TestHandleCreateVehicleDocument_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/documents", jsonBody(t, map[string]interface{}{
		"doc_type":   "insurance",
		"name":       "Police 2026",
		"object_key": "docs/police-2026.pdf",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCreateVehicleDocument(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateVehicleDocument_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/documents", jsonBody(t, map[string]interface{}{
		"doc_type":   "insurance",
		"name":       "Police 2026",
		"object_key": "docs/police-2026.pdf",
	}))
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCreateVehicleDocument(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleCreateVehicleDocument_InvalidVehicleIDUUID(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/vehicles/not-a-uuid/documents", jsonBody(t, map[string]interface{}{
		"doc_type":   "insurance",
		"name":       "Police 2026",
		"object_key": "docs/police-2026.pdf",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleCreateVehicleDocument(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleCreateVehicleDocument_MissingDocType(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/documents", jsonBody(t, map[string]interface{}{
		"name":       "Police 2026",
		"object_key": "docs/police-2026.pdf",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCreateVehicleDocument(rec, req)
	assertValidationError(t, rec, "doc_type")
}

func TestHandleCreateVehicleDocument_InvalidDocType(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/documents", jsonBody(t, map[string]interface{}{
		"doc_type":   "warranty", // not in allowed set (registration/insurance/tuev/other)
		"name":       "Police 2026",
		"object_key": "docs/police-2026.pdf",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCreateVehicleDocument(rec, req)
	assertValidationError(t, rec, "doc_type")
}

func TestHandleCreateVehicleDocument_MissingName(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/documents", jsonBody(t, map[string]interface{}{
		"doc_type":   "insurance",
		"object_key": "docs/police-2026.pdf",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCreateVehicleDocument(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleCreateVehicleDocument_MissingObjectKey(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/documents", jsonBody(t, map[string]interface{}{
		"doc_type": "insurance",
		"name":     "Police 2026",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCreateVehicleDocument(rec, req)
	assertValidationError(t, rec, "object_key")
}

func TestHandleCreateVehicleDocument_ReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/documents", jsonBody(t, map[string]interface{}{
		"doc_type":    "tuev",
		"name":        "TUEV Bericht",
		"object_key":  "docs/tuev-2026.pdf",
		"expiry_date": "2028-01-01",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCreateVehicleDocument(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleDeleteVehicleDocument
// ============================================================================

func TestHandleDeleteVehicleDocument_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/fuhrpark/documents/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteVehicleDocument(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteVehicleDocument_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/fuhrpark/documents/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteVehicleDocument(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleDeleteVehicleDocument_InvalidIDUUID(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/fuhrpark/documents/not-a-uuid", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteVehicleDocument(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleDeleteVehicleDocument_ReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/fuhrpark/documents/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteVehicleDocument(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListDamages (top-level /damages -- filterable by status AND vehicle_id
// via query params, used by the fleet-wide damages overview) vs.
// HandleListVehicleDamages (/vehicles/{id}/damages -- scoped to the {id} path
// param only, no status filter, used by the per-vehicle detail view). Both
// call the same fuhrparkv1.FuhrparkServiceClient.ListDamages RPC, just with a
// different request shape -- neither is dead code.
// ============================================================================

func TestHandleListDamages_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/damages", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListDamages(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListDamages_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/damages", nil)
	req = withUserID(req, "user-123")
	routes.HandleListDamages(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListDamages_ReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/damages?status=reported&vehicle_id=550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListDamages(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListVehicleDamages
// ============================================================================

func TestHandleListVehicleDamages_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/damages", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleListVehicleDamages(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListVehicleDamages_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/damages", nil)
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleListVehicleDamages(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListVehicleDamages_InvalidVehicleIDUUID(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/vehicles/not-a-uuid/damages", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleListVehicleDamages(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleListVehicleDamages_ReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fuhrpark/vehicles/550e8400-e29b-41d4-a716-446655440000/damages", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleListVehicleDamages(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleUpdateDamage
//
// NOTE: unlike reportDamageRequest (Severity validate:"omitempty,oneof=minor
// moderate major totalled"), updateDamageRequest carries NO validate tag on
// Severity or Status, and internal/fuhrpark/service.go UpdateDamage writes
// both straight through with no enum check. An update can set severity or
// status to an arbitrary string. Filed as
// fix-fuhrpark-update-damage-missing-enum-validation rather than fixed here,
// since a coverage unit must not change behavior; the "reaches RPC" test
// below demonstrates the gap by using a value outside the create-time oneof
// set and confirming the gateway does not reject it at validation time.
// ============================================================================

func TestHandleUpdateDamage_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/fuhrpark/damages/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{
		"description": "Scratch on door",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateDamage(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateDamage_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/fuhrpark/damages/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{
		"description": "Scratch on door",
	}))
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateDamage(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleUpdateDamage_InvalidIDUUID(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/fuhrpark/damages/not-a-uuid", jsonBody(t, map[string]interface{}{
		"description": "Scratch on door",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateDamage(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateDamage_InvalidJSON(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/fuhrpark/damages/550e8400-e29b-41d4-a716-446655440000", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateDamage(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleUpdateDamage_ArbitrarySeverityReachesRPCUnvalidated(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/fuhrpark/damages/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{
		"severity": "not-a-real-severity",
		"status":   "not-a-real-status",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateDamage(rec, req)
	// No validation tag on updateDamageRequest.Severity/Status (see NOTE
	// above) -- the request reaches the RPC layer instead of being rejected
	// with 400, unlike HandleReportDamage's create-time oneof check.
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateDamage_ReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/fuhrpark/damages/550e8400-e29b-41d4-a716-446655440000", jsonBody(t, map[string]interface{}{
		"description": "Scratch repaired",
		"severity":    "minor",
		"status":      "in_repair",
		"cost_cents":  4200,
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateDamage(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleResolveDamage
// ============================================================================

func TestHandleResolveDamage_ServiceUnavailable(t *testing.T) {
	routes := NewFuhrparkRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/damages/550e8400-e29b-41d4-a716-446655440000/resolve", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleResolveDamage(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleResolveDamage_MissingTenant(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/damages/550e8400-e29b-41d4-a716-446655440000/resolve", jsonBody(t, map[string]interface{}{}))
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleResolveDamage(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleResolveDamage_InvalidIDUUID(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/damages/not-a-uuid/resolve", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleResolveDamage(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleResolveDamage_InvalidJSON(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/damages/550e8400-e29b-41d4-a716-446655440000/resolve", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleResolveDamage(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleResolveDamage_ReachesRPC(t *testing.T) {
	routes := NewFuhrparkRoutes(registryWithService("fuhrpark"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fuhrpark/damages/550e8400-e29b-41d4-a716-446655440000/resolve", jsonBody(t, map[string]interface{}{
		"resolved_by": "550e8400-e29b-41d4-a716-446655440001",
		"cost_cents":  9900,
		"notes":       "repaired at workshop",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleResolveDamage(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
