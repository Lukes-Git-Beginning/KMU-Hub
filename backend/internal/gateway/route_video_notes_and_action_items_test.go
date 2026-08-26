package gateway

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// route_video_notes_and_action_items_test.go covers the meeting-notes, action-item,
// and meeting-chat handlers of route_video.go — the second of three units splitting
// the file (see route_video_test.go for the first, route_video_breakout_cohost_test.go
// for the second-built).
//
// None of these handlers extract tenant/user IDs via an explicit gateway check beyond
// middleware.GetUserID (used to populate author_id/user_id on the outbound request) —
// authorization (host/co-host for HandleGenerateMeetingSummary, tenant scoping for
// everything else) is enforced entirely in internal/work/meeting/service.go and the
// Postgres repository (RLS), already covered there. A gateway test can only observe
// that a gRPC error is mapped to the right HTTP status, never body content, since no
// fake VideoServiceClient exists in this package.
//
// Scope finding (HandleGetPreviousMeetingNotes): the service method
// (internal/work/meeting/service.go GetPreviousMeetingNotes) takes no caller/user ID
// at all — it only checks that the meeting belongs to the caller's tenant
// (repo.GetMeeting(meetingID, tenantID)) and that it is a recurring meeting, then
// returns the most recent PUBLIC note (is_private=false) of the previous completed
// occurrence (postgres_repository.go GetPreviousMeetingNotes, filtered
// "AND mn.is_private = false"). There is no participant/attendee check on either the
// current or the previous meeting. This is NOT an isolated gap: the underlying
// Service.GetMeeting (service.go:186) has no attendee check either — meetings and
// their public notes are visible tenant-wide by design in this module, the same way
// meeting metadata already is; only PRIVATE notes are author-scoped (repo GetNotes
// filters by author_id). Verified by reading the source; no fix needed, no
// inconsistency found relative to the rest of the module's authorization model.
//
// Confirmed production bug found while covering this file (NOT fixed here — out of
// gateway scope, see notes on cov-gateway-video-notes-and-action-items in
// BACKLOG.yml): HandleGetMeetingNotes's gRPC implementation
// (internal/server/video_grpc.go GetMeetingNotes) does not actually read notes. It
// calls meetingService.SaveNotes(ctx, meetingID, userID, tenantID, "", false) — a
// WRITE with empty content — to simulate a read. SaveNotes trims and rejects empty
// content unconditionally (service.go:678-681, ErrNotesContentRequired) BEFORE ever
// touching the repository, so the call always fails, and the handler's error branch
// then fabricates an empty stub response ({meeting_id, author_id} only, no content
// field). HandleGetMeetingNotes can therefore never return a user's real notes —
// logged as a new backlog unit (see neue-units in the iteration 23 journal entry).

// ============================================================================
// Meeting Notes
// ============================================================================

