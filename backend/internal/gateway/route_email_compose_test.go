package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

// This file covers the send/compose handlers (HandleSendEmail, HandleSaveDraft,
// HandleReplyEmail, HandleForwardEmail) and the per-message/bulk action handlers
// (HandleMarkRead, HandleMarkUnread, HandleToggleStar, HandleMoveToFolder,
// HandleDeleteMessage, HandleBulkMessageAction) in route_email.go.
//
// The compose handlers decode into local DTOs with `validate` tags
// (decodeAndValidate), so their required-field/format errors are exercised
// via assertValidationError -- unlike the account handlers in
// route_email_accounts_test.go, which are proto-direct. HandleMarkRead/
// HandleMarkUnread/HandleToggleStar/HandleDeleteMessage take only a
// chi.URLParam id with no local UUID check, so an invalid id is documented as
// passing straight through to the (unreachable) RPC layer, same as
// TestHandleUpdateAccount_InvalidIDUUID.
//
// A note on the "Consent bei fehlendem contact_id-Wert" path from the backlog
// scope: consent enforcement for SendEmail lives entirely in
// internal/email/send/service.go (consentAsserter, exercised by
// internal/email/send/consent_test.go), not in the gateway handler --
// HandleSendEmail only forwards ContactId to the RPC when set. There is no
// gateway-level consent error path to test, the same service-layer boundary
// documented for the video meeting state machine in prior iterations.

// ============================================================================
// HandleMarkRead / HandleMarkUnread / HandleToggleStar
// ============================================================================

func TestHandleMarkRead_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/"+id+"/read", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleMarkRead(rec, req)
	assertStatus(t, rec, http.StatusBadGateway)
}

func TestHandleMarkRead_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/"+id+"/read", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleMarkRead(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleMarkUnread_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/"+id+"/unread", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleMarkUnread(rec, req)
	assertStatus(t, rec, http.StatusBadGateway)
}

func TestHandleMarkUnread_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/"+id+"/unread", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleMarkUnread(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleToggleStar_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/"+id+"/star", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleToggleStar(rec, req)
	assertStatus(t, rec, http.StatusBadGateway)
}

func TestHandleToggleStar_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/"+id+"/star", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleToggleStar(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleMoveToFolder
// ============================================================================

func TestHandleMoveToFolder_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/"+id+"/move", jsonBody(t, map[string]interface{}{
		"target_folder_id": uuid.New().String(),
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleMoveToFolder(rec, req)
	assertStatus(t, rec, http.StatusBadGateway)
}

