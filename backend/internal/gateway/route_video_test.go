package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/server/response"
	authv1 "github.com/kmuhub/kmuhub/proto/auth/v1"
	videov1 "github.com/kmuhub/kmuhub/proto/video/v1"
)

// TestProtoMeeting_TimestampSerialisation verifies R3-P0-1 for Meeting responses:
// google.protobuf.Timestamp fields must serialize as RFC3339 strings (not {seconds,nanos})
// when response.Proto is used. This is the serialisation path used by HandleGetMeeting,
// HandleCreateMeeting, HandleListMeetings, and all other Meeting handlers after the
// route_video.go migration.
//
// A full handler-level test is omitted because it would require a complete
// VideoServiceClient mock; the response_proto_test.go in internal/server/response
// already covers the protojson contract at the marshaller level. This test locks in
// the Meeting-specific field names and types the frontend consumes.
func TestProtoMeeting_TimestampSerialisation(t *testing.T) {
	now := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)
	meeting := &videov1.Meeting{
		Id:             uuid.New().String(),
		Title:          "Standup",
		OrganizerId:    uuid.New().String(),
		Status:         videov1.MeetingStatus_MEETING_STATUS_SCHEDULED,
		ScheduledStart: timestamppb.New(now),
		ScheduledEnd:   timestamppb.New(now.Add(time.Hour)),
		CreatedAt:      timestamppb.New(now),
		UpdatedAt:      timestamppb.New(now),
	}

	rec := httptest.NewRecorder()
	response.Proto(rec, http.StatusOK, meeting)

	body := rec.Body.String()

	// Timestamps must be RFC3339 strings, never {seconds, nanos} objects.
	for _, field := range []string{"scheduled_start", "scheduled_end", "created_at", "updated_at"} {
		want := `"` + field + `":"2026-07-01T09:0` // prefix check is enough
		if field == "scheduled_end" {
			want = `"` + field + `":"2026-07-01T10:0`
		}
		if !strings.Contains(body, want) {
			t.Errorf("field %q: expected RFC3339 timestamp prefix %q, got body: %s", field, want, body)
		}
	}
	if strings.Contains(body, `"seconds"`) {
		t.Errorf("Timestamp leaked {seconds,nanos} shape into body: %s", body)
	}

	// Status must be an integer (UseEnumNumbers=true), not a string.
	if !strings.Contains(body, `"status":1`) {
		t.Errorf("expected integer enum status:1 (MEETING_STATUS_SCHEDULED), got body: %s", body)
	}

	// Must be valid JSON.
	var sink map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &sink); err != nil {
		t.Errorf("Proto output is not valid JSON: %v", err)
	}
}

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

// ============================================================================
// Caller identity resolution for the call.incoming broadcast
// ============================================================================

// TestCallerDisplayName covers the fallback chain: first+last name, either
// alone, and the email fallback when neither name is set. Mirrors the SQL
// COALESCE(NULLIF(CONCAT_WS(...))) pattern used throughout the HR read paths.
func TestCallerDisplayName(t *testing.T) {
	tests := []struct {
		name string
		user *authv1.UserInfo
		want string
	}{
		{
			name: "first and last name set",
			user: &authv1.UserInfo{FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com"},
			want: "Ada Lovelace",
		},
		{
			name: "only first name set",
			user: &authv1.UserInfo{FirstName: "Ada", Email: "ada@example.com"},
			want: "Ada",
		},
		{
			name: "only last name set",
			user: &authv1.UserInfo{LastName: "Lovelace", Email: "ada@example.com"},
			want: "Lovelace",
		},
		{
			name: "no name falls back to email",
			user: &authv1.UserInfo{Email: "ada@example.com"},
			want: "ada@example.com",
		},
		{
			name: "blank name fields fall back to email",
			user: &authv1.UserInfo{FirstName: "  ", LastName: "", Email: "ada@example.com"},
			want: "ada@example.com",
		},
		{
			name: "nil user yields empty string",
			user: nil,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := callerDisplayName(tt.user); got != tt.want {
				t.Errorf("callerDisplayName() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestResolveCallerIdentity_AuthClientUnavailable_ReturnsEmpty covers the case
// where the auth service is not registered at all (dev without the service
// wired up). The call must not panic or propagate an error to the caller.
func TestResolveCallerIdentity_AuthClientUnavailable_ReturnsEmpty(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	name, avatar := routes.resolveCallerIdentity(context.Background(), uuid.New().String())
	if name != "" || avatar != "" {
		t.Errorf("resolveCallerIdentity() = (%q, %q), want (\"\", \"\") when auth client is unavailable", name, avatar)
	}
}

// TestResolveCallerIdentity_LookupFailure_ReturnsEmptyWithoutError belongs to
// a-video-caller-identity's third done_when criterion: a failed lookup (here,
// the RPC itself fails because "auth" resolves to a dummy unreachable address)
// must not abort the call -- the broadcast proceeds without a name.
func TestResolveCallerIdentity_LookupFailure_ReturnsEmptyWithoutError(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("auth"), "", "")
	name, avatar := routes.resolveCallerIdentity(context.Background(), uuid.New().String())
	if name != "" || avatar != "" {
		t.Errorf("resolveCallerIdentity() = (%q, %q), want (\"\", \"\") on RPC failure", name, avatar)
	}
}
