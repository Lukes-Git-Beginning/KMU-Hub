package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

// TestConfirmInitiatorConsent_ServiceUnavailable verifies 503 when the work service is not registered.
func TestConfirmInitiatorConsent_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/video/recordings/"+uuid.New().String()+"/initiator-consent", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", uuid.New().String())
	routes.HandleConfirmInitiatorConsent(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestConfirmInitiatorConsent_InvalidUUID verifies 400 when the recording ID is not a valid UUID.
func TestConfirmInitiatorConsent_InvalidUUID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/video/recordings/not-a-uuid/initiator-consent", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleConfirmInitiatorConsent(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

// TestConfirmInitiatorConsent_MissingTenantID verifies 401 when no tenant context is set.
func TestConfirmInitiatorConsent_MissingTenantID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/video/recordings/"+uuid.New().String()+"/initiator-consent", nil)
	req = withUserID(req, uuid.New().String()) // no tenant ID
	req = withChiURLParam(req, "id", uuid.New().String())
	routes.HandleConfirmInitiatorConsent(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}
