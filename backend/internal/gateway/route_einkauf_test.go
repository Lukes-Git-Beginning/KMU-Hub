package gateway

import (
	"net/http/httptest"
	"testing"

	"github.com/kmuhub/kmuhub/internal/featureflag"
)

func newEinkaufRoutes(registry *ServiceRegistry) *EinkaufRoutes {
	flags := featureflag.NewRegistry().Load(func(key string) string {
		if key == "COSMI_MODULE_EINKAUF_ENABLED" {
			return "true"
		}
		return ""
	})
	return NewEinkaufRoutes(registry, flags)
}

// --- HandleCreateSupplier ---

func TestHandleCreateSupplier_MissingName(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/einkauf/suppliers", jsonBody(t, map[string]interface{}{}))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateSupplier(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleCreateSupplier_InvalidEmail(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/einkauf/suppliers", jsonBody(t, map[string]interface{}{
		"name":  "ACME GmbH",
		"email": "not-an-email",
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateSupplier(rec, req)
	assertValidationError(t, rec, "email")
}

func TestHandleCreateSupplier_InvalidContactIDUUID(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/einkauf/suppliers", jsonBody(t, map[string]interface{}{
		"name":       "ACME GmbH",
		"contact_id": "not-a-uuid",
	}))
	req = withTenantID(req, testTenantID)
	routes.HandleCreateSupplier(rec, req)
	assertValidationError(t, rec, "contact_id")
}

// --- HandleCreatePO ---

func TestHandleCreatePO_MissingSupplierID(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/einkauf/pos", jsonBody(t, map[string]interface{}{
		"po_number": "PO-001",
	}))
	req = withTenantID(req, testTenantID)
	req = withUserID(req, "user-123")
	routes.HandleCreatePO(rec, req)
	assertValidationError(t, rec, "supplier_id")
}

func TestHandleCreatePO_MissingPONumber(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/einkauf/pos", jsonBody(t, map[string]interface{}{
		"supplier_id": "550e8400-e29b-41d4-a716-446655440000",
	}))
	req = withTenantID(req, testTenantID)
	req = withUserID(req, "user-123")
	routes.HandleCreatePO(rec, req)
	assertValidationError(t, rec, "po_number")
}

// --- HandleAddPOLine ---

func TestHandleAddPOLine_MissingProductName(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/einkauf/pos/550e8400-e29b-41d4-a716-446655440000/lines",
		jsonBody(t, map[string]interface{}{
			"quantity":   "10",
			"unit_price": "9.99",
		}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleAddPOLine(rec, req)
	assertValidationError(t, rec, "product_name")
}

func TestHandleAddPOLine_MissingQuantity(t *testing.T) {
	routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/einkauf/pos/550e8400-e29b-41d4-a716-446655440000/lines",
		jsonBody(t, map[string]interface{}{
			"product_name": "Widget",
			"unit_price":   "9.99",
		}))
	req = withTenantID(req, testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleAddPOLine(rec, req)
	assertValidationError(t, rec, "quantity")
}
