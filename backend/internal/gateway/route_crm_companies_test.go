package gateway

// route_crm_companies_test.go covers route_crm_companies.go (unit
// cov-gateway-crm-companies), which had no DEDICATED test file -- a handful
// of its handlers (create's ServiceUnavailable/NoUserID/InvalidJSON/
// MissingName, get's InvalidUUID) are already covered ad hoc in
// route_crm_test.go alongside contacts/custom-fields/tags and are not
// duplicated here. Like advisory protocols (route_crm_advisory_test.go),
// every handler is a thin pass-through to the CRM gRPC client with no tenant
// check of its own -- the tenant is threaded through context and enforced
// server-side. This file fills the rest: list, update, delete, get-contacts,
// and the invalid-tag-id and reaches-RPC branches create/get were missing.
//
// The two behavioural done_when items are proven elsewhere, deliberately not
// re-pinned here as a gateway-level fake:
//   - Delete on a company with hanging contacts returns 409, not 500:
//     internal/crm/company/postgres_repository_db_test.go
//     TestService_Delete_HangingContactsBlockDeletionButForeignTenantContactDoesNot
//     proves this against the real repository (companies.company_id on
//     contacts is ON DELETE SET NULL -- migration 000007 -- so nothing
//     RESTRICTs at the FK level; the block is Service.Delete's own
//     HasContacts check, company.ErrCompanyInUse); internal/server/crm_grpc.go
//     maps it to codes.FailedPrecondition, which internal/gateway/helpers.go
//     maps to HTTP 409. The only RESTRICT-shaped FK on companies is the
//     self-referential merged_into_id (migration 000059, no ON DELETE clause
//     = default NO ACTION) and it is unreachable from this route: merges go
//     through a dedicated merge endpoint (route_crm_ext.go), never through
//     HandleDeleteCompany.
//   - Cross-tenant read returns 404, not a foreign company's data:
//     PostgresRepository.GetByID (postgres_repository.go:39) scopes every
//     read with "WHERE id = $1 AND tenant_id = $2", and internal/crm/company/
//     rls_test.go proves the same isolation at the RLS-policy level directly
//     against the real database.

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const companyTestID = "44444444-4444-4444-4444-444444444444"

// ============================================================================
// ServiceUnavailable — every handler resolves the CRM client first.
// ============================================================================

func TestCRMCompanyRoutes_ServiceUnavailable(t *testing.T) {
	routes := NewCRMRoutes(emptyRegistry(), nil)

	handlers := map[string]http.HandlerFunc{
		"HandleCreateCompany":      routes.HandleCreateCompany,
		"HandleGetCompany":         routes.HandleGetCompany,
		"HandleListCompanies":      routes.HandleListCompanies,
		"HandleUpdateCompany":      routes.HandleUpdateCompany,
		"HandleDeleteCompany":      routes.HandleDeleteCompany,
		"HandleGetCompanyContacts": routes.HandleGetCompanyContacts,
	}

	for name, h := range handlers {
		t.Run(name, func(t *testing.T) {
			testServiceUnavailable(t, h)
		})
	}
}

// ============================================================================
// HandleCreateCompany — InvalidJSON, MissingName and ServiceUnavailable/
// NoUserID already covered in route_crm_test.go.
// ============================================================================

func TestHandleCreateCompany_InvalidTagID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/companies", jsonBody(t, map[string]interface{}{
		"name":    "Acme GmbH",
		"tag_ids": []string{"not-a-uuid"},
	}))
	routes.HandleCreateCompany(rec, req)
	assertValidationError(t, rec, "tag_ids[0]")
}

func TestHandleCreateCompany_ReachesRPC(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/companies", jsonBody(t, map[string]interface{}{
		"name": "Acme GmbH",
	}))
	routes.HandleCreateCompany(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleGetCompany — InvalidUUID already covered in route_crm_test.go.
// ============================================================================

func TestHandleGetCompany_ReachesRPC(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/"+companyTestID, nil)
	req = withChiURLParam(req, "id", companyTestID)
	routes.HandleGetCompany(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListCompanies — no required params, every filter is optional.
// ============================================================================

func TestHandleListCompanies_ReachesRPC(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies?industry=tech&search=acme&sort_by=name&sort_desc=true&tag_ids=aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa&page=2&page_size=10", nil)
	routes.HandleListCompanies(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListCompanies_DefaultsWithoutQuery(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies", nil)
	routes.HandleListCompanies(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleUpdateCompany — the UUID check runs before decode/validate.
// ============================================================================

func TestHandleUpdateCompany_InvalidID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/companies/bad", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateCompany(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateCompany_InvalidJSON(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/companies/"+companyTestID, invalidJSON())
	req = withChiURLParam(req, "id", companyTestID)
	routes.HandleUpdateCompany(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateCompany_BlankNameRejected(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/companies/"+companyTestID, jsonBody(t, map[string]interface{}{
		"name": "",
	}))
	req = withChiURLParam(req, "id", companyTestID)
	routes.HandleUpdateCompany(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleUpdateCompany_ReachesRPC(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/companies/"+companyTestID, jsonBody(t, map[string]interface{}{
		"industry": "tech",
	}))
	req = withChiURLParam(req, "id", companyTestID)
	routes.HandleUpdateCompany(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleDeleteCompany
// ============================================================================

func TestHandleDeleteCompany_InvalidID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/companies/bad", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteCompany(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleDeleteCompany_ReachesRPC(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/companies/"+companyTestID, nil)
	req = withChiURLParam(req, "id", companyTestID)
	routes.HandleDeleteCompany(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleGetCompanyContacts
// ============================================================================

func TestHandleGetCompanyContacts_InvalidID(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/bad/contacts", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetCompanyContacts(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleGetCompanyContacts_ReachesRPC(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/"+companyTestID+"/contacts?page=2&page_size=5", nil)
	req = withChiURLParam(req, "id", companyTestID)
	routes.HandleGetCompanyContacts(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
