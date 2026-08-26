package gateway

import (
	"net/http"
	"net/http/httptest"
	"strings"
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

// --- HandleListFormSchemas ---

func TestHandleListFormSchemas_MissingTenant(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/formulare/schemas", nil)
	routes.HandleListFormSchemas(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListFormSchemas_ServiceUnavailable(t *testing.T) {
	routes := newFormulareRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleListFormSchemas)
}

func TestHandleListFormSchemas_ReachesRPC(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/formulare/schemas", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListFormSchemas(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetFormSchema ---

func TestHandleGetFormSchema_InvalidIDUUID(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/formulare/schemas/not-a-uuid", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetFormSchema(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleGetFormSchema_MissingTenant(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/formulare/schemas/"+testSchemaID, nil)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleGetFormSchema(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetFormSchema_ServiceUnavailable(t *testing.T) {
	routes := newFormulareRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/formulare/schemas/"+testSchemaID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleGetFormSchema(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetFormSchema_ReachesRPC(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/formulare/schemas/"+testSchemaID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleGetFormSchema(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetSubmission ---

func TestHandleGetSubmission_InvalidIDUUID(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/formulare/submissions/not-a-uuid", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetSubmission(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleGetSubmission_MissingTenant(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/formulare/submissions/"+testSchemaID, nil)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleGetSubmission(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetSubmission_ServiceUnavailable(t *testing.T) {
	routes := newFormulareRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/formulare/submissions/"+testSchemaID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleGetSubmission(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetSubmission_ReachesRPC(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/formulare/submissions/"+testSchemaID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleGetSubmission(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListSubmissions ---

func TestHandleListSubmissions_InvalidIDUUID(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/formulare/schemas/not-a-uuid/submissions", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleListSubmissions(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleListSubmissions_MissingTenant(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/formulare/schemas/"+testSchemaID+"/submissions", nil)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleListSubmissions(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListSubmissions_ServiceUnavailable(t *testing.T) {
	routes := newFormulareRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/formulare/schemas/"+testSchemaID+"/submissions", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleListSubmissions(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListSubmissions_ReachesRPC(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/formulare/schemas/"+testSchemaID+"/submissions", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleListSubmissions(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleExportSubmissions ---

func TestHandleExportSubmissions_InvalidIDUUID(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/formulare/schemas/not-a-uuid/submissions/export", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleExportSubmissions(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleExportSubmissions_MissingTenant(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/formulare/schemas/"+testSchemaID+"/submissions/export", nil)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleExportSubmissions(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleExportSubmissions_ServiceUnavailable(t *testing.T) {
	routes := newFormulareRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/formulare/schemas/"+testSchemaID+"/submissions/export", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleExportSubmissions(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleExportSubmissions_ReachesRPC also exercises the ?format=xlsx
// query branch (default is csv) — both must reach the RPC identically, the
// format only changes the fallback content-type/filename applied to a
// successful response.
func TestHandleExportSubmissions_ReachesRPC(t *testing.T) {
	for _, format := range []string{"", "csv", "xlsx"} {
		routes := newFormulareRoutes(registryWithService("formulare"))
		rec := httptest.NewRecorder()
		url := "/api/v1/formulare/schemas/" + testSchemaID + "/submissions/export"
		if format != "" {
			url += "?format=" + format
		}
		req := httptest.NewRequest("GET", url, nil)
		req = withAuth(req, "user-123", testTenantID)
		req = withChiURLParam(req, "id", testSchemaID)
		routes.HandleExportSubmissions(rec, req)
		assertStatus(t, rec, http.StatusServiceUnavailable)
	}
}

// --- HandleGetFormStats ---

func TestHandleGetFormStats_InvalidIDUUID(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/formulare/schemas/not-a-uuid/stats", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetFormStats(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleGetFormStats_MissingTenant(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/formulare/schemas/"+testSchemaID+"/stats", nil)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleGetFormStats(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetFormStats_ServiceUnavailable(t *testing.T) {
	routes := newFormulareRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/formulare/schemas/"+testSchemaID+"/stats", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleGetFormStats(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetFormStats_ReachesRPC(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/formulare/schemas/"+testSchemaID+"/stats", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleGetFormStats(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListWebhooks ---

func TestHandleListWebhooks_InvalidIDUUID(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/formulare/schemas/not-a-uuid/webhooks", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleListWebhooks(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleListWebhooks_MissingTenant(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/formulare/schemas/"+testSchemaID+"/webhooks", nil)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleListWebhooks(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListWebhooks_ServiceUnavailable(t *testing.T) {
	routes := newFormulareRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/formulare/schemas/"+testSchemaID+"/webhooks", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleListWebhooks(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListWebhooks_ReachesRPC(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/formulare/schemas/"+testSchemaID+"/webhooks", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleListWebhooks(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleCreateWebhook ---

func TestHandleCreateWebhook_InvalidIDUUID(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/formulare/schemas/not-a-uuid/webhooks", jsonBody(t, map[string]interface{}{
		"url": "https://example.com/hook",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleCreateWebhook(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleCreateWebhook_MissingTenant(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/formulare/schemas/"+testSchemaID+"/webhooks", jsonBody(t, map[string]interface{}{
		"url": "https://example.com/hook",
	}))
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleCreateWebhook(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleCreateWebhook_InvalidJSON(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/formulare/schemas/"+testSchemaID+"/webhooks", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleCreateWebhook(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleCreateWebhook_MissingURL(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/formulare/schemas/"+testSchemaID+"/webhooks", jsonBody(t, map[string]interface{}{
		"active": true,
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleCreateWebhook(rec, req)
	assertValidationError(t, rec, "url")
}

// TestHandleCreateWebhook_MalformedURL documents the gateway-level guard
// against a syntactically broken webhook target ("url" tag on
// createWebhookRequest) — this is validation, not the SSRF check. The SSRF
// guard (internal/security/safehttp, see WebhookWorker in worker.go) applies
// only at delivery time, when the worker actually dials the URL: the gateway
// has no way to resolve DNS or classify an address as internal at request
// time, so a syntactically valid but private-network URL is accepted here
// and rejected later by safehttp when the worker tries to deliver to it.
func TestHandleCreateWebhook_MalformedURL(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/formulare/schemas/"+testSchemaID+"/webhooks", jsonBody(t, map[string]interface{}{
		"url": "not a url",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleCreateWebhook(rec, req)
	assertValidationError(t, rec, "url")
}

func TestHandleCreateWebhook_ServiceUnavailable(t *testing.T) {
	routes := newFormulareRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/formulare/schemas/"+testSchemaID+"/webhooks", jsonBody(t, map[string]interface{}{
		"url": "https://example.com/hook",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleCreateWebhook(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateWebhook_ReachesRPC(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/formulare/schemas/"+testSchemaID+"/webhooks", jsonBody(t, map[string]interface{}{
		"url": "https://example.com/hook",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleCreateWebhook(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetWebhook ---

func TestHandleGetWebhook_InvalidIDUUID(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/formulare/webhooks/not-a-uuid", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetWebhook(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleGetWebhook_MissingTenant(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/formulare/webhooks/"+testSchemaID, nil)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleGetWebhook(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetWebhook_ServiceUnavailable(t *testing.T) {
	routes := newFormulareRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/formulare/webhooks/"+testSchemaID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleGetWebhook(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetWebhook_ReachesRPC(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/formulare/webhooks/"+testSchemaID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleGetWebhook(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpdateWebhook ---

func TestHandleUpdateWebhook_InvalidIDUUID(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/formulare/webhooks/not-a-uuid", jsonBody(t, map[string]interface{}{
		"active": false,
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateWebhook(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleUpdateWebhook_MissingTenant(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/formulare/webhooks/"+testSchemaID, jsonBody(t, map[string]interface{}{
		"active": false,
	}))
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleUpdateWebhook(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleUpdateWebhook_InvalidJSON(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/formulare/webhooks/"+testSchemaID, invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleUpdateWebhook(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateWebhook_ServiceUnavailable(t *testing.T) {
	routes := newFormulareRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/formulare/webhooks/"+testSchemaID, jsonBody(t, map[string]interface{}{
		"active": false,
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleUpdateWebhook(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateWebhook_ReachesRPC(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/formulare/webhooks/"+testSchemaID, jsonBody(t, map[string]interface{}{
		"active": false,
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleUpdateWebhook(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleDeleteWebhook ---

func TestHandleDeleteWebhook_InvalidIDUUID(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/formulare/webhooks/not-a-uuid", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteWebhook(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleDeleteWebhook_MissingTenant(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/formulare/webhooks/"+testSchemaID, nil)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleDeleteWebhook(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleDeleteWebhook_ServiceUnavailable(t *testing.T) {
	routes := newFormulareRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/formulare/webhooks/"+testSchemaID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleDeleteWebhook(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteWebhook_ReachesRPC(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/formulare/webhooks/"+testSchemaID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleDeleteWebhook(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListWebhookDeliveriesForWebhook ---

func TestHandleListWebhookDeliveriesForWebhook_InvalidIDUUID(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/formulare/webhooks/not-a-uuid/deliveries", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleListWebhookDeliveriesForWebhook(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleListWebhookDeliveriesForWebhook_MissingTenant(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/formulare/webhooks/"+testSchemaID+"/deliveries", nil)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleListWebhookDeliveriesForWebhook(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListWebhookDeliveriesForWebhook_ServiceUnavailable(t *testing.T) {
	routes := newFormulareRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/formulare/webhooks/"+testSchemaID+"/deliveries", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleListWebhookDeliveriesForWebhook(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListWebhookDeliveriesForWebhook_ReachesRPC(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/formulare/webhooks/"+testSchemaID+"/deliveries", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testSchemaID)
	routes.HandleListWebhookDeliveriesForWebhook(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListWebhookDeliveries ---

func TestHandleListWebhookDeliveries_MissingTenant(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/formulare/deliveries", nil)
	routes.HandleListWebhookDeliveries(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListWebhookDeliveries_ServiceUnavailable(t *testing.T) {
	routes := newFormulareRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleListWebhookDeliveries)
}

// TestHandleListWebhookDeliveries_ReachesRPC also exercises the three
// optional filter query params (webhook_id, submission_id, status) together,
// including a non-numeric status value — parsePagination/strconv.Atoi must
// not panic on a bad status and must simply leave the filter unset.
func TestHandleListWebhookDeliveries_ReachesRPC(t *testing.T) {
	cases := []string{
		"/api/v1/formulare/deliveries",
		"/api/v1/formulare/deliveries?webhook_id=" + testSchemaID,
		"/api/v1/formulare/deliveries?submission_id=" + testSchemaID,
		"/api/v1/formulare/deliveries?status=1",
		"/api/v1/formulare/deliveries?status=not-a-number",
	}
	for _, url := range cases {
		routes := newFormulareRoutes(registryWithService("formulare"))
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", url, nil)
		req = withAuth(req, "user-123", testTenantID)
		routes.HandleListWebhookDeliveries(rec, req)
		assertStatus(t, rec, http.StatusServiceUnavailable)
	}
}

// --- HandleSubmitByShareToken ---
//
// The token-verdict logic itself (expired, revoked, quota used up, unknown —
// all collapse to the same ErrShareLinkNotFound) is a service-layer concern
// and is already exhaustively covered by
// TestSubmitByShareToken_EveryTokenVerdictIsTheSameNotFound in
// internal/formulare/form_share_test.go. The gateway is a thin passthrough
// (respondGRPCError maps codes.NotFound to 404) with exactly two pieces of
// its own logic worth a handler-level test: the empty/over-long token guard
// that runs before any RPC call, and the request-body size cap.

func TestHandleSubmitByShareToken_EmptyToken(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/public/formulare/submit/", jsonBody(t, map[string]interface{}{
		"answers": []byte(`{}`),
	}))
	req = withChiURLParam(req, "token", "")
	routes.HandleSubmitByShareToken(rec, req)
	assertStatus(t, rec, http.StatusNotFound)
}

func TestHandleSubmitByShareToken_OverLongToken(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	longToken := strings.Repeat("a", 129)
	req := httptest.NewRequest("POST", "/api/v1/public/formulare/submit/"+longToken, jsonBody(t, map[string]interface{}{
		"answers": []byte(`{}`),
	}))
	req = withChiURLParam(req, "token", longToken)
	routes.HandleSubmitByShareToken(rec, req)
	assertStatus(t, rec, http.StatusNotFound)
}

func TestHandleSubmitByShareToken_ServiceUnavailable(t *testing.T) {
	routes := newFormulareRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/public/formulare/submit/some-token", jsonBody(t, map[string]interface{}{
		"answers": []byte(`{}`),
	}))
	req = withChiURLParam(req, "token", "some-token")
	routes.HandleSubmitByShareToken(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleSubmitByShareToken_InvalidJSON(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/public/formulare/submit/some-token", invalidJSON())
	req = withChiURLParam(req, "token", "some-token")
	routes.HandleSubmitByShareToken(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleSubmitByShareToken_MissingAnswers(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/public/formulare/submit/some-token", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "token", "some-token")
	routes.HandleSubmitByShareToken(rec, req)
	assertValidationError(t, rec, "answers")
}

// TestHandleSubmitByShareToken_OversizedBody proves the MaxBytesReader cap
// (maxPublicSubmitBody, 256 KiB) actually trips before decode: an
// unauthenticated write must not be able to make the gateway buffer an
// arbitrarily large body to find out it is invalid.
func TestHandleSubmitByShareToken_OversizedBody(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	oversized := `{"answers":"` + strings.Repeat("x", maxPublicSubmitBody+1) + `"}`
	req := httptest.NewRequest("POST", "/api/v1/public/formulare/submit/some-token", strings.NewReader(oversized))
	req = withChiURLParam(req, "token", "some-token")
	routes.HandleSubmitByShareToken(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleSubmitByShareToken_ReachesRPC(t *testing.T) {
	routes := newFormulareRoutes(registryWithService("formulare"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/public/formulare/submit/some-token", jsonBody(t, map[string]interface{}{
		"answers": []byte(`{"f1":"x"}`),
	}))
	req = withChiURLParam(req, "token", "some-token")
	routes.HandleSubmitByShareToken(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
