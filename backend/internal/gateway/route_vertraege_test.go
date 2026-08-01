package gateway

import (
	"net/http/httptest"
	"testing"

	"github.com/kmuhub/kmuhub/internal/featureflag"
)

func newVertraegeRoutes(registry *ServiceRegistry) *VertraegeRoutes {
	flags := featureflag.NewRegistry().Load(func(key string) string {
		if key == "COSMI_MODULE_VERTRAEGE_ENABLED" {
			return "true"
		}
		return ""
	})
	return NewVertraegeRoutes(registry, flags)
}

// --- HandleCreateContract ---

func TestHandleCreateContract_MissingContractNumber(t *testing.T) {
	routes := NewVertraegeRoutes(registryWithService("vertraege"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/vertraege/contracts", jsonBody(t, map[string]interface{}{
		"title":         "Service Agreement",
		"contract_type": "service",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateContract(rec, req)
	assertValidationError(t, rec, "contract_number")
}

func TestHandleCreateContract_MissingTitle(t *testing.T) {
	routes := NewVertraegeRoutes(registryWithService("vertraege"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/vertraege/contracts", jsonBody(t, map[string]interface{}{
		"contract_number": "CTR-001",
		"contract_type":   "service",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateContract(rec, req)
	assertValidationError(t, rec, "title")
}

func TestHandleCreateContract_MissingContractType(t *testing.T) {
	routes := NewVertraegeRoutes(registryWithService("vertraege"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/vertraege/contracts", jsonBody(t, map[string]interface{}{
		"contract_number": "CTR-001",
		"title":           "Service Agreement",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateContract(rec, req)
	assertValidationError(t, rec, "contract_type")
}

// --- HandleAddParty ---

func TestHandleAddParty_MissingPartyType(t *testing.T) {
	routes := NewVertraegeRoutes(registryWithService("vertraege"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/vertraege/contracts/550e8400-e29b-41d4-a716-446655440000/parties",
		jsonBody(t, map[string]interface{}{
			"role_in_contract": "signatory",
		}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleAddParty(rec, req)
	assertValidationError(t, rec, "party_type")
}

func TestHandleAddParty_MissingRoleInContract(t *testing.T) {
	routes := NewVertraegeRoutes(registryWithService("vertraege"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/vertraege/contracts/550e8400-e29b-41d4-a716-446655440000/parties",
		jsonBody(t, map[string]interface{}{
			"party_type": "internal",
		}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleAddParty(rec, req)
	assertValidationError(t, rec, "role_in_contract")
}

// --- HandleCreateReminder ---

func TestHandleCreateReminder_MissingSubject(t *testing.T) {
	routes := NewVertraegeRoutes(registryWithService("vertraege"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/vertraege/contracts/550e8400-e29b-41d4-a716-446655440000/reminders",
		jsonBody(t, map[string]interface{}{
			"remind_at":     "2026-07-01T09:00:00Z",
			"reminder_type": "expiry",
		}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCreateReminder(rec, req)
	assertValidationError(t, rec, "subject")
}
