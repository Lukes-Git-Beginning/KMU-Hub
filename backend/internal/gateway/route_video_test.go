package gateway

import (
	"net/http"
	"net/http/httptest"
	"strings"
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

// ============================================================================
// Validation tests for decodeAndValidate-migrated handlers
// ============================================================================

// TestHandleCreateCall_MissingParticipants verifies validation rejects empty participant list.
func TestHandleCreateCall_MissingParticipants(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/video/calls",
		strings.NewReader(`{"call_type":"group","participant_ids":[]}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleCreateCall(rec, req)
	assertValidationError(t, rec, "participant_ids")
}

// TestHandleCreateCall_InvalidJSON verifies 400 on malformed JSON.
func TestHandleCreateCall_InvalidJSON(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/video/calls", invalidJSON())
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleCreateCall(rec, req)
	assertErrorContains(t, rec, "invalid request body")
}

// TestHandleCreateMeeting_MissingTitle verifies title is required.
func TestHandleCreateMeeting_MissingTitle(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/meetings",
		strings.NewReader(`{"scheduled_start":"2026-07-01T10:00:00Z","scheduled_end":"2026-07-01T11:00:00Z"}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleCreateMeeting(rec, req)
	assertValidationError(t, rec, "title")
}

// TestHandleCreateMeeting_MissingScheduledStart verifies scheduled_start is required.
func TestHandleCreateMeeting_MissingScheduledStart(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/meetings",
		strings.NewReader(`{"title":"Standup","scheduled_end":"2026-07-01T11:00:00Z"}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleCreateMeeting(rec, req)
	assertValidationError(t, rec, "scheduled_start")
}

// TestHandleSetPresenceStatus_InvalidValue verifies oneof validation for status.
func TestHandleSetPresenceStatus_InvalidValue(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/video/presence/status",
		strings.NewReader(`{"status":"unknown_status"}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleSetPresenceStatus(rec, req)
	assertValidationError(t, rec, "status")
}

// TestHandleSetPresenceStatus_MissingStatus verifies status is required.
func TestHandleSetPresenceStatus_MissingStatus(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/video/presence/status",
		strings.NewReader(`{}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleSetPresenceStatus(rec, req)
	assertValidationError(t, rec, "status")
}

// TestHandleConvertActionItemsToTasks_InvalidProjectID verifies project_id must be a UUID.
func TestHandleConvertActionItemsToTasks_InvalidProjectID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/meetings/"+uuid.New().String()+"/action-items/convert",
		strings.NewReader(`{"action_item_ids":["`+uuid.New().String()+`"],"project_id":"not-a-uuid"}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleConvertActionItemsToTasks(rec, req)
	assertValidationError(t, rec, "project_id")
}

// TestHandleGetBulkPresence_EmptyUserIDs verifies min=1 validation on user_ids.
func TestHandleGetBulkPresence_EmptyUserIDs(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/video/presence/bulk",
		strings.NewReader(`{"user_ids":[]}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleGetBulkPresence(rec, req)
	assertValidationError(t, rec, "user_ids")
}
