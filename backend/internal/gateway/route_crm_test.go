package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCRMRoutes_ServiceName(t *testing.T) {
	routes := NewCRMRoutes(emptyRegistry(), nil)
	if routes.ServiceName() != "crm" {
		t.Errorf("ServiceName() = %q, want %q", routes.ServiceName(), "crm")
	}
}

// --- Contacts ---

func TestHandleCreateContact_ServiceUnavailable(t *testing.T) {
	routes := NewCRMRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleCreateContact)
}

func TestHandleCreateContact_NoUserID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/contacts", jsonBody(t, map[string]interface{}{}))
	withAuthRequired(routes.HandleCreateContact)(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
	assertErrorContains(t, rec, "not authenticated")
}

func TestHandleCreateContact_InvalidJSON(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/contacts", invalidJSON())
	req = withUserID(req, "user-123")
	routes.HandleCreateContact(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleCreateContact_MissingFields(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)

	t.Run("empty", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/v1/contacts", jsonBody(t, map[string]interface{}{}))
		req = withUserID(req, "user-123")
		routes.HandleCreateContact(rec, req)
		assertStatus(t, rec, http.StatusBadRequest)
		assertValidationError(t, rec, "first_name")
	})

	t.Run("no_last_name", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/v1/contacts", jsonBody(t, map[string]interface{}{"first_name": "Max"}))
		req = withUserID(req, "user-123")
		routes.HandleCreateContact(rec, req)
		assertStatus(t, rec, http.StatusBadRequest)
		assertValidationError(t, rec, "last_name")
	})

	t.Run("no_first_name", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/v1/contacts", jsonBody(t, map[string]interface{}{"last_name": "Muster"}))
		req = withUserID(req, "user-123")
		routes.HandleCreateContact(rec, req)
		assertStatus(t, rec, http.StatusBadRequest)
		assertValidationError(t, rec, "first_name")
	})
}

func TestHandleCreateContact_InvalidEmail(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/contacts", jsonBody(t, map[string]interface{}{
		"first_name": "Max",
		"last_name":  "Muster",
		"email":      "not-an-email",
	}))
	req = withUserID(req, "user-123")
	routes.HandleCreateContact(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "email")
}

func TestHandleCreateContact_InvalidCompanyID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/contacts", jsonBody(t, map[string]interface{}{
		"first_name": "Max",
		"last_name":  "Muster",
		"company_id": "not-a-uuid",
	}))
	req = withUserID(req, "user-123")
	routes.HandleCreateContact(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "company_id")
}

func TestHandleGetContact_ServiceUnavailable(t *testing.T) {
	routes := NewCRMRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/contacts/123", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetContact(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetContact_InvalidUUID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/contacts/bad", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetContact(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleListContacts_ServiceUnavailable(t *testing.T) {
	routes := NewCRMRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/contacts", nil)
	routes.HandleListContacts(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteContact_InvalidUUID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/contacts/bad", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteContact(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

// --- Companies ---

func TestHandleCreateCompany_ServiceUnavailable(t *testing.T) {
	routes := NewCRMRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleCreateCompany)
}

func TestHandleCreateCompany_NoUserID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/companies", jsonBody(t, map[string]interface{}{}))
	withAuthRequired(routes.HandleCreateCompany)(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleCreateCompany_InvalidJSON(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/companies", invalidJSON())
	req = withUserID(req, "user-123")
	routes.HandleCreateCompany(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleCreateCompany_MissingName(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/companies", jsonBody(t, map[string]interface{}{
		"domain": "example.com",
	}))
	req = withUserID(req, "user-123")
	routes.HandleCreateCompany(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "name")
}

func TestHandleGetCompany_InvalidUUID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/companies/bad", nil)
	req = withChiURLParam(req, "id", "invalid")
	routes.HandleGetCompany(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

// --- Custom Fields ---

func TestHandleCreateCustomField_ServiceUnavailable(t *testing.T) {
	routes := NewCRMRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleCreateCustomField)
}

func TestHandleCreateCustomField_NoUserID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/custom-fields", jsonBody(t, map[string]interface{}{}))
	withAuthRequired(routes.HandleCreateCustomField)(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleCreateCustomField_MissingFields(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/custom-fields", jsonBody(t, map[string]interface{}{
		"entity_type": "contact",
	}))
	req = withUserID(req, "user-123")
	routes.HandleCreateCustomField(rec, req)
	assertValidationError(t, rec, "field_name")
}

func TestHandleGetCustomField_InvalidUUID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/custom-fields/bad", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetCustomField(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

// --- Tags ---

func TestHandleCreateTag_ServiceUnavailable(t *testing.T) {
	routes := NewCRMRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleCreateTag)
}

func TestHandleCreateTag_InvalidJSON(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tags", invalidJSON())
	routes.HandleCreateTag(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleCreateTag_MissingFields(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tags", jsonBody(t, map[string]interface{}{
		"name": "test",
	}))
	routes.HandleCreateTag(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertValidationError(t, rec, "entity_type")
}

func TestHandleGetTag_InvalidUUID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/tags/bad", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetTag(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}
