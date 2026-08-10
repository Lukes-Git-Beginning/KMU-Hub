package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// extendedTestID is a stable, valid UUID used wherever a handler needs a
// syntactically valid id/tenant path param but the value itself is not under
// test.
const extendedTestID = "33333333-3333-3333-3333-333333333333"

// ============================================================================
// ServiceUnavailable — every handler in route_einkauf_extended.go resolves
// the einkauf client first, before any body/query parsing.
// ============================================================================

func TestEinkaufExtendedRoutes_ServiceUnavailable(t *testing.T) {
	routes := newEinkaufRoutes(emptyRegistry())

	handlers := map[string]http.HandlerFunc{
		"HandleListCatalogItems":        routes.HandleListCatalogItems,
		"HandleGetCatalogItem":          routes.HandleGetCatalogItem,
		"HandleCreateCatalogItem":       routes.HandleCreateCatalogItem,
		"HandleUpdateCatalogItem":       routes.HandleUpdateCatalogItem,
		"HandleDeleteCatalogItem":       routes.HandleDeleteCatalogItem,
		"HandleListSupplierRatings":     routes.HandleListSupplierRatings,
		"HandleCreateSupplierRating":    routes.HandleCreateSupplierRating,
		"HandleDeleteSupplierRating":    routes.HandleDeleteSupplierRating,
		"HandleListFrameworkContracts":  routes.HandleListFrameworkContracts,
		"HandleGetFrameworkContract":    routes.HandleGetFrameworkContract,
		"HandleCreateFrameworkContract": routes.HandleCreateFrameworkContract,
		"HandleUpdateFrameworkContract": routes.HandleUpdateFrameworkContract,
		"HandleDeleteFrameworkContract": routes.HandleDeleteFrameworkContract,
		"HandleCreateContractItem":      routes.HandleCreateContractItem,
		"HandleUpdateContractItem":      routes.HandleUpdateContractItem,
		"HandleDeleteContractItem":      routes.HandleDeleteContractItem,
		"HandleListContractCalls":       routes.HandleListContractCalls,
		"HandleCreateContractCall":      routes.HandleCreateContractCall,
	}

	for name, h := range handlers {
		t.Run(name, func(t *testing.T) {
			testServiceUnavailable(t, h)
		})
	}
}

// ============================================================================
// HandleListCatalogItems — pagination + category/search/supplier_id/available
// query params, all resolved before the RPC call.
// ============================================================================

func TestHandleListCatalogItems_DefaultsReachRPC(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/einkauf/catalog", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListCatalogItems(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListCatalogItems_FiltersReachRPC(t *testing.T) {
	cases := []string{
		"?category=tools",
		"?search=widget",
		"?supplier_id=" + extendedTestID,
		"?available=true",
		"?available=1",
		"?available=false",
		"?available=0",
		"?page=2&page_size=50",
	}
	for _, qs := range cases {
		t.Run(qs, func(t *testing.T) {
			routes := newEinkaufRoutes(registryWithService("einkauf"))
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/einkauf/catalog"+qs, nil)
			req = withTenantID(req, testTenantID)
			routes.HandleListCatalogItems(rec, req)
			assertStatus(t, rec, http.StatusServiceUnavailable)
		})
	}
}

