package gateway

import (
	"net/http"
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

// TestPOLineTaxRate_FractionOnly: po_lines.tax_rate is NUMERIC(5,4), a fraction. A percentage
// ("19") used to overflow the column and surface as 500; the boundary now answers 400, while a
// valid fraction passes validation and reaches the RPC (503 with the dummy connection).
func TestPOLineTaxRate_FractionOnly(t *testing.T) {
	cases := []struct {
		rate    string
		wantErr bool
	}{
		{"19", true},
		{"1.0001", true},
		{"-0.19", true},
		{"0.19001", true},
		{"abc", true},
		{"", false},
		{"0", false},
		{"0.19", false},
		{"0.0700", false},
		{"1", false},
	}
	for _, tc := range cases {
		t.Run("add_"+tc.rate, func(t *testing.T) {
			routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/api/v1/einkauf/pos/"+testEinkaufPOID+"/lines",
				jsonBody(t, map[string]interface{}{
					"product_name": "Widget", "quantity": "10", "unit_price": "9.99", "tax_rate": tc.rate,
				}))
			req = withTenantID(req, testTenantID)
			req = withChiURLParam(req, "id", testEinkaufPOID)
			routes.HandleAddPOLine(rec, req)
			if tc.wantErr {
				assertValidationError(t, rec, "tax_rate")
			} else {
				assertStatus(t, rec, http.StatusServiceUnavailable)
			}
		})
		t.Run("update_"+tc.rate, func(t *testing.T) {
			routes := NewEinkaufRoutes(registryWithService("einkauf"), nil)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("PATCH", "/api/v1/einkauf/pos/"+testEinkaufPOID+"/lines/"+testEinkaufLineID,
				jsonBody(t, map[string]interface{}{"tax_rate": tc.rate}))
			req = withTenantID(req, testTenantID)
			req = withChiURLParam(req, "id", testEinkaufPOID)
			req = withChiURLParam(req, "lineId", testEinkaufLineID)
			routes.HandleUpdatePOLine(rec, req)
			// An empty string on the PATCH is a non-nil pointer to "": omitempty does not skip it
			// but rate_fraction rejects it, same as any other clear signal on a numeric column.
			if tc.wantErr || tc.rate == "" {
				assertValidationError(t, rec, "tax_rate")
			} else {
				assertStatus(t, rec, http.StatusServiceUnavailable)
			}
		})
	}
}
