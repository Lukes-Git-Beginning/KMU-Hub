package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kmuhub/kmuhub/internal/featureflag"
)

func newFormulareRoutes(registry *ServiceRegistry) *FormulareRoutes {
	flags := featureflag.NewRegistry().Load(func(key string) string {
		if key == "COSMI_MODULE_FORMULARE_ENABLED" {
			return "true"
		}
		return ""
	})
	return NewFormulareRoutes(registry, flags)
}

const testSchemaID = "550e8400-e29b-41d4-a716-446655440000"

// --- HandleCreateFormSchema ---

func TestHandleCreateFormSchema_ServiceUnavailable(t *testing.T) {
	routes := newFormulareRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleCreateFormSchema)
}

func TestHandleCreateFormSchema_MissingTenant(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/formulare/schemas", jsonBody(t, map[string]interface{}{
		"title": "Kontaktformular",
	}))
	req = withUserID(req, "user-123")
	routes.HandleCreateFormSchema(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleCreateFormSchema_InvalidJSON(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/formulare/schemas", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateFormSchema(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

// TestHandleCreateFormSchema_MissingTitle documents that Title is the only
// gateway-enforced required field (createFormSchemaRequest carries no
// validate tag for Fields — a schema's field definitions are opaque JSON
// to this handler and validated downstream, not here).
func TestHandleCreateFormSchema_MissingTitle(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/formulare/schemas", jsonBody(t, map[string]interface{}{
		"description": "ohne Titel",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateFormSchema(rec, req)
	assertValidationError(t, rec, "title")
}

func TestHandleCreateFormSchema_ReachesRPC(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/formulare/schemas", jsonBody(t, map[string]interface{}{
		"title": "Kontaktformular",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateFormSchema(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpdateFormSchema ---

func TestHandleUpdateFormSchema_InvalidIDUUID(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/formulare/schemas/not-a-uuid", jsonBody(t, map[string]interface{}{
		"title": "Updated",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateFormSchema(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleUpdateFormSchema_MissingTenant(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/formulare/schemas/"+testSchemaID, jsonBody(t, map[string]interface{}{
		"title": "Updated",
	}))
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleUpdateFormSchema(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleUpdateFormSchema_InvalidJSON(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/formulare/schemas/"+testSchemaID, invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleUpdateFormSchema(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateFormSchema_ReachesRPC(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/formulare/schemas/"+testSchemaID, jsonBody(t, map[string]interface{}{
		"title": "Updated",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleUpdateFormSchema(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleDeleteFormSchema ---

func TestHandleDeleteFormSchema_InvalidIDUUID(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/formulare/schemas/not-a-uuid", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteFormSchema(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleDeleteFormSchema_MissingTenant(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/formulare/schemas/"+testSchemaID, nil)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleDeleteFormSchema(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleDeleteFormSchema_ServiceUnavailable(t *testing.T) {
	routes := newFormulareRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/formulare/schemas/"+testSchemaID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleDeleteFormSchema(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleDuplicateFormSchema ---

func TestHandleDuplicateFormSchema_InvalidIDUUID(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/formulare/schemas/not-a-uuid/duplicate", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDuplicateFormSchema(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleDuplicateFormSchema_MissingTenant(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/formulare/schemas/"+testSchemaID+"/duplicate", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleDuplicateFormSchema(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleDuplicateFormSchema_InvalidJSON(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/formulare/schemas/"+testSchemaID+"/duplicate", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleDuplicateFormSchema(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleDuplicateFormSchema_ReachesRPC(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/formulare/schemas/"+testSchemaID+"/duplicate", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleDuplicateFormSchema(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleCreateSubmission ---

func TestHandleCreateSubmission_InvalidIDUUID(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/formulare/schemas/not-a-uuid/submissions", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleCreateSubmission(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleCreateSubmission_MissingTenant(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/formulare/schemas/"+testSchemaID+"/submissions", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleCreateSubmission(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleCreateSubmission_InvalidJSON(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/formulare/schemas/"+testSchemaID+"/submissions", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleCreateSubmission(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleCreateSubmission_ServiceUnavailable(t *testing.T) {
	routes := newFormulareRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/formulare/schemas/"+testSchemaID+"/submissions", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleCreateSubmission(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleCreateSubmission_ReachesRPC documents that CreateSubmission fails
// at the RPC (503 via the unreachable registryWithService dummy address)
// before dispatchIntake is ever attempted -- the intake-record side effect
// has no local state to observe in a handler unit test without a bufconn
// stub for FormulareServiceClient.
func TestHandleCreateSubmission_ReachesRPC(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/formulare/schemas/"+testSchemaID+"/submissions", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleCreateSubmission(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpdateSubmissionStatus ---

func TestHandleUpdateSubmissionStatus_InvalidIDUUID(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/formulare/submissions/not-a-uuid", jsonBody(t, map[string]interface{}{
		"status": "read",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateSubmissionStatus(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleUpdateSubmissionStatus_MissingTenant(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/formulare/submissions/"+testSchemaID, jsonBody(t, map[string]interface{}{
		"status": "read",
	}))
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleUpdateSubmissionStatus(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleUpdateSubmissionStatus_InvalidJSON(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/formulare/submissions/"+testSchemaID, invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleUpdateSubmissionStatus(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateSubmissionStatus_MissingStatus(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/formulare/submissions/"+testSchemaID, jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleUpdateSubmissionStatus(rec, req)
	assertValidationError(t, rec, "status")
}

// TestHandleUpdateSubmissionStatus_InvalidStatusValue covers the oneof
// validation on the status field: a syntactically valid string that is not
// one of "new"/"read"/"archived" must be rejected at the gateway, not
// silently mapped to the FORM_SUBMISSION_STATUS_UNSPECIFIED zero value and
// forwarded to the RPC.
func TestHandleUpdateSubmissionStatus_InvalidStatusValue(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/formulare/submissions/"+testSchemaID, jsonBody(t, map[string]interface{}{
		"status": "closed",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleUpdateSubmissionStatus(rec, req)
	assertValidationError(t, rec, "status")
}

func TestHandleUpdateSubmissionStatus_ReachesRPC(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/formulare/submissions/"+testSchemaID, jsonBody(t, map[string]interface{}{
		"status": "archived",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleUpdateSubmissionStatus(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