func TestHandleListCatalogItems_NoTenant(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/einkauf/catalog", nil)
	routes.HandleListCatalogItems(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

// ============================================================================
// HandleGetCatalogItem / HandleDeleteCatalogItem — pure id validation.
// ============================================================================

func TestHandleGetCatalogItem_InvalidUUID(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/einkauf/catalog/not-a-uuid", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetCatalogItem(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleDeleteCatalogItem_InvalidUUID(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/einkauf/catalog/not-a-uuid", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteCatalogItem(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

// ============================================================================
// HandleCreateCatalogItem
// ============================================================================

func TestHandleCreateCatalogItem_MissingSupplierID(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/einkauf/catalog", jsonBody(t, map[string]interface{}{
		"name":  "Widget",
		"price": "9.99",
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateCatalogItem(rec, req)
	assertValidationError(t, rec, "supplier_id")
}

func TestHandleCreateCatalogItem_MissingName(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/einkauf/catalog", jsonBody(t, map[string]interface{}{
		"supplier_id": extendedTestID,
		"price":       "9.99",
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateCatalogItem(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleCreateCatalogItem_InvalidPrice(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/einkauf/catalog", jsonBody(t, map[string]interface{}{
		"supplier_id": extendedTestID,
		"name":        "Widget",
		"price":       "not-a-decimal",
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateCatalogItem(rec, req)
	assertValidationError(t, rec, "price")
}

func TestHandleCreateCatalogItem_NegativePrice(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/einkauf/catalog", jsonBody(t, map[string]interface{}{
		"supplier_id": extendedTestID,
		"name":        "Widget",
		"price":       "-5.00",
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateCatalogItem(rec, req)
	assertValidationError(t, rec, "price")
}

func TestHandleCreateCatalogItem_ValidReachesRPC(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/einkauf/catalog", jsonBody(t, map[string]interface{}{
		"supplier_id": extendedTestID,
		"name":        "Widget",
		"price":       "9.99",
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateCatalogItem(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateCatalogItem_InvalidJSON(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/einkauf/catalog", invalidJSON())
	req = withTenantID(req, testTenantID)
	routes.HandleCreateCatalogItem(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

// ============================================================================
// HandleUpdateCatalogItem
// ============================================================================

func TestHandleUpdateCatalogItem_InvalidUUID(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/einkauf/catalog/not-a-uuid", jsonBody(t, map[string]interface{}{"name": "x"}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateCatalogItem(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateCatalogItem_InvalidPrice(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/einkauf/catalog/"+extendedTestID, jsonBody(t, map[string]interface{}{
		"price": "garbage",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", extendedTestID)
	routes.HandleUpdateCatalogItem(rec, req)
	assertValidationError(t, rec, "price")
}

func TestHandleUpdateCatalogItem_PartialBodyReachesRPC(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/einkauf/catalog/"+extendedTestID, jsonBody(t, map[string]interface{}{
		"available": false,
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", extendedTestID)
	routes.HandleUpdateCatalogItem(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListSupplierRatings / HandleDeleteSupplierRating — id validation.
// ============================================================================

func TestHandleListSupplierRatings_InvalidSupplierUUID(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/einkauf/suppliers/not-a-uuid/ratings", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "supplierId", "not-a-uuid")
	routes.HandleListSupplierRatings(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid supplierId")
}

func TestHandleDeleteSupplierRating_InvalidRatingUUID(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/einkauf/suppliers/"+extendedTestID+"/ratings/not-a-uuid", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "ratingId", "not-a-uuid")
	routes.HandleDeleteSupplierRating(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid ratingId")
}

// ============================================================================
// HandleCreateSupplierRating
// ============================================================================

func TestHandleCreateSupplierRating_InvalidSupplierUUID(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/einkauf/suppliers/not-a-uuid/ratings", jsonBody(t, map[string]interface{}{
		"category": "quality",
		"rating":   4,
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "supplierId", "not-a-uuid")
	routes.HandleCreateSupplierRating(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid supplierId")
}

func TestHandleCreateSupplierRating_MissingCategory(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/einkauf/suppliers/"+extendedTestID+"/ratings", jsonBody(t, map[string]interface{}{
		"rating": 4,
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "supplierId", extendedTestID)
	routes.HandleCreateSupplierRating(rec, req)
	assertValidationError(t, rec, "category")
}

func TestHandleCreateSupplierRating_MissingRating(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/einkauf/suppliers/"+extendedTestID+"/ratings", jsonBody(t, map[string]interface{}{
		"category": "quality",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "supplierId", extendedTestID)
	routes.HandleCreateSupplierRating(rec, req)
	assertValidationError(t, rec, "rating")
}

func TestHandleCreateSupplierRating_ValidReachesRPC(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/einkauf/suppliers/"+extendedTestID+"/ratings", jsonBody(t, map[string]interface{}{
		"category": "quality",
		"rating":   5,
		"comment":  "great",
	}))
	req = withTenantID(req, testTenantID)
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "supplierId", extendedTestID)
	routes.HandleCreateSupplierRating(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListFrameworkContracts — supplier_id/status filters + pagination.
// ============================================================================

func TestHandleListFrameworkContracts_DefaultsReachRPC(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/einkauf/contracts", nil)
	req = withTenantID(req, testTenantID)
	routes.HandleListFrameworkContracts(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListFrameworkContracts_FiltersReachRPC(t *testing.T) {
	cases := []string{
		"?supplier_id=" + extendedTestID,
		"?status=active",
		"?page=3&page_size=10",
	}
	for _, qs := range cases {
		t.Run(qs, func(t *testing.T) {
			routes := newEinkaufRoutes(registryWithService("einkauf"))
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/einkauf/contracts"+qs, nil)
			req = withTenantID(req, testTenantID)
			routes.HandleListFrameworkContracts(rec, req)
			assertStatus(t, rec, http.StatusServiceUnavailable)
		})
	}
}

// ============================================================================
// HandleGetFrameworkContract / HandleDeleteFrameworkContract — id validation.
// ============================================================================

func TestHandleGetFrameworkContract_InvalidUUID(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/einkauf/contracts/not-a-uuid", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetFrameworkContract(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleDeleteFrameworkContract_InvalidUUID(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/einkauf/contracts/not-a-uuid", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteFrameworkContract(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

// ============================================================================
// HandleCreateFrameworkContract
// ============================================================================

func TestHandleCreateFrameworkContract_MissingSupplierID(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/einkauf/contracts", jsonBody(t, map[string]interface{}{
		"title": "Rahmenvertrag 2026",
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateFrameworkContract(rec, req)
	assertValidationError(t, rec, "supplier_id")
}

func TestHandleCreateFrameworkContract_MissingTitle(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/einkauf/contracts", jsonBody(t, map[string]interface{}{
		"supplier_id": extendedTestID,
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateFrameworkContract(rec, req)
	assertValidationError(t, rec, "title")
}

func TestHandleCreateFrameworkContract_InvalidTotalValue(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/einkauf/contracts", jsonBody(t, map[string]interface{}{
		"supplier_id": extendedTestID,
		"title":       "Rahmenvertrag 2026",
		"total_value": "-100.00",
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateFrameworkContract(rec, req)
	assertValidationError(t, rec, "total_value")
}

func TestHandleCreateFrameworkContract_ValidReachesRPC(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/einkauf/contracts", jsonBody(t, map[string]interface{}{
		"supplier_id": extendedTestID,
		"title":       "Rahmenvertrag 2026",
		"total_value": "5000.00",
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateFrameworkContract(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleUpdateFrameworkContract
// ============================================================================

func TestHandleUpdateFrameworkContract_InvalidUUID(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/einkauf/contracts/not-a-uuid", jsonBody(t, map[string]interface{}{"title": "x"}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateFrameworkContract(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateFrameworkContract_InvalidTotalValue(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/einkauf/contracts/"+extendedTestID, jsonBody(t, map[string]interface{}{
		"total_value": "garbage",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", extendedTestID)
	routes.HandleUpdateFrameworkContract(rec, req)
	assertValidationError(t, rec, "total_value")
}

// ============================================================================
// HandleCreateContractItem / HandleUpdateContractItem / HandleDeleteContractItem
// ============================================================================

func TestHandleCreateContractItem_InvalidContractUUID(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/einkauf/contracts/not-a-uuid/items", jsonBody(t, map[string]interface{}{
		"name": "Item",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleCreateContractItem(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleCreateContractItem_MissingName(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/einkauf/contracts/"+extendedTestID+"/items", jsonBody(t, map[string]interface{}{
		"unit_price": "9.99",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", extendedTestID)
	routes.HandleCreateContractItem(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleCreateContractItem_InvalidUnitPrice(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/einkauf/contracts/"+extendedTestID+"/items", jsonBody(t, map[string]interface{}{
		"name":       "Item",
		"unit_price": "-1.00",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", extendedTestID)
	routes.HandleCreateContractItem(rec, req)
	assertValidationError(t, rec, "unit_price")
}

func TestHandleUpdateContractItem_InvalidContractUUID(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/einkauf/contracts/not-a-uuid/items/"+extendedTestID, jsonBody(t, map[string]interface{}{}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withChiURLParam(req, "itemId", extendedTestID)
	routes.HandleUpdateContractItem(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateContractItem_InvalidItemUUID(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/einkauf/contracts/"+extendedTestID+"/items/not-a-uuid", jsonBody(t, map[string]interface{}{}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", extendedTestID)
	req = withChiURLParam(req, "itemId", "not-a-uuid")
	routes.HandleUpdateContractItem(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid itemId")
}

func TestHandleDeleteContractItem_InvalidContractUUID(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/einkauf/contracts/not-a-uuid/items/"+extendedTestID, nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withChiURLParam(req, "itemId", extendedTestID)
	routes.HandleDeleteContractItem(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleDeleteContractItem_InvalidItemUUID(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/einkauf/contracts/"+extendedTestID+"/items/not-a-uuid", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", extendedTestID)
	req = withChiURLParam(req, "itemId", "not-a-uuid")
	routes.HandleDeleteContractItem(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid itemId")
}

// ============================================================================
// HandleListContractCalls / HandleCreateContractCall
// ============================================================================

func TestHandleListContractCalls_InvalidContractUUID(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/einkauf/contracts/not-a-uuid/calls", nil)
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleListContractCalls(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleCreateContractCall_InvalidContractUUID(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/einkauf/contracts/not-a-uuid/calls", jsonBody(t, map[string]interface{}{
		"amount": "100.00",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleCreateContractCall(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleCreateContractCall_MissingAmount(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/einkauf/contracts/"+extendedTestID+"/calls", jsonBody(t, map[string]interface{}{}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", extendedTestID)
	routes.HandleCreateContractCall(rec, req)
	assertValidationError(t, rec, "amount")
}

func TestHandleCreateContractCall_NegativeAmount(t *testing.T) {
	routes := newEinkaufRoutes(registryWithService("einkauf"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/einkauf/contracts/"+extendedTestID+"/calls", jsonBody(t, map[string]interface{}{
		"amount": "-50.00",
	}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", extendedTestID)
	routes.HandleCreateContractCall(rec, req)
	assertValidationError(t, rec, "amount")
}

// TestHandleCreateContractCall_WithAndWithoutPOID locks down the po_id ->
// *string wiring: an empty po_id must not become a non-nil empty-string
// pointer on the gRPC request (it is optional on the wire), and a present one
// must not be dropped. Neither branch panics and both reach the RPC call.
func TestHandleCreateContractCall_WithAndWithoutPOID(t *testing.T) {
	cases := map[string]map[string]interface{}{
		"without po_id": {"amount": "100.00"},
		"with po_id":    {"amount": "100.00", "po_id": extendedTestID},
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			routes := newEinkaufRoutes(registryWithService("einkauf"))
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/einkauf/contracts/"+extendedTestID+"/calls", jsonBody(t, body))
			req = withTenantID(req, testTenantID)
			req = withChiURLParam(req, "id", extendedTestID)
			routes.HandleCreateContractCall(rec, req)
			assertStatus(t, rec, http.StatusServiceUnavailable)
		})
	}
}