func TestHandleGetMeetingNotes_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/meetings/"+meetingID+"/notes", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", meetingID)
	routes.HandleGetMeetingNotes(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetMeetingNotes_InvalidMeetingID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/meetings/not-a-uuid/notes", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetMeetingNotes(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleGetMeetingNotes_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/meetings/"+meetingID+"/notes", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", meetingID)
	routes.HandleGetMeetingNotes(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleSaveMeetingNotes_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/meetings/"+meetingID+"/notes",
		strings.NewReader(`{"content":"agenda notes"}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", meetingID)
	routes.HandleSaveMeetingNotes(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleSaveMeetingNotes_InvalidMeetingID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/meetings/not-a-uuid/notes",
		strings.NewReader(`{"content":"agenda notes"}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleSaveMeetingNotes(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleSaveMeetingNotes_MissingContent(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/meetings/"+meetingID+"/notes",
		strings.NewReader(`{}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", meetingID)
	routes.HandleSaveMeetingNotes(rec, req)
	assertValidationError(t, rec, "content")
}

func TestHandleSaveMeetingNotes_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/meetings/"+meetingID+"/notes",
		strings.NewReader(`{"content":"agenda notes","is_private":true}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", meetingID)
	routes.HandleSaveMeetingNotes(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// HandleGetPreviousMeetingNotes: no fake client exists, so the gateway test can only
// observe transport-level behavior. The scope question (does the caller need to have
// been a participant of the previous meeting?) is answered in the file-level comment
// above by reading the service and repository source directly — there is no
// participant check, by design, consistent with the rest of the module.
func TestHandleGetPreviousMeetingNotes_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/meetings/"+meetingID+"/previous-notes", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", meetingID)
	routes.HandleGetPreviousMeetingNotes(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetPreviousMeetingNotes_InvalidMeetingID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/meetings/not-a-uuid/previous-notes", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetPreviousMeetingNotes(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleGetPreviousMeetingNotes_ReachesRPC_ForeignMeeting(t *testing.T) {
	// The "foreign" meeting here is any meeting ID the caller did not attend — the
	// gateway forwards it unconditionally (no participant check possible to observe
	// without a fake client); the service enforces only tenant scope (RLS) and
	// recurrence, as documented above.
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/meetings/"+meetingID+"/previous-notes", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", meetingID)
	routes.HandleGetPreviousMeetingNotes(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// Meeting AI Summary
// ============================================================================

func TestHandleGenerateMeetingSummary_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+meetingID+"/ai-summary", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", meetingID)
	routes.HandleGenerateMeetingSummary(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGenerateMeetingSummary_InvalidMeetingID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/not-a-uuid/ai-summary", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGenerateMeetingSummary(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleGenerateMeetingSummary_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+meetingID+"/ai-summary", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", meetingID)
	routes.HandleGenerateMeetingSummary(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// Action Items
// ============================================================================

func TestHandleCreateActionItem_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+meetingID+"/action-items",
		strings.NewReader(`{"description":"follow up with client"}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", meetingID)
	routes.HandleCreateActionItem(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateActionItem_InvalidMeetingID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/not-a-uuid/action-items",
		strings.NewReader(`{"description":"follow up with client"}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleCreateActionItem(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleCreateActionItem_MissingDescription(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+meetingID+"/action-items",
		strings.NewReader(`{}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", meetingID)
	routes.HandleCreateActionItem(rec, req)
	assertValidationError(t, rec, "description")
}

func TestHandleCreateActionItem_InvalidAssigneeID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+meetingID+"/action-items",
		strings.NewReader(`{"description":"follow up","assignee_id":"not-a-uuid"}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", meetingID)
	routes.HandleCreateActionItem(rec, req)
	assertValidationError(t, rec, "assignee_id")
}

func TestHandleCreateActionItem_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+meetingID+"/action-items",
		strings.NewReader(`{"description":"follow up with client","assignee_id":"`+uuid.New().String()+`","sort_order":2}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", meetingID)
	routes.HandleCreateActionItem(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateActionItem_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	itemID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/meetings/action-items/"+itemID,
		strings.NewReader(`{"is_completed":true}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "itemId", itemID)
	routes.HandleUpdateActionItem(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateActionItem_InvalidItemID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/meetings/action-items/not-a-uuid",
		strings.NewReader(`{"is_completed":true}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "itemId", "not-a-uuid")
	routes.HandleUpdateActionItem(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

// HandleUpdateActionItem has no required fields (a partial update is valid): an
// empty body must still reach the RPC, not fail validation.
func TestHandleUpdateActionItem_ReachesRPC_EmptyBody(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	itemID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/meetings/action-items/"+itemID,
		strings.NewReader(`{}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "itemId", itemID)
	routes.HandleUpdateActionItem(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateActionItem_ReachesRPC_FullBody(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	itemID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/meetings/action-items/"+itemID,
		strings.NewReader(`{"description":"revised text","assignee_id":"`+uuid.New().String()+`","is_completed":true,"sort_order":5}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "itemId", itemID)
	routes.HandleUpdateActionItem(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteActionItem_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	itemID := uuid.New().String()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/meetings/action-items/"+itemID, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "itemId", itemID)
	routes.HandleDeleteActionItem(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteActionItem_InvalidItemID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/meetings/action-items/not-a-uuid", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "itemId", "not-a-uuid")
	routes.HandleDeleteActionItem(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleDeleteActionItem_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	itemID := uuid.New().String()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/meetings/action-items/"+itemID, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "itemId", itemID)
	routes.HandleDeleteActionItem(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListActionItems_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/meetings/"+meetingID+"/action-items", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", meetingID)
	routes.HandleListActionItems(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListActionItems_InvalidMeetingID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/meetings/not-a-uuid/action-items", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleListActionItems(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleListActionItems_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/meetings/"+meetingID+"/action-items", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", meetingID)
	routes.HandleListActionItems(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// Meeting Chat
// ============================================================================

func TestHandleSaveMeetingChatMessage_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+meetingID+"/chat",
		strings.NewReader(`{"sender_name":"Alice","message":"hi all"}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "meetingId", meetingID)
	routes.HandleSaveMeetingChatMessage(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleSaveMeetingChatMessage_InvalidMeetingID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/not-a-uuid/chat",
		strings.NewReader(`{"sender_name":"Alice","message":"hi all"}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "meetingId", "not-a-uuid")
	routes.HandleSaveMeetingChatMessage(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleSaveMeetingChatMessage_MissingMessage(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+meetingID+"/chat",
		strings.NewReader(`{"sender_name":"Alice"}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "meetingId", meetingID)
	routes.HandleSaveMeetingChatMessage(rec, req)
	assertValidationError(t, rec, "message")
}

func TestHandleSaveMeetingChatMessage_ReachesRPC(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/meetings/"+meetingID+"/chat",
		strings.NewReader(`{"sender_name":"Alice","message":"hi all"}`))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "meetingId", meetingID)
	routes.HandleSaveMeetingChatMessage(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListMeetingChatMessages_ServiceUnavailable(t *testing.T) {
	routes := NewVideoRoutes(emptyRegistry(), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/meetings/"+meetingID+"/chat", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "meetingId", meetingID)
	routes.HandleListMeetingChatMessages(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListMeetingChatMessages_InvalidMeetingID(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/meetings/not-a-uuid/chat", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "meetingId", "not-a-uuid")
	routes.HandleListMeetingChatMessages(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleListMeetingChatMessages_ReachesRPC_DefaultLimit(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/meetings/"+meetingID+"/chat", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "meetingId", meetingID)
	routes.HandleListMeetingChatMessages(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListMeetingChatMessages_ReachesRPC_ExplicitLimit(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/meetings/"+meetingID+"/chat?limit=10", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "meetingId", meetingID)
	routes.HandleListMeetingChatMessages(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// A non-numeric or non-positive limit is silently ignored (Sscanf failure or
// parsed<=0 keeps the default of 100) — it must not turn into a 400.
func TestHandleListMeetingChatMessages_ReachesRPC_InvalidLimitIgnored(t *testing.T) {
	routes := NewVideoRoutes(registryWithService("work"), "", "")
	rec := httptest.NewRecorder()
	meetingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/meetings/"+meetingID+"/chat?limit=not-a-number", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "meetingId", meetingID)
	routes.HandleListMeetingChatMessages(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