func TestHandleMoveToFolder_InvalidJSON(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/"+id+"/move", invalidJSON())
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleMoveToFolder(rec, req)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleMoveToFolder_MissingTargetFolderID(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/"+id+"/move", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleMoveToFolder(rec, req)
	assertValidationError(t, rec, "target_folder_id")
}

func TestHandleMoveToFolder_InvalidTargetFolderIDFormat(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/"+id+"/move", jsonBody(t, map[string]interface{}{
		"target_folder_id": "not-a-uuid",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleMoveToFolder(rec, req)
	assertValidationError(t, rec, "target_folder_id")
}

// TestHandleMoveToFolder_InvalidMessageIDUUID documents the actual (missing)
// behaviour: the message id comes from chi.URLParam(r, "id") and is forwarded
// verbatim as MoveToFolderRequest.MessageId with no local UUID check, same
// pattern as TestHandleUpdateAccount_InvalidIDUUID for the account handlers.
func TestHandleMoveToFolder_InvalidMessageIDUUID(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/not-a-uuid/move", jsonBody(t, map[string]interface{}{
		"target_folder_id": uuid.New().String(),
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleMoveToFolder(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleMoveToFolder_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/"+id+"/move", jsonBody(t, map[string]interface{}{
		"target_folder_id": uuid.New().String(),
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleMoveToFolder(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleDeleteMessage
// ============================================================================

func TestHandleDeleteMessage_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/email/messages/"+id, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleDeleteMessage(rec, req)
	assertStatus(t, rec, http.StatusBadGateway)
}

// TestHandleDeleteMessage_InvalidIDUUID documents the same missing local
// UUID check as HandleMoveToFolder's message id: a non-UUID path segment is
// forwarded verbatim to DeleteMessageRequest.Id and fails only once it
// reaches the (unreachable) RPC layer, indistinguishable from a valid UUID.
func TestHandleDeleteMessage_InvalidIDUUID(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/email/messages/not-a-uuid", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteMessage(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteMessage_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/email/messages/"+id, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleDeleteMessage(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleBulkMessageAction
// ============================================================================

func TestHandleBulkMessageAction_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/bulk", jsonBody(t, map[string]interface{}{
		"ids":    []string{uuid.New().String()},
		"action": "read",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withPermissions(req, "email:write")
	routes.HandleBulkMessageAction(rec, req)
	assertStatus(t, rec, http.StatusBadGateway)
}

func TestHandleBulkMessageAction_InvalidJSON(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/bulk", invalidJSON())
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withPermissions(req, "email:write")
	routes.HandleBulkMessageAction(rec, req)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleBulkMessageAction_EmptyIDs(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/bulk", jsonBody(t, map[string]interface{}{
		"ids":    []string{},
		"action": "read",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withPermissions(req, "email:write")
	routes.HandleBulkMessageAction(rec, req)
	assertValidationError(t, rec, "ids")
}

func TestHandleBulkMessageAction_InvalidUUIDInIDs(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/bulk", jsonBody(t, map[string]interface{}{
		"ids":    []string{"not-a-uuid"},
		"action": "read",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withPermissions(req, "email:write")
	routes.HandleBulkMessageAction(rec, req)
	// dive reports the offending element's index, not the bare field name.
	assertValidationError(t, rec, "ids[0]")
}

func TestHandleBulkMessageAction_MissingAction(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/bulk", jsonBody(t, map[string]interface{}{
		"ids": []string{uuid.New().String()},
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withPermissions(req, "email:write")
	routes.HandleBulkMessageAction(rec, req)
	assertValidationError(t, rec, "action")
}

func TestHandleBulkMessageAction_InvalidTargetUUID(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/bulk", jsonBody(t, map[string]interface{}{
		"ids":    []string{uuid.New().String()},
		"action": "move",
		"target": "not-a-uuid",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withPermissions(req, "email:write")
	routes.HandleBulkMessageAction(rec, req)
	assertValidationError(t, rec, "target")
}

// TestHandleBulkMessageAction_DeleteWithoutPermission proves the second,
// explicit permission check described in the handler's doc comment: the
// route guard only requires email:write, so a bulk "delete" action must be
// rejected locally for a caller who has write but not email:delete --
// otherwise bulk delete would silently be a wider door than the single
// DELETE /{id} route (route_capability_guard_test.go's withPermissions is
// reused here, same package).
func TestHandleBulkMessageAction_DeleteWithoutPermission(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/bulk", jsonBody(t, map[string]interface{}{
		"ids":    []string{uuid.New().String()},
		"action": "delete",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withPermissions(req, "email:write")
	routes.HandleBulkMessageAction(rec, req)
	assertStatus(t, rec, http.StatusForbidden)
	assertErrorContains(t, rec, "insufficient permissions")
}

func TestHandleBulkMessageAction_DeleteWithPermission_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/bulk", jsonBody(t, map[string]interface{}{
		"ids":    []string{uuid.New().String()},
		"action": "delete",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withPermissions(req, "email:write", "email:delete")
	routes.HandleBulkMessageAction(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleBulkMessageAction_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/bulk", jsonBody(t, map[string]interface{}{
		"ids":    []string{uuid.New().String()},
		"action": "read",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withPermissions(req, "email:write")
	routes.HandleBulkMessageAction(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleSendEmail
// ============================================================================

func TestHandleSendEmail_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/send", jsonBody(t, map[string]interface{}{
		"to": []string{"recipient@example.com"},
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleSendEmail(rec, req)
	assertStatus(t, rec, http.StatusBadGateway)
}

func TestHandleSendEmail_InvalidJSON(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/send", invalidJSON())
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleSendEmail(rec, req)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleSendEmail_MissingTo(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/send", jsonBody(t, map[string]interface{}{
		"subject": "hi",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleSendEmail(rec, req)
	assertValidationError(t, rec, "to")
}

func TestHandleSendEmail_InvalidEmailInTo(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/send", jsonBody(t, map[string]interface{}{
		"to": []string{"not-an-email"},
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleSendEmail(rec, req)
	assertValidationError(t, rec, "to[0]")
}

func TestHandleSendEmail_InvalidContactID(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/send", jsonBody(t, map[string]interface{}{
		"to":         []string{"recipient@example.com"},
		"contact_id": "not-a-uuid",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleSendEmail(rec, req)
	assertValidationError(t, rec, "contact_id")
}

func TestHandleSendEmail_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/send", jsonBody(t, map[string]interface{}{
		"to": []string{"recipient@example.com"},
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleSendEmail(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleSaveDraft
// ============================================================================

func TestHandleSaveDraft_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/draft", jsonBody(t, map[string]interface{}{
		"subject": "draft",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleSaveDraft(rec, req)
	assertStatus(t, rec, http.StatusBadGateway)
}

func TestHandleSaveDraft_InvalidJSON(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/draft", invalidJSON())
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleSaveDraft(rec, req)
	assertErrorContains(t, rec, "invalid request body")
}

// TestHandleSaveDraft_InvalidEmailInTo proves that although "to" is optional
// for a draft (unlike HandleSendEmail), a non-empty entry still has to be a
// well-formed address -- the DTO's validate tag is "omitempty,dive,email".
func TestHandleSaveDraft_InvalidEmailInTo(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/draft", jsonBody(t, map[string]interface{}{
		"to": []string{"not-an-email"},
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleSaveDraft(rec, req)
	assertValidationError(t, rec, "to[0]")
}

func TestHandleSaveDraft_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/draft", jsonBody(t, map[string]interface{}{
		"subject": "draft",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleSaveDraft(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleReplyEmail
// ============================================================================

func TestHandleReplyEmail_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/reply", jsonBody(t, map[string]interface{}{
		"original_message_id": uuid.New().String(),
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleReplyEmail(rec, req)
	assertStatus(t, rec, http.StatusBadGateway)
}

func TestHandleReplyEmail_InvalidJSON(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/reply", invalidJSON())
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleReplyEmail(rec, req)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleReplyEmail_MissingOriginalMessageID(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/reply", jsonBody(t, map[string]interface{}{
		"body_text": "reply body",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleReplyEmail(rec, req)
	assertValidationError(t, rec, "original_message_id")
}

func TestHandleReplyEmail_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/reply", jsonBody(t, map[string]interface{}{
		"original_message_id": uuid.New().String(),
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleReplyEmail(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleForwardEmail
// ============================================================================

func TestHandleForwardEmail_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/forward", jsonBody(t, map[string]interface{}{
		"original_message_id": uuid.New().String(),
		"to":                  []string{"recipient@example.com"},
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleForwardEmail(rec, req)
	assertStatus(t, rec, http.StatusBadGateway)
}

func TestHandleForwardEmail_InvalidJSON(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/forward", invalidJSON())
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleForwardEmail(rec, req)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleForwardEmail_MissingOriginalMessageID(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/forward", jsonBody(t, map[string]interface{}{
		"to": []string{"recipient@example.com"},
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleForwardEmail(rec, req)
	assertValidationError(t, rec, "original_message_id")
}

func TestHandleForwardEmail_MissingTo(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/forward", jsonBody(t, map[string]interface{}{
		"original_message_id": uuid.New().String(),
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleForwardEmail(rec, req)
	assertValidationError(t, rec, "to")
}

func TestHandleForwardEmail_InvalidEmailInTo(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/forward", jsonBody(t, map[string]interface{}{
		"original_message_id": uuid.New().String(),
		"to":                  []string{"not-an-email"},
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleForwardEmail(rec, req)
	assertValidationError(t, rec, "to[0]")
}

func TestHandleForwardEmail_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages/forward", jsonBody(t, map[string]interface{}{
		"original_message_id": uuid.New().String(),
		"to":                  []string{"recipient@example.com"},
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleForwardEmail(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
